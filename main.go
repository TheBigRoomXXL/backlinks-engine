package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
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
	ch := make(chan Link)
	go Accumulator(ch)

	// Configure Colly
	c := colly.NewCollector(
		colly.UserAgent("backlinks-engine"),
		colly.MaxBodySize(1024*1024),
		colly.CacheDir("data/colly-cache"),
		colly.Async(true),
	)
	c.WithTransport(&http.Transport{
		DisableKeepAlives: true,
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
	})
	c.Limit(&colly.LimitRule{
		Delay: 5 * time.Second,
	})

	err = c.SetStorage(&CollySQLStorage{})
	if err != nil {
		fmt.Print("here")
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
		ch <- link

		e.Request.Visit(targetNorm)
	})

	// c.OnRequest(func(r *colly.Request) {
	// 	fmt.Println("Visiting", r.URL)
	// })

	// First run seeds
	c.Visit("https://lovergne.dev")
	c.Visit("https://en.wikipedia.org/wiki/Ted_Nelson")
	c.Visit("https://www.lemonde.fr/")
	c.Visit("https://www.bbc.com/")

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
	fmt.Print("wait")
	c.Wait()
}
