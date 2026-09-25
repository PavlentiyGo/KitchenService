package handler_companies

import (
	"avitoKitchen/internal/core/domain"
	"context"

	"github.com/google/uuid"
)
//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
type companiesService interface {
	AddNewProduct(ctx context.Context, companyId uuid.UUID, product domain.Product) (domain.Product, error)
	PatchCompanyProduct(
		ctx context.Context,
		companyId uuid.UUID,
		partnerProductId string,
		patch domain.ProductPatch,
	) (domain.Product, error)
	ChangeOrderStatus(
		ctx context.Context,
		companyId uuid.UUID,
		orderId uuid.UUID,
		status domain.OrderStatus,
	) (domain.Order, error)
}
