package crawler

import (
	"context"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/client"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
)

type Crawler struct {
	ctx    context.Context
	group  *errgroup.Group
	pg     *pgxpool.Pool
	client *client.Fetcher
}

// func NewCrawler(ctx context.Context) (*Crawler, error) {
// 	ctx := shutdown.Subscribe()
// 	group := errgroup.WithContext(ctx)
// 	pg, err := database.NewPostgres()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get postgres connection pool: %w", err)
// 	}
// 	s, err := settings.New()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get settings connection pool: %w", err)
// 	}
// 	return &Crawler{s, pg}, nil
// }

// func Crawl() error {

// }

// 	err = planner.Seed(seeds)
// 	if err != nil {
// 		return fmt.Errorf("failed to import seeds: %w", err)
// 	}

// 	for i := 0; i < 10; i++ {
// 		fmt.Print(i, " -> ")
// 		pages, err := planner.Next()
// 		if err != nil {
// 			return err
// 		}
// 		fmt.Println(pages)
// 	}

// 	return nil
// }

// type Spider struct {
// 	pg        *database.Postgres
// 	client    *http.Client
// 	errorChan *chan struct{}
// }

// func NewSpider() (*Spider, error) {
// 	pg, err := database.New()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get postgres pool: %w", err)
// 	}
// 	errorChan := make(chan struct{})
// 	client := http.DefaultClient
// 	return &Spider{pg, client, &errorChan}, nil
// }

// func (p *Spider) CrawlPage(url url.URL) {
// 	// TODO: robot.txt validation
// 	// response, err := p.client.Head(url.String())
// 	// if err != nil {
// 	// 	fmt.Errorf("failed HEAD request on %s: %w", url.String(), err)
// 	// }

// }

// type Planner struct {
// 	s  *settings.Settings
// 	pg *database.Postgres
// }

// func NewPlanner() (*Planner, error) {
// 	pg, err := database.New()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get postgres connection pool: %w", err)
// 	}
// 	s, err := settings.New()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get settings connection pool: %w", err)
// 	}
// 	return &Planner{s, pg}, nil
// }

// func (p *Planner) Seed(seeds []string) error {
// 	urls := make([]*url.URL, len(seeds))
// 	for i := 0; i < len(seeds); i++ {
// 		url, err := url.Parse(seeds[i])
// 		if err != nil {
// 			return fmt.Errorf("filed to parse seed %s: %w", seeds[i], err)
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

// 	results := p.pg.Pool.SendBatch(context.Background(), batch)
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