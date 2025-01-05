package exporter

import (
	"context"
	"fmt"
	"time"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/telemetry"
	"github.com/jackc/pgx/v5"
)

const PG_BATCH_SIZE = 128

// We define an interface with just the pool methods we use so that we can easily mock
type MinimalPostgres interface {
	CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error)
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

func (e *PostgresExporter) Insert(ctx context.Context, batch [PG_BATCH_SIZE]*LinkGroup) {
	entries := [][]any{}
	columns := []string{"source", "target"}
	tableName := "links"

	for _, group := range batch {
		if group == nil {
			// When we do a partial insert part of the values are null.
			continue
		}
		source := group.From.String()
		for _, target := range group.To {
			entries = append(entries, []any{source, target.String()})
		}
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_, err := e.pg.CopyFrom(
		ctx,
		pgx.Identifier{tableName},
		columns,
		pgx.CopyFromRows(entries),
	)
	if err != nil {
		telemetry.ErrorChan <- fmt.Errorf("error copying into %s table: %w", tableName, err)
	}
}
