package telemetry

import (
	"log/slog"
	"os"

	"github.com/mrops-br/optl-testing-api/internal/infrastructure/config"
)

// initLogger initializes a structured logger
// Note: Full OTLP log export is still experimental in Go
// For production, consider using a log exporter or bridge
func initLogger(cfg *config.OTLPConfig) *slog.Logger {
	// Create JSON handler for structured logging
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)

	logger := slog.New(handler).With(
		slog.String("service.name", cfg.ServiceName),
		slog.String("environment", cfg.Environment),
	)

	return logger
}
