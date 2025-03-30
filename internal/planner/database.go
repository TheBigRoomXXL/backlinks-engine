package planner

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

func initDb(s *Settings) (driver.Conn, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", s.DB_HOSTNAME, s.DB_PORT)},
		Auth: clickhouse.Auth{
			Database: s.DB_NAME,
			Username: s.DB_USER,
			Password: s.DB_PASSWORD,
		},
		Debugf: func(format string, v ...interface{}) {
			fmt.Printf(format, v)
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to init connection to database: %w", err)
	}

	query := `CREATE TABLE IF NOT EXISTS pages
	(
		id UUID DEFAULT generateUUIDv4(),
		protocol Enum('http', 'https'),
		host String,
		path String,
		last_visited_at Nullable(DateTime),
	)
	ENGINE = ReplacingMergeTree()
	ORDER BY (host, path)
	`

	if err := conn.Exec(context.Background(), query); err != nil {
		return nil, fmt.Errorf("failed to create necessary tables: %w", err)
	}

	return conn, nil
}
