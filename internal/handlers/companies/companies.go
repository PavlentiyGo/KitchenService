package handler_companies

import (
	"avitoKitchen/internal/core/domain"
	http_middleware "avitoKitchen/internal/core/http/middleware"
	"avitoKitchen/internal/core/http/request"
	core_response "avitoKitchen/internal/core/http/response"
	core_server "avitoKitchen/internal/core/http/server"
	"avitoKitchen/internal/handlers/dto"
	"net/http"
)

type Handler struct {
	companiesService companiesService
	validator        request.Validator
}

func NewHandler(s companiesService, validator request.Validator) *Handler {
	return &Handler{companiesService: s, validator: validator}
}

func (h *Handler) Routes() []core_server.Route {
	return []core_server.Route{
		{
			Method:  "POST",
			Path:    "/company/products",
			Handler: h.AddNewProduct,
			Middlewares: []http_middleware.Middleware{
				http_middleware.SimpleCompanyAuth(),
			},
		},
		{
			Method:  "PATCH",
			Path:    "/company/products/{partnerProductId}",
			Handler: h.PatchCompanyProduct,
			Middlewares: []http_middleware.Middleware{
				http_middleware.SimpleCompanyAuth(),
			},
		},
		{
			Method:  "PATCH",
			Path:    "/company/orders/{orderId}/status",
			Handler: h.ChangeOrderStatus,
			Middlewares: []http_middleware.Middleware{
				http_middleware.SimpleCompanyAuth(),
			},
		},
	}
}

// AddNewProduct creates a product for the authenticated company.
// @Summary Create company product
// @Description Creates a product owned by the company identified by the Authorization header.
// @Tags Company
// @Accept json
// @Produce json
// @Param Authorization header string true "Company UUID" format(uuid)
// @Param request body dto.CreateCompanyProductRequest true "Product data"
// @Success 201 {object} dto.CompanyProductDto
// @Failure 400 {object} core_response.ErrorResponse
// @Failure 401 {object} core_response.ErrorResponse
// @Failure 404 {object} core_response.ErrorResponse
// @Failure 409 {object} core_response.ErrorResponse
// @Failure 500 {object} core_response.ErrorResponse
// @Router /api/v1/company/products [post]
func (h *Handler) AddNewProduct(w *core_response.Writer, r *http.Request) {
	companyId, err := http_middleware.GetCompanyIdFromCtx(r.Context())
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	var body dto.CreateCompanyProductRequest
	err = request.DecodeAndValidate(r, &body, h.validator)
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	partnerProductId := body.PartnerProductId
	product, err := h.companiesService.AddNewProduct(
		r.Context(),
		companyId,
		domain.Product{
			PartnerProductId: &partnerProductId,
			Name:             body.Name,
			Price:            body.Price,
			Description:      body.Description,
			CurrentCount:     body.CurrentCount,
			IsAvailable:      true,
		},
	)
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	w.WriteJson(http.StatusCreated, dto.CompanyProductDtoFromDomain(product))
}

// PatchCompanyProduct updates a product of the authenticated company.
// @Summary Update company product
// @Description Partially updates a product selected by its partner product identifier.
// @Tags Company
// @Accept json
// @Produce json
// @Param Authorization header string true "Company UUID" format(uuid)
// @Param partnerProductId path string true "Partner product ID"
// @Param request body dto.PatchCompanyProductRequest true "Product fields to update"
// @Success 200 {object} dto.CompanyProductDto
// @Failure 400 {object} core_response.ErrorResponse
// @Failure 401 {object} core_response.ErrorResponse
// @Failure 404 {object} core_response.ErrorResponse
// @Failure 500 {object} core_response.ErrorResponse
// @Router /api/v1/company/products/{partnerProductId} [patch]
func (h *Handler) PatchCompanyProduct(w *core_response.Writer, r *http.Request) {
	companyId, err := http_middleware.GetCompanyIdFromCtx(r.Context())
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	partnerProductId := r.PathValue("partnerProductId")

	var body dto.PatchCompanyProductRequest
	err = request.DecodeAndValidate(r, &body, h.validator)
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	product, err := h.companiesService.PatchCompanyProduct(
		r.Context(),
		companyId,
		partnerProductId,
		domain.ProductPatch{
			Name:         body.Name,
			Price:        body.Price,
			Description:  body.Description,
			CurrentCount: body.CurrentCount,
			IsAvailable:  body.IsAvailable,
		},
	)
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	w.WriteJson(http.StatusOK, dto.CompanyProductDtoFromDomain(product))
}

// ChangeOrderStatus updates an order status for the authenticated company.
// @Summary Change company order status
// @Description Changes the status of an order belonging to the authenticated company.
// @Tags Company
// @Accept json
// @Produce json
// @Param Authorization header string true "Company UUID" format(uuid)
// @Param orderId path string true "Order UUID" format(uuid)
// @Param request body dto.ChangeOrderStatusRequest true "New order status"
// @Success 200 {object} dto.OrderDto
// @Failure 400 {object} core_response.ErrorResponse
// @Failure 401 {object} core_response.ErrorResponse
// @Failure 404 {object} core_response.ErrorResponse
// @Failure 409 {object} core_response.ErrorResponse
// @Failure 500 {object} core_response.ErrorResponse
// @Router /api/v1/company/orders/{orderId}/status [patch]
func (h *Handler) ChangeOrderStatus(w *core_response.Writer, r *http.Request) {
	companyId, err := http_middleware.GetCompanyIdFromCtx(r.Context())
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	orderId, err := request.ParseUUIDPathParam(r, "orderId")
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	var body dto.ChangeOrderStatusRequest
	err = request.DecodeAndValidate(r, &body, h.validator)
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	order, err := h.companiesService.ChangeOrderStatus(
		r.Context(),
		companyId,
		orderId,
		domain.OrderStatus(body.Status),
	)
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	w.WriteJson(http.StatusOK, dto.OrderDtoFromDomain(order))
}
