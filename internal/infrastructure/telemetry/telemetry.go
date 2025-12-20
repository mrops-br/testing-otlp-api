package telemetry

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mrops-br/optl-testing-api/internal/infrastructure/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Telemetry holds all OpenTelemetry components
type Telemetry struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *metric.MeterProvider
	Logger         *slog.Logger
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

	// Initialize meter provider
	mp, err := initMeterProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize meter provider: %w", err)
	}

	// Set global meter provider
	otel.SetMeterProvider(mp)
	logger.Info("Meter provider initialized successfully")

	return &Telemetry{
		TracerProvider: tp,
		MeterProvider:  mp,
		Logger:         logger,
	}, nil
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
