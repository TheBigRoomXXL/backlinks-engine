package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
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

func MetricLogger(req <-chan struct{}, err <-chan error) {
	l := log.New(os.Stderr, "", log.Ldate|log.Ltime)
	ticker := time.NewTicker(1 * time.Second)
	requests := 0
	errors := 0
	timeouts := 0
	fmt.Println("┌───────────────┬───────────────┬───────────────┐")
	fmt.Println("│   requests    │    errors     │   timeouts    │")
	fmt.Println("├───────────────┼───────────────┼───────────────┤")
	for {
		for {
			select {
			case <-ticker.C:
				fmt.Printf(
					"│ %13d │ %13d │ %13d │\n",
					requests, errors, timeouts,
				)
			case <-req:
				requests++
			case e := <-err:
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

	// Start the Accumulator in a goroutine
	linksAccumulator := make(chan Link)
	go Accumulator(linksAccumulator)

	// Start the MetricLogger in a goroutine
	counterRequest := make(chan struct{})
	counterError := make(chan error)
	go MetricLogger(counterRequest, counterError)

	// Configure Colly
	// We only want like with http(s) and a domaine name, no direct IP.
	urlRegex := regexp.MustCompile(`^(http|https):\/\/([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}`)
	c := colly.NewCollector(
		colly.UserAgent("backlinks-engine"),
		colly.MaxBodySize(1024*1024),
		colly.Async(true),
		colly.URLFilters(urlRegex),
		// colly.CacheDir("data/colly-cache"),
	)
	c.SetRequestTimeout(
		5 * time.Second,
	)

	c.Limit(&colly.LimitRule{
		Parallelism: 2,
		Delay:       5 * time.Second,
		RandomDelay: 5 * time.Second,
	})

	err = c.SetStorage(&CollySQLStorage{})
	if err != nil {
		log.Fatal(err)
	}

	// Add Response handler
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		// Prepare target value
		targetRaw := e.Request.AbsoluteURL(e.Attr("href"))
		if targetRaw == "" {
			return
		}
		targetNorm, err := NormalizeUrlString(targetRaw)
		if err != nil {
			return
		}

		// Prepare source vlaue
		source, err := NormalizeURL(e.Request.URL)
		if err != nil {
			return
		}

		// Push link in the queue
		link := Link{
			Target: targetNorm,
			Source: source,
		}
		linksAccumulator <- link

		e.Request.Visit(targetNorm)
	})

	c.OnRequest(func(r *colly.Request) {
		counterRequest <- struct{}{}
	})

	c.OnError(func(r *colly.Response, err error) {
		msg := fmt.Errorf("%s: %s", r.Request.URL.Hostname(), err)
		counterError <- msg
	})

	// First run seeds
	c.Visit("https://lovergne.dev")
	// c.Visit("https://en.wikipedia.org/wiki/Ted_Nelson")
	// c.Visit("https://www.lemonde.fr/")
	// c.Visit("https://www.bbc.com/")
	c.Visit("https://www.theguardian.com/europe/")
	c.Visit("https://www.liberation.fr/")

	// Next run re-create queue
	rows, err := db.Query(`
		SELECT target 
		FROM links 
		WHERE target NOT IN (SELECT source FROM links) LIMIT 1000;
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var not_visited string
		err = rows.Scan(&not_visited)
		if err != nil {
			log.Fatal()
		}
		c.Visit(not_visited)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	c.Wait()
	fmt.Println("OMG! NOT MORE LINKS TO VISIT! DID WE JUST CRAWLED THE ENTIRE INTERNET?!")
}
