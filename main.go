package main

import (
	"crypto/sha1"
	"database/sql"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/queue"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/nlnwa/whatwg-url/canonicalizer"
	"github.com/nlnwa/whatwg-url/url"
)

const BATCH_SIZE = 1024

var db *sql.DB
var err error

type Link struct {
	Target int64
	Source int64
}

type UrlDb struct {
	Sha1     int64
	Scheme   string
	Host     string
	Pathname string
	Fragment string
}

type Settings struct {
	PostgresUri string
}

func initDatabase(uri string) {

	fmt.Println(uri)
	db, err = sql.Open("postgres", uri)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			sha1 BIGINT,
			scheme TEXT,
			host TEXT,
			pathname TEXT,
			fragment TEXT,
			PRIMARY KEY (sha1)
		);

		CREATE TABLE IF NOT EXISTS links (
			target_id BIGINT,
			source_id BIGINT,
			PRIMARY KEY (target_id, source_id),
			FOREIGN KEY (target_id) REFERENCES urls (sha1),
			FOREIGN KEY (source_id) REFERENCES urls (sha1)
		);
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

func LinkAccumulator(ch <-chan [2]url.Url) {
	var linksBatch = [BATCH_SIZE]Link{}
	var urlsBatch = [BATCH_SIZE * 2]url.Url{}
	i := 0
	for v := range ch {
		source, target := v[0], v[1]
		urlsBatch[i] = source
		urlsBatch[BATCH_SIZE+i] = target
		linksBatch[i] = Link{GetUrlHash(source), GetUrlHash(target)}
		if i == 1023 {
			BulkInsertUrls(urlsBatch)
			BulkInsertLinks(linksBatch)
			i = 0
		} else {
			i++
		}
	}
}

func BulkInsertUrls(urls [2 * BATCH_SIZE]url.Url) {
	// Start building the bulk insert statement
	var values []string
	var args []interface{}

	for i, url := range urls {
		// Generate placeholders like ($1, $2, $3, $4, $5), ($6, $7, $8, $9, $10), etc.
		values = append(values, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)", i*5+1, i*5+2, i*5+3, i*5+4, i*5+5))
		args = append(args, GetUrlHash(url), url.Scheme(), url.Host(), url.Pathname(), url.Fragment())
	}

	// Combine into a single statement
	stmt := fmt.Sprintf(
		"INSERT INTO urls (sha1, scheme, host, pathname, fragment) VALUES %s ON CONFLICT DO NOTHING",
		strings.Join(values, ","),
	)

	// Execute the statement with all arguments
	_, err := db.Exec(stmt, args...)
	if err != nil {
		log.Println(err)
	}
}

func BulkInsertLinks(links [BATCH_SIZE]Link) {
	// Start building the bulk insert statement
	var values []string
	var args []interface{}

	for i, link := range links {
		// Generate placeholders like ($1, $2), ($3, $4), etc.
		values = append(values, fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
		args = append(args, int(link.Target), int(link.Source))
	}

	// Combine into a single statement
	stmt := fmt.Sprintf(
		"INSERT INTO links (target_id, source_id) VALUES %s ON CONFLICT DO NOTHING",
		strings.Join(values, ","),
	)

	// Execute the statement with all arguments
	_, err := db.Exec(stmt, args...)
	if err != nil {
		log.Println(err)
	}
}

func NormalizeUrlString(urlRaw string) (*url.Url, error) {
	url, err := canonicalizer.GoogleSafeBrowsing.Parse(urlRaw)
	if err != nil {
		return url, err
	}

	s := url.Scheme()
	if s != "http" && s != "https" {
		return url, fmt.Errorf("url scheme is not http or https: %s", s)
	}

	p := url.Port()
	if p != "" && p != "80" && p != "443" {
		return url, fmt.Errorf("port is not 80 or 443: %s", p)
	}
	url.SetPort("")

	url.SetSearch("")
	return url, nil
}

func GetUrlHash(url url.Url) int64 {
	urlNormalized := url.Href(false)
	h := sha1.New()
	h.Write([]byte(urlNormalized))
	hBytes := h.Sum(nil)
	return int64(binary.BigEndian.Uint64(hBytes))
}

func main() {
	// Load Settings
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file:", err)
	}
	s := Settings{PostgresUri: os.Getenv("POSTGRES_URI")}

	// Init Backlink engine DB Connection
	initDatabase(s.PostgresUri)
	defer db.Close()

	// Start the MetricLogger in a goroutine
	counterRequest := make(chan struct{})
	counterError := make(chan error)
	go MetricLogger(counterRequest, counterError)

	// Start the accumulator in a goroutine
	linksAccumulator := make(chan [2]url.Url)
	go LinkAccumulator(linksAccumulator)

	// Settingsure Colly
	c := colly.NewCollector()

	// Set Colly Storage
	storage := &PostgresStorage{Db: db}
	q, err := queue.New(16, storage)
	if err != nil {
		log.Fatal("Failed to init the queue:", err)
	}

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

		source, err := NormalizeUrlString(e.Request.URL.String())
		if err != nil {
			// TODO Add metric for normalization error
			return
		}
		linksAccumulator <- [2]url.Url{*source, *target}

		q.AddURL(target.Href(false))
	})

	c.OnRequest(func(r *colly.Request) {
		counterRequest <- struct{}{}
	})

	c.OnError(func(r *colly.Response, err error) {
		msg := fmt.Errorf("%s: %s", r.Request.URL.Hostname(), err)
		counterError <- msg
	})

	// Start scraping on
	q.AddURL("https://www.bbc.com/")
	q.AddURL("https://www.theguardian.com/europe/")
	q.AddURL("https://www.liberation.fr/")

	q.Run(c)
	c.Wait()
}
