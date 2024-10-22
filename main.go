package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gocolly/colly"
	"github.com/goware/urlx"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB
var err error

type Link struct {
	Target_normalized string
	Target            string
	Source            float32
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
			target_normalized TEXT NOT NULL,
			target TEXT NOT NULL, 
			source TEXT NOT NULL
		)
	`)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	initSqlite()
	defer db.Close()

	// Configure Colly
	c := colly.NewCollector(
		colly.UserAgent("backlinks-engine"),
		colly.MaxBodySize(1024*1024),
		colly.CacheDir(".colly-cache"),
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

	// Add Response handler
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		targetRaw := e.Request.AbsoluteURL(e.Attr("href"))
		if targetRaw == "" {
			return
		}

		targetNorm, err := urlx.NormalizeString(targetRaw)
		if err != nil {
			return
		}

		_, err = db.ExecContext(
			context.Background(),
			`INSERT INTO links (target_normalized, target, source) VALUES (?,?,?);`,
			targetNorm, targetRaw, e.Request.URL.String(),
		)
		if err != nil {
			log.Println("Error", err)
		}
		e.Request.Visit(targetNorm)
	})

	// Run
	c.Visit("https://lovergne.dev")
	c.Visit("https://en.wikipedia.org/wiki/Ted_Nelson")
	c.Wait()
}
