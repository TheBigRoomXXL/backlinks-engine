package crawler

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	clientpkg "github.com/TheBigRoomXXL/backlinks-engine/internal/client"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/commons"
	controllerpkg "github.com/TheBigRoomXXL/backlinks-engine/internal/controller"
	robotpkg "github.com/TheBigRoomXXL/backlinks-engine/internal/robot"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/telemetry"
	"golang.org/x/time/rate"
)

type Crawler struct {
	ctx             context.Context
	controller      *controllerpkg.Controller
	fetcher         clientpkg.Fetcher
	robot           robotpkg.RobotPolicy
	concurencyLimit int
	rateLimiters    *sync.Map
	rateLimit       rate.Limit
}

func NewCrawler(
	ctx context.Context,
	controller *controllerpkg.Controller,
	fetcher clientpkg.Fetcher,
	robot robotpkg.RobotPolicy,
	max_concurency int,
	rateLimit rate.Limit,
) *Crawler {

	return &Crawler{
		ctx:             ctx,
		controller:      controller,
		fetcher:         fetcher,
		robot:           robot,
		concurencyLimit: max_concurency,
		rateLimiters:    &sync.Map{},
		rateLimit:       rateLimit,
	}
}

func (c *Crawler) Seed(seeds []*url.URL) {
	c.controller.Seed(seeds)
}

func (c *Crawler) Run() error {
	for i := 0; i < c.concurencyLimit; i++ {
		go c.crawlPages()
	}

	<-c.ctx.Done()
	return nil
}

func (c *Crawler) crawlPages() error {
	for {
		select {
		case <-c.ctx.Done():
			return nil
		default:
			c.crawlNextPage()
		}

	}
}

func (c *Crawler) crawlNextPage() {
	t0 := time.Now()
	defer func() { telemetry.PageProcessDuration.Observe(time.Since(t0).Seconds()) }()
	defer telemetry.ProcessedURL.Add(1)

	pageUrl := c.controller.Next()

	isAllowed := c.robot.IsAllowed(pageUrl)

	if !isAllowed {
		return
	}

	err := c.WaitForRateLimit("HEAD", pageUrl.Host)
	if err != nil {
		slog.Error(fmt.Sprintf("error while waiting for rate limit: %s", err))
		return
	}

	pageUrlStr := pageUrl.String()
	resp, err := c.fetcher.Head(pageUrlStr)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	resp.Body.Close()

	if resp.StatusCode == 429 {
		c.IncreaseRateLimit(pageUrl.Host)
	}

	if err := isResponsesCrawlable(resp); err != nil {
		slog.Warn(fmt.Sprintf("uncrawlable response from HEAD %s: %s", pageUrlStr, err))
		return
	}

	resp, err = c.fetcher.Get(pageUrlStr)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	defer resp.Body.Close()

	// We double check in case the HEAD response was not representative
	if err := isResponsesCrawlable(resp); err != nil {
		slog.Warn(fmt.Sprintf("uncrawlable response from GET %s: %s", pageUrlStr, err))
		return
	}

	links, err := extractLinks(resp)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	linkSet := make(map[string]*url.URL)
	for _, link := range links {
		linkSet[link.String()] = link
	}

	c.controller.Add(&commons.LinkGroup{
		From: pageUrl,
		To:   slices.Collect(maps.Values(linkSet)),
	})
}

func (c *Crawler) WaitForRateLimit(method string, host string) error {
	v, _ := c.rateLimiters.LoadOrStore(method+"-"+host, rate.NewLimiter(c.rateLimit, 1))
	rateLimiter := v.(*rate.Limiter)

	return rateLimiter.Wait(c.ctx)
}

func (c *Crawler) IncreaseRateLimit(host string) {
	v, _ := c.rateLimiters.LoadOrStore("HEAD-"+host, rate.NewLimiter(c.rateLimit, 1))
	rateLimiter := v.(*rate.Limiter)
	rateLimiter.SetLimit(rateLimiter.Limit() / 2) // Limit is a frequency so we divide

	v, _ = c.rateLimiters.LoadOrStore("GET-"+host, rate.NewLimiter(c.rateLimit, 1))
	rateLimiter = v.(*rate.Limiter)
	rateLimiter.SetLimit(rateLimiter.Limit() / 2) // Limit is a frequency so we divide
}

func isResponsesCrawlable(resp *http.Response) error {
	if resp.StatusCode < 200 || resp.StatusCode > 299 || resp.StatusCode == 204 {
		return fmt.Errorf("resp %s has bad status %d", resp.Request.URL, resp.StatusCode)
	}

	contentType := resp.Header.Get("content-type")
	if !strings.Contains(contentType, "html") {
		return fmt.Errorf("resp %s has bad content-type %s", resp.Request.URL, resp.Header.Get("content-type"))
	}

	robotsTags := resp.Header.Get("x-robots-tag")
	if strings.Contains(robotsTags, "nofollow") || strings.Contains(robotsTags, "noindex") {
		return fmt.Errorf("resp %s has robotag %s", resp.Request.URL, robotsTags)
	}
	return nil
}

func extractLinks(resp *http.Response) ([]*url.URL, error) {
	links := make([]*url.URL, 0)
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the HTML document: %s", err)
	}

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		linkRelative, exist := s.Attr("href")
		if !exist {
			return
		}

		link, err := resp.Request.URL.Parse(linkRelative)
		if err != nil {
			slog.Warn(fmt.Sprintf("failed to parse url: %s", err))
			return
		}

		linkNormalized, err := commons.NormalizeUrl(link)
		if err != nil {
			return
		}

		links = append(links, linkNormalized)
	})

	return links, nil
}
