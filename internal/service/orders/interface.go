package service_orders

import (
	"avitoKitchen/internal/core/domain"
	"context"

	"github.com/google/uuid"
)

//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
type ordersRepository interface {
	GetUserOrders(ctx context.Context, userId uuid.UUID) ([]domain.OrderRow, error)
	GetOrder(ctx context.Context, orderId uuid.UUID) ([]domain.OrderRow, error)
	CreateOrder(ctx context.Context, userId uuid.UUID, companyId uuid.UUID) (uuid.UUID, error)
	CreateOrderItems(ctx context.Context, orderId uuid.UUID, products []domain.BasketProduct) error
	DecreaseProductsCount(ctx context.Context, companyId uuid.UUID, products []domain.BasketProduct) error
}

type companiesRepository interface {
	AddCompanyOrderStatusHistory(ctx context.Context, orderId uuid.UUID, status domain.OrderStatus) error
}

type ordersSender interface {
	SendOrderCreated(ctx context.Context, orderID uuid.UUID) error
}
type menuRepo interface {
	GetCompany(ctx context.Context, companyId uuid.UUID) (domain.Company, error)
	GetUser(ctx context.Context, userId uuid.UUID) error
}
type basketRepo interface {
	ConfirmBasketProductsPrice(
		ctx context.Context,
		userId uuid.UUID,
		companyId uuid.UUID,
		productIds []uuid.UUID,
	) error
	GetUserBasketProducts(
		ctx context.Context,
		userId uuid.UUID,
		companyId uuid.UUID,
	) ([]domain.BasketProduct, error)
	ClearBasket(ctx context.Context, userId uuid.UUID, companyId uuid.UUID) error
}

type txManager interface {
	WithinTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
