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
)

func TestGetBasket_CompanyNotFound(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")

	deps.menuRepo.
		EXPECT().
		GetCompany(ctx, companyId).
		Return(domain.Company{}, core_errors.ErrNotFound)

	basket, err := deps.service.GetBasket(ctx, userId, companyId)

	require.Error(t, err)
	assert.ErrorIs(t, err, core_errors.ErrNotFound)
	assert.ErrorContains(t, err, "failed to get company")
	assert.Empty(t, basket)
}

func TestGetBasket_UserNotFound(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")

	deps.menuRepo.
		EXPECT().
		GetCompany(ctx, companyId).
		Return(domain.Company{}, nil)
	deps.menuRepo.
		EXPECT().
		GetUser(ctx, userId).
		Return(core_errors.ErrNotFound)

	basket, err := deps.service.GetBasket(ctx, userId, companyId)

	require.Error(t, err)
	assert.ErrorIs(t, err, core_errors.ErrNotFound)
	assert.ErrorContains(t, err, "failed to get user")
	assert.Empty(t, basket)
}

func TestGetBasket_InternalError(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	internalErr := errors.New("internal error")

	deps.menuRepo.
		EXPECT().
		GetCompany(ctx, companyId).
		Return(domain.Company{}, nil)
	deps.menuRepo.
		EXPECT().
		GetUser(ctx, userId).
		Return(nil)
	deps.basketRepo.
		EXPECT().
		GetUserBasketProducts(ctx, userId, companyId).
		Return(nil, internalErr)

	basket, err := deps.service.GetBasket(ctx, userId, companyId)

	require.Error(t, err)
	assert.ErrorIs(t, err, internalErr)
	assert.ErrorContains(t, err, "failed to get user basket")
	assert.Empty(t, basket)
}

func TestGetBasket_Success(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	products := []domain.BasketProduct{
		{
			Product: domain.Product{
				Id:          uuid.MustParse("33333333-3333-4333-8333-000000000001"),
				CompanyId:   companyId,
				Name:        "available product",
				Price:       120,
				IsAvailable: true,
			},
			ConfirmedPrice: 100,
			ChosenCount:    2,
		},
		{
			Product: domain.Product{
				Id:          uuid.MustParse("33333333-3333-4333-8333-000000000002"),
				CompanyId:   companyId,
				Name:        "another available product",
				Price:       75,
				IsAvailable: true,
			},
			ConfirmedPrice: 60,
			ChosenCount:    3,
		},
		{
			Product: domain.Product{
				Id:          uuid.MustParse("33333333-3333-4333-8333-000000000003"),
				CompanyId:   companyId,
				Name:        "unavailable product",
				Price:       1000,
				IsAvailable: false,
			},
			ConfirmedPrice: 900,
			ChosenCount:    5,
		},
	}
	expectedBasket := domain.Basket{
		Products:           products,
		FullPrice:          465,
		FullConfirmedPrice: 380,
	}

	deps.menuRepo.
		EXPECT().
		GetCompany(ctx, companyId).
		Return(domain.Company{}, nil)
	deps.menuRepo.
		EXPECT().
		GetUser(ctx, userId).
		Return(nil)
	deps.basketRepo.
		EXPECT().
		GetUserBasketProducts(ctx, userId, companyId).
		Return(products, nil)

	basket, err := deps.service.GetBasket(ctx, userId, companyId)

	require.NoError(t, err)
	assert.Equal(t, expectedBasket, basket)
}
