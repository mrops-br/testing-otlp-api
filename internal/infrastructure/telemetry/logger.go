package telemetry

import (
	"context"
	"log/slog"
	"os"

	"github.com/mrops-br/testing-otlp-api/internal/infrastructure/config"
	"go.opentelemetry.io/otel/trace"
)

// Context key for storing HTTP route
type contextKey string

const httpRouteKey contextKey = "http.route"

// WithHTTPRoute adds the HTTP route to the context
func WithHTTPRoute(ctx context.Context, route string) context.Context {
	return context.WithValue(ctx, httpRouteKey, route)
}

// HTTPRouteFromContext extracts the HTTP route from context
func HTTPRouteFromContext(ctx context.Context) string {
	if route, ok := ctx.Value(httpRouteKey).(string); ok {
		return route
	}
	return ""
}

// traceContextHandler is a custom slog handler that injects trace context
type traceContextHandler struct {
	handler slog.Handler
}

// Enabled reports whether the handler handles records at the given level
func (h *traceContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// Handle adds trace_id, span_id, and http.route to log records from the context
func (h *traceContextHandler) Handle(ctx context.Context, r slog.Record) error {
	// Add trace context if available
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		r.AddAttrs(
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)
	}

	// Add HTTP route if available in context
	if route := HTTPRouteFromContext(ctx); route != "" {
		r.AddAttrs(slog.String("http.route", route))
	}

	return h.handler.Handle(ctx, r)
}

// WithAttrs returns a new handler with additional attributes
func (h *traceContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceContextHandler{
		handler: h.handler.WithAttrs(attrs),
	}
}

// WithGroup returns a new handler with the given group name
func (h *traceContextHandler) WithGroup(name string) slog.Handler {
	return &traceContextHandler{
		handler: h.handler.WithGroup(name),
	}
}

// initLogger initializes a structured logger with trace context injection
func initLogger(cfg *config.OTLPConfig) *slog.Logger {
	// Create JSON handler for structured logging
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	jsonHandler := slog.NewJSONHandler(os.Stdout, opts)

	// Wrap with trace context handler
	handler := &traceContextHandler{handler: jsonHandler}

	logger := slog.New(handler).With(
		slog.String("service.name", cfg.ServiceName),
		slog.String("environment", cfg.Environment),
	)

	return logger
}
