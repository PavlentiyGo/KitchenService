package service_companies

import (
	"avitoKitchen/internal/core/domain"
	"context"
	"time"

	"github.com/google/uuid"
)

//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
type companiesRepository interface {
	CreateProduct(ctx context.Context, companyId uuid.UUID, product domain.Product) (domain.Product, error)
	PatchProduct(
		ctx context.Context,
		companyId uuid.UUID,
		partnerProductId string,
		patch domain.ProductPatch,
	) (domain.Product, error)
	UpdateCompanyOrderStatus(
		ctx context.Context,
		companyId uuid.UUID,
		orderId uuid.UUID,
		status domain.OrderStatus,
	) (time.Time, error)
	AddCompanyOrderStatusHistory(ctx context.Context, orderId uuid.UUID, status domain.OrderStatus) error
}

type ordersRepository interface {
	GetOrder(ctx context.Context, orderId uuid.UUID) ([]domain.OrderRow, error)
}

type txManager interface {
	WithinTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
