package shared

import (
	"database/sql"
	"log"
	"sync"

	_ "github.com/marcboeker/go-duckdb/v2"
)

var once sync.Once
var db *sql.DB

func initDb() {
	var err error
	db, err = sql.Open("duckdb", "backlinks.db")
	if err != nil {
		log.Fatalf("failed to init database: %s\n", err)
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
	
	CREATE OR REPLACE VIEW hosts AS
	SELECT 
		host,
		SUM(CASE WHEN visited_at IS NULL THEN 1 ELSE 0 END) AS unvisited_count
	FROM pages
	GROUP BY host;
	`

	if _, err := db.Exec(query); err != nil {
		log.Fatalf("failed to init database: %s\n", err)
	}
}

func GetDatabase() *sql.DB {
	once.Do(initDb)
	return db
}
