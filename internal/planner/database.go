package planner

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/marcboeker/go-duckdb/v2"
)

func initDb() (*sql.DB, error) {
	db, err := sql.Open("duckdb", "backlinks.db")
	if err != nil {
		log.Fatal(err)
	}

	query := `
	CREATE SEQUENCE IF NOT EXISTS seq_page_id;
	CREATE TABLE IF NOT EXISTS pages (
		id BIGINT PRIMARY KEY DEFAULT nextval('seq_page_id'),
		protocol TEXT NOT NULL CHECK (protocol IN ('http', 'https')),
		host TEXT NOT NULL,
		path TEXT NOT NULL,
		visited_at TIMESTAMP
	);
	CREATE UNIQUE INDEX IF NOT EXISTS pages_host_path_idx ON pages (host,path);
	`

	if _, err := db.Exec(query); err != nil {
		return nil, fmt.Errorf("failed to create necessary tables: %w", err)
	}

	return db, nil
}
