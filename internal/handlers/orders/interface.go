package handler_orders

import (
	"avitoKitchen/internal/core/domain"
	"context"

	"github.com/google/uuid"
)
//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
type ordersService interface {
	GetMyOrders(ctx context.Context, userId uuid.UUID) ([]domain.Order, error)
	GetUserOrder(ctx context.Context, userId uuid.UUID, orderId uuid.UUID) (domain.Order, error)
	CreateOrder(ctx context.Context, userId uuid.UUID, companyId uuid.UUID, basketItems []domain.Product) error
}
