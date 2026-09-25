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

func TestConfirmBasketProducts_EmptyProducts(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")

	basket, err := deps.service.ConfirmBasketProducts(ctx, nil, companyId, userId)

	require.Error(t, err)
	assert.ErrorIs(t, err, core_errors.ErrInvalidRequest)
	assert.ErrorContains(t, err, "products for confirmation are empty")
	assert.Nil(t, basket)
}

func TestConfirmBasketProducts_UserLookupError(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	productsForChange := []domain.Product{
		{
			Id:    productId,
			Name:  "Пицца Маргарита",
			Price: 49900,
		},
	}
	userErr := errors.New("get user error")

	deps.menuRepo.
		EXPECT().
		GetUser(ctx, userId).
		Return(userErr)

	basket, err := deps.service.ConfirmBasketProducts(
		ctx,
		productsForChange,
		companyId,
		userId,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, userErr)
	assert.ErrorContains(t, err, "failed to get user")
	assert.Nil(t, basket)
}

func TestConfirmBasketProducts_CompanyLookupError(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	productsForChange := []domain.Product{
		{
			Id:    productId,
			Name:  "Пицца Маргарита",
			Price: 49900,
		},
	}
	companyErr := errors.New("get company error")

	deps.menuRepo.
		EXPECT().
		GetUser(ctx, userId).
		Return(nil)
	deps.menuRepo.
		EXPECT().
		GetCompany(ctx, companyId).
		Return(domain.Company{}, companyErr)

	basket, err := deps.service.ConfirmBasketProducts(
		ctx,
		productsForChange,
		companyId,
		userId,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, companyErr)
	assert.ErrorContains(t, err, "failed to get company")
	assert.Nil(t, basket)
}

func TestConfirmBasketProducts_DuplicateProduct(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	productsForChange := []domain.Product{
		{
			Id:    productId,
			Name:  "Пицца Маргарита",
			Price: 49900,
		},
		{
			Id:    productId,
			Name:  "Пицца Маргарита",
			Price: 49900,
		},
	}

	deps.menuRepo.
		EXPECT().
		GetUser(ctx, userId).
		Return(nil)
	deps.menuRepo.
		EXPECT().
		GetCompany(ctx, companyId).
		Return(domain.Company{}, nil)

	basket, err := deps.service.ConfirmBasketProducts(
		ctx,
		productsForChange,
		companyId,
		userId,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, core_errors.ErrInvalidRequest)
	assert.ErrorContains(t, err, "is duplicated")
	assert.Nil(t, basket)
}

func TestConfirmBasketProducts_TransactionError(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	productsForChange := []domain.Product{
		{
			Id:    productId,
			Name:  "Пицца Маргарита",
			Price: 49900,
		},
	}
	transactionErr := errors.New("transaction error")

	deps.menuRepo.
		EXPECT().
		GetUser(ctx, userId).
		Return(nil)
	deps.menuRepo.
		EXPECT().
		GetCompany(ctx, companyId).
		Return(domain.Company{}, nil)
	deps.txManager.
		EXPECT().
		WithinTx(ctx, gomock.Any()).
		Return(transactionErr)

	basket, err := deps.service.ConfirmBasketProducts(
		ctx,
		productsForChange,
		companyId,
		userId,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, transactionErr)
	assert.Nil(t, basket)
}

func TestConfirmBasketProducts_ConfirmPricesError(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	txCtx := context.WithValue(ctx, transactionContextKey{}, true)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	productsForChange := []domain.Product{
		{
			Id:    productId,
			Name:  "Пицца Маргарита",
			Price: 49900,
		},
	}
	productIds := []uuid.UUID{productId}
	confirmPriceErr := errors.New("confirm basket products price error")

	deps.menuRepo.
		EXPECT().
		GetUser(ctx, userId).
		Return(nil)
	deps.menuRepo.
		EXPECT().
		GetCompany(ctx, companyId).
		Return(domain.Company{}, nil)
	deps.txManager.
		EXPECT().
		WithinTx(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
			return fn(txCtx)
		})
	deps.basketRepo.
		EXPECT().
		ConfirmBasketProductsPrice(txCtx, userId, companyId, productIds).
		Return(confirmPriceErr)

	basket, err := deps.service.ConfirmBasketProducts(
		ctx,
		productsForChange,
		companyId,
		userId,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, confirmPriceErr)
	assert.ErrorContains(t, err, "failed to update products prices")
	assert.Nil(t, basket)
}

func TestConfirmBasketProducts_GetBasketError(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	txCtx := context.WithValue(ctx, transactionContextKey{}, true)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	productsForChange := []domain.Product{
		{
			Id:    productId,
			Name:  "Пицца Маргарита",
			Price: 49900,
		},
	}
	productIds := []uuid.UUID{productId}
	getBasketErr := errors.New("get user basket error")

	deps.menuRepo.
		EXPECT().
		GetUser(ctx, userId).
		Return(nil)
	deps.menuRepo.
		EXPECT().
		GetCompany(ctx, companyId).
		Return(domain.Company{}, nil)
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
		Return(nil, getBasketErr)

	basket, err := deps.service.ConfirmBasketProducts(
		ctx,
		productsForChange,
		companyId,
		userId,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, getBasketErr)
	assert.ErrorContains(t, err, "failed to get user basket")
	assert.Nil(t, basket)
}

func TestConfirmBasketProducts_ProductMissingFromBasket(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	txCtx := context.WithValue(ctx, transactionContextKey{}, true)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	productsForChange := []domain.Product{
		{
			Id:    productId,
			Name:  "Пицца Маргарита",
			Price: 49900,
		},
	}
	productIds := []uuid.UUID{productId}
	userBasket := []domain.BasketProduct{}

	deps.menuRepo.
		EXPECT().
		GetUser(ctx, userId).
		Return(nil)
	deps.menuRepo.
		EXPECT().
		GetCompany(ctx, companyId).
		Return(domain.Company{}, nil)
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
		Return(userBasket, nil)

	basket, err := deps.service.ConfirmBasketProducts(
		ctx,
		productsForChange,
		companyId,
		userId,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, core_errors.ErrNotFound)
	assert.ErrorContains(t, err, "is not in user basket")
	assert.Nil(t, basket)
}

func TestConfirmBasketProducts_ProductPriceIncreased(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	txCtx := context.WithValue(ctx, transactionContextKey{}, true)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	productsForChange := []domain.Product{
		{
			Id:    productId,
			Name:  "Пицца Маргарита",
			Price: 49900,
		},
	}
	productIds := []uuid.UUID{productId}
	userBasket := []domain.BasketProduct{
		{
			Product: domain.Product{
				Id:        productId,
				CompanyId: companyId,
				Name:      "Пицца Маргарита",
				Price:     50000,
			},
			ConfirmedPrice: 50000,
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
		Return(domain.Company{}, nil)
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
		Return(userBasket, nil)

	basket, err := deps.service.ConfirmBasketProducts(
		ctx,
		productsForChange,
		companyId,
		userId,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, core_errors.ErrConflict)
	assert.ErrorContains(t, err, "price changed from 49900 to 50000")
	assert.Nil(t, basket)
}

func TestConfirmBasketProducts_Success(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	txCtx := context.WithValue(ctx, transactionContextKey{}, true)
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	firstProductId := uuid.MustParse("33333333-3333-4333-8333-000000000001")
	secondProductId := uuid.MustParse("33333333-3333-4333-8333-000000000002")
	productsForChange := []domain.Product{
		{
			Id:    firstProductId,
			Name:  "Пицца Маргарита",
			Price: 49900,
		},
		{
			Id:    secondProductId,
			Name:  "Пицца Пепперони",
			Price: 59900,
		},
	}
	productIds := []uuid.UUID{firstProductId, secondProductId}
	userBasket := []domain.BasketProduct{
		{
			Product: domain.Product{
				Id:          firstProductId,
				CompanyId:   companyId,
				Name:        "Пицца Маргарита",
				Price:       49900,
				IsAvailable: true,
			},
			ConfirmedPrice: 49900,
			ChosenCount:    1,
		},
		{
			Product: domain.Product{
				Id:          secondProductId,
				CompanyId:   companyId,
				Name:        "Пицца Пепперони",
				Price:       54900,
				IsAvailable: true,
			},
			ConfirmedPrice: 54900,
			ChosenCount:    2,
		},
	}

	deps.menuRepo.
		EXPECT().
		GetUser(ctx, userId).
		Return(nil)
	deps.menuRepo.
		EXPECT().
		GetCompany(ctx, companyId).
		Return(domain.Company{}, nil)
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
		Return(userBasket, nil)

	basket, err := deps.service.ConfirmBasketProducts(
		ctx,
		productsForChange,
		companyId,
		userId,
	)

	require.NoError(t, err)
	assert.Equal(t, userBasket, basket)
}
