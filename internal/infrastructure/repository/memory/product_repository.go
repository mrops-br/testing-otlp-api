package memory

import (
	"context"
	"log/slog"
	"sync"

	"github.com/mrops-br/optl-testing-api/internal/domain"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ProductRepository is an in-memory implementation of domain.ProductRepository
type ProductRepository struct {
	mu       sync.RWMutex
	products map[string]*domain.Product
	tracer   trace.Tracer
	logger   *slog.Logger
}

// NewProductRepository creates a new in-memory product repository
func NewProductRepository(tracer trace.Tracer, logger *slog.Logger) *ProductRepository {
	return &ProductRepository{
		products: make(map[string]*domain.Product),
		tracer:   tracer,
		logger:   logger,
	}
}

// Create stores a new product
func (r *ProductRepository) Create(ctx context.Context, product *domain.Product) error {
	ctx, span := r.tracer.Start(ctx, "ProductRepository.Create")
	defer span.End()

	span.SetAttributes(
		attribute.String("product.id", product.ID),
		attribute.String("product.name", product.Name),
	)

	r.mu.Lock()
	defer r.mu.Unlock()

	r.products[product.ID] = product

	r.logger.InfoContext(ctx, "Product created in repository",
		slog.String("product_id", product.ID),
		slog.String("product_name", product.Name),
	)

	span.SetStatus(codes.Ok, "Product created successfully")
	return nil
}

// FindByID retrieves a product by ID
func (r *ProductRepository) FindByID(ctx context.Context, id string) (*domain.Product, error) {
	ctx, span := r.tracer.Start(ctx, "ProductRepository.FindByID")
	defer span.End()

	span.SetAttributes(attribute.String("product.id", id))

	r.mu.RLock()
	defer r.mu.RUnlock()

	product, exists := r.products[id]
	if !exists {
		span.RecordError(domain.ErrProductNotFound)
		span.SetStatus(codes.Error, "Product not found")
		r.logger.WarnContext(ctx, "Product not found",
			slog.String("product_id", id),
		)
		return nil, domain.ErrProductNotFound
	}

	r.logger.DebugContext(ctx, "Product found in repository",
		slog.String("product_id", id),
		slog.String("product_name", product.Name),
	)

	span.SetStatus(codes.Ok, "Product found")
	return product, nil
}

// FindAll retrieves all products
func (r *ProductRepository) FindAll(ctx context.Context) ([]*domain.Product, error) {
	ctx, span := r.tracer.Start(ctx, "ProductRepository.FindAll")
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	products := make([]*domain.Product, 0, len(r.products))
	for _, product := range r.products {
		products = append(products, product)
	}

	span.SetAttributes(attribute.Int("product.count", len(products)))

	r.logger.InfoContext(ctx, "Products retrieved from repository",
		slog.Int("count", len(products)),
	)

	span.SetStatus(codes.Ok, "Products retrieved successfully")
	return products, nil
}
