package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidProductName  = errors.New("product name is required")
	ErrInvalidProductPrice = errors.New("product price must be positive")
)

// Product represents the product entity
type Product struct {
	ID          string
	Name        string
	Description string
	Price       float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewProduct creates a new product with validation
func NewProduct(name, description string, price float64) (*Product, error) {
	product := &Product{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Price:       price,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := product.Validate(); err != nil {
		return nil, err
	}

	return product, nil
}

// Validate performs business validation on the product
func (p *Product) Validate() error {
	if p.Name == "" {
		return ErrInvalidProductName
	}
	if p.Price <= 0 {
		return ErrInvalidProductPrice
	}
	return nil
}
