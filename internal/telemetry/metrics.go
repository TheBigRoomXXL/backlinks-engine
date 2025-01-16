package telemetry

import (
	"context"
	"expvar"
	"fmt"
	_ "net/http/pprof"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	ProcessedURL    = expvar.NewInt("PocessedURL")
	Errors          = expvar.NewInt("Errors")
	Warnings        = expvar.NewInt("Warnings")
	QueueSize       = expvar.NewInt("QueueSize")
	RobotAllowed    = expvar.NewInt("RobotAllowed")
	RobotDisallowed = expvar.NewInt("RobotDisallowed")
	Links           = expvar.NewInt("Links")

	PageProcessDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "backlinkbot",
			Name:      "page_processing_duration_second",
			Help:      "How many second it takes to fully process a page",
		},
	)
	NextDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "backlinkbot",
			Name:      "next_duration_second",
			Help:      "How many second it takes to get the next page to visit",
		},
	)
	RobotDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "backlinkbot",
			Name:      "robot_duration_second",
			Help:      "How many second it takes to check is a robot policy allow visit",
		},
	)
	HeadDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "backlinkbot",
			Name:      "head_duration_second",
			Help:      "How many second it takes to make the HEAD request",
		},
	)
	IsCrawlableDuration1 = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "backlinkbot",
			Name:      "is_crawlable_1_duration_second",
			Help:      "How many second it takes to determined if it's a crawlable response",
		},
	)
	GetDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "backlinkbot",
			Name:      "get_duration_second",
			Help:      "How many second it takes to make the GET request",
		},
	)
	IsCrawlableDuration2 = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "backlinkbot",
			Name:      "is_crawlable_2_duration_second",
			Help:      "How many second it takes to determined if it's a crawlable response",
		},
	)
	ExtractLinksDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "backlinkbot",
			Name:      "extract_links_duration_second",
			Help:      "How many second it takes to extract links from a response body",
		},
	)
	AddDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "backlinkbot",
			Name:      "add_duration_second",
			Help:      "How many second it takes to save links",
		},
	)
)

func init() {
	prometheus.MustRegister(PageProcessDuration)
	prometheus.MustRegister(NextDuration)
	prometheus.MustRegister(RobotDuration)
	prometheus.MustRegister(HeadDuration)
	prometheus.MustRegister(IsCrawlableDuration1)
	prometheus.MustRegister(GetDuration)
	prometheus.MustRegister(IsCrawlableDuration2)
	prometheus.MustRegister(ExtractLinksDuration)
	prometheus.MustRegister(AddDuration)
}

func MetricsReport(ctx context.Context) {
	start := time.Now()
	fmt.Println("┌───────────────┬───────────────┬───────────────┬───────────────┬───────────────┬───────────────┐")
	fmt.Println("│     Time      │   processed   │    errors     │   warnings    │  queue size   │  link pairs   │")
	fmt.Println("├───────────────┼───────────────┼───────────────┼───────────────┼───────────────┼───────────────┤")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			uptime := time.Since(start).Round(time.Second)
			fmt.Printf(
				"\n│ %13s │ %13d │ %13d │ %13d │ %13d │ %13d │\n",
				uptime,
				ProcessedURL.Value(),
				Errors.Value(),
				Warnings.Value(),
				QueueSize.Value(),
				Links.Value(),
			)
			fmt.Println("\r└───────────────┴───────────────┴───────────────┴───────────────┴───────────────┴───────────────┘")
			return
		case <-ticker.C:
			uptime := time.Since(start).Round(time.Second)
			fmt.Printf(
				"\r│ %13s │ %13d │ %13d │ %13d │ %13d │ %13d │",
				uptime,
				ProcessedURL.Value(),
				Errors.Value(),
				Warnings.Value(),
				QueueSize.Value(),
				Links.Value(),
			)
			if int(uptime.Seconds())%30 == 0 {
				fmt.Print("\n")
			}
		}
	}
}
