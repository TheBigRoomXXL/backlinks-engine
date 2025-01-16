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
	"time"

	"github.com/PuerkitoBio/goquery"
	clientpkg "github.com/TheBigRoomXXL/backlinks-engine/internal/client"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/commons"
	controllerpkg "github.com/TheBigRoomXXL/backlinks-engine/internal/controller"
	robotpkg "github.com/TheBigRoomXXL/backlinks-engine/internal/robot"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/telemetry"
)

type Crawler struct {
	ctx             context.Context
	controller      *controllerpkg.Controller
	fetcher         clientpkg.Fetcher
	robot           robotpkg.RobotPolicy
	concurencyLimit int
}

func NewCrawler(
	ctx context.Context,
	controller *controllerpkg.Controller,
	fetcher clientpkg.Fetcher,
	robot robotpkg.RobotPolicy,
	max_concurency int,
) *Crawler {

	return &Crawler{
		ctx:             ctx,
		controller:      controller,
		fetcher:         fetcher,
		robot:           robot,
		concurencyLimit: max_concurency,
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
	t1 := time.Now()
	telemetry.NextDuration.Observe(t1.Sub(t0).Seconds())

	isAllowed := !c.robot.IsAllowed(pageUrl)

	if isAllowed {
		telemetry.RobotDisallowed.Add(1)
		telemetry.RobotDuration.Observe(time.Since(t1).Seconds())
		return
	}
	telemetry.RobotAllowed.Add(1)
	t2 := time.Now()
	telemetry.RobotDuration.Observe(t2.Sub(t1).Seconds())

	pageUrlStr := pageUrl.String()
	resp, err := c.fetcher.Head(pageUrlStr)
	if err != nil {
		slog.Error(err.Error())
		telemetry.HeadDuration.Observe(time.Since(t2).Seconds())
		return
	}
	resp.Body.Close()
	t3 := time.Now()
	telemetry.HeadDuration.Observe(t3.Sub(t2).Seconds())

	if err := isResponsesCrawlable(resp); err != nil {
		slog.Warn(fmt.Sprintf("uncrawlable response from HEAD %s: %s", pageUrlStr, err))
		telemetry.IsCrawlableDuration1.Observe(time.Since(t3).Seconds())
		return
	}
	t4 := time.Now()
	telemetry.IsCrawlableDuration1.Observe(t4.Sub(t3).Seconds())

	resp, err = c.fetcher.Get(pageUrlStr)
	if err != nil {
		telemetry.GetDuration.Observe(time.Since(t4).Seconds())
		slog.Error(err.Error())
		return
	}
	defer resp.Body.Close()
	t5 := time.Now()
	telemetry.GetDuration.Observe(t5.Sub(t4).Seconds())

	// We double check in case the HEAD response was not representative
	if err := isResponsesCrawlable(resp); err != nil {
		telemetry.IsCrawlableDuration2.Observe(time.Since(t5).Seconds())
		slog.Warn(fmt.Sprintf("uncrawlable response from GET %s: %s", pageUrlStr, err))
		return
	}
	t6 := time.Now()
	telemetry.IsCrawlableDuration2.Observe(t6.Sub(t5).Seconds())

	links, err := extractLinks(resp)
	if err != nil {
		telemetry.ExtractLinksDuration.Observe(time.Since(t6).Seconds())
		slog.Error(err.Error())
		return
	}

	linkSet := make(map[string]*url.URL)
	for _, link := range links {
		linkSet[link.String()] = link
	}
	t7 := time.Now()
	telemetry.ExtractLinksDuration.Observe(t7.Sub(t6).Seconds())

	c.controller.Add(&commons.LinkGroup{
		From: pageUrl,
		To:   slices.Collect(maps.Values(linkSet)),
	})
	telemetry.AddDuration.Observe(time.Since(t7).Seconds())
	telemetry.Links.Add(int64(len(linkSet)))
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
