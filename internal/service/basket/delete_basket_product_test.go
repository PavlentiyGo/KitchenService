package service_basket_test

import (
	"context"
	"errors"
	"foodDelivery/internal/core/domain"
	core_errors "foodDelivery/internal/core/errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteBasketProduct_UserLookupError(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	userErr := errors.New("get user error")

	deps.menuRepo.
		EXPECT().
		GetUser(ctx, userId).
		Return(userErr)

	err := deps.service.DeleteBasketProduct(ctx, userId, companyId, productId)

	require.Error(t, err)
	assert.ErrorIs(t, err, userErr)
	assert.ErrorContains(t, err, "failed to get user")
}

func TestDeleteBasketProduct_CompanyLookupError(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	companyErr := errors.New("get company error")

	deps.menuRepo.
		EXPECT().
		GetUser(ctx, userId).
		Return(nil)
	deps.menuRepo.
		EXPECT().
		GetCompany(ctx, companyId).
		Return(domain.Company{}, companyErr)

	err := deps.service.DeleteBasketProduct(ctx, userId, companyId, productId)

	require.Error(t, err)
	assert.ErrorIs(t, err, companyErr)
	assert.ErrorContains(t, err, "failed to get company")
}

func TestDeleteBasketProduct_ProductLookupError(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	productErr := errors.New("get product error")

	deps.menuRepo.
		EXPECT().
		GetUser(ctx, userId).
		Return(nil)
	deps.menuRepo.
		EXPECT().
		GetCompany(ctx, companyId).
		Return(domain.Company{}, nil)
	deps.menuRepo.
		EXPECT().
		GetCompanyProduct(ctx, productId).
		Return(domain.Product{}, productErr)

	err := deps.service.DeleteBasketProduct(ctx, userId, companyId, productId)

	require.Error(t, err)
	assert.ErrorIs(t, err, productErr)
	assert.ErrorContains(t, err, "failed to get product")
}

func TestDeleteBasketProduct_ProductBelongsToAnotherCompany(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	otherCompanyId := uuid.MustParse("11111111-1111-4111-8111-000000000002")
	productId := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	product := domain.Product{
		Id:           productId,
		CompanyId:    otherCompanyId,
		Name:         "Пицца Маргарита",
		Price:        49900,
		CurrentCount: 5,
		IsAvailable:  true,
	}

	deps.menuRepo.
		EXPECT().
		GetUser(ctx, userId).
		Return(nil)
	deps.menuRepo.
		EXPECT().
		GetCompany(ctx, companyId).
		Return(domain.Company{}, nil)
	deps.menuRepo.
		EXPECT().
		GetCompanyProduct(ctx, productId).
		Return(product, nil)

	err := deps.service.DeleteBasketProduct(ctx, userId, companyId, productId)

	require.Error(t, err)
	assert.ErrorIs(t, err, core_errors.ErrNotFound)
	assert.ErrorContains(t, err, "does not belong to company")
}

func TestDeleteBasketProduct_RepositoryError(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	product := domain.Product{
		Id:           productId,
		CompanyId:    companyId,
		Name:         "Пицца Маргарита",
		Price:        49900,
		CurrentCount: 5,
		IsAvailable:  true,
	}
	repositoryErr := errors.New("delete basket product error")

	deps.menuRepo.
		EXPECT().
		GetUser(ctx, userId).
		Return(nil)
	deps.menuRepo.
		EXPECT().
		GetCompany(ctx, companyId).
		Return(domain.Company{}, nil)
	deps.menuRepo.
		EXPECT().
		GetCompanyProduct(ctx, productId).
		Return(product, nil)
	deps.basketRepo.
		EXPECT().
		DeleteBasketProduct(ctx, userId, companyId, productId).
		Return(repositoryErr)

	err := deps.service.DeleteBasketProduct(ctx, userId, companyId, productId)

	require.Error(t, err)
	assert.ErrorIs(t, err, repositoryErr)
	assert.ErrorContains(t, err, "failed to delete product from basket")
}

func TestDeleteBasketProduct_Success(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	userId := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	productId := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	product := domain.Product{
		Id:           productId,
		CompanyId:    companyId,
		Name:         "Пицца Маргарита",
		Price:        49900,
		CurrentCount: 5,
		IsAvailable:  true,
	}

	deps.menuRepo.
		EXPECT().
		GetUser(ctx, userId).
		Return(nil)
	deps.menuRepo.
		EXPECT().
		GetCompany(ctx, companyId).
		Return(domain.Company{}, nil)
	deps.menuRepo.
		EXPECT().
		GetCompanyProduct(ctx, productId).
		Return(product, nil)
	deps.basketRepo.
		EXPECT().
		DeleteBasketProduct(ctx, userId, companyId, productId).
		Return(nil)

	err := deps.service.DeleteBasketProduct(ctx, userId, companyId, productId)

	require.NoError(t, err)
}
