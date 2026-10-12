package repository_companies

import (
	"context"
	"errors"
	"fmt"
	"foodDelivery/internal/core/domain"
	core_errors "foodDelivery/internal/core/errors"
	"foodDelivery/internal/repository"
	"foodDelivery/internal/repository/postgres/models"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Repository struct {
	txManager repository.TxManager
}

func NewRepository(txManager repository.TxManager) *Repository {
	return &Repository{txManager: txManager}
}

func (r *Repository) CreateProduct(ctx context.Context, companyId uuid.UUID, product domain.Product) (domain.Product, error) {
	ctx, cancel := context.WithTimeout(ctx, r.txManager.GetTimeout())
	defer cancel()

	sqlQuery := `
		INSERT INTO company_products (
			company_id,
			partner_product_id,
			name,
			price,
			description,
			current_count,
			is_available
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING
			id,
			company_id,
			partner_product_id,
			name,
			price,
			description,
			current_count,
			is_available
	`

	var productModel models.ProductModel
	err := r.txManager.GetExecutor(ctx).QueryRow(
		ctx,
		sqlQuery,
		companyId,
		product.PartnerProductId,
		product.Name,
		product.Price,
		product.Description,
		product.CurrentCount,
		product.IsAvailable,
	).Scan(
		&productModel.Id,
		&productModel.CompanyId,
		&productModel.PartnerProductId,
		&productModel.Name,
		&productModel.Price,
		&productModel.Description,
		&productModel.CurrentCount,
		&productModel.IsAvailable,
	)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) {
			switch postgresError.Code {
			case "23503":
				return domain.Product{}, fmt.Errorf("company %s not found: %w", companyId, core_errors.ErrNotFound)
			case "23505":
				return domain.Product{}, fmt.Errorf(
					"product with partner id %q already exists: %w",
					*product.PartnerProductId,
					core_errors.ErrConflict,
				)
			}
		}
		return domain.Product{}, fmt.Errorf("failed to execute query: %w", err)
	}

	return models.ProductDomainFromModel(productModel), nil
}

func (r *Repository) PatchProduct(ctx context.Context, companyId uuid.UUID, partnerProductId string, patch domain.ProductPatch) (domain.Product, error) {
	ctx, cancel := context.WithTimeout(ctx, r.txManager.GetTimeout())
	defer cancel()

	sqlQuery := `
		UPDATE company_products
		SET
			name = COALESCE($3, name),
			price = COALESCE($4, price),
			description = COALESCE($5, description),
			current_count = COALESCE($6, current_count),
			is_available = COALESCE($7, is_available)
		WHERE company_id = $1
		  AND partner_product_id = $2
		RETURNING
			id,
			company_id,
			partner_product_id,
			name,
			price,
			description,
			current_count,
			is_available
	`

	var productModel models.ProductModel
	err := r.txManager.GetExecutor(ctx).QueryRow(
		ctx,
		sqlQuery,
		companyId,
		partnerProductId,
		patch.Name,
		patch.Price,
		patch.Description,
		patch.CurrentCount,
		patch.IsAvailable,
	).Scan(
		&productModel.Id,
		&productModel.CompanyId,
		&productModel.PartnerProductId,
		&productModel.Name,
		&productModel.Price,
		&productModel.Description,
		&productModel.CurrentCount,
		&productModel.IsAvailable,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Product{}, fmt.Errorf("product with partner id %q not found: %w", partnerProductId, core_errors.ErrNotFound)
	}
	if err != nil {
		return domain.Product{}, fmt.Errorf("failed to execute query: %w", err)
	}

	return models.ProductDomainFromModel(productModel), nil
}

func (r *Repository) UpdateCompanyOrderStatus(ctx context.Context, companyId uuid.UUID, orderId uuid.UUID, status domain.OrderStatus) (time.Time, error) {
	ctx, cancel := context.WithTimeout(ctx, r.txManager.GetTimeout())
	defer cancel()

	sqlQuery := `
		UPDATE user_orders
		SET status = $3::user_order_status,
			updated_at = now()
		WHERE id = $1
		  AND company_id = $2
		  AND status <> 'rejected'
		RETURNING updated_at
	`

	var updatedAt time.Time
	err := r.txManager.GetExecutor(ctx).QueryRow(
		ctx,
		sqlQuery,
		orderId,
		companyId,
		status,
	).Scan(&updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, fmt.Errorf("order %s cannot change status: %w", orderId, core_errors.ErrConflict)
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to update company order: %w", err)
	}

	return updatedAt, nil
}

func (r *Repository) AddCompanyOrderStatusHistory(ctx context.Context, orderId uuid.UUID, status domain.OrderStatus) error {
	ctx, cancel := context.WithTimeout(ctx, r.txManager.GetTimeout())
	defer cancel()

	sqlQuery := `
		INSERT INTO user_order_status_history (order_id, status)
		VALUES ($1, $2::user_order_status)
		ON CONFLICT (order_id, status) DO NOTHING
	`

	_, err := r.txManager.GetExecutor(ctx).Exec(ctx, sqlQuery, orderId, status)
	if err != nil {
		return fmt.Errorf("failed to insert order status history: %w", err)
	}

	return nil
}
