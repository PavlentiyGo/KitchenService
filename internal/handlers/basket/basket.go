package handler_basket

import (
	"avitoKitchen/internal/core/domain"
	core_errors "avitoKitchen/internal/core/errors"
	http_middleware "avitoKitchen/internal/core/http/middleware"
	"avitoKitchen/internal/core/http/request"
	core_response "avitoKitchen/internal/core/http/response"
	core_server "avitoKitchen/internal/core/http/server"
	"avitoKitchen/internal/handlers/dto"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

type Handler struct {
	basketService basketService
	validator     request.Validator
}

func NewHandler(service1 basketService, validator request.Validator) *Handler {
	return &Handler{basketService: service1, validator: validator}
}

func (h *Handler) Routes() []core_server.Route {
	return []core_server.Route{
		{
			Method:  "GET",
			Path:    "/companies/{companyId}/basket",
			Handler: h.GetBasket,
			Middlewares: []http_middleware.Middleware{
				http_middleware.SimpleAuth(),
			},
		},
		{
			Method:  "PUT",
			Path:    "/companies/{companyId}/basket/items/{productId}",
			Handler: h.PutBasketProduct,
			Middlewares: []http_middleware.Middleware{
				http_middleware.SimpleAuth(),
			},
		},
		{
			Method:  "DELETE",
			Path:    "/companies/{companyId}/basket/items/{productId}",
			Handler: h.DeleteBasketProduct,
			Middlewares: []http_middleware.Middleware{
				http_middleware.SimpleAuth(),
			},
		},
		{
			Method:  "PATCH",
			Path:    "/companies/{companyId}/basket/price-confirmation",
			Handler: h.ConfirmProductsPrice,
			Middlewares: []http_middleware.Middleware{
				http_middleware.SimpleAuth(),
			},
		},
	}
}

// GetBasket returns the current user basket for a company.
// @Summary Get basket
// @Description Returns the authenticated user basket and its current and confirmed totals.
// @Tags Basket
// @Produce json
// @Param Authorization header string true "User UUID" format(uuid)
// @Param companyId path string true "Company UUID" format(uuid)
// @Success 200 {object} dto.UserBasketDto
// @Failure 400 {object} core_response.ErrorResponse
// @Failure 401 {object} core_response.ErrorResponse
// @Failure 404 {object} core_response.ErrorResponse
// @Failure 500 {object} core_response.ErrorResponse
// @Router /api/v1/companies/{companyId}/basket [get]
func (h *Handler) GetBasket(w *core_response.Writer, r *http.Request) {
	ctx := r.Context()
	userId, err := http_middleware.GetUserIdFromCtx(ctx)
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	companyId, err := request.ParseUUIDPathParam(r, "companyId")
	if err != nil {
		w.ErrorResponse(err)
		return
	}
	basket, err := h.basketService.GetBasket(ctx, userId, companyId)
	if err != nil {
		w.ErrorResponse(err)
		return
	}
	w.WriteJson(http.StatusOK, dto.BasketDtoFromDomain(basket))
}

// PutBasketProduct adds a product to the current user basket or changes its quantity.
// @Summary Add or update basket product
// @Description Sets the requested quantity of a product in the authenticated user basket.
// @Tags Basket
// @Accept json
// @Produce json
// @Param Authorization header string true "User UUID" format(uuid)
// @Param companyId path string true "Company UUID" format(uuid)
// @Param productId path string true "Product UUID" format(uuid)
// @Param request body dto.PutBasketProductRequest true "Product quantity"
// @Success 204 "No Content"
// @Failure 400 {object} core_response.ErrorResponse
// @Failure 401 {object} core_response.ErrorResponse
// @Failure 404 {object} core_response.ErrorResponse
// @Failure 409 {object} core_response.ErrorResponse
// @Failure 500 {object} core_response.ErrorResponse
// @Router /api/v1/companies/{companyId}/basket/items/{productId} [put]
func (h *Handler) PutBasketProduct(w *core_response.Writer, r *http.Request) {
	ctx := r.Context()
	userId, err := http_middleware.GetUserIdFromCtx(ctx)
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	companyId, err := request.ParseUUIDPathParam(r, "companyId")
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	productId, err := request.ParseUUIDPathParam(r, "productId")
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	var body dto.PutBasketProductRequest
	err = request.DecodeAndValidate(r, &body, h.validator)
	if err != nil {
		w.ErrorResponse(err)
		return
	}
	err = h.basketService.PutBasketProduct(ctx, userId, companyId, productId, body.Count)
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteBasketProduct removes a product from the current user basket.
// @Summary Delete basket product
// @Description Removes the specified product from the authenticated user basket.
// @Tags Basket
// @Produce json
// @Param Authorization header string true "User UUID" format(uuid)
// @Param companyId path string true "Company UUID" format(uuid)
// @Param productId path string true "Product UUID" format(uuid)
// @Success 204 "No Content"
// @Failure 400 {object} core_response.ErrorResponse
// @Failure 401 {object} core_response.ErrorResponse
// @Failure 404 {object} core_response.ErrorResponse
// @Failure 500 {object} core_response.ErrorResponse
// @Router /api/v1/companies/{companyId}/basket/items/{productId} [delete]
func (h *Handler) DeleteBasketProduct(w *core_response.Writer, r *http.Request) {
	ctx := r.Context()
	userId, err := http_middleware.GetUserIdFromCtx(ctx)
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	companyId, err := request.ParseUUIDPathParam(r, "companyId")
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	productId, err := request.ParseUUIDPathParam(r, "productId")
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	err = h.basketService.DeleteBasketProduct(ctx, userId, companyId, productId)
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	w.WriteHeader(http.StatusNoContent)

}

// ConfirmProductsPrice confirms the current prices of products in the basket.
// @Summary Confirm basket prices
// @Description Confirms current prices and returns the updated basket products.
// @Tags Basket
// @Accept json
// @Produce json
// @Param Authorization header string true "User UUID" format(uuid)
// @Param companyId path string true "Company UUID" format(uuid)
// @Param request body dto.ConfirmProductsPriceRequest true "Products and expected prices"
// @Success 200 {array} dto.BasketProductDto
// @Failure 400 {object} core_response.ErrorResponse
// @Failure 401 {object} core_response.ErrorResponse
// @Failure 404 {object} core_response.ErrorResponse
// @Failure 409 {object} core_response.ErrorResponse
// @Failure 500 {object} core_response.ErrorResponse
// @Router /api/v1/companies/{companyId}/basket/price-confirmation [patch]
func (h *Handler) ConfirmProductsPrice(w *core_response.Writer, r *http.Request) {
	ctx := r.Context()
	userId, err := http_middleware.GetUserIdFromCtx(ctx)
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	companyId, err := request.ParseUUIDPathParam(r, "companyId")
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	var body dto.ConfirmProductsPriceRequest
	err = request.DecodeAndValidate(r, &body, h.validator)
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	productsForChange := make([]domain.Product, len(body.Products))
	for ind, productDto := range body.Products {
		id, err := uuid.Parse(productDto.Id)
		if err != nil {
			w.ErrorResponse(fmt.Errorf("wrong id: %s: %w", productDto.Id, core_errors.ErrInvalidRequest))
			return
		}
		productsForChange[ind] = domain.Product{
			Id:    id,
			Price: productDto.Price,
		}
	}
	basket, err := h.basketService.ConfirmBasketProducts(
		ctx,
		productsForChange,
		companyId,
		userId,
	)
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	w.WriteJson(http.StatusOK, dto.BasketProductsDtoFromDomains(basket))
}
