package service_orders

import (
	"avitoKitchen/internal/core/domain"
	core_errors "avitoKitchen/internal/core/errors"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type Service struct {
	txManager     txManager
	ordersRepo    ordersRepository
	companiesRepo companiesRepository
	menuRepo      menuRepo
	basketRepo    basketRepo
	sender        ordersSender
}

func NewService(
	txManager txManager,
	ordersRepo ordersRepository,
	companiesRepo companiesRepository,
	sender ordersSender,
	menuRepo menuRepo,
	basketRepo basketRepo,
) *Service {
	return &Service{
		txManager:     txManager,
		ordersRepo:    ordersRepo,
		companiesRepo: companiesRepo,
		sender:        sender,
		menuRepo:      menuRepo,
		basketRepo:    basketRepo,
	}
}

func (s *Service) CreateOrder(ctx context.Context, userId uuid.UUID, companyId uuid.UUID, basketItems []domain.Product) error {
	if len(basketItems) == 0 {
		return fmt.Errorf("products for confirmation are empty: %w", core_errors.ErrInvalidRequest)
	}

	err := s.menuRepo.GetUser(ctx, userId)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	_, err = s.menuRepo.GetCompany(ctx, companyId)
	if err != nil {
		return fmt.Errorf("failed to get company: %w", err)
	}

	productIds := make([]uuid.UUID, len(basketItems))
	productIdsMap := make(map[uuid.UUID]domain.Product, len(basketItems))
	for i := 0; i < len(basketItems); i++ {
		productIds[i] = basketItems[i].Id
		productIdsMap[basketItems[i].Id] = basketItems[i]
	}

	err = s.txManager.WithinTx(ctx, func(txCtx context.Context) error {
		err = s.basketRepo.ConfirmBasketProductsPrice(txCtx, userId, companyId, productIds)
		if err != nil {
			return fmt.Errorf("failed to confirm products price: %w", err)
		}
		basketProducts, err := s.basketRepo.GetUserBasketProducts(txCtx, userId, companyId)
		if err != nil {
			return fmt.Errorf("failed to get user full basket: %w", err)
		}
		if len(basketProducts) != len(basketItems) {
			return fmt.Errorf("basket in request and user basket aren't equal: %w", core_errors.ErrInvalidRequest)
		}
		for _, product := range basketProducts {
			mapProduct, ok := productIdsMap[product.Id]
			if !ok {
				return fmt.Errorf("not all products in request: %w", core_errors.ErrInvalidRequest)
			}
			if mapProduct.Price < product.ConfirmedPrice {
				return fmt.Errorf(
					"product %s price changed from %d to %d: %w",
					product.Id,
					mapProduct.Price,
					product.ConfirmedPrice,
					core_errors.ErrConflict,
				)
			}
		}
		err = s.ordersRepo.DecreaseProductsCount(txCtx, companyId, basketProducts)
		if err != nil {
			return err
		}
		orderId, err := s.ordersRepo.CreateOrder(txCtx, userId, companyId)
		if err != nil {
			return err
		}
		err = s.ordersRepo.CreateOrderItems(txCtx, orderId, basketProducts)
		if err != nil {
			return err
		}
		err = s.companiesRepo.AddCompanyOrderStatusHistory(
			txCtx,
			orderId,
			domain.OrderStatusCreated,
		)
		if err != nil {
			return err
		}
		err = s.basketRepo.ClearBasket(txCtx, userId, companyId)
		if err != nil {
			return err
		}
		err = s.sender.SendOrderCreated(txCtx, orderId)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil

}

func (s *Service) GetMyOrders(ctx context.Context, userId uuid.UUID) ([]domain.Order, error) {
	err := s.menuRepo.GetUser(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	orderRows, err := s.ordersRepo.GetUserOrders(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to get user orders: %w", err)
	}

	return ordersFromRows(orderRows), nil
}

func (s *Service) GetUserOrder(ctx context.Context, userId uuid.UUID, orderId uuid.UUID) (domain.Order, error) {
	err := s.menuRepo.GetUser(ctx, userId)
	if err != nil {
		return domain.Order{}, fmt.Errorf("failed to get user: %w", err)
	}

	orderRows, err := s.ordersRepo.GetOrder(ctx, orderId)
	if err != nil {
		return domain.Order{}, fmt.Errorf("failed to get order: %w", err)
	}

	order := ordersFromRows(orderRows)[0]
	if order.UserId != userId {
		return domain.Order{}, fmt.Errorf("this isn't your order: %w", core_errors.ErrForbidden)
	}

	return order, nil
}

func ordersFromRows(orderRows []domain.OrderRow) []domain.Order {
	orders := make([]domain.Order, 0)
	orderIndexes := make(map[uuid.UUID]int)
	for _, row := range orderRows {
		orderIndex, exists := orderIndexes[row.OrderId]
		if !exists {
			orderIndex = len(orders)
			orderIndexes[row.OrderId] = orderIndex
			orders = append(orders, domain.Order{
				Id:              row.OrderId,
				UserId:          row.UserId,
				CompanyId:       row.CompanyId,
				Status:          row.Status,
				RejectionReason: row.RejectionReason,
				Products:        make([]domain.OrderProduct, 0),
				CreatedAt:       row.CreatedAt,
				UpdatedAt:       row.UpdatedAt,
			})
		}
		orders[orderIndex].Products = append(orders[orderIndex].Products, row.Product)
		orders[orderIndex].FullPrice += row.Product.Price * int64(row.Product.Count)
	}

	return orders
}
