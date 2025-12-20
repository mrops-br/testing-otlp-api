package middleware

import (
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// TracingMiddleware adds tracing to HTTP requests
func TracingMiddleware(tracer trace.Tracer) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracer.Start(r.Context(), "HTTP "+r.Method+" "+r.URL.Path)
			defer span.End()

			span.SetAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
				attribute.String("http.route", r.URL.Path),
				attribute.String("http.user_agent", r.UserAgent()),
			)

			// Create a custom response writer to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r.WithContext(ctx))

			span.SetAttributes(attribute.Int("http.status_code", rw.statusCode))
		})
	}
}

// MetricsMiddleware adds metrics to HTTP requests
func MetricsMiddleware(meter metric.Meter) func(next http.Handler) http.Handler {
	// Initialize metrics
	requestCounter, _ := meter.Int64Counter(
		"http.server.request.count",
		metric.WithDescription("Total number of HTTP requests"),
	)

	requestDuration, _ := meter.Int64Histogram(
		"http.server.request.duration",
		metric.WithDescription("HTTP request duration in milliseconds"),
		metric.WithUnit("ms"),
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a custom response writer to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r)

			duration := time.Since(start).Milliseconds()

			// Record metrics
			attrs := []attribute.KeyValue{
				attribute.String("http.method", r.Method),
				attribute.String("http.route", r.URL.Path),
				attribute.Int("http.status_code", rw.statusCode),
			}

			requestCounter.Add(r.Context(), 1, metric.WithAttributes(attrs...))
			requestDuration.Record(r.Context(), duration, metric.WithAttributes(attrs...))
		})
	}
}

// responseWriter is a custom response writer that captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
