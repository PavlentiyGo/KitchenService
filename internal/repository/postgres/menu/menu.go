package repository_menu

import (
	"avitoKitchen/internal/core/domain"
	core_errors "avitoKitchen/internal/core/errors"
	"avitoKitchen/internal/repository"
	"avitoKitchen/internal/repository/postgres/models"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct {
	txManager repository.TxManager
}

func NewRepository(txManager repository.TxManager) *Repository {
	return &Repository{txManager: txManager}
}

func (r *Repository) GetCompany(ctx context.Context, companyId uuid.UUID) (domain.Company, error) {
	ctx, cancel := context.WithTimeout(ctx, r.txManager.GetTimeout())
	defer cancel()

	sqlQuery := `SELECT id,name,is_active,company_source_url FROM companies WHERE id=$1`

	var model models.CompanyModel
	err := r.txManager.GetExecutor(ctx).QueryRow(ctx, sqlQuery, companyId).Scan(
		&model.Id,
		&model.Name,
		&model.IsActive,
		&model.Url,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Company{}, fmt.Errorf("company %s not found: %w", companyId, core_errors.ErrNotFound)
		}
		return domain.Company{}, fmt.Errorf("failed to execute query: %w", err)
	}

	return models.CompanyDomainFromModel(model), nil
}

func (r *Repository) GetUser(ctx context.Context, userId uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, r.txManager.GetTimeout())
	defer cancel()

	sqlQuery := `SELECT id FROM users WHERE id=$1`

	var foundUserId uuid.UUID
	err := r.txManager.GetExecutor(ctx).QueryRow(ctx, sqlQuery, userId).Scan(&foundUserId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("user %s not found: %w", userId, core_errors.ErrNotFound)
		}
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}

func (r *Repository) GetCompanies(ctx context.Context) ([]domain.Company, error) {
	ctx, cancel := context.WithTimeout(ctx, r.txManager.GetTimeout())
	defer cancel()

	sqlQuery := `SELECT id,name,is_active,company_source_url FROM companies`

	rows, err := r.txManager.GetExecutor(ctx).Query(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	var companies []domain.Company
	for rows.Next() {
		var model models.CompanyModel

		err = rows.Scan(
			&model.Id,
			&model.Name,
			&model.IsActive,
			&model.Url,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		companies = append(companies, models.CompanyDomainFromModel(model))
	}

	return companies, nil
}

func (r *Repository) GetCompanyProducts(ctx context.Context, companyId uuid.UUID) ([]domain.Product, error) {
	ctx, cancel := context.WithTimeout(ctx, r.txManager.GetTimeout())
	defer cancel()

	sqlQuery := `SELECT id,company_id,partner_product_id,name,price,description,current_count,is_available FROM company_products WHERE company_id=$1`

	rows, err := r.txManager.GetExecutor(ctx).Query(ctx, sqlQuery, companyId)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	var products []domain.Product
	for rows.Next() {
		var model models.ProductModel

		err = rows.Scan(
			&model.Id,
			&model.CompanyId,
			&model.PartnerProductId,
			&model.Name,
			&model.Price,
			&model.Description,
			&model.CurrentCount,
			&model.IsAvailable,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		products = append(products, models.ProductDomainFromModel(model))
	}

	return products, nil
}

func (r *Repository) GetCompanyProduct(ctx context.Context, productId uuid.UUID) (domain.Product, error) {
	ctx, cancel := context.WithTimeout(ctx, r.txManager.GetTimeout())
	defer cancel()

	sqlQuery := `SELECT id,company_id,partner_product_id,name,price,description,current_count,is_available FROM company_products WHERE id=$1`

	var model models.ProductModel
	err := r.txManager.GetExecutor(ctx).QueryRow(ctx, sqlQuery, productId).Scan(
		&model.Id,
		&model.CompanyId,
		&model.PartnerProductId,
		&model.Name,
		&model.Price,
		&model.Description,
		&model.CurrentCount,
		&model.IsAvailable,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Product{}, fmt.Errorf("no product with such id: %w: %w", err, core_errors.ErrNotFound)
		}
		return domain.Product{}, fmt.Errorf("failed to execute query: %w", err)
	}

	return models.ProductDomainFromModel(model), nil
}
