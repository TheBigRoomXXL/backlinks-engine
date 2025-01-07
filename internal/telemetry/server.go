package telemetry

import (
	_ "expvar"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func StartTelemetryServer(listen string) {
	fmt.Printf("Telemetry server is listening on http://%s/debug/pprof and http://%s/debug/vars \n", listen, listen)
	log.Fatalf("pprof crashed: %s", http.ListenAndServe(listen, nil))
}
