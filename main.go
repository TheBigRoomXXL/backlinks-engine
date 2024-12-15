package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/joho/godotenv"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/nlnwa/whatwg-url/canonicalizer"
)

type Settings struct {
	NEO4J_USER     string
	NEO4J_PASSWORD string
}

func newSettings() *Settings {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	return &Settings{
		NEO4J_USER:     os.Getenv("NEO4J_USER"),
		NEO4J_PASSWORD: os.Getenv("NEO4J_PASSWORD"),
	}

}

func newNeo4j(s *Settings) (neo4j.DriverWithContext, error) {
	uri := "neo4j://localhost:7687" // TODO: add to settings

	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(s.NEO4J_USER, s.NEO4J_PASSWORD, ""))
	if err != nil {
		return nil, err
	}

	// Test the connection by verifying the authentication
	ctx := context.Background()
	if err := driver.VerifyConnectivity(ctx); err != nil {
		driver.Close(ctx)
		return nil, fmt.Errorf("failed to verify connection: %w", err)
	}

	return driver, nil
}

func upsertPage(db neo4j.DriverWithContext, source string, targets []string) error {
	ctx := context.Background()
	session := db.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	_, err := neo4j.ExecuteWrite(ctx, session, func(tx neo4j.ManagedTransaction) (any, error) {
		// Step 1: Create the source node
		sourceQuery := "MERGE (source:Page {url: $source})"
		if _, err := tx.Run(ctx, sourceQuery, map[string]any{"source": source}); err != nil {
			return struct{}{}, err
		}

		// Step 2: Create target nodes and relationships
		targetQuery := `
			UNWIND $targets AS targetUrl
			MERGE (target:Page {url: targetUrl})
			MERGE (source:Page {url: $source})-[:LINKS_TO]->(target)
		`
		if _, err := tx.Run(ctx, targetQuery, map[string]any{"source": source, "targets": targets}); err != nil {
			return struct{}{}, err
		}

		// Step 3: Remove edges to nodes not in the targets list
		cleanupQuery := `
			MATCH (source:Page {url: $source})-[r:LINKS_TO]->(target:Page)
			WHERE NOT target.url IN $targets
			DELETE r
		`
		if _, err := tx.Run(ctx, cleanupQuery, map[string]any{"source": source, "targets": targets}); err != nil {
			return struct{}{}, err
		}

		return struct{}{}, nil
	})

	return err
}

func MetricLogger(reqChan <-chan struct{}, errChan <-chan error) {
	logFile, err := os.Create("errors.log")
	if err != nil {
		panic(err)
	}
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

func NormalizeUrlString(urlRaw string) (string, error) {
	url, err := canonicalizer.GoogleSafeBrowsing.Parse(urlRaw)
	if err != nil {
		return "", err
	}

	s := url.Scheme()
	if s != "http" && s != "https" {
		return "", fmt.Errorf("url scheme is not http or https: %s", s)
	}

	p := url.Port()
	if p != "" && p != "80" && p != "443" {
		return "", fmt.Errorf("port is not 80 or 443: %s", p)
	}
	url.SetPort("")

	url.SetSearch("")
	url.SetHash("")

	return url.Href(true), nil
}

func main() {
	// Init DB connection
	s := newSettings()
	db, err := newNeo4j(s)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close(context.Background())

	// Start the MetricLogger in a goroutine
	counterRequest := make(chan struct{})
	counterError := make(chan error)
	go MetricLogger(counterRequest, counterError)

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

		err = upsertPage(db, source, targets)
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
	c.Visit("https://www.bbc.com/")
	c.Visit("https://www.theguardian.com/europe/")
	c.Visit("https://www.liberation.fr/")

	c.Wait()
}
