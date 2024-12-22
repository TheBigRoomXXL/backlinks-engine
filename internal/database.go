package internal

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

func NewDatabase(s *Settings) (driver.Conn, error) {
	var (
		ctx       = context.Background()
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{fmt.Sprintf("%s:%s", s.DB_HOSTNAME, s.DB_PORT)},
			Auth: clickhouse.Auth{
				Database: s.DB_NAME,
				Username: s.DB_USER,
				Password: s.DB_PASSWORD,
			},
			Protocol: clickhouse.Native,
			Debugf: func(format string, v ...interface{}) {
				fmt.Printf(format, v)
			},
		})
	)

	if err != nil {
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		return nil, err
	}

	const ddl = `
	CREATE TABLE IF NOT EXISTS links (
		target String,
		source String,
	)
	ENGINE = ReplacingMergeTree
	PRIMARY KEY (target, source)
	`
	if err := conn.Exec(ctx, ddl); err != nil {
		return nil, err
	}

	return conn, nil
}
