package telemetry

import (
	"expvar"
	"fmt"
	_ "net/http/pprof"
	"time"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/shutdown"
)

var (
	ProcessedURL *expvar.Int
	Errors       *expvar.Int
)

func init() {
	ProcessedURL = expvar.NewInt("PocessedURL")
	Errors = expvar.NewInt("Errors")

	go MetricsReport()
}

func MetricsReport() {
	start := time.Now()
	fmt.Println("┌───────────────┬───────────────┬───────────────┐")
	fmt.Println("│     Time      │   processed   │    errors     │")
	fmt.Println("├───────────────┼───────────────┼───────────────┤")

	done := make(chan struct{})
	shutdown.Subscribe(done)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			defer func() { done <- struct{}{} }()
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
