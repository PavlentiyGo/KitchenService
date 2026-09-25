package handler_menu

import (
	"avitoKitchen/internal/core/http/request"
	core_response "avitoKitchen/internal/core/http/response"
	core_server "avitoKitchen/internal/core/http/server"
	"avitoKitchen/internal/handlers/dto"
	"net/http"
)

type Handler struct {
	menuService menuService
	validator   request.Validator
}

func NewHandler(service1 menuService, validator request.Validator) *Handler {
	return &Handler{menuService: service1, validator: validator}
}

func (h *Handler) Routes() []core_server.Route {
	return []core_server.Route{
		{
			Method:  "GET",
			Path:    "/companies",
			Handler: h.GetCompanies,
		},
		{
			Method:  "GET",
			Path:    "/companies/{companyId}/products",
			Handler: h.GetCompanyProducts,
		},
		{
			Method:  "GET",
			Path:    "/products/{productId}",
			Handler: h.GetCompanyProduct,
		},
	}
}

// GetCompanies returns all companies.
// @Summary List companies
// @Description Returns all companies available in the service.
// @Tags Menu
// @Produce json
// @Success 200 {object} dto.GetCompaniesResponse
// @Failure 500 {object} core_response.ErrorResponse
// @Router /api/v1/companies [get]
func (h *Handler) GetCompanies(w *core_response.Writer, r *http.Request) {
	ctx := r.Context()

	companies, err := h.menuService.GetCompanies(ctx)
	if err != nil {
		w.ErrorResponse(err)
		return
	}
	w.WriteJson(http.StatusOK, dto.CompaniesDtoFromDomains(companies))
}

// GetCompanyProducts returns the products of a company.
// @Summary List company products
// @Description Returns all products belonging to the specified company.
// @Tags Menu
// @Produce json
// @Param companyId path string true "Company UUID" format(uuid)
// @Success 200 {object} dto.GetCompanyProductsResponse
// @Failure 400 {object} core_response.ErrorResponse
// @Failure 500 {object} core_response.ErrorResponse
// @Router /api/v1/companies/{companyId}/products [get]
func (h *Handler) GetCompanyProducts(w *core_response.Writer, r *http.Request) {
	ctx := r.Context()

	companyId, err := request.ParseUUIDPathParam(r, "companyId")
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	products, err := h.menuService.GetCompanyProducts(ctx, companyId)
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	w.WriteJson(http.StatusOK, dto.ProductsDtoFromDomains(products))
}

// GetCompanyProduct returns a product by its identifier.
// @Summary Get product
// @Description Returns a single product by UUID.
// @Tags Menu
// @Produce json
// @Param productId path string true "Product UUID" format(uuid)
// @Success 200 {object} dto.ProductDto
// @Failure 400 {object} core_response.ErrorResponse
// @Failure 404 {object} core_response.ErrorResponse
// @Failure 500 {object} core_response.ErrorResponse
// @Router /api/v1/products/{productId} [get]
func (h *Handler) GetCompanyProduct(w *core_response.Writer, r *http.Request) {
	ctx := r.Context()

	productId, err := request.ParseUUIDPathParam(r, "productId")
	if err != nil {
		w.ErrorResponse(err)
		return
	}

	product, err := h.menuService.GetCompanyProduct(ctx, productId)
	if err != nil {
		w.ErrorResponse(err)
		return
	}
	w.WriteJson(http.StatusOK, dto.ProductDtoFromDomain(product))
}
