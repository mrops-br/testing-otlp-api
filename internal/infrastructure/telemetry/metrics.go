package telemetry

import (
	"context"
	"fmt"

	"github.com/mrops-br/optl-testing-api/internal/infrastructure/config"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// initMeterProvider initializes the OpenTelemetry meter provider
func initMeterProvider(cfg *config.OTLPConfig) (*metric.MeterProvider, error) {
	ctx := context.Background()

	// Create OTLP metric exporter
	conn, err := grpc.NewClient(cfg.Endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	exporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

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

	// Create meter provider
	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter)),
		metric.WithResource(res),
	)

	return mp, nil
}
