package repository_basket

import (
	"avitoKitchen/internal/core/domain"
	core_errors "avitoKitchen/internal/core/errors"
	"avitoKitchen/internal/repository"
	"avitoKitchen/internal/repository/postgres/models"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type Repository struct {
	txManager repository.TxManager
}

func NewRepository(txManager repository.TxManager) *Repository {
	return &Repository{txManager: txManager}
}

func (r *Repository) GetUserBasketProducts(ctx context.Context, userId uuid.UUID, companyId uuid.UUID) ([]domain.BasketProduct, error) {
	ctx, cancel := context.WithTimeout(ctx, r.txManager.GetTimeout())
	defer cancel()

	sqlQuery := `
	SELECT cp.id,cp.name,cp.price,cp.description,cp.is_available,cp.current_count,ubp.count,ubp.confirmed_price FROM user_baskets ub
	JOIN user_basket_products as ubp
	ON ub.id = ubp.basket_id
	JOIN company_products cp
	ON ubp.product_id = cp.id
	WHERE ub.company_id = $1 AND ub.user_id = $2;
	`

	rows, err := r.txManager.GetExecutor(ctx).Query(ctx, sqlQuery, companyId, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	products := make([]domain.BasketProduct, 0, 10)
	for rows.Next() {
		var model models.BasketProductModel
		err = rows.Scan(
			&model.Id,
			&model.Name,
			&model.CurrentPrice,
			&model.Description,
			&model.IsAvailable,
			&model.CurrentCount,
			&model.ChosenCount,
			&model.ConfirmedPrice,
		)
		if err != nil {
			return nil, fmt.Errorf("row scan: %w", err)
		}
		products = append(products, models.BasketProductDomainFromModel(model))
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return products, nil
}

func (r *Repository) PutBasketProduct(
	ctx context.Context,
	userId uuid.UUID,
	companyId uuid.UUID,
	productId uuid.UUID,
	count int,
	confirmedPrice int64,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.txManager.GetTimeout())
	defer cancel()

	sqlQuery := `
	WITH basket AS (
		INSERT INTO user_baskets (company_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (company_id, user_id)
		DO UPDATE SET user_id = EXCLUDED.user_id
		RETURNING id
	)
	INSERT INTO user_basket_products (basket_id, product_id, count, confirmed_price)
	SELECT id, $3, $4, $5 FROM basket
	ON CONFLICT (basket_id, product_id)
	DO UPDATE SET 
	count = EXCLUDED.count, confirmed_price = EXCLUDED.confirmed_price
	`

	_, err := r.txManager.GetExecutor(ctx).Exec(
		ctx,
		sqlQuery,
		companyId,
		userId,
		productId,
		count,
		confirmedPrice,
	)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}

func (r *Repository) DeleteBasketProduct(ctx context.Context, userId uuid.UUID, companyId uuid.UUID, productId uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, r.txManager.GetTimeout())
	defer cancel()

	sqlQuery := `
	DELETE FROM user_basket_products AS ubp
	USING user_baskets AS ub
	WHERE ubp.basket_id = ub.id
	  AND ub.user_id = $1
	  AND ub.company_id = $2
	  AND ubp.product_id = $3
	`

	_, err := r.txManager.GetExecutor(ctx).Exec(
		ctx,
		sqlQuery,
		userId,
		companyId,
		productId,
	)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}

func (r *Repository) ClearBasket(ctx context.Context, userId uuid.UUID, companyId uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, r.txManager.GetTimeout())
	defer cancel()

	sqlQuery := `
		DELETE FROM user_basket_products AS ubp
		USING user_baskets AS ub
		WHERE ubp.basket_id = ub.id
		  AND ub.user_id = $1
		  AND ub.company_id = $2
	`

	cmdTag, err := r.txManager.GetExecutor(ctx).Exec(
		ctx,
		sqlQuery,
		userId,
		companyId,
	)
	if err != nil {
		return fmt.Errorf("failed to clear basket: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("basket is clear: %w", core_errors.ErrConflict)
	}

	return nil
}

func (r *Repository) ConfirmBasketProductsPrice(ctx context.Context, userId uuid.UUID, companyId uuid.UUID, productIds []uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, r.txManager.GetTimeout())
	defer cancel()

	sqlQuery := `
	UPDATE user_basket_products AS ubp
	SET confirmed_price = cp.price
	FROM user_baskets AS ub,
		 company_products AS cp
	WHERE ubp.basket_id = ub.id
	  AND ubp.product_id = cp.id
	  AND ub.user_id = $1
	  AND ub.company_id = $2
	  AND cp.company_id = ub.company_id
	  AND ubp.product_id = ANY($3::uuid[])
	`

	_, err := r.txManager.GetExecutor(ctx).Exec(
		ctx,
		sqlQuery,
		userId,
		companyId,
		productIds,
	)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}
