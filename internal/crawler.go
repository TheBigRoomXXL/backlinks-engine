package internal

import (
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/PuerkitoBio/goquery"
	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/client"
)

func HTMLHandler(g *geziyor.Geziyor, r *client.Response) {
	counterRequest <- struct{}{}
	r.HTMLDoc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exist := s.Attr("href")
		if !exist {
			counterError <- fmt.Errorf("no link in <a> element")
			return
		}

		targetAbsolute, err := r.Request.URL.Parse(href)
		if err != nil {
			counterError <- fmt.Errorf("failed to parse absolute link %s", href)
			return
		}

		target, err := NormalizeUrlString(targetAbsolute.String())
		if err != nil {
			counterError <- fmt.Errorf("failed to normalize link %s", target)
			return
		}
		g.Get(target, HTMLHandler)
	})
}

func ErrorHandler(g *geziyor.Geziyor, r *client.Request, err error) {
	counterError <- fmt.Errorf("geziyor error: %w", err)
}

func Crawl(s *Settings, db driver.Conn, seeds []string) {
	// Start the metrics logger
	initLogger(s)

	// Start the link acculator in goroutine
	sourcesChan := make(chan Link)
	go LinksAccumulator(sourcesChan, db)

	// Setup Colly
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs:   seeds,
		ParseFunc:   HTMLHandler,
		LogDisabled: true,
	}).Start()

	fmt.Println("└───────────────┴───────────────┴───────────────┴───────────────┘")
}
