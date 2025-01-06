package queue

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/commons"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/telemetry"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const ADD_BULK_SIZE = 64
const NEXT_BULK_SIZE = 128

type PostgresQueue struct {
	pg          *pgxpool.Pool
	ctx         context.Context
	plannerChan chan *url.URL
}

func NewPostgresQueue(ctx context.Context, pg *pgxpool.Pool) *PostgresQueue {
	q := &PostgresQueue{pg: pg}
	go q.Planner()
	return q
}

func (q *PostgresQueue) Add(url *url.URL) error {
	stmt := `
		INSERT INTO pages (scheme, hostname_reversed, path)
		VALUES (@scheme, @hostReversed, @path)
		ON CONFLICT DO NOTHING;
	`
	args := pgx.NamedArgs{
		"scheme":       url.Scheme,
		"hostReversed": commons.ReverseHostname(url.Hostname()),
		"path":         url.Path,
	}

	ctx, cancel := context.WithTimeout(q.ctx, time.Second*30)
	defer cancel()

	_, err := q.pg.Exec(ctx, stmt, args)
	if err != nil {
		return fmt.Errorf("unable to insert row: %w", err)
	}
	return nil
}

func (q *PostgresQueue) Next() (*url.URL, error) {
	url := <-q.plannerChan
	return url, nil
}

func (q *PostgresQueue) Planner() {
	for {
		query := `
			UPDATE pages
			SET latest_visit = NOW()
			WHERE id IN (
				SELECT id
				FROM pages
				WHERE latest_visit IS NULL
				FOR UPDATE SKIP LOCKED
				LIMIT 1024
			)
			RETURNING scheme, hostname_reversed, path;
		`

		ctx, cancel := context.WithTimeout(q.ctx, time.Second*30)
		defer cancel()

		rows, err := q.pg.Query(ctx, query)
		if err != nil {
			telemetry.ErrorChan <- fmt.Errorf("error in planner: unable to get next pages: %w", err)
			continue
		}
		defer rows.Close()

		for rows.Next() {
			// Marshall the url
			var scheme string
			var hostReversed string
			var path string
			err := rows.Scan(&scheme, &hostReversed, &path)
			if err != nil {
				telemetry.ErrorChan <- fmt.Errorf("error in planner: unable to scan row: %w", err)
				continue
			}
			host := commons.ReverseHostname(hostReversed)
			url := &url.URL{Scheme: scheme, Host: host, Path: path}

			// Yield the url or stop if app is shutting down
			select {
			case <-q.ctx.Done():
				close(q.plannerChan)
				return
			case q.plannerChan <- url:
			}
		}
	}
}
