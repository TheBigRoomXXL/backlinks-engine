package controller

// Pool management adapted from
// https://donchev.is/post/working-with-postgresql-in-go-using-pgx/#connection-pool-setup

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/commons"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	pool        *pgxpool.Pool
	initError   error
	initOnce    sync.Once
	ctx         context.Context
	postgresURI string
)

func newPostgres(initCtx context.Context, initPostgresURI string) (*pgxpool.Pool, error) {
	ctx = initCtx
	postgresURI = initPostgresURI
	initOnce.Do(initDatabase)
	return pool, initError
}

func initDatabase() {
	// Ensure connection will be closed gracefully
	go func() {
		<-ctx.Done()
		if pool != nil {
			slog.Debug("closing postgres pool")
			pool.Close()
		}
	}()

	// Create connection pool
	var err error
	pool, err = pgxpool.New(ctx, postgresURI)
	if err != nil {
		initError = fmt.Errorf("failed to create connection pool: %w", err)
		return
	}
	slog.Debug("postgres pool aquired")

	// Test connection pool
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	err = pool.Ping(ctx)
	if err != nil {
		initError = fmt.Errorf("ping failed after pool creation: %w", err)
		return
	}
	slog.Debug("pool pinged successfully")

	// Ensure Schema is initialized
	_, err = pool.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS host (
				host_reversed	text PRIMARY KEY,
				robot 				text NOT NULL
			);

			CREATE TABLE IF NOT EXISTS pages (
			 	id 				uuid DEFAULT gen_random_uuid(),
				scheme			text NOT NULL,
				host_reversed	text NOT NULL,
				path 			text NOT NULL,
				latest_visit 	timestamp,

				PRIMARY KEY(host_reversed, path)
			);

			CREATE TABLE IF NOT EXISTS links (
				source	text,
				target	text,

				PRIMARY KEY (source, target)
			);
		`)
	if err != nil {
		initError = fmt.Errorf("failed to initialize postgres tables: %w", err)
		return
	}
}

func insertPages(ctx context.Context, db *pgxpool.Pool, pages []*url.URL) {
	if len(pages) == 0 {
		return
	}

	var (
		stmtBuilder strings.Builder
		args        []any
	)

	stmtBuilder.WriteString("INSERT INTO pages (scheme, host_reversed, path) VALUES ")
	for i, page := range pages {
		if i > 0 {
			stmtBuilder.WriteString(", ")
		}
		paramIndex := i * 3
		stmtBuilder.WriteString(fmt.Sprintf("($%d, $%d, $%d)", paramIndex+1, paramIndex+2, paramIndex+3))
		args = append(args, page.Scheme, commons.ReverseHostname(page.Hostname()), page.Path)
	}
	stmtBuilder.WriteString(" ON CONFLICT DO NOTHING;")
	stmt := stmtBuilder.String()

	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	_, err := db.Exec(ctx, stmt, args...)
	if err != nil {
		slog.Error(fmt.Sprintf("unable to insert pages: %s", err))
	}
}

func insertLinks(ctx context.Context, db *pgxpool.Pool, links []commons.Link) {
	if len(links) == 0 {
		return
	}

	var (
		stmtBuilder strings.Builder
		args        []any
	)

	stmtBuilder.WriteString("INSERT INTO links (source, target) VALUES ")
	for i, link := range links {
		if i > 0 {
			stmtBuilder.WriteString(", ")
		}
		paramIndex := i * 2
		stmtBuilder.WriteString(fmt.Sprintf("($%d, $%d)", paramIndex+1, paramIndex+2))
		args = append(args, link.From, link.To)
	}
	stmtBuilder.WriteString(" ON CONFLICT DO NOTHING;")
	stmt := stmtBuilder.String()

	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	_, err := db.Exec(ctx, stmt, args...)
	if err != nil {
		slog.Error(fmt.Sprintf("unable to insert links: %s", err))
	}
}

func updatePages(ctx context.Context, db *pgxpool.Pool, pages []*url.URL) {
	// TODO: what do we want to save? status, hash, cache?
}

func selectNextPages(ctx context.Context, db *pgxpool.Pool) (pgx.Rows, error) {
	query := `
		WITH next_pages AS (
			SELECT DISTINCT ON (host_reversed) id
			FROM pages
			WHERE latest_visit IS NULL
			LIMIT 2048
		)
		UPDATE pages
		SET latest_visit = NOW()
		FROM next_pages
		WHERE pages.id = next_pages.id
		RETURNING scheme, host_reversed, path;
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	return db.Query(ctx, query)
}
