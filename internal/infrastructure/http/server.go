package http

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/mrops-br/optl-testing-api/internal/infrastructure/config"
	"github.com/mrops-br/optl-testing-api/internal/infrastructure/http/handler"
	"github.com/mrops-br/optl-testing-api/internal/infrastructure/http/middleware"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Server represents the HTTP server
type Server struct {
	router  *chi.Mux
	config  *config.ServerConfig
	handler *handler.ProductHandler
	tracer  trace.Tracer
	meter   metric.Meter
	logger  *slog.Logger
}

// NewServer creates a new HTTP server
func NewServer(
	cfg *config.ServerConfig,
	handler *handler.ProductHandler,
	tracer trace.Tracer,
	meter metric.Meter,
	logger *slog.Logger,
) *Server {
	s := &Server{
		router:  chi.NewRouter(),
		config:  cfg,
		handler: handler,
		tracer:  tracer,
		meter:   meter,
		logger:  logger,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// setupMiddleware configures the middleware chain
func (s *Server) setupMiddleware() {
	s.router.Use(chimiddleware.Logger)
	s.router.Use(chimiddleware.Recoverer)
	s.router.Use(chimiddleware.RequestID)
	s.router.Use(middleware.TracingMiddleware(s.tracer))
	s.router.Use(middleware.MetricsMiddleware(s.meter))
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
		w.Write([]byte("OK"))
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%s", s.config.Host, s.config.Port)
	s.logger.Info("Starting HTTP server",
		slog.String("address", addr),
	)

	return http.ListenAndServe(addr, s.router)
}
