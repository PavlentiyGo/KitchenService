package service_companies_test

import (
	"avitoKitchen/internal/core/domain"
	core_errors "avitoKitchen/internal/core/errors"
	service_companies "avitoKitchen/internal/service/companies"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type testDeps struct {
	service       *service_companies.Service
	companiesRepo *MockcompaniesRepository
}

func newTestDeps(t *testing.T) *testDeps {
	t.Helper()

	ctrl := gomock.NewController(t)
	txManager := NewMocktxManager(ctrl)
	companiesRepo := NewMockcompaniesRepository(ctrl)
	ordersRepo := NewMockordersRepository(ctrl)

	return &testDeps{
		service:       service_companies.NewService(txManager, companiesRepo, ordersRepo),
		companiesRepo: companiesRepo,
	}
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
	createdProduct := domain.Product{
		Id:               productId,
		CompanyId:        companyId,
		PartnerProductId: &partnerProductId,
		Name:             "Пицца Маргарита",
		Price:            49900,
		Description:      "Томатный соус, моцарелла, томаты и базилик",
		CurrentCount:     18,
		IsAvailable:      true,
	}

	deps.companiesRepo.
		EXPECT().
		CreateProduct(gomock.Any(), companyId, productForCreate).
		Return(createdProduct, nil)

	product, err := deps.service.AddNewProduct(
		context.Background(),
		companyId,
		productForCreate,
	)

	require.NoError(t, err)
	assert.Equal(t, createdProduct, product)
}

func TestAddNewProduct_InvalidFields(t *testing.T) {
	partnerProductId := "pizza-margherita"
	emptyPartnerProductId := "   "
	productWithoutPartnerId := domain.Product{
		PartnerProductId: nil,
		Name:             "Пицца Маргарита",
		Price:            49900,
		Description:      "Томатный соус и моцарелла",
		CurrentCount:     18,
		IsAvailable:      true,
	}
	productWithEmptyPartnerId := domain.Product{
		PartnerProductId: &emptyPartnerProductId,
		Name:             "Пицца Маргарита",
		Price:            49900,
		Description:      "Томатный соус и моцарелла",
		CurrentCount:     18,
		IsAvailable:      true,
	}
	productWithEmptyName := domain.Product{
		PartnerProductId: &partnerProductId,
		Name:             "   ",
		Price:            49900,
		Description:      "Томатный соус и моцарелла",
		CurrentCount:     18,
		IsAvailable:      true,
	}
	productWithInvalidPrice := domain.Product{
		PartnerProductId: &partnerProductId,
		Name:             "Пицца Маргарита",
		Price:            0,
		Description:      "Томатный соус и моцарелла",
		CurrentCount:     18,
		IsAvailable:      true,
	}
	productWithInvalidCount := domain.Product{
		PartnerProductId: &partnerProductId,
		Name:             "Пицца Маргарита",
		Price:            49900,
		Description:      "Томатный соус и моцарелла",
		CurrentCount:     -1,
		IsAvailable:      true,
	}

	testCases := []struct {
		name                 string
		product              domain.Product
		expectedErrorMessage string
	}{
		{
			name:                 "missing partner product id",
			product:              productWithoutPartnerId,
			expectedErrorMessage: "partner product id is empty",
		},
		{
			name:                 "empty partner product id",
			product:              productWithEmptyPartnerId,
			expectedErrorMessage: "partner product id is empty",
		},
		{
			name:                 "empty name",
			product:              productWithEmptyName,
			expectedErrorMessage: "product name is empty",
		},
		{
			name:                 "invalid price",
			product:              productWithInvalidPrice,
			expectedErrorMessage: "product price must be greater than zero",
		},
		{
			name:                 "invalid current count",
			product:              productWithInvalidCount,
			expectedErrorMessage: "product count cannot be negative",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			deps := newTestDeps(t)

			product, err := deps.service.AddNewProduct(
				context.Background(),
				uuid.New(),
				testCase.product,
			)

			require.Error(t, err)
			assert.ErrorIs(t, err, core_errors.ErrInvalidRequest)
			assert.ErrorContains(t, err, testCase.expectedErrorMessage)
			assert.Empty(t, product)
		})
	}
}

func TestAddNewProduct_RepositoryError(t *testing.T) {
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
	repositoryErr := errors.New("create product error")

	deps.companiesRepo.
		EXPECT().
		CreateProduct(gomock.Any(), companyId, productForCreate).
		Return(domain.Product{}, repositoryErr)

	product, err := deps.service.AddNewProduct(
		context.Background(),
		companyId,
		productForCreate,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, repositoryErr)
	assert.ErrorContains(t, err, "failed to create product")
	assert.Empty(t, product)
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

	deps.companiesRepo.
		EXPECT().
		PatchProduct(gomock.Any(), companyId, partnerProductId, patch).
		Return(updatedProduct, nil)

	product, err := deps.service.PatchCompanyProduct(
		context.Background(),
		companyId,
		partnerProductId,
		patch,
	)

	require.NoError(t, err)
	assert.Equal(t, updatedProduct, product)
}

func TestPatchCompanyProduct_InvalidFields(t *testing.T) {
	emptyName := "   "
	invalidPrice := int64(0)
	invalidCount := -1

	testCases := []struct {
		name                 string
		partnerProductId     string
		patch                domain.ProductPatch
		expectedErrorMessage string
	}{
		{
			name:                 "empty partner product id",
			partnerProductId:     "   ",
			patch:                domain.ProductPatch{},
			expectedErrorMessage: "partner product id is empty",
		},
		{
			name:             "empty name",
			partnerProductId: "pizza-margherita",
			patch: domain.ProductPatch{
				Name: &emptyName,
			},
			expectedErrorMessage: "product name is empty",
		},
		{
			name:             "invalid price",
			partnerProductId: "pizza-margherita",
			patch: domain.ProductPatch{
				Price: &invalidPrice,
			},
			expectedErrorMessage: "product price must be greater than zero",
		},
		{
			name:             "invalid current count",
			partnerProductId: "pizza-margherita",
			patch: domain.ProductPatch{
				CurrentCount: &invalidCount,
			},
			expectedErrorMessage: "product count cannot be negative",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			deps := newTestDeps(t)

			product, err := deps.service.PatchCompanyProduct(
				context.Background(),
				uuid.New(),
				testCase.partnerProductId,
				testCase.patch,
			)

			require.Error(t, err)
			assert.ErrorIs(t, err, core_errors.ErrInvalidRequest)
			assert.ErrorContains(t, err, testCase.expectedErrorMessage)
			assert.Empty(t, product)
		})
	}
}

func TestPatchCompanyProduct_RepositoryError(t *testing.T) {
	deps := newTestDeps(t)
	companyId := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	partnerProductId := "pizza-margherita"
	price := int64(59900)
	patch := domain.ProductPatch{Price: &price}
	repositoryErr := errors.New("patch product error")

	deps.companiesRepo.
		EXPECT().
		PatchProduct(gomock.Any(), companyId, partnerProductId, patch).
		Return(domain.Product{}, repositoryErr)

	product, err := deps.service.PatchCompanyProduct(
		context.Background(),
		companyId,
		partnerProductId,
		patch,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, repositoryErr)
	assert.ErrorContains(t, err, "failed to patch product")
	assert.Empty(t, product)
}
