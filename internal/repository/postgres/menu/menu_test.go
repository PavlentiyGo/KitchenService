package repository_menu_test

import (
	"avitoKitchen/internal/core/domain"
	core_errors "avitoKitchen/internal/core/errors"
	"avitoKitchen/internal/repository"
	repository_menu "avitoKitchen/internal/repository/postgres/menu"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
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
	repo *repository_menu.Repository
}

func newTestDeps(t *testing.T) *testDeps {
	t.Helper()
	pool, err := pgxmock.NewPool()
	require.NoError(t, err)
	txManager := &txManagerMock{pool: pool}
	return &testDeps{
		pool: pool,
		repo: repository_menu.NewRepository(txManager),
	}
}

func TestGetUser_Success(t *testing.T) {
	deps := newTestDeps(t)

	userId := uuid.Nil

	deps.pool.ExpectQuery(
		"SELECT id FROM users",
	).WithArgs(userId).
		WillReturnRows(
			pgxmock.NewRows(
				[]string{
					"id",
				}).AddRow(
				userId,
			),
		)
	err := deps.repo.GetUser(t.Context(), userId)
	require.NoError(t, err)
	assert.NoError(t, deps.assertExpectations())
}
func TestGetUser_Fail(t *testing.T) {
	deps := newTestDeps(t)

	userId := uuid.Nil

	deps.pool.ExpectQuery(
		"SELECT id FROM users",
	).WithArgs(userId).
		WillReturnRows(
			pgxmock.NewRows(
				[]string{
					"id",
				}),
		)
	err := deps.repo.GetUser(t.Context(), userId)
	require.ErrorIs(t, err, core_errors.ErrNotFound)
	assert.NoError(t, deps.assertExpectations())
}
func TestGetUser_Internal(t *testing.T) {
	deps := newTestDeps(t)

	userId := uuid.Nil

	unexpectedErr := errors.New("internal error")

	deps.pool.ExpectQuery(
		"SELECT id FROM users",
	).WithArgs(userId).
		WillReturnError(unexpectedErr)
	err := deps.repo.GetUser(t.Context(), userId)
	require.ErrorIs(t, err, unexpectedErr)
	assert.NoError(t, deps.assertExpectations())
}
func TestGetCompany_Success(t *testing.T) {
	deps := newTestDeps(t)

	companyExpect := domain.Company{
		Id:       uuid.Nil,
		Name:     "company_test",
		IsActive: true,
		Url:      "https://company.com",
	}

	deps.pool.ExpectQuery(
		"SELECT id,name,is_active,company_source_url FROM companies",
	).WithArgs(companyExpect.Id).
		WillReturnRows(
			pgxmock.NewRows(
				[]string{
					"id", "name", "is_active", "company_source_url",
				}).AddRow(
				companyExpect.Id, companyExpect.Name, companyExpect.IsActive, companyExpect.Url,
			),
		)
	company, err := deps.repo.GetCompany(t.Context(), companyExpect.Id)
	require.NoError(t, err)
	require.Equal(t, companyExpect, company)
	assert.NoError(t, deps.assertExpectations())
}
func TestGetCompany_Fail(t *testing.T) {
	deps := newTestDeps(t)

	companyExpect := domain.Company{
		Id:       uuid.Nil,
		Name:     "company_test",
		IsActive: true,
		Url:      "https://company.com",
	}

	deps.pool.ExpectQuery(
		"SELECT id,name,is_active,company_source_url FROM companies",
	).WithArgs(companyExpect.Id).
		WillReturnRows(
			pgxmock.NewRows(
				[]string{
					"id", "name", "is_active", "company_source_url",
				}),
		)
	_, err := deps.repo.GetCompany(t.Context(), companyExpect.Id)
	require.ErrorIs(t, err, core_errors.ErrNotFound)
	assert.NoError(t, deps.assertExpectations())
}

func TestGetCompany_Internal(t *testing.T) {
	deps := newTestDeps(t)

	companyExpect := domain.Company{
		Id:       uuid.Nil,
		Name:     "company_test",
		IsActive: true,
		Url:      "https://company.com",
	}

	unexpectedErr := errors.New("internal error")

	deps.pool.ExpectQuery(
		"SELECT id,name,is_active,company_source_url FROM companies",
	).WithArgs(companyExpect.Id).WillReturnError(unexpectedErr)

	_, err := deps.repo.GetCompany(t.Context(), companyExpect.Id)
	require.ErrorIs(t, err, unexpectedErr)
	assert.NoError(t, deps.assertExpectations())
}

func TestGetCompanies_Success(t *testing.T) {
	deps := newTestDeps(t)

	companiesExpect := []domain.Company{
		{
			Id:       uuid.New(),
			Name:     "company_test",
			IsActive: true,
			Url:      "https://company.com",
		},
		{
			Id:       uuid.New(),
			Name:     "company_test2",
			IsActive: true,
			Url:      "https://company.com",
		},
	}

	deps.pool.ExpectQuery(
		"SELECT id,name,is_active,company_source_url FROM companies",
	).WillReturnRows(
		pgxmock.NewRows(
			[]string{
				"id", "name", "is_active", "company_source_url",
			}).AddRow(
			companiesExpect[0].Id, companiesExpect[0].Name, companiesExpect[0].IsActive, companiesExpect[0].Url,
		).AddRow(companiesExpect[1].Id, companiesExpect[1].Name, companiesExpect[1].IsActive, companiesExpect[1].Url),
	)
	companies, err := deps.repo.GetCompanies(t.Context())
	require.NoError(t, err)
	require.Len(t, companiesExpect, len(companies))
	assert.NoError(t, deps.assertExpectations())
}

func TestGetCompanies_Internal(t *testing.T) {
	deps := newTestDeps(t)

	unexpectedErr := errors.New("internal error")
	deps.pool.ExpectQuery(
		"SELECT id,name,is_active,company_source_url FROM companies",
	).WillReturnError(unexpectedErr)
	_, err := deps.repo.GetCompanies(t.Context())
	require.ErrorIs(t, err, unexpectedErr)
	assert.NoError(t, deps.assertExpectations())
}

func TestGetCompanyProducts_Success(t *testing.T) {
	deps := newTestDeps(t)

	companyId := uuid.New()
	firstPartnerProductId := "first_partner_product_id"
	secondPartnerProductId := "second_partner_product_id"
	productsExpect := []domain.Product{
		{
			Id:               uuid.New(),
			CompanyId:        companyId,
			PartnerProductId: &firstPartnerProductId,
			Name:             "first_product",
			Price:            100,
			Description:      "first description",
			CurrentCount:     10,
			IsAvailable:      true,
		},
		{
			Id:               uuid.New(),
			CompanyId:        companyId,
			PartnerProductId: &secondPartnerProductId,
			Name:             "second_product",
			Price:            200,
			Description:      "second description",
			CurrentCount:     20,
			IsAvailable:      false,
		},
	}

	deps.pool.ExpectQuery(
		"SELECT id,company_id,partner_product_id,name,price,description,current_count,is_available FROM company_products",
	).WithArgs(companyId).
		WillReturnRows(
			pgxmock.NewRows(
				[]string{
					"id", "company_id", "partner_product_id", "name", "price", "description", "current_count", "is_available",
				}).AddRow(
				productsExpect[0].Id,
				productsExpect[0].CompanyId,
				productsExpect[0].PartnerProductId,
				productsExpect[0].Name,
				productsExpect[0].Price,
				productsExpect[0].Description,
				productsExpect[0].CurrentCount,
				productsExpect[0].IsAvailable,
			).AddRow(
				productsExpect[1].Id,
				productsExpect[1].CompanyId,
				productsExpect[1].PartnerProductId,
				productsExpect[1].Name,
				productsExpect[1].Price,
				productsExpect[1].Description,
				productsExpect[1].CurrentCount,
				productsExpect[1].IsAvailable,
			),
		)

	products, err := deps.repo.GetCompanyProducts(t.Context(), companyId)
	require.NoError(t, err)
	require.Equal(t, productsExpect, products)
	assert.NoError(t, deps.assertExpectations())
}

func TestGetCompanyProducts_Internal(t *testing.T) {
	deps := newTestDeps(t)

	companyId := uuid.New()
	unexpectedErr := errors.New("internal error")

	deps.pool.ExpectQuery(
		"SELECT id,company_id,partner_product_id,name,price,description,current_count,is_available FROM company_products",
	).WithArgs(companyId).
		WillReturnError(unexpectedErr)

	_, err := deps.repo.GetCompanyProducts(t.Context(), companyId)
	require.ErrorIs(t, err, unexpectedErr)
	assert.NoError(t, deps.assertExpectations())
}

func TestGetCompanyProduct_Success(t *testing.T) {
	deps := newTestDeps(t)

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

	deps.pool.ExpectQuery(
		"SELECT id,company_id,partner_product_id,name,price,description,current_count,is_available FROM company_products",
	).WithArgs(productExpect.Id).
		WillReturnRows(
			pgxmock.NewRows(
				[]string{
					"id", "company_id", "partner_product_id", "name", "price", "description", "current_count", "is_available",
				}).AddRow(
				productExpect.Id,
				productExpect.CompanyId,
				productExpect.PartnerProductId,
				productExpect.Name,
				productExpect.Price,
				productExpect.Description,
				productExpect.CurrentCount,
				productExpect.IsAvailable,
			),
		)

	product, err := deps.repo.GetCompanyProduct(t.Context(), productExpect.Id)
	require.NoError(t, err)
	require.Equal(t, productExpect, product)
	assert.NoError(t, deps.assertExpectations())
}

func TestGetCompanyProduct_Fail(t *testing.T) {
	deps := newTestDeps(t)

	productId := uuid.New()

	deps.pool.ExpectQuery(
		"SELECT id,company_id,partner_product_id,name,price,description,current_count,is_available FROM company_products",
	).WithArgs(productId).
		WillReturnRows(
			pgxmock.NewRows(
				[]string{
					"id", "company_id", "partner_product_id", "name", "price", "description", "current_count", "is_available",
				}),
		)

	_, err := deps.repo.GetCompanyProduct(t.Context(), productId)
	require.ErrorIs(t, err, core_errors.ErrNotFound)
	assert.NoError(t, deps.assertExpectations())
}

func TestGetCompanyProduct_Internal(t *testing.T) {
	deps := newTestDeps(t)

	productId := uuid.New()
	unexpectedErr := errors.New("internal error")

	deps.pool.ExpectQuery(
		"SELECT id,company_id,partner_product_id,name,price,description,current_count,is_available FROM company_products",
	).WithArgs(productId).
		WillReturnError(unexpectedErr)

	_, err := deps.repo.GetCompanyProduct(t.Context(), productId)
	require.ErrorIs(t, err, unexpectedErr)
	assert.NoError(t, deps.assertExpectations())
}
