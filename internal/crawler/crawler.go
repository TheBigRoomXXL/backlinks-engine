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
	fetcher         *clientpkg.CrawlClient
	robot           *robotpkg.InMemoryRobotPolicy
	concurencyLimit int
}

func NewCrawler(
	ctx context.Context,
	controller *controllerpkg.Controller,
	fetcher *clientpkg.CrawlClient,
	robot *robotpkg.InMemoryRobotPolicy,
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
	ctx, span := telemetry.Tracer.Start(context.Background(), "crawlNextPage")
	defer span.End()
	defer telemetry.ProcessedURL.Add(1)

	pageUrl := c.controller.Next(ctx)

	if !c.robot.IsAllowed(ctx, pageUrl) {
		telemetry.RobotDisallowed.Add(1)
		return
	} else {
		telemetry.RobotAllowed.Add(1)
	}

	pageUrlStr := pageUrl.String()
	resp, err := c.fetcher.Head(ctx, pageUrlStr)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	resp.Body.Close()

	if err := isResponsesCrawlable(resp); err != nil {
		slog.Warn(fmt.Sprintf("uncrawlable response from HEAD %s: %s", pageUrlStr, err))
		return
	}

	resp, err = c.fetcher.Get(ctx, pageUrlStr)
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

	linkGroup, err := extractLinks(ctx, resp)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	c.controller.Add(ctx, linkGroup)

	telemetry.Links.Add(int64(len(linkGroup.To)))
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

func extractLinks(ctx context.Context, resp *http.Response) (*commons.LinkGroup, error) {
	_, span := telemetry.Tracer.Start(ctx, "extractLinks")
	defer span.End()

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

	linkSet := make(map[string]*url.URL)
	for _, link := range links {
		linkSet[link.String()] = link
	}

	return &commons.LinkGroup{
		From: resp.Request.URL,
		To:   slices.Collect(maps.Values(linkSet)),
	}, nil
}
