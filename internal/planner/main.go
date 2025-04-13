package planner

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
)

type Planner struct {
	ctx context.Context
	db  *sql.DB
}

func New() (*Planner, error) {
	ctx := context.Background()
	db, err := initDb()
	if err != nil {
		return nil, fmt.Errorf("failed to init planner database: %w", err)
	}

	return &Planner{
		ctx: ctx,
		db:  db,
	}, nil
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
	valueStrings := make([]string, len(hosts))
	args := make([]interface{}, len(hosts))
	for i := 0; i < len(hosts); i++ {
		valueStrings[i] = "('https', ?, '/')"
		args[i] = hosts[i]
	}
	stmt := fmt.Sprintf(
		"INSERT OR IGNORE INTO pages (protocol, host,path) VALUES %s",
		strings.Join(valueStrings, ","),
	)
	_, err = p.db.Exec(stmt, args...)
	return err
}
