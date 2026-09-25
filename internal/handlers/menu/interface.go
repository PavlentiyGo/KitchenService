package handler_menu

import (
	"avitoKitchen/internal/core/domain"
	"context"

	"github.com/google/uuid"
)

//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
type menuService interface {
	GetCompanies(ctx context.Context) ([]domain.Company, error)
	GetCompanyProducts(ctx context.Context, companyId uuid.UUID) ([]domain.Product, error)
	GetCompanyProduct(ctx context.Context, productId uuid.UUID) (domain.Product, error)
}
