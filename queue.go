package main

import (
	"database/sql"

	_ "github.com/lib/pq"
)

// PostgresStorage is the default implementation of the Storage interface.
// PostgresStorage holds the request queue in memory.
type PostgresStorage struct {
	Db *sql.DB
}

// Init implements Storage.Init() function
func (q *PostgresStorage) Init() error {
	_, err = q.Db.Exec(`
		CREATE TABLE IF NOT EXISTS colly_queue (
			id UUID PRIMARY KEY,
			inserted_at      TIMESTAMP,
			request            bytea
		);
		CREATE INDEX IF NOT EXISTS idx ON colly_queue (inserted_at ASC);
	`)
	return err
}

// AddRequest implements Storage.AddRequest() function
func (q *PostgresStorage) AddRequest(r []byte) error {
	stmt := `
		INSERT INTO colly_queue (id, inserted_at, request)
		VALUES (gen_random_uuid(), current_timestamp, $1 );
	`
	_, err = q.Db.Exec(stmt, r)
	return err
}

// GetRequest implements Storage.GetRequest() function
func (q *PostgresStorage) GetRequest() ([]byte, error) {
	// The use of select for update skip locked ensures that concurrent clients can access
	// the table concurrently and do not block each other on existing locks.
	stmt := `DELETE
             FROM colly_queue 
             WHERE colly_queue.id =
                 (SELECT colly_queue_inner.id
                  FROM colly_queue AS colly_queue_inner
                  ORDER BY colly_queue_inner.inserted_at ASC
                  FOR UPDATE SKIP LOCKED
                  LIMIT 1)
             RETURNING colly_queue.request;`

	// Execute the query and retrieve the result
	var request []byte
	err := q.Db.QueryRow(stmt).Scan(&request)
	if err != nil {
		return nil, err
	}
	return request, nil
}

// QueueSize implements Storage.QueueSize() function
func (q *PostgresStorage) QueueSize() (int, error) {
	var count int
	err := q.Db.QueryRow("SELECT COUNT(id) FROM colly_queue").Scan(&count)
	return count, err
}
