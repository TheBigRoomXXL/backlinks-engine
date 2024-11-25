package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/purell"
	"github.com/gocolly/colly"
	"github.com/goware/urlx"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB
var err error

type Link struct {
	Target string
	Source string
}

func initSqlite() {
	db, err = sql.Open("sqlite3", "./backlinks.db")
	if err != nil {
		log.Fatal(err)
	}
	_, err := db.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA busy_timeout = 5000;
		PRAGMA cache_size = -20000;
		PRAGMA foreign_keys = ON;
		PRAGMA temp_store = MEMORY;
	`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS links (
			id INTEGER PRIMARY KEY AUTOINCREMENT, 
			target TEXT NOT NULL, 
			source TEXT NOT NULL
		);
		CREATE UNIQUE INDEX IF NOT EXISTS links_target_source_idx ON links (target, source);
		CREATE INDEX IF NOT EXISTS target_idx ON links (target);
	`)
	if err != nil {
		log.Fatal(err)
	}
}

func MetricLogger(reqChan <-chan struct{}, errChan <-chan error) {
	logFile, err := os.Create("errors.log")
	if err != nil {
		panic(err)
	}
	l := log.New(logFile, "", log.Ldate|log.Ltime)
	ticker := time.NewTicker(1 * time.Second)
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

func Accumulator(ch <-chan Link) {
	batchSize := 1024
	var batch = make([]Link, 0, batchSize)
	for v := range ch {
		batch = append(batch, v)
		if len(batch) == batchSize { // full
			BulkInsertLinks(batch)
			batch = make([]Link, 0, batchSize) // reset
		}
	}
}

func BulkInsertLinks(links []Link) {
	// Start building the bulk insert statement
	var values []string
	var args []interface{}

	for _, link := range links {
		values = append(values, "(?, ?)")
		args = append(args, link.Target, link.Source)
	}

	// Combine into a single statement
	stmt := fmt.Sprintf(
		"INSERT  INTO links (target, source) VALUES %s ON CONFLICT DO NOTHING", strings.Join(values, ","),
	)

	// Prepare the statement
	preparedStmt, err := db.Prepare(stmt)
	if err != nil {
		log.Println(err)
	}
	defer preparedStmt.Close()

	// Execute the statement with all arguments
	_, err = preparedStmt.Exec(args...)
	if err != nil {
		log.Println(err)
	}
}

func NormalizeUrlString(urlRaw string) (string, error) {
	url, err := urlx.Parse(urlRaw)
	if err != nil {
		return "", err
	}
	return NormalizeURL(url)
}

func NormalizeURL(url *url.URL) (string, error) {
	url.RawQuery = ""
	url.User = nil
	flags := purell.FlagsSafe | purell.FlagDecodeDWORDHost | purell.FlagDecodeOctalHost | purell.FlagDecodeHexHost | purell.FlagRemoveUnnecessaryHostDots | purell.FlagRemoveEmptyPortSeparator
	return purell.NormalizeURL(url, flags), nil
}

func main() {
	initSqlite()
	defer db.Close()

	// Start the MetricLogger in a goroutine
	counterRequest := make(chan struct{})
	counterError := make(chan error)
	go MetricLogger(counterRequest, counterError)

	// Start the Accumulator in a goroutine
	linksAccumulator := make(chan Link)
	go Accumulator(linksAccumulator)

	// Configure Colly
	c := colly.NewCollector(
		colly.Async(true),
	)
	c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 8})
	c.SetRequestTimeout(5 * time.Second)

	// Handlers
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		target := e.Attr("href")

		linksAccumulator <- Link{
			Source: e.Request.URL.String(),
			Target: target,
		}
		e.Request.Visit(target)
	})

	c.OnRequest(func(r *colly.Request) {
		counterRequest <- struct{}{}
	})

	c.OnError(func(r *colly.Response, err error) {
		msg := fmt.Errorf("%s: %s", r.Request.URL.Hostname(), err)
		counterError <- msg
	})

	// Start scraping on
	// c.Visit("https://www.lemonde.fr/")
	c.Visit("https://www.bbc.com/")
	c.Visit("https://www.theguardian.com/europe/")
	c.Visit("https://www.liberation.fr/")

	c.Wait()
}
