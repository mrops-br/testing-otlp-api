package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mrops-br/testing-otlp-api/internal/infrastructure/telemetry"
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

// ActiveRequestsMiddleware tracks active HTTP requests using OpenTelemetry metrics
// This middleware should be registered AFTER routing middleware to have access to route patterns
func ActiveRequestsMiddleware(meter metric.Meter) func(next http.Handler) http.Handler {
	// Create an UpDownCounter for tracking active requests
	activeRequests, err := meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithDescription("Number of active HTTP server requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		// If metric creation fails, return a pass-through middleware
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a custom response writer to extract route after handler processes request
			wrapper := &routeAwareWriter{
				ResponseWriter: w,
				request:        r,
				activeRequests: activeRequests,
			}

			// Process the request - route will be available after otelhttp.WithRouteTag processes it
			next.ServeHTTP(wrapper, r)

			// Ensure decrement happens even if Write/WriteHeader were never called
			wrapper.ensureDecrement()
		})
	}
}

// routeAwareWriter captures the route and tracks active requests
type routeAwareWriter struct {
	http.ResponseWriter
	request        *http.Request
	activeRequests metric.Int64UpDownCounter
	incrementDone  bool
	decrementDone  bool
}

func (w *routeAwareWriter) WriteHeader(statusCode int) {
	w.incrementIfNeeded()
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *routeAwareWriter) Write(b []byte) (int, error) {
	w.incrementIfNeeded()
	return w.ResponseWriter.Write(b)
}

func (w *routeAwareWriter) incrementIfNeeded() {
	if w.incrementDone {
		return
	}
	w.incrementDone = true

	// Extract route pattern - try Chi context first, then fall back to URL path
	routePattern := w.request.URL.Path
	if rctx := chi.RouteContext(w.request.Context()); rctx != nil {
		if pattern := rctx.RoutePattern(); pattern != "" {
			routePattern = pattern
		}
	}

	// Create attributes for the metric
	attrs := []attribute.KeyValue{
		attribute.String("http.request.method", w.request.Method),
		attribute.String("http.route", routePattern),
		attribute.String("server.address", w.request.Host),
	}

	// Increment active requests
	w.activeRequests.Add(w.request.Context(), 1, metric.WithAttributes(attrs...))
}

func (w *routeAwareWriter) ensureDecrement() {
	if w.decrementDone {
		return
	}
	w.decrementDone = true

	// Only decrement if we actually incremented
	if !w.incrementDone {
		w.incrementIfNeeded()
	}

	// Extract route pattern (same logic as increment)
	routePattern := w.request.URL.Path
	if rctx := chi.RouteContext(w.request.Context()); rctx != nil {
		if pattern := rctx.RoutePattern(); pattern != "" {
			routePattern = pattern
		}
	}

	// Create attributes for the metric (must match increment attributes exactly)
	attrs := []attribute.KeyValue{
		attribute.String("http.request.method", w.request.Method),
		attribute.String("http.route", routePattern),
		attribute.String("server.address", w.request.Host),
	}

	// Decrement active requests
	w.activeRequests.Add(w.request.Context(), -1, metric.WithAttributes(attrs...))
}


// DurationMillisecondsMiddleware records HTTP request duration in milliseconds
// This is a custom metric in addition to the standard OTel seconds-based metric
func DurationMillisecondsMiddleware(meter metric.Meter) func(next http.Handler) http.Handler {
	// Create a histogram for duration in milliseconds
	durationHistogram, err := meter.Float64Histogram(
		"http.server.request.duration.ms",
		metric.WithDescription("HTTP server request duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		// If metric creation fails, return a pass-through middleware
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Process the request
			next.ServeHTTP(rw, r)

			// Calculate duration in milliseconds
			duration := float64(time.Since(start).Milliseconds())

			// Extract route pattern from Chi context
			routePattern := r.URL.Path
			if rctx := chi.RouteContext(r.Context()); rctx != nil {
				if pattern := rctx.RoutePattern(); pattern != "" {
					routePattern = pattern
				}
			}

			// Record the metric
			durationHistogram.Record(r.Context(), duration,
				metric.WithAttributes(
					attribute.String("http.request.method", r.Method),
					attribute.String("http.route", routePattern),
					attribute.Int("http.response.status_code", rw.statusCode),
					attribute.String("server.address", r.Host),
				),
			)
		})
	}
}

// HTTPRouteContext adds the HTTP route pattern to the request context
// This allows all logs during request processing to include the http.route attribute
func HTTPRouteContext() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract route pattern from Chi context
			routePattern := r.URL.Path
			if rctx := chi.RouteContext(r.Context()); rctx != nil {
				if pattern := rctx.RoutePattern(); pattern != "" {
					routePattern = pattern
				}
			}

			// Add route to context so it's available in all logs
			ctx := telemetry.WithHTTPRoute(r.Context(), routePattern)

			// Continue with enriched context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// StructuredLogger creates a structured JSON logging middleware
// This replaces Chi's default logger to maintain consistent JSON log format
func StructuredLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status and bytes written
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Process request
			next.ServeHTTP(ww, r)

			// Calculate duration
			duration := time.Since(start)

			// Extract trace context from span (if available)
			span := trace.SpanFromContext(r.Context())
			spanCtx := span.SpanContext()

			// Extract route pattern
			routePattern := r.URL.Path
			if rctx := chi.RouteContext(r.Context()); rctx != nil {
				if pattern := rctx.RoutePattern(); pattern != "" {
					routePattern = pattern
				}
			}

			// Build log attributes
			attrs := []any{
				slog.String("http.request.method", r.Method),
				slog.String("http.route", routePattern),
				slog.String("url.path", r.URL.Path),
				slog.String("url.query", r.URL.RawQuery),
				slog.Int("http.response.status_code", ww.Status()),
				slog.Int("http.response.body.size", ww.BytesWritten()),
				slog.String("duration", duration.String()),
				slog.Float64("duration_ms", float64(duration.Milliseconds())),
				slog.String("client.address", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
			}

			// Add trace context if available
			if spanCtx.IsValid() {
				attrs = append(attrs,
					slog.String("trace_id", spanCtx.TraceID().String()),
					slog.String("span_id", spanCtx.SpanID().String()),
				)
			}

			// Log at appropriate level based on status code
			logLevel := slog.LevelInfo
			if ww.Status() >= 500 {
				logLevel = slog.LevelError
			} else if ww.Status() >= 400 {
				logLevel = slog.LevelWarn
			}

			logger.Log(r.Context(), logLevel, "HTTP request completed", attrs...)
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
