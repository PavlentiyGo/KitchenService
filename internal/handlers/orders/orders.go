package handler_orders

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
	ordersService ordersService
	validator     request.Validator
}

func NewHandler(service1 ordersService, validator request.Validator) *Handler {
	return &Handler{ordersService: service1, validator: validator}
}

func (h *Handler) Routes() []core_server.Route {
	return []core_server.Route{
		{
			Method:  "POST",
			Path:    "/companies/{companyId}/orders",
			Handler: h.PostOrder,
			Middlewares: []http_middleware.Middleware{
				http_middleware.SimpleAuth(),
			},
		},
		{
			Method:  "GET",
			Path:    "/orders/my",
			Handler: h.GetMyOrders,
			Middlewares: []http_middleware.Middleware{
				http_middleware.SimpleAuth(),
			},
		},
		{
			Method:  "GET",
			Path:    "/orders/{orderId}",
			Handler: h.GetOrder,
			Middlewares: []http_middleware.Middleware{
				http_middleware.SimpleAuth(),
			},
		},
	}
}

// PostOrder creates an order from the current user basket.
// @Summary Create order
// @Description Creates an order after validating the basket contents and confirmed prices.
// @Tags Orders
// @Accept json
// @Produce json
// @Param Authorization header string true "User UUID" format(uuid)
// @Param companyId path string true "Company UUID" format(uuid)
// @Param request body dto.CreateOrderRequest true "Products and confirmed prices"
// @Success 201 "Created"
// @Failure 400 {object} core_response.ErrorResponse
// @Failure 401 {object} core_response.ErrorResponse
// @Failure 404 {object} core_response.ErrorResponse
// @Failure 409 {object} core_response.ErrorResponse
// @Failure 500 {object} core_response.ErrorResponse
// @Router /api/v1/companies/{companyId}/orders [post]
func (h *Handler) PostOrder(w *core_response.Writer, r *http.Request) {
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

	var body dto.CreateOrderRequest
	err = request.DecodeAndValidate(r, &body, h.validator)
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	products := make([]domain.Product, len(body.Products))
	for i, product := range body.Products {
		productId, err := uuid.Parse(product.Id)
		if err != nil {
			w.ErrorResponse(fmt.Errorf(
				"invalid product id %q: %w",
				product.Id,
				core_errors.ErrInvalidRequest,
			))
			return
		}

		products[i] = domain.Product{
			Id:    productId,
			Price: product.Price,
		}
	}

	err = h.ordersService.CreateOrder(ctx, userId, companyId, products)
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// GetMyOrders returns all orders of the current user.
// @Summary List my orders
// @Description Returns all orders belonging to the authenticated user.
// @Tags Orders
// @Produce json
// @Param Authorization header string true "User UUID" format(uuid)
// @Success 200 {object} dto.GetMyOrdersResponse
// @Failure 400 {object} core_response.ErrorResponse
// @Failure 401 {object} core_response.ErrorResponse
// @Failure 404 {object} core_response.ErrorResponse
// @Failure 500 {object} core_response.ErrorResponse
// @Router /api/v1/orders/my [get]
func (h *Handler) GetMyOrders(w *core_response.Writer, r *http.Request) {
	userId, err := http_middleware.GetUserIdFromCtx(r.Context())
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	orders, err := h.ordersService.GetMyOrders(r.Context(), userId)
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	w.WriteJson(http.StatusOK, dto.GetMyOrdersResponseFromDomains(orders))
}

// GetOrder returns an order of the current user.
// @Summary Get my order
// @Description Returns an order when it belongs to the authenticated user.
// @Tags Orders
// @Produce json
// @Param Authorization header string true "User UUID" format(uuid)
// @Param orderId path string true "Order UUID" format(uuid)
// @Success 200 {object} dto.OrderDto
// @Failure 400 {object} core_response.ErrorResponse
// @Failure 401 {object} core_response.ErrorResponse
// @Failure 403 {object} core_response.ErrorResponse
// @Failure 404 {object} core_response.ErrorResponse
// @Failure 500 {object} core_response.ErrorResponse
// @Router /api/v1/orders/{orderId} [get]
func (h *Handler) GetOrder(w *core_response.Writer, r *http.Request) {
	userId, err := http_middleware.GetUserIdFromCtx(r.Context())
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	orderId, err := request.ParseUUIDPathParam(r, "orderId")
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	order, err := h.ordersService.GetUserOrder(r.Context(), userId, orderId)
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	w.WriteJson(http.StatusOK, dto.OrderDtoFromDomain(order))
}
