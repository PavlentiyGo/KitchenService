package repository_orders

import (
	"avitoKitchen/internal/core/domain"
	core_errors "avitoKitchen/internal/core/errors"
	core_messaging "avitoKitchen/internal/core/messaging"
	"avitoKitchen/internal/repository"
	"avitoKitchen/internal/repository/postgres/models"
	"context"
	"errors"
	"fmt"

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

func (r *Repository) DecreaseProductsCount(ctx context.Context, companyId uuid.UUID, products []domain.BasketProduct) error {
	ctx, cancel := context.WithTimeout(ctx, r.txManager.GetTimeout())
	defer cancel()

	if len(products) == 0 {
		return fmt.Errorf("products are empty: %w", core_errors.ErrInvalidRequest)
	}

	productIds := make([]uuid.UUID, len(products))
	counts := make([]int, len(products))
	for i, product := range products {
		if product.Id == uuid.Nil {
			return fmt.Errorf("product id is empty: %w", core_errors.ErrInvalidRequest)
		}
		if product.ChosenCount <= 0 {
			return fmt.Errorf(
				"product %s count must be greater than zero: %w",
				product.Id,
				core_errors.ErrInvalidRequest,
			)
		}
		productIds[i] = product.Id
		counts[i] = product.ChosenCount
	}

	sqlQuery := `
		UPDATE company_products AS cp
		SET current_count = cp.current_count - requested.count
		FROM unnest($2::uuid[], $3::integer[])
			AS requested(product_id, count)
		WHERE cp.company_id = $1
		  AND cp.id = requested.product_id
		  AND cp.is_available
	`

	commandTag, err := r.txManager.GetExecutor(ctx).Exec(
		ctx,
		sqlQuery,
		companyId,
		productIds,
		counts,
	)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.Code == "23514" {
			return fmt.Errorf("not enough products in stock: %w", core_errors.ErrInsufficientStock)
		}
		return fmt.Errorf("failed to decrease products count: %w", err)
	}
	if commandTag.RowsAffected() != int64(len(products)) {
		return fmt.Errorf("some products are unavailable: %w", core_errors.ErrProductUnavailable)
	}

	return nil
}

func (r *Repository) CancelOrder(ctx context.Context, orderId uuid.UUID, cancelReason core_messaging.Reason) error {
	ctx, cancel := context.WithTimeout(ctx, r.txManager.GetTimeout())
	defer cancel()

	sqlQuery := `
		UPDATE user_orders
		SET status = 'rejected',
			rejection_reason = $2,
			updated_at = now()
		WHERE id = $1
		  AND status = 'created'
	`

	commandTag, err := r.txManager.GetExecutor(ctx).Exec(
		ctx,
		sqlQuery,
		orderId,
		cancelReason,
	)
	if err != nil {
		return fmt.Errorf("failed to cancel order %s: %w", orderId, err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("order %s not found: %w", orderId, core_errors.ErrNotFound)
	}

	return nil
}

func (r *Repository) RestoreOrderProducts(ctx context.Context, orderId uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, r.txManager.GetTimeout())
	defer cancel()

	sqlQuery := `
		UPDATE company_products AS cp
		SET current_count = cp.current_count + oi.count
		FROM user_order_items AS oi
		WHERE oi.order_id = $1
		  AND cp.id = oi.product_id
	`

	commandTag, err := r.txManager.GetExecutor(ctx).Exec(ctx, sqlQuery, orderId)
	if err != nil {
		return fmt.Errorf("failed to restore products for order %s: %w", orderId, err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("products for order %s not found: %w", orderId, core_errors.ErrNotFound)
	}

	return nil
}

func (r *Repository) CreateOrder(ctx context.Context, userId uuid.UUID, companyId uuid.UUID) (uuid.UUID, error) {
	ctx, cancel := context.WithTimeout(ctx, r.txManager.GetTimeout())
	defer cancel()

	sqlQuery := `
		INSERT INTO user_orders (user_id, company_id)
		VALUES ($1, $2)
		RETURNING id
	`

	var orderId uuid.UUID
	err := r.txManager.GetExecutor(ctx).QueryRow(
		ctx,
		sqlQuery,
		userId,
		companyId,
	).Scan(&orderId)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create order: %w", err)
	}

	return orderId, nil
}

func (r *Repository) CreateOrderItems(ctx context.Context, orderId uuid.UUID, products []domain.BasketProduct) error {
	ctx, cancel := context.WithTimeout(ctx, r.txManager.GetTimeout())
	defer cancel()

	if len(products) == 0 {
		return fmt.Errorf("order products are empty: %w", core_errors.ErrInvalidRequest)
	}

	productIds := make([]uuid.UUID, len(products))
	counts := make([]int, len(products))
	productNames := make([]string, len(products))
	productPrices := make([]int64, len(products))
	for i, product := range products {
		productIds[i] = product.Id
		counts[i] = product.ChosenCount
		productNames[i] = product.Name
		productPrices[i] = product.ConfirmedPrice
	}

	sqlQuery := `
		INSERT INTO user_order_items (
			order_id,
			product_id,
			count,
			product_name,
			product_price
		)
		SELECT
			$1,
			item.product_id,
			item.count,
			item.product_name,
			item.product_price
		FROM unnest(
			$2::uuid[],
			$3::integer[],
			$4::text[],
			$5::bigint[]
		) AS item(product_id, count, product_name, product_price)
	`

	_, err := r.txManager.GetExecutor(ctx).Exec(
		ctx,
		sqlQuery,
		orderId,
		productIds,
		counts,
		productNames,
		productPrices,
	)
	if err != nil {
		return fmt.Errorf("failed to create order items: %w", err)
	}

	return nil
}

func (r *Repository) GetOrderForSend(ctx context.Context, orderID uuid.UUID) (domain.OrderForSend, error) {
	ctx, cancel := context.WithTimeout(ctx, r.txManager.GetTimeout())
	defer cancel()

	sqlQuery := `
		SELECT
			uo.id,
			c.company_source_url,
			cp.partner_product_id,
			oi.count
		FROM user_orders AS uo
		JOIN companies AS c
			ON c.id = uo.company_id
		JOIN user_order_items AS oi
			ON oi.order_id = uo.id
		JOIN company_products AS cp
			ON cp.id = oi.product_id
		WHERE uo.id = $1
		ORDER BY oi.id
	`

	rows, err := r.txManager.GetExecutor(ctx).Query(ctx, sqlQuery, orderID)
	if err != nil {
		return domain.OrderForSend{}, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var order domain.OrderForSend
	found := false
	for rows.Next() {
		var productModel models.ProductForSendModel
		err = rows.Scan(
			&productModel.OrderId,
			&productModel.CompanyURL,
			&productModel.PartnerProductId,
			&productModel.Count,
		)
		if err != nil {
			return domain.OrderForSend{}, fmt.Errorf("row scan: %w", err)
		}

		if !found {
			order = models.OrderForSendDomainFromModel(productModel)
			found = true
			continue
		}

		order.Products = append(
			order.Products,
			models.ProductForSendDomainFromModel(productModel),
		)
	}
	err = rows.Err()
	if err != nil {
		return domain.OrderForSend{}, fmt.Errorf("rows iteration: %w", err)
	}
	if !found {
		return domain.OrderForSend{}, fmt.Errorf("order %s: %w", orderID, core_errors.ErrNotFound)
	}

	return order, nil
}

func (r *Repository) GetUserOrders(ctx context.Context, userId uuid.UUID) ([]domain.OrderRow, error) {
	ctx, cancel := context.WithTimeout(ctx, r.txManager.GetTimeout())
	defer cancel()

	sqlQuery := `
		SELECT
			uo.id,
			uo.user_id,
			uo.company_id,
			uo.status::text,
			uo.rejection_reason,
			uo.created_at,
			uo.updated_at,
			oi.product_id,
			oi.product_name,
			oi.product_price,
			oi.count
		FROM user_orders AS uo
		JOIN user_order_items AS oi
			ON oi.order_id = uo.id
		WHERE uo.user_id = $1
		ORDER BY uo.created_at DESC, uo.id, oi.id
	`

	rows, err := r.txManager.GetExecutor(ctx).Query(ctx, sqlQuery, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to get user orders: %w", err)
	}
	defer rows.Close()

	return scanOrderRows(rows)
}

func (r *Repository) GetOrder(ctx context.Context, orderId uuid.UUID) ([]domain.OrderRow, error) {
	ctx, cancel := context.WithTimeout(ctx, r.txManager.GetTimeout())
	defer cancel()

	sqlQuery := `
		SELECT
			uo.id,
			uo.user_id,
			uo.company_id,
			uo.status::text,
			uo.rejection_reason,
			uo.created_at,
			uo.updated_at,
			oi.product_id,
			oi.product_name,
			oi.product_price,
			oi.count
		FROM user_orders AS uo
		JOIN user_order_items AS oi
			ON oi.order_id = uo.id
		WHERE uo.id = $1
	`

	rows, err := r.txManager.GetExecutor(ctx).Query(ctx, sqlQuery, orderId)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	defer rows.Close()

	orderRows, err := scanOrderRows(rows)
	if err != nil {
		return nil, err
	}
	if len(orderRows) == 0 {
		return nil, fmt.Errorf("order %s: %w", orderId, core_errors.ErrNotFound)
	}

	return orderRows, nil
}

func scanOrderRows(rows pgx.Rows) ([]domain.OrderRow, error) {
	orderRows := make([]domain.OrderRow, 0)
	for rows.Next() {
		var orderModel models.OrderModel
		var productModel models.OrderProductModel
		err := rows.Scan(
			&orderModel.Id,
			&orderModel.UserId,
			&orderModel.CompanyId,
			&orderModel.Status,
			&orderModel.RejectionReason,
			&orderModel.CreatedAt,
			&orderModel.UpdatedAt,
			&productModel.Id,
			&productModel.Name,
			&productModel.Price,
			&productModel.Count,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user order: %w", err)
		}

		orderRows = append(
			orderRows,
			models.OrderRowDomainFromModels(orderModel, productModel),
		)
	}
	err := rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to iterate user orders: %w", err)
	}

	return orderRows, nil
}
