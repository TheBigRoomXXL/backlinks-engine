package client

import (
	"context"
	"fmt"
	"math"
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
	ctx              context.Context
	client           *http.Client
	headRatelimiters map[string]*rate.Limiter
	getRatelimiters  map[string]*rate.Limiter
	rateLimit        rate.Limit
	retryLimit       int
	lock             *sync.RWMutex
}

func NewCrawlClient(
	ctx context.Context,
	transport http.RoundTripper,
	rateLimiter rate.Limit,
	retryLimit int,
	timeout time.Duration,
) *CrawlClient {
	c := &CrawlClient{
		ctx:              ctx,
		client:           &http.Client{Transport: transport, Timeout: timeout},
		rateLimit:        rateLimiter,
		headRatelimiters: make(map[string]*rate.Limiter, 1024),
		getRatelimiters:  make(map[string]*rate.Limiter, 1024),
		retryLimit:       retryLimit,
		lock:             &sync.RWMutex{},
	}
	return c
}

func (c *CrawlClient) Do(req *http.Request) (*http.Response, error) {
	rateLimiter := c.getRateLimiter(req.Method, req.Host)
	err := rateLimiter.Wait(c.ctx) // This is a blocking call. Honors the rate limit
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
	req.Header.Set("User-Agent", "BacklinksBot")
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *CrawlClient) Head(url string) (resp *http.Response, err error) {
	req, err := http.NewRequest("HEAD", url, nil)
	req.Header.Set("User-Agent", "BacklinksBot")
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Return the matching ratelimiter in a concurency-safe way.
// Return nil if bad method is given. Method must be uppercase.
func (c *CrawlClient) getRateLimiter(method string, hostname string) *rate.Limiter {
	var rateLimiter *rate.Limiter
	var ok bool
	c.lock.RLock()
	switch method {
	case "GET":
		rateLimiter, ok = c.getRatelimiters[hostname]
	case "HEAD":
		rateLimiter, ok = c.headRatelimiters[hostname]
	default:
		return nil
	}
	c.lock.RUnlock()
	if !ok {
		c.lock.Lock()
		if method == "HEAD" {
			rateLimiter = rate.NewLimiter(c.rateLimit, 1)
			c.headRatelimiters[hostname] = rateLimiter
		} else {
			rateLimiter = rate.NewLimiter(c.rateLimit, 1)
			c.getRatelimiters[hostname] = rateLimiter
		}
		c.lock.Unlock()
	}
	return rateLimiter
}

// Return the duration for next retry based on an exponential of the rate limit
func exponentialBackoff(limit rate.Limit, retry int) float64 {
	// Limit is a frequency but we want the periode so we need the inverse.
	retryPeriode := 1 / float64(limit)
	return retryPeriode * math.Pow10(retry)
}
