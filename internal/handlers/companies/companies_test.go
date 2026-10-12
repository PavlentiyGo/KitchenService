package handler_companies_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"foodDelivery/internal/core/domain"
	core_errors "foodDelivery/internal/core/errors"
	core_response "foodDelivery/internal/core/http/response"
	core_server "foodDelivery/internal/core/http/server"
	core_logger "foodDelivery/internal/core/logger"
	handler_companies "foodDelivery/internal/handlers/companies"
	"foodDelivery/internal/handlers/dto"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

type testDeps struct {
	router           *http.ServeMux
	companiesService *MockcompaniesService
}

func newTestDeps(t *testing.T) *testDeps {
	t.Helper()

	ctrl := gomock.NewController(t)
	companiesService := NewMockcompaniesService(ctrl)

	handler := handler_companies.NewHandler(companiesService, validator.New())

	router := core_server.NewApiVersionRouter("v1")
	router.ChainRoutes(handler.Routes()...)
	return &testDeps{
		router:           core_server.RegisterRouters(router),
		companiesService: companiesService,
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

func TestAddNewProduct_Success(t *testing.T) {
	deps := newTestDeps(t)
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	partnerProductId := "pizza-margherita"
	productForCreate := domain.Product{
		PartnerProductId: &partnerProductId,
		Name:             "Пицца Маргарита",
		Price:            49900,
		Description:      "Томатный соус, моцарелла, томаты и базилик",
		CurrentCount:     18,
		IsAvailable:      true,
	}
	createdProduct := productForCreate
	createdProduct.Id = productId
	createdProduct.CompanyId = companyId

	deps.companiesService.
		EXPECT().
		AddNewProduct(gomock.Any(), companyId, productForCreate).
		Return(createdProduct, nil)

	body := []byte(`{
		"partner_product_id":"pizza-margherita",
		"name":"Пицца Маргарита",
		"price":49900,
		"description":"Томатный соус, моцарелла, томаты и базилик",
		"current_count":18
	}`)
	responseRecorder := makeRequest(
		deps.router,
		http.MethodPost,
		"/api/v1/company/products",
		body,
		map[string]string{"Authorization": companyId.String()},
	)

	assert.Equal(t, http.StatusCreated, responseRecorder.Code)
	var response dto.CompanyProductDto
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)

	expected := dto.CompanyProductDto{
		Id:               productId.String(),
		PartnerProductId: &partnerProductId,
		Name:             "Пицца Маргарита",
		Price:            49900,
		Description:      "Томатный соус, моцарелла, томаты и базилик",
		CurrentCount:     18,
		IsAvailable:      true,
	}
	assert.Equal(t, expected, response)
}

func TestAddNewProduct_InvalidBodyFields(t *testing.T) {
	testCases := []struct {
		name string
		body map[string]any
	}{
		{
			name: "missing partner product id",
			body: map[string]any{
				"name":          "Пицца Маргарита",
				"price":         49900,
				"description":   "Томатный соус и моцарелла",
				"current_count": 18,
			},
		},
		{
			name: "partner product id is too long",
			body: map[string]any{
				"partner_product_id": strings.Repeat("a", 256),
				"name":               "Пицца Маргарита",
				"price":              49900,
				"description":        "Томатный соус и моцарелла",
				"current_count":      18,
			},
		},
		{
			name: "missing name",
			body: map[string]any{
				"partner_product_id": "pizza-margherita",
				"price":              49900,
				"description":        "Томатный соус и моцарелла",
				"current_count":      18,
			},
		},
		{
			name: "name is too long",
			body: map[string]any{
				"partner_product_id": "pizza-margherita",
				"name":               strings.Repeat("a", 256),
				"price":              49900,
				"description":        "Томатный соус и моцарелла",
				"current_count":      18,
			},
		},
		{
			name: "invalid price",
			body: map[string]any{
				"partner_product_id": "pizza-margherita",
				"name":               "Пицца Маргарита",
				"price":              0,
				"description":        "Томатный соус и моцарелла",
				"current_count":      18,
			},
		},
		{
			name: "description is too long",
			body: map[string]any{
				"partner_product_id": "pizza-margherita",
				"name":               "Пицца Маргарита",
				"price":              49900,
				"description":        strings.Repeat("a", 256),
				"current_count":      18,
			},
		},
		{
			name: "invalid current count",
			body: map[string]any{
				"partner_product_id": "pizza-margherita",
				"name":               "Пицца Маргарита",
				"price":              49900,
				"description":        "Томатный соус и моцарелла",
				"current_count":      -1,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			deps := newTestDeps(t)
			companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
			body, err := json.Marshal(testCase.body)
			require.NoError(t, err)

			responseRecorder := makeRequest(
				deps.router,
				http.MethodPost,
				"/api/v1/company/products",
				body,
				map[string]string{"Authorization": companyId.String()},
			)

			assert.Equal(t, http.StatusBadRequest, responseRecorder.Code)
			var response core_response.ErrorResponse
			err = json.NewDecoder(responseRecorder.Body).Decode(&response)
			require.NoError(t, err)
			assert.Equal(t, core_errors.ErrInvalidRequest.Error(), response.Error.Code)
			assert.Contains(t, response.Error.Message, "failed to validate request")
		})
	}
}

func TestAddNewProduct_DuplicatePartnerProductId(t *testing.T) {
	deps := newTestDeps(t)
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	partnerProductId := "pizza-margherita"
	productForCreate := domain.Product{
		PartnerProductId: &partnerProductId,
		Name:             "Пицца Маргарита",
		Price:            49900,
		Description:      "Томатный соус и моцарелла",
		CurrentCount:     18,
		IsAvailable:      true,
	}
	serviceErr := fmt.Errorf(
		"failed to create product: product with partner id %q already exists: %w",
		partnerProductId,
		core_errors.ErrConflict,
	)

	deps.companiesService.
		EXPECT().
		AddNewProduct(gomock.Any(), companyId, productForCreate).
		Return(domain.Product{}, serviceErr)

	body := []byte(`{
		"partner_product_id":"pizza-margherita",
		"name":"Пицца Маргарита",
		"price":49900,
		"description":"Томатный соус и моцарелла",
		"current_count":18
	}`)
	responseRecorder := makeRequest(
		deps.router,
		http.MethodPost,
		"/api/v1/company/products",
		body,
		map[string]string{"Authorization": companyId.String()},
	)

	assert.Equal(t, http.StatusConflict, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "conflict", response.Error.Code)
	assert.Equal(t, serviceErr.Error(), response.Error.Message)
}

func TestAddNewProduct_CompanyNotFound(t *testing.T) {
	deps := newTestDeps(t)
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	partnerProductId := "pizza-margherita"
	productForCreate := domain.Product{
		PartnerProductId: &partnerProductId,
		Name:             "Пицца Маргарита",
		Price:            49900,
		Description:      "Томатный соус и моцарелла",
		CurrentCount:     18,
		IsAvailable:      true,
	}
	serviceErr := fmt.Errorf("failed to create product: company %s not found: %w", companyId, core_errors.ErrNotFound)

	deps.companiesService.
		EXPECT().
		AddNewProduct(gomock.Any(), companyId, productForCreate).
		Return(domain.Product{}, serviceErr)

	body := []byte(`{
		"partner_product_id":"pizza-margherita",
		"name":"Пицца Маргарита",
		"price":49900,
		"description":"Томатный соус и моцарелла",
		"current_count":18
	}`)
	responseRecorder := makeRequest(
		deps.router,
		http.MethodPost,
		"/api/v1/company/products",
		body,
		map[string]string{"Authorization": companyId.String()},
	)

	assert.Equal(t, http.StatusNotFound, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, core_errors.ErrNotFound.Error(), response.Error.Code)
	assert.Equal(t, serviceErr.Error(), response.Error.Message)
}

func TestAddNewProduct_InternalError(t *testing.T) {
	deps := newTestDeps(t)
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	partnerProductId := "pizza-margherita"
	productForCreate := domain.Product{
		PartnerProductId: &partnerProductId,
		Name:             "Пицца Маргарита",
		Price:            49900,
		Description:      "Томатный соус и моцарелла",
		CurrentCount:     18,
		IsAvailable:      true,
	}
	internalErr := errors.New("internal server error")

	deps.companiesService.
		EXPECT().
		AddNewProduct(gomock.Any(), companyId, productForCreate).
		Return(domain.Product{}, internalErr)

	body := []byte(`{
		"partner_product_id":"pizza-margherita",
		"name":"Пицца Маргарита",
		"price":49900,
		"description":"Томатный соус и моцарелла",
		"current_count":18
	}`)
	responseRecorder := makeRequest(
		deps.router,
		http.MethodPost,
		"/api/v1/company/products",
		body,
		map[string]string{"Authorization": companyId.String()},
	)

	assert.Equal(t, http.StatusInternalServerError, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Empty(t, response.Error.Code)
	assert.Equal(t, internalErr.Error(), response.Error.Message)
}

func TestPatchCompanyProduct_Success(t *testing.T) {
	deps := newTestDeps(t)
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	partnerProductId := "pizza-margherita"
	name := "Пицца Маргарита большая"
	price := int64(59900)
	description := "Томатный соус, двойная моцарелла, томаты и базилик"
	currentCount := 25
	isAvailable := false
	patch := domain.ProductPatch{
		Name:         &name,
		Price:        &price,
		Description:  &description,
		CurrentCount: &currentCount,
		IsAvailable:  &isAvailable,
	}
	updatedProduct := domain.Product{
		Id:               productId,
		CompanyId:        companyId,
		PartnerProductId: &partnerProductId,
		Name:             name,
		Price:            price,
		Description:      description,
		CurrentCount:     currentCount,
		IsAvailable:      isAvailable,
	}

	deps.companiesService.
		EXPECT().
		PatchCompanyProduct(gomock.Any(), companyId, partnerProductId, patch).
		Return(updatedProduct, nil)

	body := []byte(`{
		"name":"Пицца Маргарита большая",
		"price":59900,
		"description":"Томатный соус, двойная моцарелла, томаты и базилик",
		"current_count":25,
		"is_available":false
	}`)
	responseRecorder := makeRequest(
		deps.router,
		http.MethodPatch,
		"/api/v1/company/products/"+partnerProductId,
		body,
		map[string]string{"Authorization": companyId.String()},
	)

	assert.Equal(t, http.StatusOK, responseRecorder.Code)
	var response dto.CompanyProductDto
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)

	expected := dto.CompanyProductDto{
		Id:               productId.String(),
		PartnerProductId: &partnerProductId,
		Name:             name,
		Price:            price,
		Description:      description,
		CurrentCount:     currentCount,
		IsAvailable:      isAvailable,
	}
	assert.Equal(t, expected, response)
}

func TestPatchCompanyProduct_InvalidBodyFields(t *testing.T) {
	testCases := []struct {
		name                 string
		body                 map[string]any
		expectedErrorMessage string
	}{
		{
			name: "name is too long",
			body: map[string]any{
				"name": strings.Repeat("a", 256),
			},
			expectedErrorMessage: "failed to validate request",
		},
		{
			name: "invalid price",
			body: map[string]any{
				"price": 0,
			},
			expectedErrorMessage: "failed to validate request",
		},
		{
			name: "description is too long",
			body: map[string]any{
				"description": strings.Repeat("a", 256),
			},
			expectedErrorMessage: "failed to validate request",
		},
		{
			name: "invalid current count",
			body: map[string]any{
				"current_count": -1,
			},
			expectedErrorMessage: "failed to validate request",
		},
		{
			name: "invalid is available type",
			body: map[string]any{
				"is_available": "yes",
			},
			expectedErrorMessage: "failed to decode request",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			deps := newTestDeps(t)
			companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
			body, err := json.Marshal(testCase.body)
			require.NoError(t, err)

			responseRecorder := makeRequest(
				deps.router,
				http.MethodPatch,
				"/api/v1/company/products/pizza-margherita",
				body,
				map[string]string{"Authorization": companyId.String()},
			)

			assert.Equal(t, http.StatusBadRequest, responseRecorder.Code)
			var response core_response.ErrorResponse
			err = json.NewDecoder(responseRecorder.Body).Decode(&response)
			require.NoError(t, err)
			assert.Equal(t, core_errors.ErrInvalidRequest.Error(), response.Error.Code)
			assert.Contains(t, response.Error.Message, testCase.expectedErrorMessage)
		})
	}
}

func TestPatchCompanyProduct_EmptyName(t *testing.T) {
	deps := newTestDeps(t)
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	partnerProductId := "pizza-margherita"
	name := ""
	patch := domain.ProductPatch{Name: &name}
	serviceErr := fmt.Errorf("product name is empty: %w", core_errors.ErrInvalidRequest)

	deps.companiesService.
		EXPECT().
		PatchCompanyProduct(gomock.Any(), companyId, partnerProductId, patch).
		Return(domain.Product{}, serviceErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodPatch,
		"/api/v1/company/products/"+partnerProductId,
		[]byte(`{"name":""}`),
		map[string]string{"Authorization": companyId.String()},
	)

	assert.Equal(t, http.StatusBadRequest, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, core_errors.ErrInvalidRequest.Error(), response.Error.Code)
	assert.Equal(t, serviceErr.Error(), response.Error.Message)
}

func TestPatchCompanyProduct_ProductNotFound(t *testing.T) {
	deps := newTestDeps(t)
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	partnerProductId := "unknown-product"
	price := int64(59900)
	patch := domain.ProductPatch{Price: &price}
	serviceErr := fmt.Errorf(
		"failed to patch product: product with partner id %q not found: %w",
		partnerProductId,
		core_errors.ErrNotFound,
	)

	deps.companiesService.
		EXPECT().
		PatchCompanyProduct(gomock.Any(), companyId, partnerProductId, patch).
		Return(domain.Product{}, serviceErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodPatch,
		"/api/v1/company/products/"+partnerProductId,
		[]byte(`{"price":59900}`),
		map[string]string{"Authorization": companyId.String()},
	)

	assert.Equal(t, http.StatusNotFound, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, core_errors.ErrNotFound.Error(), response.Error.Code)
	assert.Equal(t, serviceErr.Error(), response.Error.Message)
}

func TestChangeOrderStatus_Success(t *testing.T) {
	deps := newTestDeps(t)
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	orderId := uuid.MustParse("44444444-4444-4444-8444-444444444444")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	createdAt := time.Date(2026, time.September, 8, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.September, 8, 12, 5, 0, 0, time.UTC)
	order := domain.Order{
		Id:        orderId,
		UserId:    userId,
		CompanyId: companyId,
		Status:    domain.OrderStatusAccepted,
		Products: []domain.OrderProduct{
			{
				Id:    productId,
				Name:  "Пицца Маргарита",
				Price: 49900,
				Count: 2,
			},
		},
		FullPrice: 99800,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	deps.companiesService.
		EXPECT().
		ChangeOrderStatus(gomock.Any(), companyId, orderId, domain.OrderStatusAccepted).
		Return(order, nil)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodPatch,
		"/api/v1/company/orders/"+orderId.String()+"/status",
		[]byte(`{"status":"accepted"}`),
		map[string]string{"Authorization": companyId.String()},
	)

	assert.Equal(t, http.StatusOK, responseRecorder.Code)
	var response dto.OrderDto
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)

	expected := dto.OrderDto{
		Id:        orderId.String(),
		CompanyId: companyId.String(),
		Status:    string(domain.OrderStatusAccepted),
		Products: []dto.OrderProductDto{
			{
				Id:    productId.String(),
				Name:  "Пицца Маргарита",
				Price: 49900,
				Count: 2,
			},
		},
		FullPrice: 99800,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
	assert.Equal(t, expected, response)
}

func TestChangeOrderStatus_InvalidOrderId(t *testing.T) {
	deps := newTestDeps(t)
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")

	responseRecorder := makeRequest(
		deps.router,
		http.MethodPatch,
		"/api/v1/company/orders/not-a-uuid/status",
		[]byte(`{"status":"accepted"}`),
		map[string]string{"Authorization": companyId.String()},
	)

	assert.Equal(t, http.StatusBadRequest, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, core_errors.ErrInvalidRequest.Error(), response.Error.Code)
	assert.Contains(t, response.Error.Message, "invalid orderId path parameter")
}

func TestChangeOrderStatus_OrderNotFound(t *testing.T) {
	deps := newTestDeps(t)
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	orderId := uuid.MustParse("44444444-4444-4444-8444-444444444444")
	serviceErr := fmt.Errorf("failed to get company order: %w", core_errors.ErrNotFound)

	deps.companiesService.
		EXPECT().
		ChangeOrderStatus(gomock.Any(), companyId, orderId, domain.OrderStatusAccepted).
		Return(domain.Order{}, serviceErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodPatch,
		"/api/v1/company/orders/"+orderId.String()+"/status",
		[]byte(`{"status":"accepted"}`),
		map[string]string{"Authorization": companyId.String()},
	)

	assert.Equal(t, http.StatusNotFound, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, core_errors.ErrNotFound.Error(), response.Error.Code)
	assert.Equal(t, serviceErr.Error(), response.Error.Message)
}

func TestChangeOrderStatus_InvalidStatus(t *testing.T) {
	deps := newTestDeps(t)
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	orderId := uuid.MustParse("44444444-4444-4444-8444-444444444444")

	responseRecorder := makeRequest(
		deps.router,
		http.MethodPatch,
		"/api/v1/company/orders/"+orderId.String()+"/status",
		[]byte(`{"status":"unknown"}`),
		map[string]string{"Authorization": companyId.String()},
	)

	assert.Equal(t, http.StatusBadRequest, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, core_errors.ErrInvalidRequest.Error(), response.Error.Code)
	assert.Contains(t, response.Error.Message, "failed to validate request")
}

func TestChangeOrderStatus_RejectedOrderConflict(t *testing.T) {
	deps := newTestDeps(t)
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	orderId := uuid.MustParse("44444444-4444-4444-8444-444444444444")
	serviceErr := fmt.Errorf(
		"rejected order %s cannot change status: %w",
		orderId,
		core_errors.ErrConflict,
	)

	deps.companiesService.
		EXPECT().
		ChangeOrderStatus(gomock.Any(), companyId, orderId, domain.OrderStatusAccepted).
		Return(domain.Order{}, serviceErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodPatch,
		"/api/v1/company/orders/"+orderId.String()+"/status",
		[]byte(`{"status":"accepted"}`),
		map[string]string{"Authorization": companyId.String()},
	)

	assert.Equal(t, http.StatusConflict, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "conflict", response.Error.Code)
	assert.Equal(t, serviceErr.Error(), response.Error.Message)
}

func TestChangeOrderStatus_InternalError(t *testing.T) {
	deps := newTestDeps(t)
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	orderId := uuid.MustParse("44444444-4444-4444-8444-444444444444")
	internalErr := errors.New("internal server error")

	deps.companiesService.
		EXPECT().
		ChangeOrderStatus(gomock.Any(), companyId, orderId, domain.OrderStatusAccepted).
		Return(domain.Order{}, internalErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodPatch,
		"/api/v1/company/orders/"+orderId.String()+"/status",
		[]byte(`{"status":"accepted"}`),
		map[string]string{"Authorization": companyId.String()},
	)

	assert.Equal(t, http.StatusInternalServerError, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Empty(t, response.Error.Code)
	assert.Equal(t, internalErr.Error(), response.Error.Message)
}
