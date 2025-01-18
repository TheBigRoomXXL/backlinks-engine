package telemetry

import (
	"context"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

var Tracer = otel.Tracer("")
var TracerProvider trace.TracerProvider

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

	TracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(ressource),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(TracerProvider)

	// Finally, set the tracer that can be used for this package.
	Tracer = TracerProvider.Tracer("github.com/TheBigRoomXXL/backlinks-engine")

	return func() { TracerProvider.Shutdown(ctx) }

}
