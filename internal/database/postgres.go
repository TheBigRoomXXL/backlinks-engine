package database

// Pool management adapted from
// https://donchev.is/post/working-with-postgresql-in-go-using-pgx/#connection-pool-setup

import (
	"context"
	"fmt"
	"sync"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/settings"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgres struct {
	Pool *pgxpool.Pool
}

var (
	pg     *postgres
	pgOnce sync.Once
)

func NewPostgres(ctx context.Context, s *settings.Settings) (*postgres, error) {
	var err error
	pgOnce.Do(func() {
		var pool *pgxpool.Pool
		uri := fmt.Sprintf(
			"postgresql://%s:%s@%s:%s/%s",
			s.DB_USER,
			s.DB_PASSWORD,
			s.DB_HOSTNAME,
			s.DB_PORT,
			s.DB_NAME,
		)
		pool, err = pgxpool.New(ctx, uri)
		if err != nil {
			err = fmt.Errorf("failed to create connection pool: %w", err)
			return
		}
		pg = &postgres{pool}

		_, err = pg.Pool.Exec(context.Background(), `
			CREATE TABLE IF NOT EXISTS domains (
				hostname_reversed	text PRIMARY KEY,
				robot 				text NOT NULL
			);

			CREATE TABLE IF NOT EXISTS urls (
				id bigserial		PRIMARY KEY,
				scheme				text NOT NULL,
				hostname_reversed	text NOT NULL,
				path 				text NOT NULL
			);

			CREATE TABLE IF NOT EXISTS links (
				source	text,
				target	text,

				PRIMARY KEY (source, target)
			);
		`)
		if err != nil {
			err = fmt.Errorf("failed to initialize postgres tables: %w", err)
			return
		}
	})

	if err != nil {
		return nil, err
	}

	err = pg.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("ping failed after pool creation: %w", err)
	}

	return pg, nil
}

func (pg *postgres) Ping(ctx context.Context) error {
	return pg.Pool.Ping(ctx)
}

func (pg *postgres) Close() {
	pg.Pool.Close()
}
