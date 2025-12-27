package service

import (
	"context"
	"log/slog"

	"github.com/mrops-br/testing-otlp-api/internal/app/dto"
	"github.com/mrops-br/testing-otlp-api/internal/domain"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// ProductService handles product use cases
type ProductService struct {
	repo                  domain.ProductRepository
	tracer                trace.Tracer
	logger                *slog.Logger
	productCreatedCounter metric.Int64Counter
	productOperations     metric.Int64Counter
}

// NewProductService creates a new product service
func NewProductService(
	repo domain.ProductRepository,
	tracer trace.Tracer,
	meter metric.Meter,
	logger *slog.Logger,
) *ProductService {
	// Initialize metrics
	productCreatedCounter, _ := meter.Int64Counter(
		"products.created.total",
		metric.WithDescription("Total number of products created"),
	)

	productOperations, _ := meter.Int64Counter(
		"products.operations",
		metric.WithDescription("Total number of product operations"),
	)

	return &ProductService{
		repo:                  repo,
		tracer:                tracer,
		logger:                logger,
		productCreatedCounter: productCreatedCounter,
		productOperations:     productOperations,
	}
}

// CreateProduct creates a new product
func (s *ProductService) CreateProduct(ctx context.Context, req *dto.CreateProductRequest) (*dto.ProductResponse, error) {
	ctx, span := s.tracer.Start(ctx, "ProductService.CreateProduct")
	defer span.End()

	span.SetAttributes(
		attribute.String("product.name", req.Name),
		attribute.Float64("product.price", req.Price),
	)

	s.logger.InfoContext(ctx, "Creating product",
		slog.String("name", req.Name),
		slog.Float64("price", req.Price),
	)

	// Create domain entity
	product, err := domain.NewProduct(req.Name, req.Description, req.Price)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Validation failed")
		s.logger.ErrorContext(ctx, "Failed to create product",
			slog.String("error", err.Error()),
		)
		s.productOperations.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("operation", "create"),
				attribute.String("result", "failure"),
			),
		)
		return nil, err
	}

	span.SetAttributes(attribute.String("product.id", product.ID))

	// Store in repository
	if err := s.repo.Create(ctx, product); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to store product")
		s.logger.ErrorContext(ctx, "Failed to store product",
			slog.String("error", err.Error()),
		)
		s.productOperations.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("operation", "create"),
				attribute.String("result", "failure"),
			),
		)
		return nil, err
	}

	// Record metrics
	s.productCreatedCounter.Add(ctx, 1)
	s.productOperations.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("operation", "create"),
			attribute.String("result", "success"),
		),
	)

	s.logger.InfoContext(ctx, "Product created successfully",
		slog.String("product_id", product.ID),
	)

	span.SetStatus(codes.Ok, "Product created successfully")
	return dto.ToProductResponse(product), nil
}

// GetProductByID retrieves a product by ID
func (s *ProductService) GetProductByID(ctx context.Context, id string) (*dto.ProductResponse, error) {
	ctx, span := s.tracer.Start(ctx, "ProductService.GetProductByID")
	defer span.End()

	span.SetAttributes(attribute.String("product.id", id))

	s.logger.InfoContext(ctx, "Getting product by ID",
		slog.String("product_id", id),
	)

	product, err := s.repo.FindByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Product not found")
		s.logger.WarnContext(ctx, "Product not found",
			slog.String("product_id", id),
		)
		s.productOperations.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("operation", "read"),
				attribute.String("result", "not_found"),
			),
		)
		return nil, err
	}

	s.productOperations.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("operation", "read"),
			attribute.String("result", "success"),
		),
	)

	s.logger.InfoContext(ctx, "Product retrieved successfully",
		slog.String("product_id", id),
	)

	span.SetStatus(codes.Ok, "Product retrieved successfully")
	return dto.ToProductResponse(product), nil
}

// ListProducts retrieves all products
func (s *ProductService) ListProducts(ctx context.Context) ([]*dto.ProductResponse, error) {
	ctx, span := s.tracer.Start(ctx, "ProductService.ListProducts")
	defer span.End()

	s.logger.InfoContext(ctx, "Listing all products")

	products, err := s.repo.FindAll(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to retrieve products")
		s.logger.ErrorContext(ctx, "Failed to list products",
			slog.String("error", err.Error()),
		)
		s.productOperations.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("operation", "list"),
				attribute.String("result", "failure"),
			),
		)
		return nil, err
	}

	span.SetAttributes(attribute.Int("product.count", len(products)))

	s.productOperations.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("operation", "list"),
			attribute.String("result", "success"),
		),
	)

	s.logger.InfoContext(ctx, "Products listed successfully",
		slog.Int("count", len(products)),
	)

	span.SetStatus(codes.Ok, "Products listed successfully")
	return dto.ToProductResponseList(products), nil
}
