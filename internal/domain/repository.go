package domain

import (
	"context"
	"errors"
)

var (
	ErrProductNotFound = errors.New("product not found")
)

// ProductRepository defines the contract for product storage
type ProductRepository interface {
	Create(ctx context.Context, product *Product) error
	FindByID(ctx context.Context, id string) (*Product, error)
	FindAll(ctx context.Context) ([]*Product, error)
}
