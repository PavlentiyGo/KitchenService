package handler_basket

import (
	"context"
	"foodDelivery/internal/core/domain"

	"github.com/google/uuid"
)

//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
type basketService interface {
	GetBasket(ctx context.Context, userId uuid.UUID, companyId uuid.UUID) (domain.Basket, error)
	PutBasketProduct(
		ctx context.Context,
		userId uuid.UUID,
		companyId uuid.UUID,
		productId uuid.UUID,
		count int,
	) error
	DeleteBasketProduct(ctx context.Context, userId uuid.UUID, companyId uuid.UUID, productId uuid.UUID) error
	ConfirmBasketProducts(
		ctx context.Context,
		productsForChange []domain.Product,
		companyId uuid.UUID,
		userId uuid.UUID,
	) ([]domain.BasketProduct, error)
}
