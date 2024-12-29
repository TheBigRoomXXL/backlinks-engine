package internal

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

const BULK_SIZE = 512

type Link struct {
	Source  string
	Targets []string
}

type Backlink struct {
	Target  string
	Sources []string
}

func LinksAccumulator(sourcesChan <-chan Link, db driver.Conn) {
	var sources [BULK_SIZE]Link
	i := 0
	for {
		s := <-sourcesChan
		sources[i] = s
		i++
		if i >= BULK_SIZE {
			i = 0
			go LinksBulkInsert(db, sources)
		}
	}
}

func LinksBulkInsert(db driver.Conn, sources [BULK_SIZE]Link) {
	// 1. delete any existing link from the source
	var sourcesUrls [BULK_SIZE]string
	for i := 0; i < len(sources); i++ {
		sourcesUrls[i] = sources[i].Source
	}
	ctx := context.Background()
	query := `DELETE FROM links WHERE source in ?`
	err := db.Exec(ctx, query, sourcesUrls)
	if err != nil {
		counterError <- fmt.Errorf("failed to insert links: %w", err)
		return
	}

	// 2. Insert the new links
	batch, err := db.PrepareBatch(ctx, "INSERT INTO links")
	if err != nil {
		counterError <- fmt.Errorf("failed to insert links: %w", err)
		return
	}
	for i := 0; i < len(sources); i++ {
		for j := 0; j < len(sources[i].Targets); j++ {
			err := batch.Append(
				sources[i].Source,
				sources[i].Targets[j],
			)
			if err != nil {
				counterError <- fmt.Errorf("failed to insert links: %w", err)
				return
			}
		}
	}
}

func NormalizeUrlString(urlRaw string) (string, error) {
	// TODO: bring back the rules from GoogleSafeBrowsing in a performant way
	// url, err := canonicalizer.GoogleSafeBrowsing.Parse(urlRaw)
	url, err := url.Parse(urlRaw)
	if err != nil {
		return "", err
	}

	if url.Scheme != "http" && url.Scheme != "https" {
		return "", fmt.Errorf("url scheme is not http or https: %s", url.Scheme)
	}

	p := url.Port()
	if p != "" && p != "80" && p != "443" {
		return "", fmt.Errorf("port is not 80 or 443: %s", p)
	}

	url.Fragment = ""
	url.RawQuery = ""

	return url.String(), nil
}
