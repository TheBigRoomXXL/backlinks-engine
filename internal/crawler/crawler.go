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
	"golang.org/x/sync/errgroup"
)

type Crawler struct {
	ctx             context.Context
	controller      *controllerpkg.Controller
	fetcher         clientpkg.Fetcher
	robot           robotpkg.RobotPolicy
	group           *errgroup.Group
	concurencyLimit int
}

func NewCrawler(
	ctx context.Context,
	controller *controllerpkg.Controller,
	fetcher clientpkg.Fetcher,
	robot robotpkg.RobotPolicy,
	max_concurency int,
) *Crawler {
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(max_concurency)

	return &Crawler{
		ctx:             ctx,
		controller:      controller,
		group:           group,
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
		c.group.Go(c.crawlNextPage)
	}

	ticker := time.NewTicker(time.Millisecond * 500)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return nil
		case <-ticker.C:
			for i := 0; i < c.concurencyLimit; i++ {
				ok := c.group.TryGo(c.crawlNextPage)
				if !ok {
					break
				}
			}
		}
	}
}

func (c *Crawler) crawlNextPage() error {
	pageUrl := c.controller.Next()
	defer telemetry.ProcessedURL.Add(1)

	if !c.robot.IsAllowed(pageUrl) {
		telemetry.RobotDisallowed.Add(1)
		return nil
	} else {
		telemetry.RobotAllowed.Add(1)
	}

	pageUrlStr := pageUrl.String()
	resp, err := c.fetcher.Head(pageUrlStr)
	if err != nil {
		slog.Debug(fmt.Sprintf("HEAD %s failed: %s\n", pageUrlStr, err))
		telemetry.ErrorChan <- err
		return nil
	}
	resp.Body.Close()

	if !isResponsesCrawlable(resp) {
		slog.Debug(fmt.Sprintf("HEAD %s response is not crawlable\n", pageUrlStr))
		return nil
	}

	resp, err = c.fetcher.Get(pageUrlStr)
	if err != nil {
		slog.Debug(fmt.Sprintf("Get %s failed: %s\n", pageUrlStr, err))
		telemetry.ErrorChan <- err
		return nil
	}
	defer resp.Body.Close()

	// We double check in case the HEAD response was not representative
	if !isResponsesCrawlable(resp) {
		slog.Debug(fmt.Sprintf("GET %s response is not crawlable\n", pageUrlStr))
		return nil
	}

	links, err := extractLinks(resp)
	if err != nil {
		slog.Debug(fmt.Sprintf("failed to extract links from %s : %s\n", pageUrlStr, err))
		telemetry.ErrorChan <- err
		return nil
	}

	linkSet := make(map[string]*url.URL)
	for _, link := range links {
		linkSet[link.String()] = link
	}

	c.controller.Add(&commons.LinkGroup{
		From: pageUrl,
		To:   slices.Collect(maps.Values(linkSet)),
	})

	telemetry.Links.Add(int64(len(linkSet)))
	return nil
}

func isResponsesCrawlable(resp *http.Response) bool {
	if resp.StatusCode < 200 || resp.StatusCode > 299 || resp.StatusCode == 204 {
		slog.Debug(fmt.Sprintf("resp %s has bad status: %d\n", resp.Request.URL, resp.StatusCode))
		return false
	}

	contentType := resp.Header.Get("content-type")
	if !strings.Contains(contentType, "html") {
		slog.Debug(fmt.Sprintf("resp %s has bad content-type: %s\n", resp.Request.URL, resp.Header.Get("content-type")))
		return false
	}

	robotsTags := resp.Header.Get("x-robots-tag")
	if strings.Contains(robotsTags, "nofollow") || strings.Contains(robotsTags, "noindex") {
		slog.Debug(fmt.Sprintf("resp %s has bad robotag: %s\n", resp.Request.URL, robotsTags))
		return false
	}
	return true
}

func extractLinks(resp *http.Response) ([]*url.URL, error) {
	links := make([]*url.URL, 0)
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the HTML document: %w", err)
	}

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		linkRelative, exist := s.Attr("href")
		if !exist {
			return
		}

		link, err := resp.Request.URL.Parse(linkRelative)
		if err != nil {
			return
		}

		link, err = commons.NormalizeUrl(link)
		if err != nil {
			return
		}

		links = append(links, link)
	})

	return links, nil
}
