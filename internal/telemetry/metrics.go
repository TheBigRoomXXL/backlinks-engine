package telemetry

import (
	"context"
	"expvar"
	"fmt"
	_ "net/http/pprof"
	"time"
)

var (
	ProcessedURL *expvar.Int
	Errors       *expvar.Int
)

func init() {
	ProcessedURL = expvar.NewInt("PocessedURL")
	Errors = expvar.NewInt("Errors")
}

func MetricsReport(ctx context.Context) {
	start := time.Now()
	fmt.Println("┌───────────────┬───────────────┬───────────────┐")
	fmt.Println("│     Time      │   processed   │    errors     │")
	fmt.Println("├───────────────┼───────────────┼───────────────┤")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\r└───────────────┴───────────────┴───────────────┘")
			return
		case <-ticker.C:
			time := time.Since(start).Round(time.Second)
			fmt.Printf(
				"│ %13s │ %13d │ %13d │\n",
				time, ProcessedURL.Value(), Errors.Value(),
			)
		}
	}
}
