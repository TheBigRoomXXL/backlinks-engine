package crawler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/client"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/commons"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/queue"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/telemetry"
	"golang.org/x/sync/errgroup"
)

type Crawler struct {
	ctx     context.Context
	queue   queue.Queue
	fetcher client.Fetcher
	group   *errgroup.Group
}

func NewCrawler(ctx context.Context, queue queue.Queue, fetcher client.Fetcher) (*Crawler, error) {
	group, ctx := errgroup.WithContext(ctx)

	return &Crawler{
		ctx:     ctx,
		group:   group,
		queue:   queue,
		fetcher: fetcher,
	}, nil
}

func (c *Crawler) AddUrl(url *url.URL) error {
	// slog.Debug(fmt.Sprintf("adding %s to queue\n", url))
	return c.queue.Add(url)
}

func (c *Crawler) Run() error {
	limit := 2048
	c.group.SetLimit(limit)
	for i := 0; i < limit; i++ {
		c.group.Go(c.crawlNextPage)
	}

	ticker := time.NewTicker(time.Millisecond * 500)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return nil
		case <-ticker.C:
			for i := 0; i < limit; i++ {
				ok := c.group.TryGo(c.crawlNextPage)
				if !ok {
					break
				}
			}
		}
	}
}

func (c *Crawler) crawlNextPage() error {
	url, err := c.queue.Next()
	if err != nil {
		return fmt.Errorf("error getting next element in queue: %w", err)
	}
	if url == nil {
		return nil
	}
	defer telemetry.ProcessedURL.Add(1)

	// TODO: Robot.txt validation

	urlStr := url.String()
	slog.Debug(fmt.Sprintf("processing %s", url))

	resp, err := c.fetcher.Head(urlStr)
	if err != nil {
		slog.Debug(fmt.Sprintf("HEAD %s failed: %s\n", urlStr, err))
		telemetry.ErrorChan <- err
		return nil
	}
	resp.Body.Close()
	slog.Debug(fmt.Sprintf("HEAD %s done\n", urlStr))

	if !isResponsesCrawlable(resp) {
		slog.Debug(fmt.Sprintf("HEAD %s response is not crawlable\n", urlStr))
		return nil
	}

	resp, err = c.fetcher.Get(urlStr)
	if err != nil {
		slog.Debug(fmt.Sprintf("Get %s failed: %s\n", urlStr, err))
		telemetry.ErrorChan <- err
		return nil
	}
	defer resp.Body.Close()
	slog.Debug(fmt.Sprintf("GET %s done\n", urlStr))

	// We double check in case the HEAD response was not representative
	if !isResponsesCrawlable(resp) {
		slog.Debug(fmt.Sprintf("GET %s response is not crawlable\n", urlStr))
		return nil
	}

	links, err := extractLinks(resp)
	if err != nil {
		slog.Debug(fmt.Sprintf("failed to extract links from %s : %s\n", urlStr, err))
		telemetry.ErrorChan <- err
		return nil
	}
	slog.Debug(fmt.Sprintf("%d links extacted from %s\n", len(links), urlStr))

	for _, link := range links {
		c.AddUrl(link)
	}

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
