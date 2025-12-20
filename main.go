package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mrops-br/testing-otlp-api/internal/app/service"
	"github.com/mrops-br/testing-otlp-api/internal/infrastructure/config"
	"github.com/mrops-br/testing-otlp-api/internal/infrastructure/http"
	"github.com/mrops-br/testing-otlp-api/internal/infrastructure/http/handler"
	"github.com/mrops-br/testing-otlp-api/internal/infrastructure/repository/memory"
	"github.com/mrops-br/testing-otlp-api/internal/infrastructure/telemetry"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize OpenTelemetry (if enabled)
	var telem *telemetry.Telemetry
	var err error

	if cfg.OTLP.Enabled {
		telem, err = telemetry.NewTelemetry(&cfg.OTLP)
		if err != nil {
			log.Fatalf("Failed to initialize telemetry: %v", err)
		}

		// Ensure telemetry is shutdown on exit
		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if err := telem.Shutdown(shutdownCtx); err != nil {
				log.Printf("Error shutting down telemetry: %v", err)
			}
		}()
	} else {
		// Create minimal telemetry without OTLP exporters
		telem = telemetry.NewNoOpTelemetry(&cfg.OTLP)
		log.Println("OpenTelemetry export disabled (OTEL_ENABLED=false)")
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get tracer, meter, and logger instances
	tracer := telem.TracerProvider.Tracer("products-api")
	meter := telem.MeterProvider.Meter("products-api")
	logger := telem.Logger

	logger.Info("Starting Products API")

	// Initialize repository (dependency injection)
	repo := memory.NewProductRepository(tracer, logger)

	// Initialize service
	productService := service.NewProductService(repo, tracer, meter, logger)

	// Initialize handler
	productHandler := handler.NewProductHandler(productService, logger)

	// Initialize HTTP server with otelhttp instrumentation
	// otelhttp automatically provides HTTP metrics (active_requests, duration, etc.)
	server := http.NewServer(&cfg.Server, productHandler, tracer, logger, telem)

	// Start server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			logger.Error("Server error", "error", err.Error())
			cancel()
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		logger.Info("Shutting down server...")
	case <-ctx.Done():
		logger.Info("Context cancelled, shutting down...")
	}

	logger.Info("Server stopped")
}
