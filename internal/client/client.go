package client

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/commons"
	"golang.org/x/time/rate"
)

var retryStatus = []int{408, 425, 429, 500, 502, 503, 504}

type Fetcher interface {
	Head(url string) (resp *http.Response, err error)
	Get(url string) (resp *http.Response, err error)
}

// Wrap the default http client with crawling specific feature:
//   - per domain rate limiting
//   - automated retry
//   - custom user agent
//
// NOTE: HEAD and GET requests have different request limiters otherwise all GET requests
// will be stopped by HEAD requests (wich are queued first) instead of working in tandem.
type CrawlClient struct {
	ctx          context.Context
	client       *http.Client
	rateLimiters *sync.Map
	rateLimit    rate.Limit
	retryLimit   int
	lock         *sync.RWMutex
}

func NewCrawlClient(
	ctx context.Context,
	rateLimiter rate.Limit,
	retryLimit int,
	timeout time.Duration,
) *CrawlClient {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100, // Default: 100
		MaxIdleConnsPerHost:   2,   // Default: 2
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	c := &CrawlClient{
		ctx:          ctx,
		client:       &http.Client{Transport: transport, Timeout: timeout},
		rateLimiters: &sync.Map{},
		rateLimit:    rateLimiter,
		retryLimit:   retryLimit,
		lock:         &sync.RWMutex{},
	}
	return c
}

func (c *CrawlClient) Do(req *http.Request) (*http.Response, error) {
	v, _ := c.rateLimiters.LoadOrStore(req.Method+req.Host, rate.NewLimiter(c.rateLimit, 1))
	rateLimiter := v.(*rate.Limiter)

	t0 := time.Now()
	err := rateLimiter.Wait(c.ctx) // This is a blocking call. Honors the rate limit
	slog.Info(fmt.Sprintf("waited %s for a limit of %0.2f on %s %s ", time.Since(t0), rateLimiter.Limit(), req.Method, req.Host))
	if err != nil {
		return nil, fmt.Errorf("error while waiting for rate limit: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	// Dynamically adjust rate limit
	if resp.StatusCode == 429 {
		rateLimiter.SetLimit(rateLimiter.Limit() / 2) // Limit is a frequency so we divide,
	}

	// Automatically retry with exponential backoff based on the current rate limit
	if slices.Contains(retryStatus, resp.StatusCode) {
		for retry := 0; retry < c.retryLimit; retry++ {
			backoff := exponentialBackoff(rateLimiter.Limit(), retry)
			err = commons.Delay(c.ctx, time.Duration(backoff*float64(time.Second)))
			if err != nil {
				return nil, fmt.Errorf("error while waiting for delay between retries: %w", err)
			}

			err = rateLimiter.Wait(c.ctx)
			if err != nil {
				return nil, fmt.Errorf("error while waiting for rate limit: %w", err)
			}

			resp, err = c.client.Do(req)
			if err != nil {
				return nil, fmt.Errorf("error after %d retry: %w", retry, err)
			}

			if !slices.Contains(retryStatus, resp.StatusCode) {
				break
			}
		}
	}
	return resp, nil
}

func (c *CrawlClient) Get(url string) (resp *http.Response, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "	")
	return c.Do(req)
}

func (c *CrawlClient) Head(url string) (resp *http.Response, err error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "BacklinksBot")
	return c.Do(req)
}

// Return the duration for next retry based on an exponential of the rate limit
func exponentialBackoff(limit rate.Limit, retry int) float64 {
	// Limit is a frequency but we want the periode so we need the inverse.
	retryPeriode := 1 / float64(limit)
	return retryPeriode * math.Pow10(retry)
}
