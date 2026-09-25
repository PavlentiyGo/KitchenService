package repository_companies_test

import (
	"avitoKitchen/internal/core/domain"
	core_errors "avitoKitchen/internal/core/errors"
	"avitoKitchen/internal/repository"
	repository_companies "avitoKitchen/internal/repository/postgres/companies"
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
	repo *repository_companies.Repository
}

func newTestDeps(t *testing.T) *testDeps {
	t.Helper()
	pool, err := pgxmock.NewPool()
	require.NoError(t, err)
	txManager := &txManagerMock{pool: pool}

	return &testDeps{
		pool: pool,
		repo: repository_companies.NewRepository(txManager),
	}
}
func TestCreateProduct_Success(t *testing.T) {
	deps := newTestDeps(t)

	companyId := uuid.New()
	partnerProductId := "partner_product_id"
	productForCreate := domain.Product{
		PartnerProductId: &partnerProductId,
		Name:             "product",
		Price:            100,
		Description:      "description",
		CurrentCount:     10,
		IsAvailable:      true,
	}
	productExpect := productForCreate
	productExpect.Id = uuid.New()
	productExpect.CompanyId = companyId

	deps.pool.ExpectQuery(
		"INSERT INTO company_products",
	).WithArgs(
		companyId,
		productForCreate.PartnerProductId,
		productForCreate.Name,
		productForCreate.Price,
		productForCreate.Description,
		productForCreate.CurrentCount,
		productForCreate.IsAvailable,
	).WillReturnRows(
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

	product, err := deps.repo.CreateProduct(t.Context(), companyId, productForCreate)
	require.NoError(t, err)
	require.Equal(t, productExpect, product)
	assert.NoError(t, deps.assertExpectations())
}

func TestCreateProduct_CompanyNotFound(t *testing.T) {
	deps := newTestDeps(t)

	companyId := uuid.New()
	partnerProductId := "partner_product_id"
	productForCreate := domain.Product{
		PartnerProductId: &partnerProductId,
		Name:             "product",
		Price:            100,
		Description:      "description",
		CurrentCount:     10,
		IsAvailable:      true,
	}

	deps.pool.ExpectQuery(
		"INSERT INTO company_products",
	).WithArgs(
		companyId,
		productForCreate.PartnerProductId,
		productForCreate.Name,
		productForCreate.Price,
		productForCreate.Description,
		productForCreate.CurrentCount,
		productForCreate.IsAvailable,
	).WillReturnError(&pgconn.PgError{Code: "23503"})

	_, err := deps.repo.CreateProduct(t.Context(), companyId, productForCreate)
	require.ErrorIs(t, err, core_errors.ErrNotFound)
	assert.NoError(t, deps.assertExpectations())
}

func TestCreateProduct_Conflict(t *testing.T) {
	deps := newTestDeps(t)

	companyId := uuid.New()
	partnerProductId := "partner_product_id"
	productForCreate := domain.Product{
		PartnerProductId: &partnerProductId,
		Name:             "product",
		Price:            100,
		Description:      "description",
		CurrentCount:     10,
		IsAvailable:      true,
	}

	deps.pool.ExpectQuery(
		"INSERT INTO company_products",
	).WithArgs(
		companyId,
		productForCreate.PartnerProductId,
		productForCreate.Name,
		productForCreate.Price,
		productForCreate.Description,
		productForCreate.CurrentCount,
		productForCreate.IsAvailable,
	).WillReturnError(&pgconn.PgError{Code: "23505"})

	_, err := deps.repo.CreateProduct(t.Context(), companyId, productForCreate)
	require.ErrorIs(t, err, core_errors.ErrConflict)
	assert.NoError(t, deps.assertExpectations())
}

func TestCreateProduct_Internal(t *testing.T) {
	deps := newTestDeps(t)

	companyId := uuid.New()
	partnerProductId := "partner_product_id"
	productForCreate := domain.Product{
		PartnerProductId: &partnerProductId,
		Name:             "product",
		Price:            100,
		Description:      "description",
		CurrentCount:     10,
		IsAvailable:      true,
	}
	unexpectedErr := errors.New("internal error")

	deps.pool.ExpectQuery(
		"INSERT INTO company_products",
	).WithArgs(
		companyId,
		productForCreate.PartnerProductId,
		productForCreate.Name,
		productForCreate.Price,
		productForCreate.Description,
		productForCreate.CurrentCount,
		productForCreate.IsAvailable,
	).WillReturnError(unexpectedErr)

	_, err := deps.repo.CreateProduct(t.Context(), companyId, productForCreate)
	require.ErrorIs(t, err, unexpectedErr)
	assert.NoError(t, deps.assertExpectations())
}

func TestPatchProduct_Success(t *testing.T) {
	deps := newTestDeps(t)

	companyId := uuid.New()
	partnerProductId := "partner_product_id"
	name := "updated product"
	price := int64(200)
	description := "updated description"
	currentCount := 20
	isAvailable := false
	patch := domain.ProductPatch{
		Name:         &name,
		Price:        &price,
		Description:  &description,
		CurrentCount: &currentCount,
		IsAvailable:  &isAvailable,
	}
	productExpect := domain.Product{
		Id:               uuid.New(),
		CompanyId:        companyId,
		PartnerProductId: &partnerProductId,
		Name:             name,
		Price:            price,
		Description:      description,
		CurrentCount:     currentCount,
		IsAvailable:      isAvailable,
	}

	deps.pool.ExpectQuery(
		"UPDATE company_products",
	).WithArgs(
		companyId,
		partnerProductId,
		patch.Name,
		patch.Price,
		patch.Description,
		patch.CurrentCount,
		patch.IsAvailable,
	).WillReturnRows(
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

	product, err := deps.repo.PatchProduct(t.Context(), companyId, partnerProductId, patch)
	require.NoError(t, err)
	require.Equal(t, productExpect, product)
	assert.NoError(t, deps.assertExpectations())
}

func TestPatchProduct_Fail(t *testing.T) {
	deps := newTestDeps(t)

	companyId := uuid.New()
	partnerProductId := "partner_product_id"
	patch := domain.ProductPatch{}

	deps.pool.ExpectQuery(
		"UPDATE company_products",
	).WithArgs(
		companyId,
		partnerProductId,
		patch.Name,
		patch.Price,
		patch.Description,
		patch.CurrentCount,
		patch.IsAvailable,
	).WillReturnRows(
		pgxmock.NewRows(
			[]string{
				"id", "company_id", "partner_product_id", "name", "price", "description", "current_count", "is_available",
			}),
	)

	_, err := deps.repo.PatchProduct(t.Context(), companyId, partnerProductId, patch)
	require.ErrorIs(t, err, core_errors.ErrNotFound)
	assert.NoError(t, deps.assertExpectations())
}

func TestPatchProduct_Internal(t *testing.T) {
	deps := newTestDeps(t)

	companyId := uuid.New()
	partnerProductId := "partner_product_id"
	patch := domain.ProductPatch{}
	unexpectedErr := errors.New("internal error")

	deps.pool.ExpectQuery(
		"UPDATE company_products",
	).WithArgs(
		companyId,
		partnerProductId,
		patch.Name,
		patch.Price,
		patch.Description,
		patch.CurrentCount,
		patch.IsAvailable,
	).WillReturnError(unexpectedErr)

	_, err := deps.repo.PatchProduct(t.Context(), companyId, partnerProductId, patch)
	require.ErrorIs(t, err, unexpectedErr)
	assert.NoError(t, deps.assertExpectations())
}

func TestUpdateCompanyOrderStatus_Success(t *testing.T) {
	deps := newTestDeps(t)

	companyId := uuid.New()
	orderId := uuid.New()
	status := domain.OrderStatusAccepted
	updatedAtExpect := time.Date(2026, time.September, 8, 12, 0, 0, 0, time.UTC)

	deps.pool.ExpectQuery(
		"UPDATE user_orders",
	).WithArgs(orderId, companyId, status).
		WillReturnRows(
			pgxmock.NewRows(
				[]string{
					"updated_at",
				}).AddRow(updatedAtExpect),
		)

	updatedAt, err := deps.repo.UpdateCompanyOrderStatus(t.Context(), companyId, orderId, status)
	require.NoError(t, err)
	require.Equal(t, updatedAtExpect, updatedAt)
	assert.NoError(t, deps.assertExpectations())
}

func TestUpdateCompanyOrderStatus_Fail(t *testing.T) {
	deps := newTestDeps(t)

	companyId := uuid.New()
	orderId := uuid.New()
	status := domain.OrderStatusAccepted

	deps.pool.ExpectQuery(
		"UPDATE user_orders",
	).WithArgs(orderId, companyId, status).
		WillReturnRows(
			pgxmock.NewRows(
				[]string{
					"updated_at",
				}),
		)

	_, err := deps.repo.UpdateCompanyOrderStatus(t.Context(), companyId, orderId, status)
	require.ErrorIs(t, err, core_errors.ErrConflict)
	assert.NoError(t, deps.assertExpectations())
}

func TestUpdateCompanyOrderStatus_Internal(t *testing.T) {
	deps := newTestDeps(t)

	companyId := uuid.New()
	orderId := uuid.New()
	status := domain.OrderStatusAccepted
	unexpectedErr := errors.New("internal error")

	deps.pool.ExpectQuery(
		"UPDATE user_orders",
	).WithArgs(orderId, companyId, status).
		WillReturnError(unexpectedErr)

	_, err := deps.repo.UpdateCompanyOrderStatus(t.Context(), companyId, orderId, status)
	require.ErrorIs(t, err, unexpectedErr)
	assert.NoError(t, deps.assertExpectations())
}

func TestAddCompanyOrderStatusHistory_Success(t *testing.T) {
	deps := newTestDeps(t)

	orderId := uuid.New()
	status := domain.OrderStatusCreated

	deps.pool.ExpectExec(
		"INSERT INTO user_order_status_history",
	).WithArgs(orderId, status).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := deps.repo.AddCompanyOrderStatusHistory(t.Context(), orderId, status)
	require.NoError(t, err)
	assert.NoError(t, deps.assertExpectations())
}

func TestAddCompanyOrderStatusHistory_Internal(t *testing.T) {
	deps := newTestDeps(t)

	orderId := uuid.New()
	status := domain.OrderStatusCreated
	unexpectedErr := errors.New("internal error")

	deps.pool.ExpectExec(
		"INSERT INTO user_order_status_history",
	).WithArgs(orderId, status).
		WillReturnError(unexpectedErr)

	err := deps.repo.AddCompanyOrderStatusHistory(t.Context(), orderId, status)
	require.ErrorIs(t, err, unexpectedErr)
	assert.NoError(t, deps.assertExpectations())
}
