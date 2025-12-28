package http

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/mrops-br/testing-otlp-api/internal/infrastructure/config"
	"github.com/mrops-br/testing-otlp-api/internal/infrastructure/http/handler"
	"github.com/mrops-br/testing-otlp-api/internal/infrastructure/http/middleware"
	"github.com/mrops-br/testing-otlp-api/internal/infrastructure/telemetry"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Server represents the HTTP server
type Server struct {
	router    *chi.Mux
	config    *config.ServerConfig
	handler   *handler.ProductHandler
	tracer    trace.Tracer
	logger    *slog.Logger
	telemetry *telemetry.Telemetry
}

// NewServer creates a new HTTP server
func NewServer(
	cfg *config.ServerConfig,
	handler *handler.ProductHandler,
	tracer trace.Tracer,
	logger *slog.Logger,
	telem *telemetry.Telemetry,
) *Server {
	s := &Server{
		router:    chi.NewRouter(),
		config:    cfg,
		handler:   handler,
		tracer:    tracer,
		logger:    logger,
		telemetry: telem,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// setupMiddleware configures the middleware chain
func (s *Server) setupMiddleware() {
	// Structured JSON logging middleware (replaces chimiddleware.Logger)
	s.router.Use(middleware.StructuredLogger(s.logger))
	s.router.Use(chimiddleware.Recoverer)
	s.router.Use(chimiddleware.RequestID)

	// Add HTTP route to context so all logs include it automatically
	s.router.Use(middleware.HTTPRouteContext())

	// Add OpenTelemetry active requests tracking
	meter := s.telemetry.MeterProvider.Meter("products-api")
	s.router.Use(middleware.ActiveRequestsMiddleware(meter))

	// OPTIONAL: Add custom milliseconds duration metric (in addition to standard seconds metric)
	// Uncomment the line below if you prefer milliseconds-based duration metrics
	// s.router.Use(middleware.DurationMillisecondsMiddleware(meter))
}

// setupRoutes configures the API routes
func (s *Server) setupRoutes() {
	s.router.Route("/products", func(r chi.Router) {
		r.Post("/", s.handler.CreateProduct)
		r.Get("/", s.handler.ListProducts)
		r.Get("/{id}", s.handler.GetProduct)
	})

	// Health check endpoint
	s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = w.Write([]byte("OK"))
	})

	// Prometheus metrics endpoint - exposes OpenTelemetry metrics
	s.router.Get("/metrics", promhttp.Handler().ServeHTTP)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%s", s.config.Host, s.config.Port)
	s.logger.Info("Starting HTTP server",
		slog.String("address", addr),
	)

	// Wrap the entire router with otelhttp for automatic HTTP metrics and tracing
	// This provides: http.server.request.duration, http.server.request.body.size, etc.
	handler := otelhttp.NewHandler(s.router, "http-server",
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		}),
		otelhttp.WithMeterProvider(s.telemetry.MeterProvider),
		// Add route pattern to metrics attributes
		otelhttp.WithMetricAttributesFn(func(r *http.Request) []attribute.KeyValue {
			// Extract route pattern from Chi context
			routePattern := r.URL.Path
			if rctx := chi.RouteContext(r.Context()); rctx != nil {
				if pattern := rctx.RoutePattern(); pattern != "" {
					routePattern = pattern
				}
			}
			return []attribute.KeyValue{
				attribute.String("http.route", routePattern),
			}
		}),
	)

	return http.ListenAndServe(addr, handler)
}
