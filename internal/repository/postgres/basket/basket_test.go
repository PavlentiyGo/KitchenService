package repository_basket_test

import (
	"avitoKitchen/internal/core/domain"
	core_errors "avitoKitchen/internal/core/errors"
	"avitoKitchen/internal/repository"
	repository_basket "avitoKitchen/internal/repository/postgres/basket"
	"context"
	"errors"
	"fmt"
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
	repo *repository_basket.Repository
}

func newTestDeps(t *testing.T) *testDeps {
	t.Helper()
	pool, err := pgxmock.NewPool()
	require.NoError(t, err)
	txManager := &txManagerMock{pool: pool}

	return &testDeps{
		pool: pool,
		repo: repository_basket.NewRepository(txManager),
	}
}
func TestDeleteBasketProduct_Success(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.New()
	partnerProductId := "partner_product_id"
	productExpect := domain.Product{
		Id:               uuid.New(),
		CompanyId:        uuid.New(),
		PartnerProductId: &partnerProductId,
		Name:             "product",
		Price:            100,
		Description:      "description",
		CurrentCount:     10,
		IsAvailable:      true,
	}

	deps.pool.ExpectExec(
		"DELETE FROM user_basket_products",
	).WithArgs(userId, productExpect.CompanyId, productExpect.Id).WillReturnResult(pgconn.NewCommandTag("nice"))

	err := deps.repo.DeleteBasketProduct(t.Context(), userId, productExpect.CompanyId, productExpect.Id)
	require.NoError(t, err)
	assert.NoError(t, deps.assertExpectations())
}

func TestDeleteBasketProduct_Internal(t *testing.T) {
	deps := newTestDeps(t)
	userId := uuid.New()
	partnerProductId := "partner_product_id"
	productExpect := domain.Product{
		Id:               uuid.New(),
		CompanyId:        uuid.New(),
		PartnerProductId: &partnerProductId,
		Name:             "product",
		Price:            100,
		Description:      "description",
		CurrentCount:     10,
		IsAvailable:      true,
	}
	unexpectedErr := errors.New("unexpected error")
	deps.pool.ExpectExec(
		"DELETE FROM user_basket_products",
	).WithArgs(userId, productExpect.CompanyId, productExpect.Id).WillReturnError(unexpectedErr)

	err := deps.repo.DeleteBasketProduct(t.Context(), userId, productExpect.CompanyId, productExpect.Id)
	require.ErrorIs(t, err, unexpectedErr)
	assert.NoError(t, deps.assertExpectations())
}

func TestGetUserBasketProducts_Success(t *testing.T) {
	deps := newTestDeps(t)

	userId := uuid.New()
	companyId := uuid.New()
	productExpect := domain.BasketProduct{
		Product: domain.Product{
			Id:           uuid.New(),
			Name:         "product",
			Price:        100,
			Description:  "description",
			CurrentCount: 10,
			IsAvailable:  true,
		},
		ChosenCount:    2,
		ConfirmedPrice: 90,
	}

	deps.pool.ExpectQuery(
		"SELECT cp.id,cp.name,cp.price,cp.description,cp.is_available,cp.current_count,ubp.count,ubp.confirmed_price",
	).WithArgs(companyId, userId).
		WillReturnRows(
			pgxmock.NewRows(
				[]string{
					"id", "name", "price", "description", "is_available", "current_count", "count", "confirmed_price",
				}).AddRow(
				productExpect.Id,
				productExpect.Name,
				productExpect.Price,
				productExpect.Description,
				productExpect.IsAvailable,
				productExpect.CurrentCount,
				productExpect.ChosenCount,
				productExpect.ConfirmedPrice,
			),
		)

	products, err := deps.repo.GetUserBasketProducts(t.Context(), userId, companyId)
	require.NoError(t, err)
	require.Equal(t, []domain.BasketProduct{productExpect}, products)
	assert.NoError(t, deps.assertExpectations())
}

func TestGetUserBasketProducts_Internal(t *testing.T) {
	deps := newTestDeps(t)

	userId := uuid.New()
	companyId := uuid.New()
	unexpectedErr := errors.New("unexpected error")

	deps.pool.ExpectQuery(
		"SELECT cp.id,cp.name,cp.price,cp.description,cp.is_available,cp.current_count,ubp.count,ubp.confirmed_price",
	).WithArgs(companyId, userId).
		WillReturnError(unexpectedErr)

	_, err := deps.repo.GetUserBasketProducts(t.Context(), userId, companyId)
	require.ErrorIs(t, err, unexpectedErr)
	assert.NoError(t, deps.assertExpectations())
}

func TestPutBasketProduct_Success(t *testing.T) {
	deps := newTestDeps(t)

	userId := uuid.New()
	companyId := uuid.New()
	productId := uuid.New()
	count := 2
	confirmedPrice := int64(100)

	deps.pool.ExpectExec(
		"WITH basket AS",
	).WithArgs(companyId, userId, productId, count, confirmedPrice).
		WillReturnResult(pgconn.NewCommandTag("INSERT 0 1"))

	err := deps.repo.PutBasketProduct(
		t.Context(),
		userId,
		companyId,
		productId,
		count,
		confirmedPrice,
	)
	require.NoError(t, err)
	assert.NoError(t, deps.assertExpectations())
}

func TestPutBasketProduct_Internal(t *testing.T) {
	deps := newTestDeps(t)

	userId := uuid.New()
	companyId := uuid.New()
	productId := uuid.New()
	count := 2
	confirmedPrice := int64(100)
	unexpectedErr := errors.New("unexpected error")

	deps.pool.ExpectExec(
		"WITH basket AS",
	).WithArgs(companyId, userId, productId, count, confirmedPrice).
		WillReturnError(unexpectedErr)

	err := deps.repo.PutBasketProduct(
		t.Context(),
		userId,
		companyId,
		productId,
		count,
		confirmedPrice,
	)
	require.ErrorIs(t, err, unexpectedErr)
	assert.NoError(t, deps.assertExpectations())
}

func TestClearBasket_Success(t *testing.T) {
	deps := newTestDeps(t)

	userId := uuid.New()
	companyId := uuid.New()

	deps.pool.ExpectExec(
		"DELETE FROM user_basket_products",
	).WithArgs(userId, companyId).
		WillReturnResult(pgconn.NewCommandTag("DELETE 1"))

	err := deps.repo.ClearBasket(t.Context(), userId, companyId)
	require.NoError(t, err)
	assert.NoError(t, deps.assertExpectations())
}

func TestClearBasket_Fail(t *testing.T) {
	deps := newTestDeps(t)

	userId := uuid.New()
	companyId := uuid.New()

	deps.pool.ExpectExec(
		"DELETE FROM user_basket_products",
	).WithArgs(userId, companyId).
		WillReturnResult(pgconn.NewCommandTag("DELETE 0"))

	err := deps.repo.ClearBasket(t.Context(), userId, companyId)
	require.ErrorIs(t, err, core_errors.ErrConflict)
	assert.NoError(t, deps.assertExpectations())
}

func TestClearBasket_Internal(t *testing.T) {
	deps := newTestDeps(t)

	userId := uuid.New()
	companyId := uuid.New()
	unexpectedErr := errors.New("unexpected error")

	deps.pool.ExpectExec(
		"DELETE FROM user_basket_products",
	).WithArgs(userId, companyId).
		WillReturnError(unexpectedErr)

	err := deps.repo.ClearBasket(t.Context(), userId, companyId)
	require.ErrorIs(t, err, unexpectedErr)
	assert.NoError(t, deps.assertExpectations())
}

func TestConfirmBasketProductsPrice_Success(t *testing.T) {
	deps := newTestDeps(t)

	userId := uuid.New()
	companyId := uuid.New()
	productIds := []uuid.UUID{uuid.New(), uuid.New()}

	deps.pool.ExpectExec(
		"UPDATE user_basket_products",
	).WithArgs(userId, companyId, productIds).
		WillReturnResult(pgconn.NewCommandTag("UPDATE 2"))

	err := deps.repo.ConfirmBasketProductsPrice(
		t.Context(),
		userId,
		companyId,
		productIds,
	)
	require.NoError(t, err)
	assert.NoError(t, deps.assertExpectations())
}

func TestConfirmBasketProductsPrice_Internal(t *testing.T) {
	deps := newTestDeps(t)

	userId := uuid.New()
	companyId := uuid.New()
	productIds := []uuid.UUID{uuid.New(), uuid.New()}
	unexpectedErr := errors.New("unexpected error")

	deps.pool.ExpectExec(
		"UPDATE user_basket_products",
	).WithArgs(userId, companyId, productIds).
		WillReturnError(unexpectedErr)

	err := deps.repo.ConfirmBasketProductsPrice(
		t.Context(),
		userId,
		companyId,
		productIds,
	)
	require.ErrorIs(t, err, unexpectedErr)
	assert.NoError(t, deps.assertExpectations())
}
