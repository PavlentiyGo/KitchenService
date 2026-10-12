package handler_orders_test

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
	"foodDelivery/internal/handlers/dto"
	handler_orders "foodDelivery/internal/handlers/orders"
	"net/http"
	"net/http/httptest"
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
	router        *http.ServeMux
	ordersService *MockordersService
}

func newTestDeps(t *testing.T) *testDeps {
	t.Helper()

	ctrl := gomock.NewController(t)
	ordersService := NewMockordersService(ctrl)

	handler := handler_orders.NewHandler(ordersService, validator.New())

	router := core_server.NewApiVersionRouter("v1")
	router.ChainRoutes(handler.Routes()...)

	return &testDeps{
		router:        core_server.RegisterRouters(router),
		ordersService: ordersService,
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
	for header, value := range headers {
		req.Header.Set(header, value)
	}

	responseRecorder := httptest.NewRecorder()
	handler.ServeHTTP(responseRecorder, req.WithContext(ctx))
	return responseRecorder
}

func TestGetOrder_Success(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
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

	deps.ordersService.
		EXPECT().
		GetUserOrder(gomock.Any(), userId, orderId).
		Return(order, nil)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodGet,
		"/api/v1/orders/"+orderId.String(),
		nil,
		map[string]string{"Authorization": userId.String()},
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

func TestGetOrder_InvalidOrderId(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")

	responseRecorder := makeRequest(
		deps.router,
		http.MethodGet,
		"/api/v1/orders/not-a-uuid",
		nil,
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusBadRequest, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, core_errors.ErrInvalidRequest.Error(), response.Error.Code)
	assert.Contains(t, response.Error.Message, "invalid orderId path parameter")
}

func TestGetOrder_DoesNotBelongToUser(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	orderId := uuid.MustParse("44444444-4444-4444-8444-444444444444")
	serviceErr := fmt.Errorf("this isn't your order: %w", core_errors.ErrForbidden)

	deps.ordersService.
		EXPECT().
		GetUserOrder(gomock.Any(), userId, orderId).
		Return(domain.Order{}, serviceErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodGet,
		"/api/v1/orders/"+orderId.String(),
		nil,
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusForbidden, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, core_errors.ErrForbidden.Error(), response.Error.Code)
	assert.Equal(t, serviceErr.Error(), response.Error.Message)
}

func TestGetOrder_UserNotFound(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	orderId := uuid.MustParse("44444444-4444-4444-8444-444444444444")
	serviceErr := fmt.Errorf("failed to get user: %w", core_errors.ErrNotFound)

	deps.ordersService.
		EXPECT().
		GetUserOrder(gomock.Any(), userId, orderId).
		Return(domain.Order{}, serviceErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodGet,
		"/api/v1/orders/"+orderId.String(),
		nil,
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusNotFound, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, core_errors.ErrNotFound.Error(), response.Error.Code)
	assert.Equal(t, serviceErr.Error(), response.Error.Message)
}

func TestGetOrder_OrderNotFound(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	orderId := uuid.MustParse("44444444-4444-4444-8444-444444444444")
	serviceErr := fmt.Errorf("failed to get order: %w", core_errors.ErrNotFound)

	deps.ordersService.
		EXPECT().
		GetUserOrder(gomock.Any(), userId, orderId).
		Return(domain.Order{}, serviceErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodGet,
		"/api/v1/orders/"+orderId.String(),
		nil,
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusNotFound, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, core_errors.ErrNotFound.Error(), response.Error.Code)
	assert.Equal(t, serviceErr.Error(), response.Error.Message)
}

func TestGetOrder_InternalError(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	orderId := uuid.MustParse("44444444-4444-4444-8444-444444444444")
	internalErr := errors.New("internal server error")

	deps.ordersService.
		EXPECT().
		GetUserOrder(gomock.Any(), userId, orderId).
		Return(domain.Order{}, internalErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodGet,
		"/api/v1/orders/"+orderId.String(),
		nil,
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusInternalServerError, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Empty(t, response.Error.Code)
	assert.Equal(t, internalErr.Error(), response.Error.Message)
}

func TestGetMyOrders_Success(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	orderId := uuid.MustParse("44444444-4444-4444-8444-444444444444")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	createdAt := time.Date(2026, time.September, 8, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.September, 8, 12, 5, 0, 0, time.UTC)
	orders := []domain.Order{
		{
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
		},
	}

	deps.ordersService.
		EXPECT().
		GetMyOrders(gomock.Any(), userId).
		Return(orders, nil)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodGet,
		"/api/v1/orders/my",
		nil,
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusOK, responseRecorder.Code)
	var response dto.GetMyOrdersResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)

	expected := dto.GetMyOrdersResponse{
		Orders: []dto.OrderDto{
			{
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
			},
		},
	}
	assert.Equal(t, expected, response)
}

func TestGetMyOrders_UserNotFound(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	serviceErr := fmt.Errorf("failed to get user: %w", core_errors.ErrNotFound)

	deps.ordersService.
		EXPECT().
		GetMyOrders(gomock.Any(), userId).
		Return(nil, serviceErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodGet,
		"/api/v1/orders/my",
		nil,
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusNotFound, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, core_errors.ErrNotFound.Error(), response.Error.Code)
	assert.Equal(t, serviceErr.Error(), response.Error.Message)
}

func TestGetMyOrders_InternalError(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	internalErr := errors.New("internal server error")

	deps.ordersService.
		EXPECT().
		GetMyOrders(gomock.Any(), userId).
		Return(nil, internalErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodGet,
		"/api/v1/orders/my",
		nil,
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusInternalServerError, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Empty(t, response.Error.Code)
	assert.Equal(t, internalErr.Error(), response.Error.Message)
}

func TestPostOrder_Success(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	products := []domain.Product{
		{
			Id:    productId,
			Price: 49900,
		},
	}

	deps.ordersService.
		EXPECT().
		CreateOrder(gomock.Any(), userId, companyId, products).
		Return(nil)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodPost,
		"/api/v1/companies/"+companyId.String()+"/orders",
		[]byte(`{"products":[{"id":"`+productId.String()+`","price":49900}]}`),
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusCreated, responseRecorder.Code)
	assert.Empty(t, responseRecorder.Body.String())
}

func TestPostOrder_InvalidRequest(t *testing.T) {
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	validTarget := "/api/v1/companies/" + companyId.String() + "/orders"
	validBody := []byte(`{"products":[{"id":"` + productId.String() + `","price":49900}]}`)

	testCases := []struct {
		name           string
		target         string
		body           []byte
		headers        map[string]string
		expectedStatus int
	}{
		{
			name:   "invalid company id",
			target: "/api/v1/companies/not-a-uuid/orders",
			body:   validBody,
			headers: map[string]string{
				"Authorization": userId.String(),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "malformed json",
			target: validTarget,
			body:   []byte(`{"products":[`),
			headers: map[string]string{
				"Authorization": userId.String(),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "empty products",
			target: validTarget,
			body:   []byte(`{"products":[]}`),
			headers: map[string]string{
				"Authorization": userId.String(),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "invalid product id",
			target: validTarget,
			body:   []byte(`{"products":[{"id":"not-a-uuid","price":49900}]}`),
			headers: map[string]string{
				"Authorization": userId.String(),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "zero product price",
			target: validTarget,
			body:   []byte(`{"products":[{"id":"` + productId.String() + `","price":0}]}`),
			headers: map[string]string{
				"Authorization": userId.String(),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "negative product price",
			target: validTarget,
			body:   []byte(`{"products":[{"id":"` + productId.String() + `","price":-1}]}`),
			headers: map[string]string{
				"Authorization": userId.String(),
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			deps := newTestDeps(t)

			responseRecorder := makeRequest(
				deps.router,
				http.MethodPost,
				testCase.target,
				testCase.body,
				testCase.headers,
			)

			assert.Equal(t, testCase.expectedStatus, responseRecorder.Code)
			var response core_response.ErrorResponse
			err := json.NewDecoder(responseRecorder.Body).Decode(&response)
			require.NoError(t, err)
			assert.Equal(t, core_errors.ErrInvalidRequest.Error(), response.Error.Code)
			assert.NotEmpty(t, response.Error.Message)
		})
	}
}

func TestPostOrder_ServiceErrors(t *testing.T) {
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	products := []domain.Product{
		{
			Id:    productId,
			Price: 49900,
		},
	}

	testCases := []struct {
		name           string
		serviceErr     error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "user not found",
			serviceErr:     fmt.Errorf("failed to get user: %w", core_errors.ErrNotFound),
			expectedStatus: http.StatusNotFound,
			expectedCode:   core_errors.ErrNotFound.Error(),
		},
		{
			name:           "company not found",
			serviceErr:     fmt.Errorf("failed to get company: %w", core_errors.ErrNotFound),
			expectedStatus: http.StatusNotFound,
			expectedCode:   core_errors.ErrNotFound.Error(),
		},
		{
			name: "basket size does not match request",
			serviceErr: fmt.Errorf(
				"basket in request and user basket aren't equal: %w",
				core_errors.ErrInvalidRequest,
			),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   core_errors.ErrInvalidRequest.Error(),
		},
		{
			name:           "basket contains another product",
			serviceErr:     fmt.Errorf("not all products in request: %w", core_errors.ErrInvalidRequest),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   core_errors.ErrInvalidRequest.Error(),
		},
		{
			name:           "product price changed",
			serviceErr:     fmt.Errorf("product price changed: %w", core_errors.ErrConflict),
			expectedStatus: http.StatusConflict,
			expectedCode:   "conflict",
		},
		{
			name:           "internal error",
			serviceErr:     errors.New("internal server error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			deps := newTestDeps(t)
			deps.ordersService.
				EXPECT().
				CreateOrder(gomock.Any(), userId, companyId, products).
				Return(testCase.serviceErr)

			responseRecorder := makeRequest(
				deps.router,
				http.MethodPost,
				"/api/v1/companies/"+companyId.String()+"/orders",
				[]byte(`{"products":[{"id":"`+productId.String()+`","price":49900}]}`),
				map[string]string{"Authorization": userId.String()},
			)

			assert.Equal(t, testCase.expectedStatus, responseRecorder.Code)
			var response core_response.ErrorResponse
			err := json.NewDecoder(responseRecorder.Body).Decode(&response)
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedCode, response.Error.Code)
			assert.Equal(t, testCase.serviceErr.Error(), response.Error.Message)
		})
	}
}
