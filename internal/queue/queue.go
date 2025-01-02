package queue

import (
	"container/list"
	"net/url"
	"sync"
)

type Queue interface {
	Add(*url.URL) error
	Next() (*url.URL, error)
}

// FIFOQueue represents a thread-safe FIFO queue implemented using a linked list.
type FIFOQueue struct {
	mu   *sync.Mutex
	list *list.List
}

// NewFIFOQueue creates a new FIFOQueue.
func NewFIFOQueue() *FIFOQueue {
	return &FIFOQueue{
		mu:   &sync.Mutex{},
		list: list.New(),
	}
}

// Add adds an element to the end of the list.
func (q *FIFOQueue) Add(url *url.URL) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.list.PushBack(url)
	return nil
}

// Next removes and returns the element from the front of the list.
// If the queue is empty, it returns nil.
func (q *FIFOQueue) Next() (*url.URL, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	element := q.list.Front()
	if element != nil {
		q.list.Remove(element)
		// fmt.Println("next is ", element.Value)
		return element.Value.(*url.URL), nil // Type assertion
	}
	return nil, nil
}

// func (c *Crawler) Add(url *url.URL) error {
// 	urls := make([]*url.URL, len(seeds))
// 	for i := 0; i < len(seeds); i++ {
// 		url, err := url.Parse(seeds[i])
// 		if err != nil {
// 			return fmt.Errorf("failed to parse seed %s: %w", seeds[i], err)
// 		}
// 		urls[i] = url
// 	}
// 	stmt := `
// 		INSERT INTO pages (scheme, hostname_reversed, path)
// 		VALUES (@scheme, @hostnameReversed, @path)
// 	`

// 	batch := &pgx.Batch{}
// 	for _, url := range urls {
// 		args := pgx.NamedArgs{
// 			"scheme":           url.Scheme,
// 			"hostnameReversed": commons.ReverseHostname(url.Hostname()),
// 			"path":             url.Path,
// 		}
// 		batch.Queue(stmt, args)
// 	}

// 	results := c.pg.SendBatch(context.Background(), batch)
// 	defer results.Close()

// 	for _, url := range urls {
// 		_, err := results.Exec()
// 		if err != nil {
// 			return fmt.Errorf("unable to insert url %s: %w", url, err)
// 		}
// 	}

// 	return results.Close()
// }

// func (p *Planner) Next() ([]url.URL, error) {
// 	query := `
// 		UPDATE pages
// 		SET latest_visited = NOW()
// 		WHERE id IN (
// 			SELECT id
// 			FROM pages
// 			WHERE latest_visited IS NULL
// 			FOR UPDATE SKIP LOCKED
// 			LIMIT 1024
// 		)
// 		RETURNING scheme, hostname_reversed, path;
//     `

// 	rows, err := p.pg.Pool.Query(context.Background(), query)
// 	if err != nil {
// 		return nil, fmt.Errorf("unable to get next pages: %w", err)
// 	}
// 	defer rows.Close()

// 	urls := []url.URL{}
// 	for rows.Next() {
// 		var scheme string
// 		var hostnameReversed string
// 		var path string
// 		err := rows.Scan(&scheme, &hostnameReversed, &path)
// 		if err != nil {
// 			return nil, fmt.Errorf("unable to scan row: %w", err)
// 		}
// 		host := commons.ReverseHostname(hostnameReversed)
// 		urls = append(urls, url.URL{Scheme: scheme, Host: host, Path: path})
// 	}

// 	return urls, nil
// }
