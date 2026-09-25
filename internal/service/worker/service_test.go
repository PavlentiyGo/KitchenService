package service_worker_test

import (
	core_config "avitoKitchen/internal/core/config"
	"avitoKitchen/internal/core/domain"
	core_logger "avitoKitchen/internal/core/logger"
	core_messaging "avitoKitchen/internal/core/messaging"
	service_worker "avitoKitchen/internal/service/worker"
	transport_dto "avitoKitchen/internal/transport/http/company"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

type testDeps struct {
	worker                 *service_worker.Worker
	txManager              *MocktxManager
	ordersRepo             *MockordersRepo
	companiesRepo          *MockcompaniesRepo
	createdConsumer        *Mockconsumer
	oneSecondConsumer      *Mockconsumer
	fiveSecondsConsumer    *Mockconsumer
	fifteenSecondsConsumer *Mockconsumer
	cancellerConsumer      *Mockconsumer
}

func newTestDeps(t *testing.T) *testDeps {
	t.Helper()

	ctrl := gomock.NewController(t)
	config := &core_config.WorkerConfig{
		CompanyRequestTimeout: time.Second,
		RetryOperationDelay:   time.Millisecond,
	}
	txManager := NewMocktxManager(ctrl)
	ordersRepo := NewMockordersRepo(ctrl)
	companiesRepo := NewMockcompaniesRepo(ctrl)
	menuRepo := NewMockmenuRepo(ctrl)
	createdConsumer := NewMockconsumer(ctrl)
	oneSecondConsumer := NewMockconsumer(ctrl)
	fiveSecondsConsumer := NewMockconsumer(ctrl)
	fifteenSecondsConsumer := NewMockconsumer(ctrl)
	cancellerConsumer := NewMockconsumer(ctrl)
	writer := NewMockwriter(ctrl)

	return &testDeps{
		worker: service_worker.NewWorker(
			config,
			txManager,
			ordersRepo,
			companiesRepo,
			menuRepo,
			createdConsumer,
			oneSecondConsumer,
			fiveSecondsConsumer,
			fifteenSecondsConsumer,
			cancellerConsumer,
			writer,
		),
		txManager:              txManager,
		ordersRepo:             ordersRepo,
		companiesRepo:          companiesRepo,
		createdConsumer:        createdConsumer,
		oneSecondConsumer:      oneSecondConsumer,
		fiveSecondsConsumer:    fiveSecondsConsumer,
		fifteenSecondsConsumer: fifteenSecondsConsumer,
		cancellerConsumer:      cancellerConsumer,
	}
}

type transactionContextKey struct{}

type receivedRequest struct {
	method      string
	path        string
	contentType string
	body        transport_dto.OrderDto
	err         error
}

func TestWorker_SendsCreatedOrderAndCommitsMessage(t *testing.T) {
	received := make(chan receivedRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body transport_dto.OrderDto
		err := json.NewDecoder(request.Body).Decode(&body)
		received <- receivedRequest{
			method:      request.Method,
			path:        request.URL.Path,
			contentType: request.Header.Get("Content-Type"),
			body:        body,
			err:         err,
		}
		writer.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	deps := newTestDeps(t)
	ctx, cancel := context.WithCancel(context.Background())
	orderId := uuid.MustParse("44444444-4444-4444-8444-444444444444")
	order := domain.OrderForSend{
		OrderId:    orderId,
		CompanyUrl: server.URL,
		Products: []domain.ProductForSend{
			{PartnerProductId: "pizza-margherita", Count: 2},
			{PartnerProductId: "lemonade", Count: 1},
		},
	}
	eventValue, err := json.Marshal(core_messaging.OrderCreatedEvent{OrderID: orderId})
	require.NoError(t, err)
	message := core_messaging.Message{
		Topic: "orders.created", Partition: 1, Offset: 42, Key: []byte(orderId.String()), Value: eventValue,
	}
	committed := make(chan struct{}, 1)

	var consumersStopped sync.WaitGroup
	consumersStopped.Add(5)
	blockUntilCanceled := func(ctx context.Context) (core_messaging.Message, error) {
		defer consumersStopped.Done()
		<-ctx.Done()
		return core_messaging.Message{}, ctx.Err()
	}

	deps.createdConsumer.EXPECT().ReadMessage(gomock.Any()).Return(message, nil)
	deps.createdConsumer.EXPECT().ReadMessage(gomock.Any()).DoAndReturn(blockUntilCanceled)
	deps.oneSecondConsumer.EXPECT().ReadMessage(gomock.Any()).DoAndReturn(blockUntilCanceled)
	deps.fiveSecondsConsumer.EXPECT().ReadMessage(gomock.Any()).DoAndReturn(blockUntilCanceled)
	deps.fifteenSecondsConsumer.EXPECT().ReadMessage(gomock.Any()).DoAndReturn(blockUntilCanceled)
	deps.cancellerConsumer.EXPECT().ReadMessage(gomock.Any()).DoAndReturn(blockUntilCanceled)
	deps.ordersRepo.EXPECT().GetOrderForSend(gomock.Any(), orderId).Return(order, nil)
	deps.createdConsumer.
		EXPECT().
		CommitMessage(gomock.Any(), message).
		DoAndReturn(func(context.Context, core_messaging.Message) error {
			committed <- struct{}{}
			return nil
		})

	deps.worker.Start(ctx, &core_logger.Logger{Logger: zap.NewNop()})
	committedInTime := waitForSignal(committed)
	cancel()
	waitForConsumers(t, &consumersStopped)
	require.True(t, committedInTime, "worker did not commit the delivered message")

	request := <-received
	require.NoError(t, request.err)
	assert.Equal(t, http.MethodPost, request.method)
	assert.Equal(t, "/orders", request.path)
	assert.Equal(t, "application/json", request.contentType)
	assert.Equal(t, transport_dto.OrderDto{
		OrderId: orderId.String(),
		Products: []transport_dto.ProductDto{
			{ProductId: "pizza-margherita", Count: 2},
			{ProductId: "lemonade", Count: 1},
		},
	}, request.body)
}

func TestWorker_CancelsOrderFromDLQAndCommitsMessage(t *testing.T) {
	deps := newTestDeps(t)
	ctx, cancel := context.WithCancel(context.Background())
	txCtx := context.WithValue(ctx, transactionContextKey{}, true)
	orderId := uuid.MustParse("44444444-4444-4444-8444-444444444444")
	event := core_messaging.OrderCreatedEvent{
		OrderID:     orderId,
		RetryReason: core_messaging.RetryReasonCompanyRejected,
	}
	eventValue, err := json.Marshal(event)
	require.NoError(t, err)
	message := core_messaging.Message{
		Topic: "orders.dlq", Partition: 2, Offset: 84, Key: []byte(orderId.String()), Value: eventValue,
	}
	committed := make(chan struct{}, 1)

	var consumersStopped sync.WaitGroup
	consumersStopped.Add(5)
	blockUntilCanceled := func(ctx context.Context) (core_messaging.Message, error) {
		defer consumersStopped.Done()
		<-ctx.Done()
		return core_messaging.Message{}, ctx.Err()
	}

	deps.createdConsumer.EXPECT().ReadMessage(gomock.Any()).DoAndReturn(blockUntilCanceled)
	deps.oneSecondConsumer.EXPECT().ReadMessage(gomock.Any()).DoAndReturn(blockUntilCanceled)
	deps.fiveSecondsConsumer.EXPECT().ReadMessage(gomock.Any()).DoAndReturn(blockUntilCanceled)
	deps.fifteenSecondsConsumer.EXPECT().ReadMessage(gomock.Any()).DoAndReturn(blockUntilCanceled)
	deps.cancellerConsumer.EXPECT().ReadMessage(gomock.Any()).Return(message, nil)
	deps.cancellerConsumer.EXPECT().ReadMessage(gomock.Any()).DoAndReturn(blockUntilCanceled)
	deps.txManager.
		EXPECT().
		WithinTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
			return fn(txCtx)
		})
	deps.ordersRepo.
		EXPECT().
		CancelOrder(txCtx, orderId, core_messaging.RetryReasonCompanyRejected).
		Return(nil)
	deps.ordersRepo.
		EXPECT().
		RestoreOrderProducts(txCtx, orderId).
		Return(nil)
	deps.companiesRepo.
		EXPECT().
		AddCompanyOrderStatusHistory(txCtx, orderId, domain.OrderStatusRejected).
		Return(nil)
	deps.cancellerConsumer.
		EXPECT().
		CommitMessage(gomock.Any(), message).
		DoAndReturn(func(context.Context, core_messaging.Message) error {
			committed <- struct{}{}
			return nil
		})

	deps.worker.Start(ctx, &core_logger.Logger{Logger: zap.NewNop()})
	committedInTime := waitForSignal(committed)
	cancel()
	waitForConsumers(t, &consumersStopped)
	require.True(t, committedInTime, "worker did not commit the cancelled order message")
}

func waitForSignal(signal <-chan struct{}) bool {
	select {
	case <-signal:
		return true
	case <-time.After(2 * time.Second):
		return false
	}
}

func waitForConsumers(t *testing.T, consumersStopped *sync.WaitGroup) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		consumersStopped.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker consumers did not stop after context cancellation")
	}
}
