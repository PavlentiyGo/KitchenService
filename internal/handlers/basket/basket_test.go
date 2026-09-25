package handler_basket_test

import (
	"avitoKitchen/internal/core/domain"
	core_errors "avitoKitchen/internal/core/errors"
	core_response "avitoKitchen/internal/core/http/response"
	core_server "avitoKitchen/internal/core/http/server"
	core_logger "avitoKitchen/internal/core/logger"
	handler_basket "avitoKitchen/internal/handlers/basket"
	"avitoKitchen/internal/handlers/dto"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	router        *http.ServeMux
	basketService *MockbasketService
}

func newTestDeps(t *testing.T) *testDeps {
	t.Helper()

	ctrl := gomock.NewController(t)
	basketService := NewMockbasketService(ctrl)

	handler := handler_basket.NewHandler(basketService, validator.New())

	router := core_server.NewApiVersionRouter("v1")
	router.ChainRoutes(handler.Routes()...)
	return &testDeps{
		router:        core_server.RegisterRouters(router),
		basketService: basketService,
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

func TestGetBasket_Success(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	basket := domain.Basket{
		Products: []domain.BasketProduct{
			{
				Product: domain.Product{
					Id:           productId,
					CompanyId:    companyId,
					Name:         "Пицца Маргарита",
					Price:        49900,
					Description:  "Томатный соус, моцарелла, томаты и базилик",
					CurrentCount: 18,
					IsAvailable:  true,
				},
				ConfirmedPrice: 49900,
				ChosenCount:    2,
			},
		},
		FullPrice:          99800,
		FullConfirmedPrice: 99800,
	}

	deps.basketService.
		EXPECT().
		GetBasket(gomock.Any(), userId, companyId).
		Return(basket, nil)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodGet,
		"/api/v1/companies/"+companyId.String()+"/basket",
		nil,
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusOK, responseRecorder.Code)
	var response dto.UserBasketDto
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)

	expected := dto.BasketDtoFromDomain(basket)
	assert.Equal(t, expected, response)
}

func TestGetBasket_InvalidUserId(t *testing.T) {
	deps := newTestDeps(t)
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")

	responseRecorder := makeRequest(
		deps.router,
		http.MethodGet,
		"/api/v1/companies/"+companyId.String()+"/basket",
		nil,
		map[string]string{"Authorization": "not-a-uuid"},
	)

	assert.Equal(t, http.StatusBadRequest, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, core_errors.ErrInvalidRequest.Error(), response.Error.Code)
	assert.Equal(t, core_errors.ErrInvalidAuthorization.Error(), response.Error.Message)
}

func TestGetBasket_InvalidCompanyId(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")

	responseRecorder := makeRequest(
		deps.router,
		http.MethodGet,
		"/api/v1/companies/not-a-uuid/basket",
		nil,
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusBadRequest, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, core_errors.ErrInvalidRequest.Error(), response.Error.Code)
	assert.Contains(t, response.Error.Message, "invalid companyId path parameter")
}

func TestGetBasket_InternalError(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	internalErr := errors.New("internal server error")

	deps.basketService.
		EXPECT().
		GetBasket(gomock.Any(), userId, companyId).
		Return(domain.Basket{}, internalErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodGet,
		"/api/v1/companies/"+companyId.String()+"/basket",
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

func TestPutBasketProduct_Success(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	count := 2

	deps.basketService.
		EXPECT().
		PutBasketProduct(gomock.Any(), userId, companyId, productId, count).
		Return(nil)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodPut,
		"/api/v1/companies/"+companyId.String()+"/basket/items/"+productId.String(),
		[]byte(`{"count":2}`),
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusNoContent, responseRecorder.Code)
	assert.Empty(t, responseRecorder.Body.String())
}

func TestPutBasketProduct_InvalidCount(t *testing.T) {
	testCases := []struct {
		name string
		body []byte
	}{
		{
			name: "zero count",
			body: []byte(`{"count":0}`),
		},
		{
			name: "negative count",
			body: []byte(`{"count":-1}`),
		},
		{
			name: "missing count",
			body: []byte(`{}`),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			deps := newTestDeps(t)
			userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
			companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
			productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")

			responseRecorder := makeRequest(
				deps.router,
				http.MethodPut,
				"/api/v1/companies/"+companyId.String()+"/basket/items/"+productId.String(),
				testCase.body,
				map[string]string{"Authorization": userId.String()},
			)

			assert.Equal(t, http.StatusBadRequest, responseRecorder.Code)
			var response core_response.ErrorResponse
			err := json.NewDecoder(responseRecorder.Body).Decode(&response)
			require.NoError(t, err)
			assert.Equal(t, core_errors.ErrInvalidRequest.Error(), response.Error.Code)
			assert.Contains(t, response.Error.Message, "failed to validate request")
		})
	}
}

func TestPutBasketProduct_InvalidUUID(t *testing.T) {
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	testCases := []struct {
		name          string
		target        string
		authorization string
	}{
		{
			name:          "user id",
			target:        "/api/v1/companies/" + companyId.String() + "/basket/items/" + productId.String(),
			authorization: "not-a-uuid",
		},
		{
			name:          "company id",
			target:        "/api/v1/companies/not-a-uuid/basket/items/" + productId.String(),
			authorization: userId.String(),
		},
		{
			name:          "product id",
			target:        "/api/v1/companies/" + companyId.String() + "/basket/items/not-a-uuid",
			authorization: userId.String(),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			deps := newTestDeps(t)

			responseRecorder := makeRequest(
				deps.router,
				http.MethodPut,
				testCase.target,
				[]byte(`{"count":2}`),
				map[string]string{"Authorization": testCase.authorization},
			)

			assert.Equal(t, http.StatusBadRequest, responseRecorder.Code)
			var response core_response.ErrorResponse
			err := json.NewDecoder(responseRecorder.Body).Decode(&response)
			require.NoError(t, err)
			assert.Equal(t, core_errors.ErrInvalidRequest.Error(), response.Error.Code)
		})
	}
}

func TestPutBasketProduct_ProductDoesNotBelongToCompany(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	serviceErr := fmt.Errorf("product does not belong to company: %w", core_errors.ErrNotFound)

	deps.basketService.
		EXPECT().
		PutBasketProduct(gomock.Any(), userId, companyId, productId, 2).
		Return(serviceErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodPut,
		"/api/v1/companies/"+companyId.String()+"/basket/items/"+productId.String(),
		[]byte(`{"count":2}`),
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusNotFound, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, core_errors.ErrNotFound.Error(), response.Error.Code)
	assert.Equal(t, serviceErr.Error(), response.Error.Message)
}

func TestPutBasketProduct_InsufficientStock(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	serviceErr := fmt.Errorf("requested count exceeds current stock: %w", core_errors.ErrInsufficientStock)

	deps.basketService.
		EXPECT().
		PutBasketProduct(gomock.Any(), userId, companyId, productId, 20).
		Return(serviceErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodPut,
		"/api/v1/companies/"+companyId.String()+"/basket/items/"+productId.String(),
		[]byte(`{"count":20}`),
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusConflict, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "conflict", response.Error.Code)
	assert.Equal(t, serviceErr.Error(), response.Error.Message)
}

func TestPutBasketProduct_ProductUnavailable(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	serviceErr := fmt.Errorf("product is unavailable: %w", core_errors.ErrProductUnavailable)

	deps.basketService.
		EXPECT().
		PutBasketProduct(gomock.Any(), userId, companyId, productId, 2).
		Return(serviceErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodPut,
		"/api/v1/companies/"+companyId.String()+"/basket/items/"+productId.String(),
		[]byte(`{"count":2}`),
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusConflict, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "conflict", response.Error.Code)
	assert.Equal(t, serviceErr.Error(), response.Error.Message)
}

func TestPutBasketProduct_InternalError(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	internalErr := errors.New("internal server error")

	deps.basketService.
		EXPECT().
		PutBasketProduct(gomock.Any(), userId, companyId, productId, 2).
		Return(internalErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodPut,
		"/api/v1/companies/"+companyId.String()+"/basket/items/"+productId.String(),
		[]byte(`{"count":2}`),
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusInternalServerError, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Empty(t, response.Error.Code)
	assert.Equal(t, internalErr.Error(), response.Error.Message)
}

func TestDeleteBasketProduct_Success(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")

	deps.basketService.
		EXPECT().
		DeleteBasketProduct(gomock.Any(), userId, companyId, productId).
		Return(nil)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodDelete,
		"/api/v1/companies/"+companyId.String()+"/basket/items/"+productId.String(),
		nil,
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusNoContent, responseRecorder.Code)
	assert.Empty(t, responseRecorder.Body.String())
}

func TestDeleteBasketProduct_InvalidUUID(t *testing.T) {
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	testCases := []struct {
		name          string
		target        string
		authorization string
	}{
		{
			name:          "user id",
			target:        "/api/v1/companies/" + companyId.String() + "/basket/items/" + productId.String(),
			authorization: "not-a-uuid",
		},
		{
			name:          "company id",
			target:        "/api/v1/companies/not-a-uuid/basket/items/" + productId.String(),
			authorization: userId.String(),
		},
		{
			name:          "product id",
			target:        "/api/v1/companies/" + companyId.String() + "/basket/items/not-a-uuid",
			authorization: userId.String(),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			deps := newTestDeps(t)

			responseRecorder := makeRequest(
				deps.router,
				http.MethodDelete,
				testCase.target,
				nil,
				map[string]string{"Authorization": testCase.authorization},
			)

			assert.Equal(t, http.StatusBadRequest, responseRecorder.Code)
			var response core_response.ErrorResponse
			err := json.NewDecoder(responseRecorder.Body).Decode(&response)
			require.NoError(t, err)
			assert.Equal(t, core_errors.ErrInvalidRequest.Error(), response.Error.Code)
		})
	}
}

func TestDeleteBasketProduct_ProductDoesNotBelongToCompany(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	serviceErr := fmt.Errorf(
		"product %s does not belong to company %s: %w",
		productId,
		companyId,
		core_errors.ErrNotFound,
	)

	deps.basketService.
		EXPECT().
		DeleteBasketProduct(gomock.Any(), userId, companyId, productId).
		Return(serviceErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodDelete,
		"/api/v1/companies/"+companyId.String()+"/basket/items/"+productId.String(),
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

func TestDeleteBasketProduct_ProductNotFound(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	serviceErr := fmt.Errorf("failed to get product: %w", core_errors.ErrNotFound)

	deps.basketService.
		EXPECT().
		DeleteBasketProduct(gomock.Any(), userId, companyId, productId).
		Return(serviceErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodDelete,
		"/api/v1/companies/"+companyId.String()+"/basket/items/"+productId.String(),
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

func TestDeleteBasketProduct_InternalError(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	internalErr := errors.New("internal server error")

	deps.basketService.
		EXPECT().
		DeleteBasketProduct(gomock.Any(), userId, companyId, productId).
		Return(internalErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodDelete,
		"/api/v1/companies/"+companyId.String()+"/basket/items/"+productId.String(),
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

func TestConfirmBasketProducts_Success(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	productsForChange := []domain.Product{
		{
			Id:    productId,
			Price: 49900,
		},
	}
	basketProducts := []domain.BasketProduct{
		{
			Product: domain.Product{
				Id:           productId,
				CompanyId:    companyId,
				Name:         "Пицца Маргарита",
				Price:        49900,
				Description:  "Томатный соус, моцарелла, томаты и базилик",
				CurrentCount: 18,
				IsAvailable:  true,
			},
			ConfirmedPrice: 49900,
			ChosenCount:    2,
		},
	}

	deps.basketService.
		EXPECT().
		ConfirmBasketProducts(gomock.Any(), productsForChange, companyId, userId).
		Return(basketProducts, nil)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodPatch,
		"/api/v1/companies/"+companyId.String()+"/basket/price-confirmation",
		[]byte(`{"products":[{"id":"`+productId.String()+`","price":49900}]}`),
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusOK, responseRecorder.Code)
	var response []dto.BasketProductDto
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)

	expected := dto.BasketProductsDtoFromDomains(basketProducts)
	assert.Equal(t, expected, response)
}

func TestConfirmBasketProducts_UserNotFound(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	productsForChange := []domain.Product{{Id: productId, Price: 49900}}
	serviceErr := fmt.Errorf("failed to get user: %w", core_errors.ErrNotFound)

	deps.basketService.
		EXPECT().
		ConfirmBasketProducts(gomock.Any(), productsForChange, companyId, userId).
		Return(nil, serviceErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodPatch,
		"/api/v1/companies/"+companyId.String()+"/basket/price-confirmation",
		[]byte(`{"products":[{"id":"`+productId.String()+`","price":49900}]}`),
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusNotFound, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, core_errors.ErrNotFound.Error(), response.Error.Code)
	assert.Equal(t, serviceErr.Error(), response.Error.Message)
}

func TestConfirmBasketProducts_CompanyNotFound(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	productsForChange := []domain.Product{{Id: productId, Price: 49900}}
	serviceErr := fmt.Errorf("failed to get company: %w", core_errors.ErrNotFound)

	deps.basketService.
		EXPECT().
		ConfirmBasketProducts(gomock.Any(), productsForChange, companyId, userId).
		Return(nil, serviceErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodPatch,
		"/api/v1/companies/"+companyId.String()+"/basket/price-confirmation",
		[]byte(`{"products":[{"id":"`+productId.String()+`","price":49900}]}`),
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusNotFound, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, core_errors.ErrNotFound.Error(), response.Error.Code)
	assert.Equal(t, serviceErr.Error(), response.Error.Message)
}

func TestConfirmBasketProducts_InvalidUUID(t *testing.T) {
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	testCases := []struct {
		name          string
		target        string
		authorization string
		productId     string
	}{
		{
			name:          "user id",
			target:        "/api/v1/companies/" + companyId.String() + "/basket/price-confirmation",
			authorization: "not-a-uuid",
			productId:     productId.String(),
		},
		{
			name:          "company id",
			target:        "/api/v1/companies/not-a-uuid/basket/price-confirmation",
			authorization: userId.String(),
			productId:     productId.String(),
		},
		{
			name:          "product id",
			target:        "/api/v1/companies/" + companyId.String() + "/basket/price-confirmation",
			authorization: userId.String(),
			productId:     "not-a-uuid",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			deps := newTestDeps(t)

			responseRecorder := makeRequest(
				deps.router,
				http.MethodPatch,
				testCase.target,
				[]byte(`{"products":[{"id":"`+testCase.productId+`","price":49900}]}`),
				map[string]string{"Authorization": testCase.authorization},
			)

			assert.Equal(t, http.StatusBadRequest, responseRecorder.Code)
			var response core_response.ErrorResponse
			err := json.NewDecoder(responseRecorder.Body).Decode(&response)
			require.NoError(t, err)
			assert.Equal(t, core_errors.ErrInvalidRequest.Error(), response.Error.Code)
		})
	}
}

func TestConfirmBasketProducts_DuplicatedProducts(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	productsForChange := []domain.Product{
		{Id: productId, Price: 49900},
		{Id: productId, Price: 49900},
	}
	serviceErr := fmt.Errorf("product %s is duplicated: %w", productId, core_errors.ErrInvalidRequest)

	deps.basketService.
		EXPECT().
		ConfirmBasketProducts(gomock.Any(), productsForChange, companyId, userId).
		Return(nil, serviceErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodPatch,
		"/api/v1/companies/"+companyId.String()+"/basket/price-confirmation",
		[]byte(
			`{"products":[`+
				`{"id":"`+productId.String()+`","price":49900},`+
				`{"id":"`+productId.String()+`","price":49900}`+
				`]}`,
		),
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusBadRequest, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, core_errors.ErrInvalidRequest.Error(), response.Error.Code)
	assert.Equal(t, serviceErr.Error(), response.Error.Message)
}

func TestConfirmBasketProducts_InvalidPrice(t *testing.T) {
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	testCases := []struct {
		name  string
		price int64
	}{
		{
			name:  "zero price",
			price: 0,
		},
		{
			name:  "negative price",
			price: -1,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			deps := newTestDeps(t)
			body := fmt.Sprintf(
				`{"products":[{"id":"%s","price":%d}]}`,
				productId,
				testCase.price,
			)

			responseRecorder := makeRequest(
				deps.router,
				http.MethodPatch,
				"/api/v1/companies/"+companyId.String()+"/basket/price-confirmation",
				[]byte(body),
				map[string]string{"Authorization": userId.String()},
			)

			assert.Equal(t, http.StatusBadRequest, responseRecorder.Code)
			var response core_response.ErrorResponse
			err := json.NewDecoder(responseRecorder.Body).Decode(&response)
			require.NoError(t, err)
			assert.Equal(t, core_errors.ErrInvalidRequest.Error(), response.Error.Code)
			assert.Contains(t, response.Error.Message, "failed to validate request")
		})
	}
}

func TestConfirmBasketProducts_PriceChanged(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	productsForChange := []domain.Product{{Id: productId, Price: 49900}}
	serviceErr := fmt.Errorf("product price changed from 49900 to 59900: %w", core_errors.ErrConflict)

	deps.basketService.
		EXPECT().
		ConfirmBasketProducts(gomock.Any(), productsForChange, companyId, userId).
		Return(nil, serviceErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodPatch,
		"/api/v1/companies/"+companyId.String()+"/basket/price-confirmation",
		[]byte(`{"products":[{"id":"`+productId.String()+`","price":49900}]}`),
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusConflict, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "conflict", response.Error.Code)
	assert.Equal(t, serviceErr.Error(), response.Error.Message)
}

func TestConfirmBasketProducts_ProductNotInBasket(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	productsForChange := []domain.Product{{Id: productId, Price: 49900}}
	serviceErr := fmt.Errorf("product %s is not in user basket: %w", productId, core_errors.ErrNotFound)

	deps.basketService.
		EXPECT().
		ConfirmBasketProducts(gomock.Any(), productsForChange, companyId, userId).
		Return(nil, serviceErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodPatch,
		"/api/v1/companies/"+companyId.String()+"/basket/price-confirmation",
		[]byte(`{"products":[{"id":"`+productId.String()+`","price":49900}]}`),
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusNotFound, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, core_errors.ErrNotFound.Error(), response.Error.Code)
	assert.Equal(t, serviceErr.Error(), response.Error.Message)
}

func TestConfirmBasketProducts_InternalError(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	productsForChange := []domain.Product{{Id: productId, Price: 49900}}
	internalErr := errors.New("internal server error")

	deps.basketService.
		EXPECT().
		ConfirmBasketProducts(gomock.Any(), productsForChange, companyId, userId).
		Return(nil, internalErr)

	responseRecorder := makeRequest(
		deps.router,
		http.MethodPatch,
		"/api/v1/companies/"+companyId.String()+"/basket/price-confirmation",
		[]byte(`{"products":[{"id":"`+productId.String()+`","price":49900}]}`),
		map[string]string{"Authorization": userId.String()},
	)

	assert.Equal(t, http.StatusInternalServerError, responseRecorder.Code)
	var response core_response.ErrorResponse
	err := json.NewDecoder(responseRecorder.Body).Decode(&response)
	require.NoError(t, err)
	assert.Empty(t, response.Error.Code)
	assert.Equal(t, internalErr.Error(), response.Error.Message)
}
