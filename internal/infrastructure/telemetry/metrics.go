package telemetry

import (
	"context"
	"fmt"

	"github.com/mrops-br/testing-otlp-api/internal/infrastructure/config"
	prometheusExporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)


// initMeterProvider initializes OpenTelemetry MeterProvider with DUAL exporters
// - OTLP exporter: Sends to Alloy for centralized collection
// - Prometheus exporter: Exposes /metrics endpoint for scraping
func initMeterProvider(cfg *config.OTLPConfig) (*metric.MeterProvider, error) {
	ctx := context.Background()

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion("1.0.0"),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create OTLP metric exporter (for Alloy)
	conn, err := grpc.NewClient(cfg.Endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	otlpExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
	}

	// Create Prometheus exporter (for /metrics endpoint)
	promExporter, err := prometheusExporter.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	// // Configure histogram buckets for HTTP duration (in seconds)
	// // Custom buckets for typical API latencies: 1ms to 10s
	// // Buckets: 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
	// durationView := metric.NewView(
	// 	metric.Instrument{
	// 		Name: "http.server.request.duration",
	// 		Kind: metric.InstrumentKindHistogram,
	// 	},
	// 	metric.Stream{
	// 		Aggregation: metric.AggregationExplicitBucketHistogram{
	// 			Boundaries: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	// 		},
	// 	},
	// )

	// Create meter provider with BOTH exporters and custom views
	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(otlpExporter)),  // OTLP push
		metric.WithReader(promExporter),                             // Prometheus pull
		metric.WithResource(res),
		// metric.WithView(durationView),  // Custom histogram buckets
	)

	return mp, nil
}
