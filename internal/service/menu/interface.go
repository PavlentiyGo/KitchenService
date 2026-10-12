package service_menu

import (
	"context"
	"foodDelivery/internal/core/domain"

	"github.com/google/uuid"
)

//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
type menuRepository interface {
	GetCompany(ctx context.Context, companyId uuid.UUID) (domain.Company, error)
	GetUser(ctx context.Context, userId uuid.UUID) error
	GetCompanies(ctx context.Context) ([]domain.Company, error)
	GetCompanyProducts(ctx context.Context, companyId uuid.UUID) ([]domain.Product, error)
	GetCompanyProduct(ctx context.Context, productId uuid.UUID) (domain.Product, error)
}

type txManager interface {
	WithinTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
