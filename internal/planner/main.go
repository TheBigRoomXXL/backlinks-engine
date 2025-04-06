package planner

import (
	"context"
	"database/sql"
	"fmt"
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

func (p *Planner) Seed(seed string) error {
	_, err := p.db.Exec(
		"INSERT INTO pages (protocol, host, path, last_visited_at) VALUES (?, ?, ?, ?)",
		"https", seed, "/", nil,
	)
	if err != nil {
		return fmt.Errorf("failed to seed db: %w", err)
	}
	fmt.Printf("inserted seed %s\n", seed)
	return nil
}
