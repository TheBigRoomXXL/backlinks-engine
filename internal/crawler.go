package internal

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func MetricLogger(reqChan <-chan struct{}, errChan <-chan error, logFile *os.File) {
	l := log.New(logFile, "", log.Ldate|log.Ltime)
	ticker := time.NewTicker(10 * time.Second)
	start := time.Now()
	requests := 0
	errors := 0
	timeouts := 0
	fmt.Println("┌───────────────┬───────────────┬───────────────┬───────────────┐")
	fmt.Println("│     Time      │   requests    │    errors     │   timeouts    │")
	fmt.Println("├───────────────┼───────────────┼───────────────┼───────────────┤")
	for {
		for {
			select {
			case <-ticker.C:
				time := time.Since(start).Round(time.Second)
				fmt.Printf(
					"│ %13s │ %13d │ %13d │ %13d │\n",
					time, requests, errors, timeouts,
				)
			case <-reqChan:
				requests++
			case e := <-errChan:
				errors++
				l.Println(e.Error())
				if strings.Contains(strings.ToLower(e.Error()), "timeout") {
					timeouts++
				}
			}
		}
	}
}

func Crawl(s *Settings, db neo4j.DriverWithContext, seeds []string) {
	// Start the MetricLogger in a goroutine
	counterRequest := make(chan struct{})
	counterError := make(chan error)
	logFile, err := os.Create(s.LOG_PATH)
	if err != nil {
		log.Fatal(err)
	}
	go MetricLogger(counterRequest, counterError, logFile)

	// Settingsure Colly
	c := colly.NewCollector(
		colly.Async(true),
	)
	c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 8})
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
			// TODO Add metric for normalization error
			return
		}

		err = PutPage(db, source, targets)
		if err != nil {
			counterError <- err
		}
	})

	c.OnRequest(func(r *colly.Request) {
		counterRequest <- struct{}{}
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
