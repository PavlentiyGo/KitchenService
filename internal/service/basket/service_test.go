package service_basket_test

import (
	service_basket "foodDelivery/internal/service/basket"
	"testing"

	"go.uber.org/mock/gomock"
)

type testDeps struct {
	service    *service_basket.Service
	txManager  *MocktxManager
	menuRepo   *MockmenuRepository
	basketRepo *MockbasketRepository
}

func newTestDeps(t *testing.T) *testDeps {
	t.Helper()

	ctrl := gomock.NewController(t)
	txManager := NewMocktxManager(ctrl)
	menuRepo := NewMockmenuRepository(ctrl)
	basketRepo := NewMockbasketRepository(ctrl)

	return &testDeps{
		service:    service_basket.NewService(txManager, menuRepo, basketRepo),
		txManager:  txManager,
		menuRepo:   menuRepo,
		basketRepo: basketRepo,
	}
}

type transactionContextKey struct{}
