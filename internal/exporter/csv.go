package exporter

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/url"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/telemetry"
)

var CSV_BATCH_SIZE = 128

type CSVExporter struct {
	stream io.WriteCloser
	csv    *csv.Writer
}

func NewCSVExporter(stream io.WriteCloser) *CSVExporter {
	return &CSVExporter{stream, csv.NewWriter(stream)}
}

func (e *CSVExporter) Listen(ctx context.Context, urlChan chan url.URL) {
	var url url.URL
	i := 0
	for {
		select {
		case <-ctx.Done():
			e.csv.Flush()
			e.stream.Close()
			return
		case url = <-urlChan:
			// Write to buffer
			err := e.csv.Write([]string{url.String()})
			if err != nil {
				telemetry.ErrorChan <- fmt.Errorf("failed to export url to csv: %w", err)
			}

			// Flush buffer periodically
			i++
			if i%CSV_BATCH_SIZE == 0 {
				e.csv.Flush()
				i = 0
			}
		}
	}
}
