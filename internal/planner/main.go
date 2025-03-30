package planner

import (
	"context"
	"fmt"
	"log"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type Planner struct {
	ctx context.Context
	db  driver.Conn
}

func New() (*Planner, error) {
	ctx := context.Background()
	s, err := newSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to init planner settings: %w", err)
	}

	db, err := initDb(s)
	if err != nil {
		return nil, fmt.Errorf("failed to init planner database: %w", err)
	}

	return &Planner{
		ctx: ctx,
		db:  db,
	}, nil
}

func (p *Planner) Run() {
	rows, err := p.db.Query(p.ctx, "SELECT name,toString(uuid) as uuid_str FROM system.tables LIMIT 5")
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var name, uuid string
		if err := rows.Scan(&name, &uuid); err != nil {
			log.Fatal(err)
		}
		log.Printf("name: %s, uuid: %s", name, uuid)
	}
}

func (p *Planner) Seed(seed string) error {
	query := `INSERT INTO pages (protocol, host, path, last_visited_at) VALUES (?, ?, ?, ?)`
	err := p.db.Exec(p.ctx, query,
		"https", // protocol
		seed,    // host
		"/",     // path
		nil,     // last_visited_at
	)
	if err != nil {
		return fmt.Errorf("failed to seed db: %w", err)
	}
	fmt.Printf("inserted seed %s", seed)
	return nil
}
