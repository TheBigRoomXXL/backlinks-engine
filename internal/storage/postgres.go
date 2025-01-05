package storage

// Pool management adapted from
// https://donchev.is/post/working-with-postgresql-in-go-using-pgx/#connection-pool-setup

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

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
				scheme				text NOT NULL,
				host_reversed	text NOT NULL,
				path 				text NOT NULL,
				latest_visited 		timestamp,
				latest_status 		smallint,

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
