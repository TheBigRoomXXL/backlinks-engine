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
	QueueSize       *expvar.Int
	TCPTimeout      *expvar.Int
	RobotAllowed    *expvar.Int
	RobotDisallowed *expvar.Int
	LinkPaire       *expvar.Int
)

func init() {
	ProcessedURL = expvar.NewInt("PocessedURL")
	Errors = expvar.NewInt("Errors")
	TCPTimeout = expvar.NewInt("TCPTimeout")
	QueueSize = expvar.NewInt("QueueSize")
	RobotAllowed = expvar.NewInt("RobotAllowed")
	RobotDisallowed = expvar.NewInt("RobotDisallowed")
	LinkPaire = expvar.NewInt("LinkPaire")
}

func MetricsReport(ctx context.Context) {
	start := time.Now()
	fmt.Println("┌───────────────┬───────────────┬───────────────┬───────────────┬───────────────┬───────────────┐")
	fmt.Println("│     Time      │   processed   │    errors     │  tcp timeout  │  queue size   │  link pairs   │")
	fmt.Println("├───────────────┼───────────────┼───────────────┼───────────────┼───────────────┼───────────────┤")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\r└───────────────┴───────────────┴───────────────┴───────────────┴───────────────┴───────────────┘")
			return
		case <-ticker.C:
			uptime := time.Since(start).Round(time.Second)
			fmt.Printf(
				"\r│ %13s │ %13d │ %13d │ %13d │ %13d │ %13d │",
				uptime, ProcessedURL.Value(), Errors.Value(), TCPTimeout.Value(), QueueSize.Value(), LinkPaire.Value(),
			)
			if int(uptime.Seconds())%30 == 0 {
				fmt.Print("\n")
			}
		}
	}
}
