package planner

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/shared"
)

type Planner struct {
	ctx         context.Context
	db          *sql.DB
	lockedHosts []string
}

func New() *Planner {
	return &Planner{
		ctx:         context.Background(),
		db:          shared.GetDatabase(),
		lockedHosts: []string{},
	}
}

// This function expect a CSV file with a list of whitespace separated hosts.
func (p *Planner) Seed(seedsPath string) error {
	file, err := os.Open(seedsPath)
	if err != nil {
		return fmt.Errorf("failed to seed the database: %w", err)
	}
	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to seed the database: %w", err)
	}
	hosts := strings.Fields(string(content))
	placeholders := make([]string, len(hosts))
	args := make([]interface{}, len(hosts))
	for i := 0; i < len(hosts); i++ {
		placeholders[i] = "('https', ?, '/')"
		args[i] = hosts[i]
	}
	stmt := fmt.Sprintf(
		"INSERT OR IGNORE INTO pages (protocol, host,path) VALUES %s",
		strings.Join(placeholders, ","),
	)
	_, err = p.db.Exec(stmt, args...)
	return err
}

type CrawlTask struct {
	Host   string
	Budget int
	Pages  []url.URL
}

func (p *Planner) NextCrawl() *CrawlTask {
	nextHost, err := p.nextCrawlHost()
	if err != nil {
		log.Printf("failed to prepare next crawl: %s\n", err)
		return nil
	}
	if nextHost == "" {
		log.Println("failed to prepare next crawl: no free host found")
		return nil
	}

	rows, err := p.db.Query(`
		SELECT protocol, host, path
		FROM pages WHERE host = $1
		ORDER BY visited_at ASC NULLS FIRST;`,
		nextHost,
	)
	if err != nil {
		log.Printf("failed to prepare next crawl: %s\n", err)
		return nil
	}
	defer rows.Close()

	var pages []url.URL
	for rows.Next() {
		var protocol, host, path string
		if err := rows.Scan(&protocol, &host, &path); err != nil {
			log.Printf("failed to prepare next crawl: %s\n", err)
			return nil
		}
		urlStr := fmt.Sprintf("%s://%s%s", protocol, host, path)
		parsed, err := url.Parse(urlStr)
		shared.Assert(
			err == nil,
			fmt.Sprintf("url from db should never be malformed: %s: %s", urlStr, err),
		)
		pages = append(pages, *parsed)
	}
	if err := rows.Err(); err != nil {
		log.Printf("failed to prepare next crawl: %s\n", err)
		return nil
	}
	return &CrawlTask{
		Host:   nextHost,
		Budget: 1000,
		Pages:  pages,
	}

}

func (p *Planner) nextCrawlHost() (string, error) {
	var query string
	args := make([]interface{}, len(p.lockedHosts))
	if len(p.lockedHosts) == 0 {
		query = `
			SELECT host 
			FROM hosts
			ORDER BY unvisited_count DESC
			LIMIT 1;`
	} else {
		placeholders := make([]string, len(p.lockedHosts))
		for i := 0; i < len(p.lockedHosts); i++ {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args[i] = p.lockedHosts[i]
		}
		query = fmt.Sprintf(`
			SELECT host 
			FROM hosts
			WHERE host NOT IN (%s)
			ORDER BY unvisited_count DESC
			LIMIT 1;`,
			strings.Join(placeholders, ","),
		)
	}
	var nextHost string
	err := p.db.QueryRow(query, args...).Scan(&nextHost)
	if err != nil && err == sql.ErrNoRows {
		return "", nil // No result found
	}
	return nextHost, err
}
