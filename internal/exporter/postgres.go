package exporter

import (
	"context"
	"fmt"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/telemetry"
	"github.com/jackc/pgx/v5"
)

const PG_BATCH_SIZE = 16

// We define an interface with just the pool methods we use so that we can easily mock
type MinimalPostgres interface {
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

type PostgresExporter struct {
	pg MinimalPostgres
}

func NewPostgresExporter(pg MinimalPostgres) *PostgresExporter {
	return &PostgresExporter{pg: pg}
}

func (e *PostgresExporter) Listen(ctx context.Context, urlChan chan *LinkGroup) {
	var group *LinkGroup
	batch := [PG_BATCH_SIZE]*LinkGroup{}
	i := 0
	for {
		select {
		case <-ctx.Done():
			// Empty the pointers from the precedent batch
			for i < PG_BATCH_SIZE {
				batch[i] = nil
				i++
			}
			// Then insert our partial batch
			e.Insert(ctx, batch)
			return
		case group = <-urlChan:
			batch[i] = group
			i++
			if i == PG_BATCH_SIZE {
				go e.Insert(ctx, batch)
				i = 0
			}
		}
	}
}

func (e *PostgresExporter) Insert(ctx context.Context, groups [PG_BATCH_SIZE]*LinkGroup) {
	query := `
		INSERT INTO links (source, target)
		VALUES (@source, @target)
		ON CONFLICT DO NOTHING;
	`
	batch := &pgx.Batch{}

	for _, group := range groups {
		if group == nil {
			// When we do a partial insert part of the values are null.
			continue
		}
		source := group.From.String()
		for _, target := range group.To {
			args := pgx.NamedArgs{
				"source": source,
				"target": target.String(),
			}
			batch.Queue(query, args)
		}
	}

	results := e.pg.SendBatch(ctx, batch)
	defer results.Close()

	for _, group := range groups {
		if group == nil {
			continue
		}

		_, err := results.Exec()
		if err != nil {
			telemetry.ErrorChan <- fmt.Errorf("unable to insert row in links: %w", err)
		}
	}
}
