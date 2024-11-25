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
	"github.com/gocolly/colly/queue"
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
		colly.URLFilters(urlRegex),
		colly.Async(true),
		// colly.CacheDir("data/colly-cache"),
	)
	c.Limit(&colly.LimitRule{
		Parallelism: 1,
		Delay:       5 * time.Second,
		RandomDelay: 5 * time.Second,
	})
	c.SetRequestTimeout(
		5 * time.Second,
	)

	queue, _ := queue.New(
		1, // Number of consumer threads
		&queue.InMemoryQueueStorage{MaxSize: 1024 * 1024}, // Use default queue storage
	)

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
	// c.Visit("https://lovergne.dev")
	// c.Visit("https://en.wikipedia.org/wiki/Ted_Nelson")
	// c.Visit("https://www.lemonde.fr/")
	// c.Visit("https://www.bbc.com/")
	// c.Visit("https://www.theguardian.com/europe/")
	// c.Visit("https://www.liberation.fr/")
	queue.AddURL("https://lovergne.dev")
	queue.AddURL("https://en.wikipedia.org/wiki/Ted_Nelson")
	queue.AddURL("https://www.lemonde.fr/")
	queue.AddURL("https://www.bbc.com/")
	queue.AddURL("https://www.theguardian.com/europe/")
	queue.AddURL("https://www.liberation.fr/")
	// queue.AddURL("https://lovergne.dev/rss")
	// queue.AddURL("https://htmx.org/atom")
	// queue.AddURL("https://www.bitecode.dev")
	// queue.AddURL("https://rednafi.com/index")
	// queue.AddURL("https://blog.danslimmon.com")
	// queue.AddURL("https://joshcollinsworth.com/api")
	// queue.AddURL("http://feeds.feedburner.com")
	// queue.AddURL("https://martinfowler.com/feed")
	// queue.AddURL("https://notes.eatonphil.com")
	// queue.AddURL("https://feeds.feedburner.com")
	// queue.AddURL("https://research.swtch.com")
	// queue.AddURL("https://sirupsen.com/atom")
	// queue.AddURL("https://bitbashing.io/feed")
	// queue.AddURL("https://andy-bell.co")
	// queue.AddURL("https://words.filippo.io")
	// queue.AddURL("http://len.falken.directory")
	// queue.AddURL("https://lukeplant.me.uk")
	// queue.AddURL("https://wizardzines.com/index")
	// queue.AddURL("https://sethmlarson.dev/rss")
	// queue.AddURL("https://www.petemillspaugh.com")
	// queue.AddURL("https://jakelazaroff.com/rss")
	// queue.AddURL("https://digest.browsertech.com")
	// queue.AddURL("https://www.htmhell.dev")
	// queue.AddURL("https://daniel.do/rss")
	// queue.AddURL("https://buttondown.email/hillelwayne")
	// queue.AddURL("https://cliffle.com/rss")
	// queue.AddURL("https://journal.stuffwithstuff.com")
	// queue.AddURL("https://ferd.ca/feed")
	// queue.AddURL("https://tonsky.me/atom")
	// queue.AddURL("https://chriscoyier.net/feed")
	// queue.AddURL("https://hamvocke.com/feed")
	// queue.AddURL("https://developer.mozilla.org")
	// queue.AddURL("https://safjan.com/feeds")
	// queue.AddURL("https://feeds.feedburner.com")
	// queue.AddURL("https://brooker.co.za")
	// queue.AddURL("https://ferd.ca/feed")
	// queue.AddURL("https://blog.google/threat")
	// queue.AddURL("https://solar.lowtechmagazine.com")
	// queue.AddURL("https://kerkour.com/feed")
	// queue.AddURL("https://neopythonic.blogspot.com")
	// queue.AddURL("https://feeds.feedburner.com")
	// queue.AddURL("https://blog.codingconfessions.com")
	// queue.AddURL("https://j3s.sh/feed")
	// queue.AddURL("https://blog.isosceles.com")
	// queue.AddURL("https://dave.cheney.net")
	// queue.AddURL("https://cargocollective.com/rss")
	// queue.AddURL("https://www.somethingsimilar.com")
	// queue.AddURL("https://ploum.net/atom_fr")
	// queue.AddURL("https://wickstrom.tech/feed")
	// queue.AddURL("https://research.swtch.com")
	// queue.AddURL("https://samcurry.net/api")

	queue.Run(c)
	c.Wait()
	fmt.Println("OMG! NOT MORE LINKS TO VISIT! DID WE JUST CRAWLED THE ENTIRE INTERNET?!")
}
