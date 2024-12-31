package internal

import (
	"fmt"
	"log"

	"net/http"
	_ "net/http/pprof"

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
	// Start the pprof server
	log.Println("Starting the pprof server of port ", s.PPROF_PORT)
	go func() {
		log.Println(http.ListenAndServe("localhost:"+s.PPROF_PORT, nil))
	}()

	// Start the metrics logger
	initLogger(s)

	// Start the link acculator in goroutine
	sourcesChan := make(chan Link)
	go LinksAccumulator(sourcesChan, db)

	// Setup Colly
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs:          seeds,
		ParseFunc:          HTMLHandler,
		ErrorFunc:          ErrorHandler,
		LogDisabled:        true,
		ConcurrentRequests: 2048,
	}).Start()

	fmt.Println("└───────────────┴───────────────┴───────────────┴───────────────┘")
}
