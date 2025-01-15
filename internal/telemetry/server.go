package telemetry

import (
	_ "expvar"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func StartTelemetryServer(listen string) {
	http.Handle("/metrics", promhttp.Handler())
	fmt.Println("Telemetry server is listening on:")
	fmt.Printf("  - http://%s/debug/pprof  \n", listen)
	fmt.Printf("  - http://%s/debug/vars \n", listen)
	fmt.Printf("  - http://%s/metrics \n", listen)
	log.Fatalf("Telemetry server crashed: %s", http.ListenAndServe(listen, nil))
}
