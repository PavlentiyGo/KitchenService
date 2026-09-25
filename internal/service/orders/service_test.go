package service_orders_test

import (
	"avitoKitchen/internal/core/domain"
	service_orders "avitoKitchen/internal/service/orders"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type testDeps struct {
	service       *service_orders.Service
	txManager     *MocktxManager
	ordersRepo    *MockordersRepository
	companiesRepo *MockcompaniesRepository
	sender        *MockordersSender
	menuRepo      *MockmenuRepo
	basketRepo    *MockbasketRepo
}

func newTestDeps(t *testing.T) *testDeps {
	t.Helper()

	ctrl := gomock.NewController(t)
	txManager := NewMocktxManager(ctrl)
	ordersRepo := NewMockordersRepository(ctrl)
	companiesRepo := NewMockcompaniesRepository(ctrl)
	sender := NewMockordersSender(ctrl)
	menuRepo := NewMockmenuRepo(ctrl)
	basketRepo := NewMockbasketRepo(ctrl)

	return &testDeps{
		service: service_orders.NewService(
			txManager,
			ordersRepo,
			companiesRepo,
			sender,
			menuRepo,
			basketRepo,
		),
		txManager:     txManager,
		ordersRepo:    ordersRepo,
		companiesRepo: companiesRepo,
		sender:        sender,
		menuRepo:      menuRepo,
		basketRepo:    basketRepo,
	}
}

type transactionContextKey struct{}

func TestCreateOrder_Success(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	txCtx := context.WithValue(ctx, transactionContextKey{}, true)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	orderId := uuid.MustParse("44444444-4444-4444-8444-444444444444")
	firstProductId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	secondProductId := uuid.MustParse("33333333-3333-4333-8333-000000000002")
	basketItems := []domain.Product{
		{Id: firstProductId, Price: 49900},
		{Id: secondProductId, Price: 24900},
	}
	productIds := []uuid.UUID{firstProductId, secondProductId}
	basketProducts := []domain.BasketProduct{
		{
			Product:        domain.Product{Id: firstProductId, CompanyId: companyId, Name: "Пицца Маргарита", Price: 49900},
			ConfirmedPrice: 49900,
			ChosenCount:    2,
		},
		{
			Product:        domain.Product{Id: secondProductId, CompanyId: companyId, Name: "Лимонад", Price: 24900},
			ConfirmedPrice: 24900,
			ChosenCount:    1,
		},
	}

	deps.menuRepo.
		EXPECT().
		GetUser(ctx, userId).
		Return(nil)
	deps.menuRepo.
		EXPECT().
		GetCompany(ctx, companyId).
		Return(domain.Company{Id: companyId}, nil)
	deps.txManager.
		EXPECT().
		WithinTx(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
			return fn(txCtx)
		})
	deps.basketRepo.
		EXPECT().
		ConfirmBasketProductsPrice(txCtx, userId, companyId, productIds).
		Return(nil)
	deps.basketRepo.
		EXPECT().
		GetUserBasketProducts(txCtx, userId, companyId).
		Return(basketProducts, nil)
	deps.ordersRepo.
		EXPECT().
		DecreaseProductsCount(txCtx, companyId, basketProducts).
		Return(nil)
	deps.ordersRepo.
		EXPECT().
		CreateOrder(txCtx, userId, companyId).
		Return(orderId, nil)
	deps.ordersRepo.
		EXPECT().
		CreateOrderItems(txCtx, orderId, basketProducts).
		Return(nil)
	deps.companiesRepo.
		EXPECT().
		AddCompanyOrderStatusHistory(txCtx, orderId, domain.OrderStatusCreated).
		Return(nil)
	deps.basketRepo.
		EXPECT().
		ClearBasket(txCtx, userId, companyId).
		Return(nil)
	deps.sender.
		EXPECT().
		SendOrderCreated(txCtx, orderId).
		Return(nil)

	err := deps.service.CreateOrder(ctx, userId, companyId, basketItems)

	require.NoError(t, err)
}

func TestGetMyOrders_Success(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	firstOrderId := uuid.MustParse("44444444-4444-4444-8444-000000000001")
	secondOrderId := uuid.MustParse("44444444-4444-4444-8444-000000000002")
	createdAt := time.Date(2026, time.September, 8, 10, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(5 * time.Minute)
	rows := []domain.OrderRow{
		{
			OrderId: firstOrderId, UserId: userId, CompanyId: companyId,
			Status: domain.OrderStatusCreated, Product: domain.OrderProduct{Id: uuid.New(), Name: "Пицца", Price: 500, Count: 2},
			CreatedAt: createdAt, UpdatedAt: updatedAt,
		},
		{
			OrderId: firstOrderId, UserId: userId, CompanyId: companyId,
			Status: domain.OrderStatusCreated, Product: domain.OrderProduct{Id: uuid.New(), Name: "Сок", Price: 100, Count: 3},
			CreatedAt: createdAt, UpdatedAt: updatedAt,
		},
		{
			OrderId: secondOrderId, UserId: userId, CompanyId: companyId,
			Status: domain.OrderStatusCompleted, Product: domain.OrderProduct{Id: uuid.New(), Name: "Салат", Price: 250, Count: 1},
			CreatedAt: createdAt.Add(time.Hour), UpdatedAt: updatedAt.Add(time.Hour),
		},
	}
	expected := []domain.Order{
		{
			Id: firstOrderId, UserId: userId, CompanyId: companyId, Status: domain.OrderStatusCreated,
			Products: []domain.OrderProduct{rows[0].Product, rows[1].Product}, FullPrice: 1300,
			CreatedAt: createdAt, UpdatedAt: updatedAt,
		},
		{
			Id: secondOrderId, UserId: userId, CompanyId: companyId, Status: domain.OrderStatusCompleted,
			Products: []domain.OrderProduct{rows[2].Product}, FullPrice: 250,
			CreatedAt: createdAt.Add(time.Hour), UpdatedAt: updatedAt.Add(time.Hour),
		},
	}

	deps.menuRepo.
		EXPECT().
		GetUser(ctx, userId).
		Return(nil)
	deps.ordersRepo.
		EXPECT().
		GetUserOrders(ctx, userId).
		Return(rows, nil)

	orders, err := deps.service.GetMyOrders(ctx, userId)

	require.NoError(t, err)
	assert.Equal(t, expected, orders)
}

func TestGetUserOrder_Success(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	orderId := uuid.MustParse("44444444-4444-4444-8444-444444444444")
	createdAt := time.Date(2026, time.September, 8, 10, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(5 * time.Minute)
	rows := []domain.OrderRow{
		{
			OrderId: orderId, UserId: userId, CompanyId: companyId,
			Status: domain.OrderStatusAccepted, Product: domain.OrderProduct{Id: uuid.New(), Name: "Пицца", Price: 500, Count: 2},
			CreatedAt: createdAt, UpdatedAt: updatedAt,
		},
		{
			OrderId: orderId, UserId: userId, CompanyId: companyId,
			Status: domain.OrderStatusAccepted, Product: domain.OrderProduct{Id: uuid.New(), Name: "Сок", Price: 100, Count: 1},
			CreatedAt: createdAt, UpdatedAt: updatedAt,
		},
	}
	expected := domain.Order{
		Id: orderId, UserId: userId, CompanyId: companyId, Status: domain.OrderStatusAccepted,
		Products: []domain.OrderProduct{rows[0].Product, rows[1].Product}, FullPrice: 1100,
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	}

	deps.menuRepo.
		EXPECT().
		GetUser(ctx, userId).
		Return(nil)
	deps.ordersRepo.
		EXPECT().
		GetOrder(ctx, orderId).
		Return(rows, nil)

	order, err := deps.service.GetUserOrder(ctx, userId, orderId)

	require.NoError(t, err)
	assert.Equal(t, expected, order)
}
