package client

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

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
	ctx    context.Context
	client *http.Client
}

func NewCrawlClient(
	ctx context.Context,
	timeout time.Duration,
) *CrawlClient {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          2000,
		MaxIdleConnsPerHost:   1,
		IdleConnTimeout:       10 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	http_client := &http.Client{Transport: transport, Timeout: timeout}

	inFlightGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "backlinkbot_client_requests_in_flight",
		Help: "A gauge of in-flight requests for the wrapped client.",
	})

	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "backlinkbot_client_requests_total",
			Help: "A counter for requests from the wrapped client.",
		},
		[]string{"code", "method"},
	)

	// dnsLatencyVec uses custom buckets based on expected dns durations.
	// It has an instance label "event", which is set in the
	// DNSStart and DNSDonehook functions defined in the
	// InstrumentTrace struct below.
	dnsLatencyVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "backlinkbot_client_dns_duration_seconds",
			Help:    "Trace dns latency histogram.",
			Buckets: []float64{.005, .01, .025, .05},
		},
		[]string{"event"},
	)

	// tlsLatencyVec uses custom buckets based on expected tls durations.
	// It has an instance label "event", which is set in the
	// TLSHandshakeStart and TLSHandshakeDone hook functions defined in the
	// InstrumentTrace struct below.
	tlsLatencyVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "backlinkbot_client_tls_duration_seconds",
			Help:    "Trace tls latency histogram.",
			Buckets: []float64{.05, .1, .25, .5},
		},
		[]string{"event"},
	)

	// histVec has no labels, making it a zero-dimensional ObserverVec.
	histVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_duration_seconds",
			Help:    "A histogram of request latencies.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{},
	)

	// Register all of the metrics in the standard registry.
	prometheus.MustRegister(counter, tlsLatencyVec, dnsLatencyVec, histVec, inFlightGauge)

	// Define functions for the available httptrace.ClientTrace hook
	// functions that we want to instrument.
	trace := &promhttp.InstrumentTrace{
		DNSStart: func(t float64) {
			dnsLatencyVec.WithLabelValues("dns_start").Observe(t)
		},
		DNSDone: func(t float64) {
			dnsLatencyVec.WithLabelValues("dns_done").Observe(t)
		},
		TLSHandshakeStart: func(t float64) {
			tlsLatencyVec.WithLabelValues("tls_handshake_start").Observe(t)
		},
		TLSHandshakeDone: func(t float64) {
			tlsLatencyVec.WithLabelValues("tls_handshake_done").Observe(t)
		},
	}

	// Wrap the default RoundTripper with middleware.
	roundTripper := promhttp.InstrumentRoundTripperInFlight(inFlightGauge,
		promhttp.InstrumentRoundTripperCounter(counter,
			promhttp.InstrumentRoundTripperTrace(trace,
				promhttp.InstrumentRoundTripperDuration(histVec, http.DefaultTransport),
			),
		),
	)

	// Set the RoundTripper on our client.
	http_client.Transport = roundTripper

	return &CrawlClient{
		ctx:    ctx,
		client: http_client,
	}
}

func (c *CrawlClient) Do(req *http.Request) (*http.Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
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
