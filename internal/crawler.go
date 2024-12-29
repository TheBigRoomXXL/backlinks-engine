package internal

import (
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/gocolly/colly/v2"
)

func Crawl(s *Settings, db driver.Conn, seeds []string) {
	// Start the metrics logger
	initLogger(s)

	// Start the link acculator in goroutine
	sourcesChan := make(chan Link)
	go LinksAccumulator(sourcesChan, db)

	// Setup Colly
	c := colly.NewCollector(
		colly.Async(true),
	)
	c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: s.MAX_PARALLELISM})
	c.SetRequestTimeout(5 * time.Second)

	// Handlers
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		targetAbsolute := e.Request.AbsoluteURL(e.Attr("href"))
		if targetAbsolute == "" {
			return
		}

		target, err := NormalizeUrlString(targetAbsolute)
		if err != nil {
			// TODO Add metric for normalization error
			return
		}

		e.Response.Ctx.Put(target, "target")
	})

	c.OnScraped(func(r *colly.Response) {
		counterRequest <- struct{}{}
		var targets []string
		r.Ctx.ForEach(func(key string, value interface{}) interface{} {
			if value == "target" {
				targets = append(targets, key)
			}
			return nil
		})

		// Continue to Crawl
		for _, target := range targets {
			r.Request.Visit(target)
		}

		source, err := NormalizeUrlString(r.Request.URL.String())
		if err != nil {
			counterError <- fmt.Errorf("failed to normalized link '%s' from page %s: %w", r.Request.URL.String(), source, err)
			return
		}

		sourcesChan <- Link{source, targets}
	})

	c.OnError(func(r *colly.Response, err error) {
		msg := fmt.Errorf("%s: %s", r.Request.URL.Hostname(), err)
		counterError <- msg
	})

	// Start scraping on
	for _, seed := range seeds {
		c.Visit(seed)
	}

	c.Wait()

	fmt.Println("└───────────────┴───────────────┴───────────────┴───────────────┘")
}
