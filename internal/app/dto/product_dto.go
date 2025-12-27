package dto

import (
	"time"

	"github.com/mrops-br/testing-otlp-api/internal/domain"
)

// CreateProductRequest represents the request to create a product
type CreateProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

// ProductResponse represents the product response
type ProductResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToProductResponse converts a domain Product to ProductResponse
func ToProductResponse(p *domain.Product) *ProductResponse {
	return &ProductResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// ToProductResponseList converts a list of domain Products to ProductResponse list
func ToProductResponseList(products []*domain.Product) []*ProductResponse {
	responses := make([]*ProductResponse, len(products))
	for i, p := range products {
		responses[i] = ToProductResponse(p)
	}
	return responses
}
