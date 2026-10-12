package service_basket

import (
	"context"
	"foodDelivery/internal/core/domain"

	"github.com/google/uuid"
)

//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
type menuRepository interface {
	GetCompany(ctx context.Context, companyId uuid.UUID) (domain.Company, error)
	GetCompanyProduct(ctx context.Context, productId uuid.UUID) (domain.Product, error)
	GetUser(ctx context.Context, userId uuid.UUID) error
}
type basketRepository interface {
	GetUserBasketProducts(ctx context.Context, userId uuid.UUID, companyId uuid.UUID) ([]domain.BasketProduct, error)
	PutBasketProduct(
		ctx context.Context,
		userId uuid.UUID,
		companyId uuid.UUID,
		productId uuid.UUID,
		count int,
		confirmedPrice int64,
	) error
	DeleteBasketProduct(ctx context.Context, userId uuid.UUID, companyId uuid.UUID, productId uuid.UUID) error
	ConfirmBasketProductsPrice(
		ctx context.Context,
		userId uuid.UUID,
		companyId uuid.UUID,
		productIds []uuid.UUID,
	) error
}
type txManager interface {
	WithinTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
