package internal

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/nlnwa/whatwg-url/canonicalizer"
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
			err := LinksBulkInsert(db, sources)
			if err != nil {
				// TODO: do some real error handling
				fmt.Println(err)
			}
		}
	}
}

func LinksBulkInsert(db driver.Conn, sources [BULK_SIZE]Link) error {
	// 1. delete any existing link from the source
	var sourcesUrls [BULK_SIZE]string
	for i := 0; i < len(sources); i++ {
		sourcesUrls[i] = sources[i].Source
	}
	ctx := context.Background()
	query := `DELETE FROM links WHERE source in ?`
	err := db.Exec(ctx, query, sourcesUrls)
	if err != nil {
		return err
	}

	// 2. Insert the new links
	batch, err := db.PrepareBatch(ctx, "INSERT INTO links")
	if err != nil {
		return err
	}
	for i := 0; i < len(sources); i++ {
		for j := 0; j < len(sources[i].Targets); j++ {
			err := batch.Append(
				sources[i].Source,
				sources[i].Targets[j],
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
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
