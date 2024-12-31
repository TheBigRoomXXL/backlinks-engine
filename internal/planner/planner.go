package planer

import (
	"context"
	"fmt"
	"net/url"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/commons"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/database"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/settings"
	"github.com/jackc/pgx/v5"
)

type Planner struct {
	s  *settings.Settings
	pg *database.Postgres
}

func New() (*Planner, error) {
	pg, err := database.New()
	if err != nil {
		return nil, fmt.Errorf("failed to get postgres connection pool: %w", err)
	}
	s, err := settings.New()
	if err != nil {
		return nil, fmt.Errorf("failed to get settings connection pool: %w", err)
	}
	return &Planner{s, pg}, nil
}

func (p *Planner) Seed(seeds []string) error {
	urls := make([]*url.URL, len(seeds))
	for i := 0; i < len(seeds); i++ {
		url, err := url.Parse(seeds[i])
		if err != nil {
			return fmt.Errorf("filed to parse seed %s: %w", seeds[i], err)
		}
		urls[i] = url
	}
	stmt := `
		INSERT INTO pages (scheme, hostname_reversed, path)
		VALUES (@scheme, @hostnameReversed, @path)
	`

	batch := &pgx.Batch{}
	for _, url := range urls {
		args := pgx.NamedArgs{
			"scheme":           url.Scheme,
			"hostnameReversed": commons.ReverseHostname(url.Hostname()),
			"path":             url.Path,
		}
		batch.Queue(stmt, args)
	}

	results := p.pg.Pool.SendBatch(context.Background(), batch)
	defer results.Close()

	for _, url := range urls {
		_, err := results.Exec()
		if err != nil {
			return fmt.Errorf("unable to insert url %s: %w", url, err)
		}
	}

	return results.Close()
}

func (p *Planner) Next() ([]url.URL, error) {
	query := `
		UPDATE pages
		SET latest_visited = NOW()
		WHERE id IN (
			SELECT id
			FROM pages
			WHERE latest_visited IS NULL
			FOR UPDATE SKIP LOCKED
			LIMIT 1024
		)
		RETURNING scheme, hostname_reversed, path;
    `

	rows, err := p.pg.Pool.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("unable to get next pages: %w", err)
	}
	defer rows.Close()

	urls := []url.URL{}
	for rows.Next() {
		var scheme string
		var hostnameReversed string
		var path string
		err := rows.Scan(&scheme, &hostnameReversed, &path)
		if err != nil {
			return nil, fmt.Errorf("unable to scan row: %w", err)
		}
		host := commons.ReverseHostname(hostnameReversed)
		urls = append(urls, url.URL{Scheme: scheme, Host: host, Path: path})
	}

	return urls, nil
}
