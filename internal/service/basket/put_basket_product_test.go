package service_basket_test

import (
	"avitoKitchen/internal/core/domain"
	core_errors "avitoKitchen/internal/core/errors"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestPutBasketProduct_ZeroCount(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-333333333333")

	err := deps.service.PutBasketProduct(ctx, userId, companyId, productId, 0)

	require.Error(t, err)
	assert.ErrorIs(t, err, core_errors.ErrInvalidRequest)
	assert.ErrorContains(t, err, "count must be greater than zero")
}

func TestPutBasketProduct_NegativeCount(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-333333333333")

	err := deps.service.PutBasketProduct(ctx, userId, companyId, productId, -1)

	require.Error(t, err)
	assert.ErrorIs(t, err, core_errors.ErrInvalidRequest)
	assert.ErrorContains(t, err, "count must be greater than zero")
}

func TestPutBasketProduct_TransactionError(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	count := 3
	transactionErr := errors.New("transaction error")

	deps.txManager.
		EXPECT().
		WithinTx(ctx, gomock.Any()).
		Return(transactionErr)

	err := deps.service.PutBasketProduct(ctx, userId, companyId, productId, count)

	require.Error(t, err)
	assert.ErrorIs(t, err, transactionErr)
}

func TestPutBasketProduct_UserLookupError(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	txCtx := context.WithValue(ctx, transactionContextKey{}, true)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	count := 3
	userErr := errors.New("get user error")

	deps.txManager.
		EXPECT().
		WithinTx(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
			return fn(txCtx)
		})
	deps.menuRepo.
		EXPECT().
		GetUser(txCtx, userId).
		Return(userErr)

	err := deps.service.PutBasketProduct(ctx, userId, companyId, productId, count)

	require.Error(t, err)
	assert.ErrorIs(t, err, userErr)
	assert.ErrorContains(t, err, "failed to get user")
}

func TestPutBasketProduct_ProductLookupError(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	txCtx := context.WithValue(ctx, transactionContextKey{}, true)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	count := 3
	productErr := errors.New("get product error")

	deps.txManager.
		EXPECT().
		WithinTx(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
			return fn(txCtx)
		})
	deps.menuRepo.
		EXPECT().
		GetUser(txCtx, userId).
		Return(nil)
	deps.menuRepo.
		EXPECT().
		GetCompanyProduct(txCtx, productId).
		Return(domain.Product{}, productErr)

	err := deps.service.PutBasketProduct(ctx, userId, companyId, productId, count)

	require.Error(t, err)
	assert.ErrorIs(t, err, productErr)
	assert.ErrorContains(t, err, "failed to get product")
}

func TestPutBasketProduct_ProductBelongsToAnotherCompany(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	txCtx := context.WithValue(ctx, transactionContextKey{}, true)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	otherCompanyId := uuid.MustParse("11111111-1111-4111-8111-000000000002")
	productId := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	count := 3
	product := domain.Product{
		Id:           productId,
		CompanyId:    otherCompanyId,
		Name:         "Пицца Маргарита",
		Price:        49900,
		CurrentCount: 5,
		IsAvailable:  true,
	}

	deps.txManager.
		EXPECT().
		WithinTx(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
			return fn(txCtx)
		})
	deps.menuRepo.
		EXPECT().
		GetUser(txCtx, userId).
		Return(nil)
	deps.menuRepo.
		EXPECT().
		GetCompanyProduct(txCtx, productId).
		Return(product, nil)

	err := deps.service.PutBasketProduct(ctx, userId, companyId, productId, count)

	require.Error(t, err)
	assert.ErrorIs(t, err, core_errors.ErrNotFound)
	assert.ErrorContains(t, err, "does not belong to company")
}

func TestPutBasketProduct_ProductUnavailable(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	txCtx := context.WithValue(ctx, transactionContextKey{}, true)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	count := 3
	product := domain.Product{
		Id:           productId,
		CompanyId:    companyId,
		Name:         "Пицца Маргарита",
		Price:        49900,
		CurrentCount: 5,
		IsAvailable:  false,
	}

	deps.txManager.
		EXPECT().
		WithinTx(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
			return fn(txCtx)
		})
	deps.menuRepo.
		EXPECT().
		GetUser(txCtx, userId).
		Return(nil)
	deps.menuRepo.
		EXPECT().
		GetCompanyProduct(txCtx, productId).
		Return(product, nil)

	err := deps.service.PutBasketProduct(ctx, userId, companyId, productId, count)

	require.Error(t, err)
	assert.ErrorIs(t, err, core_errors.ErrProductUnavailable)
	assert.ErrorContains(t, err, productId.String())
}

func TestPutBasketProduct_InsufficientStock(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	txCtx := context.WithValue(ctx, transactionContextKey{}, true)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	count := 3
	product := domain.Product{
		Id:           productId,
		CompanyId:    companyId,
		Name:         "Пицца Маргарита",
		Price:        49900,
		CurrentCount: 2,
		IsAvailable:  true,
	}

	deps.txManager.
		EXPECT().
		WithinTx(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
			return fn(txCtx)
		})
	deps.menuRepo.
		EXPECT().
		GetUser(txCtx, userId).
		Return(nil)
	deps.menuRepo.
		EXPECT().
		GetCompanyProduct(txCtx, productId).
		Return(product, nil)

	err := deps.service.PutBasketProduct(ctx, userId, companyId, productId, count)

	require.Error(t, err)
	assert.ErrorIs(t, err, core_errors.ErrInsufficientStock)
	assert.ErrorContains(t, err, "requested 3 items, only 2 available")
}

func TestPutBasketProduct_RepositoryError(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	txCtx := context.WithValue(ctx, transactionContextKey{}, true)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	count := 3
	product := domain.Product{
		Id:           productId,
		CompanyId:    companyId,
		Name:         "Пицца Маргарита",
		Price:        49900,
		CurrentCount: 5,
		IsAvailable:  true,
	}
	repositoryErr := errors.New("put basket product error")

	deps.txManager.
		EXPECT().
		WithinTx(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
			return fn(txCtx)
		})
	deps.menuRepo.
		EXPECT().
		GetUser(txCtx, userId).
		Return(nil)
	deps.menuRepo.
		EXPECT().
		GetCompanyProduct(txCtx, productId).
		Return(product, nil)
	deps.basketRepo.
		EXPECT().
		PutBasketProduct(txCtx, userId, companyId, productId, count, product.Price).
		Return(repositoryErr)

	err := deps.service.PutBasketProduct(ctx, userId, companyId, productId, count)

	require.Error(t, err)
	assert.ErrorIs(t, err, repositoryErr)
	assert.ErrorContains(t, err, "failed to put product into basket")
}

func TestPutBasketProduct_Success(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	txCtx := context.WithValue(ctx, transactionContextKey{}, true)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	count := 3
	product := domain.Product{
		Id:           productId,
		CompanyId:    companyId,
		Name:         "Пицца Маргарита",
		Price:        49900,
		CurrentCount: 5,
		IsAvailable:  true,
	}

	deps.txManager.
		EXPECT().
		WithinTx(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
			return fn(txCtx)
		})
	deps.menuRepo.
		EXPECT().
		GetUser(txCtx, userId).
		Return(nil)
	deps.menuRepo.
		EXPECT().
		GetCompanyProduct(txCtx, productId).
		Return(product, nil)
	deps.basketRepo.
		EXPECT().
		PutBasketProduct(txCtx, userId, companyId, productId, count, product.Price).
		Return(nil)

	err := deps.service.PutBasketProduct(ctx, userId, companyId, productId, count)

	require.NoError(t, err)
}
