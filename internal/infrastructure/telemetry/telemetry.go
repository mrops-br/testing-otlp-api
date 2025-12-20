package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/mrops-br/testing-otlp-api/internal/infrastructure/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Telemetry holds all OpenTelemetry components
type Telemetry struct {
	TracerProvider    *sdktrace.TracerProvider
	MeterProvider     *metric.MeterProvider
	Logger            *slog.Logger
}

// NewTelemetry initializes all OpenTelemetry components
func NewTelemetry(cfg *config.OTLPConfig) (*Telemetry, error) {
	// Initialize logger first for debugging
	logger := initLogger(cfg)

	logger.Info("Initializing OpenTelemetry",
		slog.String("endpoint", cfg.Endpoint),
		slog.String("service_name", cfg.ServiceName),
	)

	// Initialize tracer provider
	tp, err := initTracerProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracer provider: %w", err)
	}

	// Set global tracer provider
	otel.SetTracerProvider(tp)
	logger.Info("Tracer provider initialized successfully")

	// Initialize meter provider with DUAL exporters (OTLP + Prometheus)
	mp, err := initMeterProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize meter provider: %w", err)
	}

	// Set global meter provider
	otel.SetMeterProvider(mp)
	logger.Info("Meter provider initialized successfully (OTLP + Prometheus exporters)")

	return &Telemetry{
		TracerProvider:    tp,
		MeterProvider:     mp,
		Logger:            logger,
	}, nil
}

// NewNoOpTelemetry creates a telemetry instance with no-op providers (no export)
func NewNoOpTelemetry(cfg *config.OTLPConfig) *Telemetry {
	// Create logger without trace context handler
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})).With(
		slog.String("service.name", cfg.ServiceName),
		slog.String("environment", cfg.Environment),
	)

	// Create no-op tracer provider (doesn't export)
	tp := sdktrace.NewTracerProvider()

	// Create no-op meter provider (doesn't export, but Prometheus metrics still work)
	mp := metric.NewMeterProvider()

	// Set as global providers
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)

	logger.Info("Telemetry initialized in no-op mode (export disabled)")

	return &Telemetry{
		TracerProvider: tp,
		MeterProvider:  mp,
		Logger:         logger,
	}
}

// Shutdown gracefully shuts down all telemetry components
func (t *Telemetry) Shutdown(ctx context.Context) error {
	t.Logger.Info("Shutting down OpenTelemetry")

	if err := t.TracerProvider.Shutdown(ctx); err != nil {
		t.Logger.Error("Failed to shutdown tracer provider", slog.String("error", err.Error()))
		return err
	}

	if err := t.MeterProvider.Shutdown(ctx); err != nil {
		t.Logger.Error("Failed to shutdown meter provider", slog.String("error", err.Error()))
		return err
	}

	t.Logger.Info("OpenTelemetry shutdown successfully")
	return nil
}
