package client

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"slices"
	"time"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/commons"
	"golang.org/x/time/rate"
)

var retryStatus = []int{408, 425, 429, 500, 502, 503, 504}

type Fetcher interface {
	Head(url string) (resp *http.Response, err error)
	Get(url string) (resp *http.Response, err error)
}

// Wrap the default http client with per domain rate limit and automated retry
type CrawlClient struct {
	ctx          context.Context
	client       *http.Client
	ratelimiters map[string]*rate.Limiter
	rateLimit    rate.Limit
	retryLimit   int
	timeout      time.Duration
}

func NewCrawlClient(
	ctx context.Context,
	transport http.RoundTripper,
	rateLimiter rate.Limit,
	retryLimit int,
	timeout time.Duration,
) *CrawlClient {
	c := &CrawlClient{
		ctx:          ctx,
		client:       &http.Client{Transport: transport},
		rateLimit:    rateLimiter,
		ratelimiters: make(map[string]*rate.Limiter, 1024),
		retryLimit:   retryLimit,
		timeout:      timeout,
	}
	return c
}

func (c *CrawlClient) Do(req *http.Request) (*http.Response, error) {
	rateLimiter, ok := c.ratelimiters[req.URL.Host]
	if !ok {
		c.ratelimiters[req.URL.Host] = rate.NewLimiter(c.rateLimit, 1)
		rateLimiter = c.ratelimiters[req.URL.Host]
	}
	err := rateLimiter.Wait(c.ctx) // This is a blocking call. Honors the rate limit
	if err != nil {
		return nil, fmt.Errorf("error while waiting for rate limit: %w", err)
	}

	resp, err := c.do(req)
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

			resp, err = c.do(req)
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

// Add the context to every request to have timeout and cancelation support
func (c *CrawlClient) do(req *http.Request) (*http.Response, error) {
	ctx, _ := context.WithTimeout(c.ctx, c.timeout)
	req = req.WithContext(ctx)
	return c.client.Do(req)
}

func (c *CrawlClient) Get(url string) (resp *http.Response, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *CrawlClient) Head(url string) (resp *http.Response, err error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Return the duration for next retry based on an exponential of the rate limit
func exponentialBackoff(limit rate.Limit, retry int) float64 {
	// Limit is a frequency but we want the periode so we need the inverse.
	retryPeriode := 1 / float64(limit)
	return retryPeriode * math.Pow10(retry)
}
