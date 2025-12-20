package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mrops-br/optl-testing-api/internal/app/dto"
	"github.com/mrops-br/optl-testing-api/internal/app/service"
	"github.com/mrops-br/optl-testing-api/internal/domain"
	"github.com/mrops-br/optl-testing-api/internal/infrastructure/http/response"
)

// ProductHandler handles HTTP requests for products
type ProductHandler struct {
	service *service.ProductService
	logger  *slog.Logger
}

// NewProductHandler creates a new product handler
func NewProductHandler(service *service.ProductService, logger *slog.Logger) *ProductHandler {
	return &ProductHandler{
		service: service,
		logger:  logger,
	}
}

// CreateProduct handles POST /products
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.ErrorContext(r.Context(), "Failed to decode request body",
			slog.String("error", err.Error()),
		)
		response.Error(w, http.StatusBadRequest, err)
		return
	}

	product, err := h.service.CreateProduct(r.Context(), &req)
	if err != nil {
		switch err {
		case domain.ErrInvalidProductName, domain.ErrInvalidProductPrice:
			response.Error(w, http.StatusBadRequest, err)
		default:
			response.Error(w, http.StatusInternalServerError, err)
		}
		return
	}

	response.JSON(w, http.StatusCreated, product)
}

// GetProduct handles GET /products/{id}
func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	product, err := h.service.GetProductByID(r.Context(), id)
	if err != nil {
		if err == domain.ErrProductNotFound {
			response.Error(w, http.StatusNotFound, err)
		} else {
			response.Error(w, http.StatusInternalServerError, err)
		}
		return
	}

	response.JSON(w, http.StatusOK, product)
}

// ListProducts handles GET /products
func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.service.ListProducts(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err)
		return
	}

	response.JSON(w, http.StatusOK, products)
}
