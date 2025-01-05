package storage

// Pool management adapted from
// https://donchev.is/post/working-with-postgresql-in-go-using-pgx/#connection-pool-setup

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	pool        *pgxpool.Pool
	initError   error
	initOnce    sync.Once
	ctx         context.Context
	postgresURI string
)

func NewPostgres(initCtx context.Context, initPostgresURI string) (*pgxpool.Pool, error) {
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
			pool.Close()
		}
	}()

	// Create connection pool
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, postgresURI)
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
