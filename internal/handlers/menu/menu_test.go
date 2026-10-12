package handler_menu_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"foodDelivery/internal/core/domain"
	core_response "foodDelivery/internal/core/http/response"
	core_server "foodDelivery/internal/core/http/server"
	core_logger "foodDelivery/internal/core/logger"
	"foodDelivery/internal/handlers/dto"
	handler_menu "foodDelivery/internal/handlers/menu"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

type testDeps struct {
	router      *http.ServeMux
	menuService *MockmenuService
}

func newTestDeps(t *testing.T) *testDeps {
	t.Helper()

	ctrl := gomock.NewController(t)
	menuService := NewMockmenuService(ctrl)

	handler := handler_menu.NewHandler(menuService, validator.New())

	router := core_server.NewApiVersionRouter("v1")
	router.ChainRoutes(handler.Routes()...)
	return &testDeps{
		router:      core_server.RegisterRouters(router),
		menuService: menuService,
	}
}
func makeRequest(
	handler http.Handler,
	method string,
	target string,
	body []byte,
	headers map[string]string,
) *httptest.ResponseRecorder {

	req := httptest.NewRequest(method, target, bytes.NewBuffer(body))

	ctx := core_logger.CtxWithLogger(req.Context(), &core_logger.Logger{
		Logger: zap.NewNop(),
	})
	for header, val := range headers {
		req.Header.Set(header, val)
	}

	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req.WithContext(ctx))
	return w
}
func TestGetCompanies_Success(t *testing.T) {
	deps := newTestDeps(t)
	companies := []domain.Company{
		{
			Id:       uuid.MustParse("11111111-1111-4111-8111-111111111111"),
			Name:     "Demo company",
			IsActive: true,
		},
	}
	deps.menuService.
		EXPECT().
		GetCompanies(gomock.Any()).
		Return(companies, nil)

	req := makeRequest(deps.router, "GET", "/api/v1/companies", nil, nil)

	assert.Equal(t, req.Code, http.StatusOK)
	var response dto.GetCompaniesResponse
	err := json.NewDecoder(req.Body).Decode(&response)
	require.NoError(t, err)
	expected := dto.CompaniesDtoFromDomains(companies)
	assert.Equal(t, response, expected)
}
func TestGetCompanies_InternalError(t *testing.T) {
	deps := newTestDeps(t)
	errText := errors.New("internal server error")
	deps.menuService.
		EXPECT().
		GetCompanies(gomock.Any()).
		Return(nil, errText)

	req := makeRequest(deps.router, "GET", "/api/v1/companies", nil, nil)

	assert.Equal(t, req.Code, http.StatusInternalServerError)
	var response core_response.ErrorResponse
	err := json.NewDecoder(req.Body).Decode(&response)
	require.NoError(t, err)
	assert.NotEmpty(t, response.Error)
}

func TestGetProduct_Success(t *testing.T) {
	deps := newTestDeps(t)
	productID := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	product := domain.Product{
		Id:          productID,
		CompanyId:   uuid.MustParse("11111111-1111-4111-8111-111111111111"),
		Name:        "Пицца Маргарита",
		Price:       49900,
		Description: "Томатный соус, моцарелла, томаты и базилик",
		IsAvailable: true,
	}

	deps.menuService.
		EXPECT().
		GetCompanyProduct(gomock.Any(), productID).
		Return(product, nil)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodGet,
		"/api/v1/products/"+productID.String(),
		nil,
		nil,
	)

	assert.Equal(t, http.StatusOK, responseRecorder.Code)

	var response dto.ProductDto
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)

	expected := dto.ProductDto{
		Id:          productID.String(),
		Name:        "Пицца Маргарита",
		Price:       49900,
		Description: "Томатный соус, моцарелла, томаты и базилик",
		IsAvailable: true,
	}
	assert.Equal(t, expected, response)
}

func TestGetProduct_InternalError(t *testing.T) {
	deps := newTestDeps(t)
	productID := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	internalErr := errors.New("internal server error")

	deps.menuService.
		EXPECT().
		GetCompanyProduct(gomock.Any(), productID).
		Return(domain.Product{}, internalErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodGet,
		"/api/v1/products/"+productID.String(),
		nil,
		nil,
	)

	assert.Equal(t, http.StatusInternalServerError, responseRecorder.Code)

	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)

	assert.Empty(t, response.Error.Code)
	assert.Equal(t, internalErr.Error(), response.Error.Message)
}

func TestGetProducts_Success(t *testing.T) {
	deps := newTestDeps(t)
	companyID := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	firstProductID := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	secondProductID := uuid.MustParse("33333333-3333-4333-8333-000000000002")
	products := []domain.Product{
		{
			Id:          firstProductID,
			CompanyId:   companyID,
			Name:        "Пицца Маргарита",
			Price:       49900,
			Description: "Томатный соус, моцарелла, томаты и базилик",
			IsAvailable: true,
		},
		{
			Id:          secondProductID,
			CompanyId:   companyID,
			Name:        "Пицца Пепперони",
			Price:       57900,
			Description: "Пепперони, моцарелла и томатный соус",
			IsAvailable: false,
		},
	}

	deps.menuService.
		EXPECT().
		GetCompanyProducts(gomock.Any(), companyID).
		Return(products, nil)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodGet,
		"/api/v1/companies/"+companyID.String()+"/products",
		nil,
		nil,
	)

	assert.Equal(t, http.StatusOK, responseRecorder.Code)

	var response dto.GetCompanyProductsResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)

	expected := dto.GetCompanyProductsResponse{
		Products: []dto.ProductDto{
			{
				Id:          firstProductID.String(),
				Name:        "Пицца Маргарита",
				Price:       49900,
				Description: "Томатный соус, моцарелла, томаты и базилик",
				IsAvailable: true,
			},
			{
				Id:          secondProductID.String(),
				Name:        "Пицца Пепперони",
				Price:       57900,
				Description: "Пепперони, моцарелла и томатный соус",
				IsAvailable: false,
			},
		},
	}
	assert.Equal(t, expected, response)
}

func TestGetProducts_InternalError(t *testing.T) {
	deps := newTestDeps(t)
	companyID := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	internalErr := errors.New("internal server error")

	deps.menuService.
		EXPECT().
		GetCompanyProducts(gomock.Any(), companyID).
		Return(nil, internalErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodGet,
		"/api/v1/companies/"+companyID.String()+"/products",
		nil,
		nil,
	)

	assert.Equal(t, http.StatusInternalServerError, responseRecorder.Code)

	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)

	assert.Empty(t, response.Error.Code)
	assert.Equal(t, internalErr.Error(), response.Error.Message)
}

func TestGetProducts_InvalidPathUUID(t *testing.T) {
	cases := []struct {
		name   string
		target string
	}{
		{
			name:   "company id",
			target: "/api/v1/companies/not-a-uuid/products",
		},
		{
			name:   "product id",
			target: "/api/v1/products/not-a-uuid",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			deps := newTestDeps(t)

			response := makeRequest(deps.router, http.MethodGet, testCase.target, nil, nil)

			assert.Equal(t, http.StatusBadRequest, response.Code)
		})
	}
}
