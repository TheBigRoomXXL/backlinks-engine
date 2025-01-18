package telemetry

import (
	"context"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

var Tracer = otel.Tracer("")

func InitTracing(ctx context.Context) func() {
	// Create the exporter
	exporter, err := otlptracehttp.New(
		ctx,
		otlptracehttp.WithEndpoint("localhost:4318"),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		log.Panic("tracing failed to initialize exporter: ", err)
	}

	// Create a new tracer provider with a batch span processor and the given exporter.
	ressource, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("BacklinkBot"),
		),
	)

	if err != nil {
		log.Panic("tracing failed to initialize resource: ", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(ressource),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tracerProvider)

	// Finally, set the tracer that can be used for this package.
	Tracer = tracerProvider.Tracer("github.com/TheBigRoomXXL/backlinks-engine")

	return func() { tracerProvider.Shutdown(ctx) }

}
