package service_basket

import (
	"avitoKitchen/internal/core/domain"
	core_errors "avitoKitchen/internal/core/errors"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type Service struct {
	txManager  txManager
	menuRepo   menuRepository
	basketRepo basketRepository
}

func NewService(txManager txManager, menuRepo menuRepository, basketRepo basketRepository) *Service {
	return &Service{
		txManager:  txManager,
		menuRepo:   menuRepo,
		basketRepo: basketRepo,
	}
}

func (s *Service) GetBasket(ctx context.Context, userId uuid.UUID, companyId uuid.UUID) (domain.Basket, error) {
	_, err := s.menuRepo.GetCompany(ctx, companyId)
	if err != nil {
		return domain.Basket{}, fmt.Errorf("failed to get company: %w", err)
	}

	err = s.menuRepo.GetUser(ctx, userId)
	if err != nil {
		return domain.Basket{}, fmt.Errorf("failed to get user: %w", err)
	}

	products, err := s.basketRepo.GetUserBasketProducts(ctx, userId, companyId)
	if err != nil {
		return domain.Basket{}, fmt.Errorf("failed to get user basket: %w", err)
	}
	basket := domain.Basket{
		Products:           products,
		FullPrice:          0,
		FullConfirmedPrice: 0,
	}

	for i := 0; i < len(products); i++ {
		if products[i].IsAvailable {
			basket.FullPrice += products[i].Price * int64(products[i].ChosenCount)
			basket.FullConfirmedPrice += products[i].ConfirmedPrice * int64(products[i].ChosenCount)
		}
	}
	return basket, nil
}

func (s *Service) PutBasketProduct(
	ctx context.Context,
	userId uuid.UUID,
	companyId uuid.UUID,
	productId uuid.UUID,
	count int,
) error {
	if count <= 0 {
		return fmt.Errorf("count must be greater than zero: %w", core_errors.ErrInvalidRequest)
	}

	return s.txManager.WithinTx(ctx, func(txCtx context.Context) error {
		err := s.menuRepo.GetUser(txCtx, userId)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

		product, err := s.menuRepo.GetCompanyProduct(txCtx, productId)
		if err != nil {
			return fmt.Errorf("failed to get product: %w", err)
		}
		if product.CompanyId != companyId {
			return fmt.Errorf(
				"product %s does not belong to company %s: %w",
				productId,
				companyId,
				core_errors.ErrNotFound,
			)
		}
		if !product.IsAvailable {
			return fmt.Errorf("product %s: %w", productId, core_errors.ErrProductUnavailable)
		}
		if product.CurrentCount < count {
			return fmt.Errorf(
				"requested %d items, only %d available: %w",
				count,
				product.CurrentCount,
				core_errors.ErrInsufficientStock,
			)
		}

		err = s.basketRepo.PutBasketProduct(
			txCtx,
			userId,
			companyId,
			productId,
			count,
			product.Price,
		)
		if err != nil {
			return fmt.Errorf("failed to put product into basket: %w", err)
		}

		return nil
	})
}

func (s *Service) DeleteBasketProduct(ctx context.Context, userId uuid.UUID, companyId uuid.UUID, productId uuid.UUID) error {
	err := s.menuRepo.GetUser(ctx, userId)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	_, err = s.menuRepo.GetCompany(ctx, companyId)
	if err != nil {
		return fmt.Errorf("failed to get company: %w", err)
	}

	product, err := s.menuRepo.GetCompanyProduct(ctx, productId)
	if err != nil {
		return fmt.Errorf("failed to get product: %w", err)
	}
	if product.CompanyId != companyId {
		return fmt.Errorf(
			"product %s does not belong to company %s: %w",
			productId,
			companyId,
			core_errors.ErrNotFound,
		)
	}

	err = s.basketRepo.DeleteBasketProduct(ctx, userId, companyId, productId)
	if err != nil {
		return fmt.Errorf("failed to delete product from basket: %w", err)
	}

	return nil
}

func (s *Service) ConfirmBasketProducts(ctx context.Context, productsForChange []domain.Product, companyId uuid.UUID, userId uuid.UUID) ([]domain.BasketProduct, error) {
	if len(productsForChange) == 0 {
		return nil, fmt.Errorf("products for confirmation are empty: %w", core_errors.ErrInvalidRequest)
	}

	err := s.menuRepo.GetUser(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	_, err = s.menuRepo.GetCompany(ctx, companyId)
	if err != nil {
		return nil, fmt.Errorf("failed to get company: %w", err)
	}

	productIdsMap := make(map[uuid.UUID]domain.Product, len(productsForChange))
	productIds := make([]uuid.UUID, 0, len(productsForChange))
	for _, product := range productsForChange {
		if _, exists := productIdsMap[product.Id]; exists {
			return nil, fmt.Errorf(
				"product %s is duplicated: %w",
				product.Id,
				core_errors.ErrInvalidRequest,
			)
		}
		productIdsMap[product.Id] = product
		productIds = append(productIds, product.Id)
	}

	var basket []domain.BasketProduct
	err = s.txManager.WithinTx(ctx, func(txCtx context.Context) error {
		err := s.basketRepo.ConfirmBasketProductsPrice(
			txCtx,
			userId,
			companyId,
			productIds,
		)
		if err != nil {
			return fmt.Errorf("failed to update products prices: %w", err)
		}

		userBasket, err := s.basketRepo.GetUserBasketProducts(txCtx, userId, companyId)
		if err != nil {
			return fmt.Errorf("failed to get user basket: %w", err)
		}

		basketProductsById := make(map[uuid.UUID]domain.BasketProduct, len(userBasket))
		for _, product := range userBasket {
			basketProductsById[product.Id] = product
		}

		for _, confirmedProduct := range productsForChange {
			basketProduct, exists := basketProductsById[confirmedProduct.Id]
			if !exists {
				return fmt.Errorf(
					"product %s is not in user basket: %w",
					confirmedProduct.Id,
					core_errors.ErrNotFound,
				)
			}
			if basketProduct.ConfirmedPrice > confirmedProduct.Price {
				return fmt.Errorf(
					"product %s price changed from %d to %d: %w",
					confirmedProduct.Id,
					confirmedProduct.Price,
					basketProduct.ConfirmedPrice,
					core_errors.ErrConflict,
				)
			}
		}

		basket = userBasket
		return nil
	})
	if err != nil {
		return nil, err
	}

	return basket, nil
}
