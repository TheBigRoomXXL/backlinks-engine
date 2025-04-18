package controller

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/commons"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const BATCH_SIZE = 64

type Controller struct {
	pg       *pgxpool.Pool
	ctx      context.Context
	addChan  chan *commons.LinkGroup
	nextChan chan []*url.URL
}

func NewController(ctx context.Context, pgURI string) (*Controller, error) {
	pg, err := newPostgres(ctx, pgURI)
	if err != nil {
		return nil, fmt.Errorf("failed to init postgres connection pool: %w", err)
	}
	addChan := make(chan *commons.LinkGroup)
	nextChan := make(chan []*url.URL, 2048)

	c := &Controller{
		pg:       pg,
		ctx:      ctx,
		addChan:  addChan,
		nextChan: nextChan,
	}

	go c.addSubscriber()
	go c.nextProducer()

	return c, nil
}

func (c *Controller) Add(group *commons.LinkGroup) {
	c.addChan <- group
}

func (c *Controller) Next() []*url.URL {
	return <-c.nextChan
}

func (c *Controller) Seed(seeds []*url.URL) {
	insertPages(c.ctx, c.pg, seeds)
}

func (c *Controller) nextProducer() {
	for {
		query := `
		WITH random_host as (
			SELECT host_reversed
			FROM pages TABLESAMPLE SYSTEM_ROWS(1)
		),
		next_pages AS (
			SELECT id
			FROM pages, random_host
			WHERE latest_visit IS NULL
			AND pages.host_reversed = random_host.host_reversed
			LIMIT 128
		)
		UPDATE pages
		SET latest_visit = NOW()
		FROM next_pages
		WHERE pages.id = next_pages.id
		RETURNING scheme, host_reversed, path;
	`

		ctx, cancel := context.WithTimeout(c.ctx, time.Second*30)
		defer cancel()

		rows, err := c.pg.Query(ctx, query)
		if err != nil {
			if strings.Contains(err.Error(), "context canceled") {
				slog.Warn("context canceled in planner, exiting.")
				return
			}
			slog.Error(fmt.Sprintf("error in planner: unable to get next pages: %s", err))
			continue
		}
		defer rows.Close()

		urls, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (*url.URL, error) {
			var scheme string
			var hostReversed string
			var path string
			err := rows.Scan(&scheme, &hostReversed, &path)
			if err != nil {
				return nil, err
			}
			host := commons.ReverseHostname(hostReversed)
			return &url.URL{Scheme: scheme, Host: host, Path: path}, nil
		})
		if err != nil {
			slog.Error(fmt.Sprintf("error in planner: unable to scan row: %s", err))
			continue
		}

		// Yield the url or stop if app is shutting down
		select {
		case <-c.ctx.Done():
			close(c.nextChan)
			return
		case c.nextChan <- urls:
		}
	}
}

// This function listen to addChan and accumulates the new data until we can insert it in bulk
// If the context propagate a cancel we do a partial insert we what data we have in the buffer
func (c *Controller) addSubscriber() {
	var group *commons.LinkGroup
	links := [BATCH_SIZE]commons.Link{}
	newPages := [BATCH_SIZE]*url.URL{}
	visitedPages := [BATCH_SIZE]*url.URL{}
	i := 0
	j := 0
	timeout := time.After(time.Second)

	for {
		select {
		case group = <-c.addChan:
			from := group.From
			visitedPages[j] = from
			j++
			if j == BATCH_SIZE {
				updatePages(c.ctx, c.pg, visitedPages[:j])
				j = 0
			}

			for _, to := range group.To {
				links[i] = commons.Link{From: from, To: to}
				newPages[i] = to
				i++

				if i == BATCH_SIZE {
					insertLinks(c.ctx, c.pg, links[:i])
					insertPages(c.ctx, c.pg, newPages[:i])
					i = 0
				}
			}

			timeout = time.After(time.Second)
		// If not enough data come in one second we do a partial bulk insert (this avoid a
		// deadlock where Next() is starved because there is no new insert and there is
		// no new insert because Next is starved
		case <-timeout:
			// Insert our partial batch
			updatePages(c.ctx, c.pg, visitedPages[:j])
			insertLinks(c.ctx, c.pg, links[:i])
			insertPages(c.ctx, c.pg, newPages[:i])

			// Reset the current batch
			i = 0
			j = 0
		case <-ctx.Done():
			// Insert our partial batch
			updatePages(c.ctx, c.pg, visitedPages[:j])
			insertLinks(c.ctx, c.pg, links[:i])
			insertPages(c.ctx, c.pg, newPages[:i])

			// Reset the current batch
			i = 0
			j = 0
			// Stop the goroutine
			return
		}
	}
}
