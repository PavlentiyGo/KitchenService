package service_worker

import (
	"avitoKitchen/internal/core/domain"
	core_messaging "avitoKitchen/internal/core/messaging"
	"context"

	"github.com/google/uuid"
)

//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
type consumer interface {
	ReadMessage(ctx context.Context) (core_messaging.Message, error)
	CommitMessage(ctx context.Context, message core_messaging.Message) error
}

type writer interface {
	SendToNextTopic(ctx context.Context, nextTopic string, message core_messaging.Message) error
}
type ordersRepo interface {
	GetOrderForSend(ctx context.Context, orderID uuid.UUID) (domain.OrderForSend, error)
	CancelOrder(ctx context.Context, orderId uuid.UUID, cancelReason core_messaging.Reason) error
	RestoreOrderProducts(ctx context.Context, orderId uuid.UUID) error
}
type companiesRepo interface {
	AddCompanyOrderStatusHistory(ctx context.Context, orderId uuid.UUID, status domain.OrderStatus) error
}
type menuRepo interface {
	GetCompany(ctx context.Context, companyId uuid.UUID) (domain.Company, error)
}

type txManager interface {
	WithinTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
