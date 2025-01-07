package telemetry

import (
	"context"
	"expvar"
	"fmt"
	_ "net/http/pprof"
	"time"
)

var (
	ProcessedURL    *expvar.Int
	Errors          *expvar.Int
	Warnings        *expvar.Int
	QueueSize       *expvar.Int
	RobotAllowed    *expvar.Int
	RobotDisallowed *expvar.Int
	Links           *expvar.Int
)

func init() {
	ProcessedURL = expvar.NewInt("PocessedURL")
	Errors = expvar.NewInt("Errors")
	Warnings = expvar.NewInt("Warnings")
	QueueSize = expvar.NewInt("QueueSize")
	RobotAllowed = expvar.NewInt("RobotAllowed")
	RobotDisallowed = expvar.NewInt("RobotDisallowed")
	Links = expvar.NewInt("Links")
}

func MetricsReport(ctx context.Context) {
	start := time.Now()
	fmt.Println("┌───────────────┬───────────────┬───────────────┬───────────────┬───────────────┬───────────────┐")
	fmt.Println("│     Time      │   processed   │    errors     │   warnings    │  queue size   │  link pairs   │")
	fmt.Println("├───────────────┼───────────────┼───────────────┼───────────────┼───────────────┼───────────────┤")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			uptime := time.Since(start).Round(time.Second)
			fmt.Printf(
				"\n│ %13s │ %13d │ %13d │ %13d │ %13d │ %13d │\n",
				uptime,
				ProcessedURL.Value(),
				Errors.Value(),
				Warnings.Value(),
				QueueSize.Value(),
				Links.Value(),
			)
			fmt.Println("\r└───────────────┴───────────────┴───────────────┴───────────────┴───────────────┴───────────────┘")
			return
		case <-ticker.C:
			uptime := time.Since(start).Round(time.Second)
			fmt.Printf(
				"\r│ %13s │ %13d │ %13d │ %13d │ %13d │ %13d │",
				uptime,
				ProcessedURL.Value(),
				Errors.Value(),
				Warnings.Value(),
				QueueSize.Value(),
				Links.Value(),
			)
			if int(uptime.Seconds())%30 == 0 {
				fmt.Print("\n")
			}
		}
	}
}
