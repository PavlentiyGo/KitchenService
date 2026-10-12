package service_companies

import (
	"context"
	"fmt"
	"foodDelivery/internal/core/domain"
	core_errors "foodDelivery/internal/core/errors"
	"strings"

	"github.com/google/uuid"
)

type Service struct {
	txManager     txManager
	companiesRepo companiesRepository
	ordersRepo    ordersRepository
}

func NewService(txManager txManager, companiesRepo companiesRepository, ordersRepo ordersRepository) *Service {
	return &Service{
		txManager:     txManager,
		companiesRepo: companiesRepo,
		ordersRepo:    ordersRepo,
	}
}

func (s *Service) AddNewProduct(ctx context.Context, companyId uuid.UUID, product domain.Product) (domain.Product, error) {
	if product.PartnerProductId == nil || strings.TrimSpace(*product.PartnerProductId) == "" {
		return domain.Product{}, fmt.Errorf("partner product id is empty: %w", core_errors.ErrInvalidRequest)
	}
	if strings.TrimSpace(product.Name) == "" {
		return domain.Product{}, fmt.Errorf("product name is empty: %w", core_errors.ErrInvalidRequest)
	}
	if product.Price <= 0 {
		return domain.Product{}, fmt.Errorf("product price must be greater than zero: %w", core_errors.ErrInvalidRequest)
	}
	if product.CurrentCount < 0 {
		return domain.Product{}, fmt.Errorf("product count cannot be negative: %w", core_errors.ErrInvalidRequest)
	}

	createdProduct, err := s.companiesRepo.CreateProduct(ctx, companyId, product)
	if err != nil {
		return domain.Product{}, fmt.Errorf("failed to create product: %w", err)
	}

	return createdProduct, nil
}

func (s *Service) PatchCompanyProduct(ctx context.Context, companyId uuid.UUID, partnerProductId string, patch domain.ProductPatch) (domain.Product, error) {
	if strings.TrimSpace(partnerProductId) == "" {
		return domain.Product{}, fmt.Errorf("partner product id is empty: %w", core_errors.ErrInvalidRequest)
	}
	if patch.Name != nil && strings.TrimSpace(*patch.Name) == "" {
		return domain.Product{}, fmt.Errorf("product name is empty: %w", core_errors.ErrInvalidRequest)
	}
	if patch.Price != nil && *patch.Price <= 0 {
		return domain.Product{}, fmt.Errorf("product price must be greater than zero: %w", core_errors.ErrInvalidRequest)
	}
	if patch.CurrentCount != nil && *patch.CurrentCount < 0 {
		return domain.Product{}, fmt.Errorf("product count cannot be negative: %w", core_errors.ErrInvalidRequest)
	}

	product, err := s.companiesRepo.PatchProduct(ctx, companyId, partnerProductId, patch)
	if err != nil {
		return domain.Product{}, fmt.Errorf("failed to patch product: %w", err)
	}

	return product, nil
}

func (s *Service) ChangeOrderStatus(ctx context.Context, companyId uuid.UUID, orderId uuid.UUID, status domain.OrderStatus) (domain.Order, error) {
	if !isValidOrderStatus(status) {
		return domain.Order{}, fmt.Errorf("unknown order status %q: %w", status, core_errors.ErrInvalidRequest)
	}

	var order domain.Order
	err := s.txManager.WithinTx(ctx, func(txCtx context.Context) error {
		orderRows, err := s.ordersRepo.GetOrder(txCtx, orderId)
		if err != nil {
			return fmt.Errorf("failed to get company order: %w", err)
		}
		currentOrder := orderFromRows(orderRows)
		if currentOrder.CompanyId != companyId {
			return fmt.Errorf("order %s: %w", orderId, core_errors.ErrNotFound)
		}
		if currentOrder.Status == status {
			order = currentOrder
			return nil
		}
		if currentOrder.Status == domain.OrderStatusRejected {
			return fmt.Errorf("rejected order %s cannot change status: %w", orderId, core_errors.ErrConflict)
		}
		updatedAt, err := s.companiesRepo.UpdateCompanyOrderStatus(
			txCtx,
			companyId,
			orderId,
			status,
		)
		if err != nil {
			return fmt.Errorf("failed to update company order status: %w", err)
		}
		err = s.companiesRepo.AddCompanyOrderStatusHistory(txCtx, orderId, status)
		if err != nil {
			return fmt.Errorf("failed to add company order status history: %w", err)
		}

		currentOrder.Status = status
		currentOrder.UpdatedAt = updatedAt
		order = currentOrder
		return nil
	})
	if err != nil {
		return domain.Order{}, err
	}

	return order, nil
}

func orderFromRows(orderRows []domain.OrderRow) domain.Order {
	firstRow := orderRows[0]
	order := domain.Order{
		Id:              firstRow.OrderId,
		UserId:          firstRow.UserId,
		CompanyId:       firstRow.CompanyId,
		Status:          firstRow.Status,
		RejectionReason: firstRow.RejectionReason,
		Products:        make([]domain.OrderProduct, 0, len(orderRows)),
		CreatedAt:       firstRow.CreatedAt,
		UpdatedAt:       firstRow.UpdatedAt,
	}
	for _, row := range orderRows {
		order.Products = append(order.Products, row.Product)
		order.FullPrice += row.Product.Price * int64(row.Product.Count)
	}

	return order
}

func isValidOrderStatus(status domain.OrderStatus) bool {
	switch status {
	case domain.OrderStatusCreated,
		domain.OrderStatusRejected,
		domain.OrderStatusAccepted,
		domain.OrderStatusPreparing,
		domain.OrderStatusCompleted:
		return true
	default:
		return false
	}
}
