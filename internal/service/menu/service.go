package service_menu

import (
	"avitoKitchen/internal/core/domain"
	"context"

	"github.com/google/uuid"
)

type Service struct {
	txManager txManager
	menuRepo  menuRepository
}

func NewService(txManager txManager, menuRepo menuRepository) *Service {
	return &Service{
		txManager: txManager,
		menuRepo:  menuRepo,
	}
}

func (s *Service) GetCompany(ctx context.Context, companyId uuid.UUID) (domain.Company, error) {
	return s.menuRepo.GetCompany(ctx, companyId)
}

func (s *Service) GetUser(ctx context.Context, userId uuid.UUID) error {
	return s.menuRepo.GetUser(ctx, userId)
}

func (s *Service) GetCompanies(ctx context.Context) ([]domain.Company, error) {
	return s.menuRepo.GetCompanies(ctx)
}

func (s *Service) GetCompanyProducts(ctx context.Context, companyId uuid.UUID) ([]domain.Product, error) {
	return s.menuRepo.GetCompanyProducts(ctx, companyId)
}

func (s *Service) GetCompanyProduct(ctx context.Context, productId uuid.UUID) (domain.Product, error) {
	return s.menuRepo.GetCompanyProduct(ctx, productId)
}
