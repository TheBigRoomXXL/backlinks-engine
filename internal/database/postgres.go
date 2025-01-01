package database

// Pool management adapted from
// https://donchev.is/post/working-with-postgresql-in-go-using-pgx/#connection-pool-setup

import (
	"context"
	"fmt"
	"sync"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/settings"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/shutdown"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	pool      *pgxpool.Pool
	initError error
	initOnce  sync.Once
)

func NewPostgres() (*pgxpool.Pool, error) {
	initOnce.Do(initDatabase)
	return pool, initError
}

func initDatabase() {
	// Ensure connection will be closed gracefully
	done := make(chan struct{})
	shutdown.Subscribe(done)

	go func() {
		<-done
		if pool != nil {
			pool.Close()
		}
		done <- struct{}{}
	}()

	// Get settings
	s, err := settings.New()
	if err != nil {
		initError = err
		return
	}

	// Create connection pool
	uri := fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s?%s",
		s.DB_USER,
		s.DB_PASSWORD,
		s.DB_HOSTNAME,
		s.DB_PORT,
		s.DB_NAME,
		s.DB_OPTIONS,
	)
	ctx := context.Background()
	pool, err = pgxpool.New(ctx, uri)
	if err != nil {
		initError = fmt.Errorf("failed to create connection pool: %w", err)
		return
	}

	// Test connection pool
	err = pool.Ping(ctx)
	if err != nil {
		initError = fmt.Errorf("ping failed after pool creation: %w", err)
		return
	}

	// Ensure Schema is initialized
	_, err = pool.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS domains (
				hostname_reversed	text PRIMARY KEY,
				robot 				text NOT NULL
			);

			CREATE TABLE IF NOT EXISTS pages (
				id bigserial		PRIMARY KEY,
				scheme				text NOT NULL,
				hostname_reversed	text NOT NULL,
				path 				text NOT NULL,
				latest_visited 		timestamp,
				latest_status 		smallint
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
