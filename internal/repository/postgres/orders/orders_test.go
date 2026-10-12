package repository_orders_test

import (
	"context"
	"errors"
	"fmt"
	"foodDelivery/internal/core/domain"
	core_errors "foodDelivery/internal/core/errors"
	core_messaging "foodDelivery/internal/core/messaging"
	"foodDelivery/internal/repository"
	repository_orders "foodDelivery/internal/repository/postgres/orders"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type txManagerMock struct {
	pool pgxmock.PgxPoolIface
}

func (t *txManagerMock) WithinTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	return fn(ctx)
}
func (t *txManagerMock) GetExecutor(ctx context.Context) repository.Executor {
	return t.pool
}
func (t *txManagerMock) GetTimeout() time.Duration {
	return 5 * time.Second
}
func (t *testDeps) assertExpectations() error {
	if err := t.pool.ExpectationsWereMet(); err != nil {
		return fmt.Errorf("not all expectations were met: %w", err)
	}
	return nil
}

type testDeps struct {
	pool pgxmock.PgxPoolIface
	repo *repository_orders.Repository
}

func newTestDeps(t *testing.T) *testDeps {
	t.Helper()
	pool, err := pgxmock.NewPool()
	require.NoError(t, err)
	txManager := &txManagerMock{pool: pool}
	return &testDeps{
		pool: pool,
		repo: repository_orders.NewRepository(txManager),
	}
}

func TestDecreaseProductsCount_Success(t *testing.T) {
	deps := newTestDeps(t)

	companyId := uuid.New()
	products := []domain.BasketProduct{
		{Product: domain.Product{Id: uuid.New()}, ChosenCount: 2},
		{Product: domain.Product{Id: uuid.New()}, ChosenCount: 3},
	}
	productIds := []uuid.UUID{products[0].Id, products[1].Id}
	counts := []int{products[0].ChosenCount, products[1].ChosenCount}

	deps.pool.ExpectExec("UPDATE company_products").
		WithArgs(companyId, productIds, counts).
		WillReturnResult(pgconn.NewCommandTag("UPDATE 2"))

	err := deps.repo.DecreaseProductsCount(t.Context(), companyId, products)
	require.NoError(t, err)
	assert.NoError(t, deps.assertExpectations())
}

func TestDecreaseProductsCount_InvalidProducts(t *testing.T) {
	tests := []struct {
		name     string
		products []domain.BasketProduct
	}{
		{name: "empty products"},
		{
			name: "empty product id",
			products: []domain.BasketProduct{
				{Product: domain.Product{Id: uuid.Nil}, ChosenCount: 1},
			},
		},
		{
			name: "invalid product count",
			products: []domain.BasketProduct{
				{Product: domain.Product{Id: uuid.New()}, ChosenCount: 0},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deps := newTestDeps(t)

			err := deps.repo.DecreaseProductsCount(t.Context(), uuid.New(), test.products)
			require.ErrorIs(t, err, core_errors.ErrInvalidRequest)
			assert.NoError(t, deps.assertExpectations())
		})
	}
}

func TestDecreaseProductsCount_InsufficientStock(t *testing.T) {
	deps := newTestDeps(t)

	companyId := uuid.New()
	products := []domain.BasketProduct{
		{Product: domain.Product{Id: uuid.New()}, ChosenCount: 2},
	}
	productIds := []uuid.UUID{products[0].Id}
	counts := []int{products[0].ChosenCount}

	deps.pool.ExpectExec("UPDATE company_products").
		WithArgs(companyId, productIds, counts).
		WillReturnError(&pgconn.PgError{Code: "23514"})

	err := deps.repo.DecreaseProductsCount(t.Context(), companyId, products)
	require.ErrorIs(t, err, core_errors.ErrInsufficientStock)
	assert.NoError(t, deps.assertExpectations())
}

func TestDecreaseProductsCount_Fail(t *testing.T) {
	deps := newTestDeps(t)

	companyId := uuid.New()
	products := []domain.BasketProduct{
		{Product: domain.Product{Id: uuid.New()}, ChosenCount: 2},
		{Product: domain.Product{Id: uuid.New()}, ChosenCount: 3},
	}
	productIds := []uuid.UUID{products[0].Id, products[1].Id}
	counts := []int{products[0].ChosenCount, products[1].ChosenCount}

	deps.pool.ExpectExec("UPDATE company_products").
		WithArgs(companyId, productIds, counts).
		WillReturnResult(pgconn.NewCommandTag("UPDATE 1"))

	err := deps.repo.DecreaseProductsCount(t.Context(), companyId, products)
	require.ErrorIs(t, err, core_errors.ErrProductUnavailable)
	assert.NoError(t, deps.assertExpectations())
}

func TestDecreaseProductsCount_Internal(t *testing.T) {
	deps := newTestDeps(t)

	companyId := uuid.New()
	products := []domain.BasketProduct{
		{Product: domain.Product{Id: uuid.New()}, ChosenCount: 2},
	}
	productIds := []uuid.UUID{products[0].Id}
	counts := []int{products[0].ChosenCount}
	unexpectedErr := errors.New("unexpected error")

	deps.pool.ExpectExec("UPDATE company_products").
		WithArgs(companyId, productIds, counts).
		WillReturnError(unexpectedErr)

	err := deps.repo.DecreaseProductsCount(t.Context(), companyId, products)
	require.ErrorIs(t, err, unexpectedErr)
	assert.NoError(t, deps.assertExpectations())
}

func TestCancelOrder_Success(t *testing.T) {
	deps := newTestDeps(t)

	orderId := uuid.New()
	cancelReason := core_messaging.RetryReasonCompanyRejected

	deps.pool.ExpectExec("UPDATE user_orders").
		WithArgs(orderId, cancelReason).
		WillReturnResult(pgconn.NewCommandTag("UPDATE 1"))

	err := deps.repo.CancelOrder(t.Context(), orderId, cancelReason)
	require.NoError(t, err)
	assert.NoError(t, deps.assertExpectations())
}

func TestCancelOrder_Fail(t *testing.T) {
	deps := newTestDeps(t)

	orderId := uuid.New()
	cancelReason := core_messaging.RetryReasonCompanyRejected

	deps.pool.ExpectExec("UPDATE user_orders").
		WithArgs(orderId, cancelReason).
		WillReturnResult(pgconn.NewCommandTag("UPDATE 0"))

	err := deps.repo.CancelOrder(t.Context(), orderId, cancelReason)
	require.ErrorIs(t, err, core_errors.ErrNotFound)
	assert.NoError(t, deps.assertExpectations())
}

func TestCancelOrder_Internal(t *testing.T) {
	deps := newTestDeps(t)

	orderId := uuid.New()
	cancelReason := core_messaging.RetryReasonCompanyRejected
	unexpectedErr := errors.New("unexpected error")

	deps.pool.ExpectExec("UPDATE user_orders").
		WithArgs(orderId, cancelReason).
		WillReturnError(unexpectedErr)

	err := deps.repo.CancelOrder(t.Context(), orderId, cancelReason)
	require.ErrorIs(t, err, unexpectedErr)
	assert.NoError(t, deps.assertExpectations())
}

func TestRestoreOrderProducts_Success(t *testing.T) {
	deps := newTestDeps(t)

	orderId := uuid.New()

	deps.pool.ExpectExec("UPDATE company_products").
		WithArgs(orderId).
		WillReturnResult(pgconn.NewCommandTag("UPDATE 2"))

	err := deps.repo.RestoreOrderProducts(t.Context(), orderId)
	require.NoError(t, err)
	assert.NoError(t, deps.assertExpectations())
}

func TestRestoreOrderProducts_Fail(t *testing.T) {
	deps := newTestDeps(t)

	orderId := uuid.New()

	deps.pool.ExpectExec("UPDATE company_products").
		WithArgs(orderId).
		WillReturnResult(pgconn.NewCommandTag("UPDATE 0"))

	err := deps.repo.RestoreOrderProducts(t.Context(), orderId)
	require.ErrorIs(t, err, core_errors.ErrNotFound)
	assert.NoError(t, deps.assertExpectations())
}

func TestRestoreOrderProducts_Internal(t *testing.T) {
	deps := newTestDeps(t)

	orderId := uuid.New()
	unexpectedErr := errors.New("unexpected error")

	deps.pool.ExpectExec("UPDATE company_products").
		WithArgs(orderId).
		WillReturnError(unexpectedErr)

	err := deps.repo.RestoreOrderProducts(t.Context(), orderId)
	require.ErrorIs(t, err, unexpectedErr)
	assert.NoError(t, deps.assertExpectations())
}

func TestCreateOrder_Success(t *testing.T) {
	deps := newTestDeps(t)

	userId := uuid.New()
	companyId := uuid.New()
	orderIdExpect := uuid.New()

	deps.pool.ExpectQuery("INSERT INTO user_orders").
		WithArgs(userId, companyId).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(orderIdExpect))

	orderId, err := deps.repo.CreateOrder(t.Context(), userId, companyId)
	require.NoError(t, err)
	require.Equal(t, orderIdExpect, orderId)
	assert.NoError(t, deps.assertExpectations())
}

func TestCreateOrder_Internal(t *testing.T) {
	deps := newTestDeps(t)

	userId := uuid.New()
	companyId := uuid.New()
	unexpectedErr := errors.New("unexpected error")

	deps.pool.ExpectQuery("INSERT INTO user_orders").
		WithArgs(userId, companyId).
		WillReturnError(unexpectedErr)

	_, err := deps.repo.CreateOrder(t.Context(), userId, companyId)
	require.ErrorIs(t, err, unexpectedErr)
	assert.NoError(t, deps.assertExpectations())
}

func TestCreateOrderItems_Success(t *testing.T) {
	deps := newTestDeps(t)

	orderId := uuid.New()
	products := []domain.BasketProduct{
		{
			Product:        domain.Product{Id: uuid.New(), Name: "first product"},
			ChosenCount:    2,
			ConfirmedPrice: 100,
		},
		{
			Product:        domain.Product{Id: uuid.New(), Name: "second product"},
			ChosenCount:    3,
			ConfirmedPrice: 200,
		},
	}
	productIds := []uuid.UUID{products[0].Id, products[1].Id}
	counts := []int{products[0].ChosenCount, products[1].ChosenCount}
	productNames := []string{products[0].Name, products[1].Name}
	productPrices := []int64{products[0].ConfirmedPrice, products[1].ConfirmedPrice}

	deps.pool.ExpectExec("INSERT INTO user_order_items").
		WithArgs(orderId, productIds, counts, productNames, productPrices).
		WillReturnResult(pgconn.NewCommandTag("INSERT 0 2"))

	err := deps.repo.CreateOrderItems(t.Context(), orderId, products)
	require.NoError(t, err)
	assert.NoError(t, deps.assertExpectations())
}

func TestCreateOrderItems_Fail(t *testing.T) {
	deps := newTestDeps(t)

	err := deps.repo.CreateOrderItems(t.Context(), uuid.New(), nil)
	require.ErrorIs(t, err, core_errors.ErrInvalidRequest)
	assert.NoError(t, deps.assertExpectations())
}

func TestCreateOrderItems_Internal(t *testing.T) {
	deps := newTestDeps(t)

	orderId := uuid.New()
	products := []domain.BasketProduct{
		{
			Product:        domain.Product{Id: uuid.New(), Name: "product"},
			ChosenCount:    2,
			ConfirmedPrice: 100,
		},
	}
	productIds := []uuid.UUID{products[0].Id}
	counts := []int{products[0].ChosenCount}
	productNames := []string{products[0].Name}
	productPrices := []int64{products[0].ConfirmedPrice}
	unexpectedErr := errors.New("unexpected error")

	deps.pool.ExpectExec("INSERT INTO user_order_items").
		WithArgs(orderId, productIds, counts, productNames, productPrices).
		WillReturnError(unexpectedErr)

	err := deps.repo.CreateOrderItems(t.Context(), orderId, products)
	require.ErrorIs(t, err, unexpectedErr)
	assert.NoError(t, deps.assertExpectations())
}

func TestGetOrderForSend_Success(t *testing.T) {
	deps := newTestDeps(t)

	orderExpect := domain.OrderForSend{
		OrderId:    uuid.New(),
		CompanyUrl: "https://company.test",
		Products: []domain.ProductForSend{
			{PartnerProductId: "first_partner_product_id", Count: 2},
			{PartnerProductId: "second_partner_product_id", Count: 3},
		},
	}

	deps.pool.ExpectQuery("c.company_source_url").
		WithArgs(orderExpect.OrderId).
		WillReturnRows(
			pgxmock.NewRows(
				[]string{"id", "company_source_url", "partner_product_id", "count"},
			).AddRow(
				orderExpect.OrderId,
				orderExpect.CompanyUrl,
				orderExpect.Products[0].PartnerProductId,
				orderExpect.Products[0].Count,
			).AddRow(
				orderExpect.OrderId,
				orderExpect.CompanyUrl,
				orderExpect.Products[1].PartnerProductId,
				orderExpect.Products[1].Count,
			),
		)

	order, err := deps.repo.GetOrderForSend(t.Context(), orderExpect.OrderId)
	require.NoError(t, err)
	require.Equal(t, orderExpect, order)
	assert.NoError(t, deps.assertExpectations())
}

func TestGetOrderForSend_Fail(t *testing.T) {
	deps := newTestDeps(t)

	orderId := uuid.New()

	deps.pool.ExpectQuery("c.company_source_url").
		WithArgs(orderId).
		WillReturnRows(
			pgxmock.NewRows(
				[]string{"id", "company_source_url", "partner_product_id", "count"},
			),
		)

	_, err := deps.repo.GetOrderForSend(t.Context(), orderId)
	require.ErrorIs(t, err, core_errors.ErrNotFound)
	assert.NoError(t, deps.assertExpectations())
}

func TestGetOrderForSend_Internal(t *testing.T) {
	deps := newTestDeps(t)

	orderId := uuid.New()
	unexpectedErr := errors.New("unexpected error")

	deps.pool.ExpectQuery("c.company_source_url").
		WithArgs(orderId).
		WillReturnError(unexpectedErr)

	_, err := deps.repo.GetOrderForSend(t.Context(), orderId)
	require.ErrorIs(t, err, unexpectedErr)
	assert.NoError(t, deps.assertExpectations())
}

func TestGetUserOrders_Success(t *testing.T) {
	deps := newTestDeps(t)

	rejectionReason := "company rejected order"
	orderExpect := domain.OrderRow{
		OrderId:         uuid.New(),
		UserId:          uuid.New(),
		CompanyId:       uuid.New(),
		Status:          domain.OrderStatusRejected,
		RejectionReason: &rejectionReason,
		Product: domain.OrderProduct{
			Id:    uuid.New(),
			Name:  "product",
			Price: 100,
			Count: 2,
		},
		CreatedAt: time.Date(2026, time.September, 8, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.September, 8, 12, 1, 0, 0, time.UTC),
	}

	deps.pool.ExpectQuery("WHERE uo.user_id").
		WithArgs(orderExpect.UserId).
		WillReturnRows(
			pgxmock.NewRows(
				[]string{
					"order_id", "user_id", "company_id", "status", "rejection_reason",
					"created_at", "updated_at", "product_id", "product_name", "product_price", "count",
				},
			).AddRow(
				orderExpect.OrderId,
				orderExpect.UserId,
				orderExpect.CompanyId,
				string(orderExpect.Status),
				orderExpect.RejectionReason,
				orderExpect.CreatedAt,
				orderExpect.UpdatedAt,
				orderExpect.Product.Id,
				orderExpect.Product.Name,
				orderExpect.Product.Price,
				orderExpect.Product.Count,
			),
		)

	orders, err := deps.repo.GetUserOrders(t.Context(), orderExpect.UserId)
	require.NoError(t, err)
	require.Equal(t, []domain.OrderRow{orderExpect}, orders)
	assert.NoError(t, deps.assertExpectations())
}

func TestGetUserOrders_Internal(t *testing.T) {
	deps := newTestDeps(t)

	userId := uuid.New()
	unexpectedErr := errors.New("unexpected error")

	deps.pool.ExpectQuery("WHERE uo.user_id").
		WithArgs(userId).
		WillReturnError(unexpectedErr)

	_, err := deps.repo.GetUserOrders(t.Context(), userId)
	require.ErrorIs(t, err, unexpectedErr)
	assert.NoError(t, deps.assertExpectations())
}

func TestGetOrder_Success(t *testing.T) {
	deps := newTestDeps(t)

	orderExpect := domain.OrderRow{
		OrderId:   uuid.New(),
		UserId:    uuid.New(),
		CompanyId: uuid.New(),
		Status:    domain.OrderStatusAccepted,
		Product: domain.OrderProduct{
			Id:    uuid.New(),
			Name:  "product",
			Price: 100,
			Count: 2,
		},
		CreatedAt: time.Date(2026, time.September, 8, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.September, 8, 12, 1, 0, 0, time.UTC),
	}

	deps.pool.ExpectQuery("WHERE uo.id").
		WithArgs(orderExpect.OrderId).
		WillReturnRows(
			pgxmock.NewRows(
				[]string{
					"order_id", "user_id", "company_id", "status", "rejection_reason",
					"created_at", "updated_at", "product_id", "product_name", "product_price", "count",
				},
			).AddRow(
				orderExpect.OrderId,
				orderExpect.UserId,
				orderExpect.CompanyId,
				string(orderExpect.Status),
				orderExpect.RejectionReason,
				orderExpect.CreatedAt,
				orderExpect.UpdatedAt,
				orderExpect.Product.Id,
				orderExpect.Product.Name,
				orderExpect.Product.Price,
				orderExpect.Product.Count,
			),
		)

	orders, err := deps.repo.GetOrder(t.Context(), orderExpect.OrderId)
	require.NoError(t, err)
	require.Equal(t, []domain.OrderRow{orderExpect}, orders)
	assert.NoError(t, deps.assertExpectations())
}

func TestGetOrder_Fail(t *testing.T) {
	deps := newTestDeps(t)

	orderId := uuid.New()

	deps.pool.ExpectQuery("WHERE uo.id").
		WithArgs(orderId).
		WillReturnRows(
			pgxmock.NewRows(
				[]string{
					"order_id", "user_id", "company_id", "status", "rejection_reason",
					"created_at", "updated_at", "product_id", "product_name", "product_price", "count",
				},
			),
		)

	_, err := deps.repo.GetOrder(t.Context(), orderId)
	require.ErrorIs(t, err, core_errors.ErrNotFound)
	assert.NoError(t, deps.assertExpectations())
}

func TestGetOrder_Internal(t *testing.T) {
	deps := newTestDeps(t)

	orderId := uuid.New()
	unexpectedErr := errors.New("unexpected error")

	deps.pool.ExpectQuery("WHERE uo.id").
		WithArgs(orderId).
		WillReturnError(unexpectedErr)

	_, err := deps.repo.GetOrder(t.Context(), orderId)
	require.ErrorIs(t, err, unexpectedErr)
	assert.NoError(t, deps.assertExpectations())
}
