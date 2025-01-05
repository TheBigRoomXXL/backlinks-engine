package exporter

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/telemetry"
)

const CSV_BATCH_SIZE = 128

type CSVExporter struct {
	stream io.WriteCloser
	csv    *csv.Writer
}

func NewCSVExporter(stream io.WriteCloser) *CSVExporter {
	return &CSVExporter{stream, csv.NewWriter(stream)}
}

func (e *CSVExporter) Listen(ctx context.Context, urlChan chan *LinkGroup) {
	var group *LinkGroup
	i := 0
	for {
		select {
		case <-ctx.Done():
			e.csv.Flush()
			e.stream.Close()
			return
		case group = <-urlChan:
			// Write to buffer
			for _, to := range group.To {
				err := e.csv.Write([]string{group.From.String(), to.String()})
				if err != nil {
					telemetry.ErrorChan <- fmt.Errorf("failed to export url to csv: %w", err)
				}
				// Flush buffer periodically
				i++
				if i == CSV_BATCH_SIZE {
					e.csv.Flush()
					i = 0
				}
			}
		}
	}
}
