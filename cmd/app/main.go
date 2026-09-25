package main

import (
	core_config "avitoKitchen/internal/core/config"
	http_middleware "avitoKitchen/internal/core/http/middleware"
	core_server "avitoKitchen/internal/core/http/server"
	core_logger "avitoKitchen/internal/core/logger"
	"avitoKitchen/internal/handlers"
	handler_basket "avitoKitchen/internal/handlers/basket"
	handler_companies "avitoKitchen/internal/handlers/companies"
	handler_menu "avitoKitchen/internal/handlers/menu"
	handler_orders "avitoKitchen/internal/handlers/orders"
	"avitoKitchen/internal/repository/postgres"
	repository_basket "avitoKitchen/internal/repository/postgres/basket"
	repository_companies "avitoKitchen/internal/repository/postgres/companies"
	repository_menu "avitoKitchen/internal/repository/postgres/menu"
	repository_orders "avitoKitchen/internal/repository/postgres/orders"
	service_basket "avitoKitchen/internal/service/basket"
	service_companies "avitoKitchen/internal/service/companies"
	service_menu "avitoKitchen/internal/service/menu"
	service_orders "avitoKitchen/internal/service/orders"
	service_worker "avitoKitchen/internal/service/worker"
	transport_kafka "avitoKitchen/internal/transport/kafka"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "avitoKitchen/docs"

	"github.com/go-playground/validator"
	"go.uber.org/zap"
)

// @title Avito Kitchen API
// @version 1.0
// @description API сервиса Avito Kitchen.
func main() {
	time.Local = time.UTC
	config := core_config.NewConfigMust()

	logger, err := core_logger.NewLogger(config.LoggerCfg)
	if err != nil {
		log.Fatal("failed to create new logger", err)
	}
	defer func() {
		closeErr := logger.Close()
		if closeErr != nil {
			log.Printf("failed to close logger: %v", closeErr)
		}
	}()
	parentCtx := context.Background()
	shutdownCtx := parentCtx
	var shutdownCancel context.CancelFunc = func() {}
	defer func() {
		shutdownCancel()
	}()

	ctx, cancel := signal.NotifyContext(parentCtx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger.Debug("creating pool")
	pool, err := postgres.NewPool(ctx, config.DbConfig)
	if err != nil {
		logger.Error("failed to create pool", zap.Error(err))
		os.Exit(1)
	}

	logger.Debug("creating txManager")
	txManager := postgres.NewTxManager(pool, config.DbConfig.Timeout)

	logger.Debug("creating repositories")
	basketRepo := repository_basket.NewRepository(txManager)
	companiesRepo := repository_companies.NewRepository(txManager)
	menuRepo := repository_menu.NewRepository(txManager)
	ordersRepo := repository_orders.NewRepository(txManager)
	logger.Debug("creating worker")

	createdConsumer, err := transport_kafka.NewConsumer(transport_kafka.ConsumerConfig{
		Brokers: config.KafkaCfg.Brokers,
		Topic:   "orders.created",
		GroupID: "order-sender-created",
	})
	if err != nil {
		logger.Error("failed to create orders.created consumer", zap.Error(err))
		os.Exit(1)
	}
	defer func() {
		closeConsumer(shutdownCtx, "orders.created", createdConsumer.Close, logger)
	}()

	oneSecondConsumer, err := transport_kafka.NewConsumer(transport_kafka.ConsumerConfig{
		Brokers: config.KafkaCfg.Brokers,
		Topic:   "orders.retry.1s",
		GroupID: "order-sender-retry-1s",
	})
	if err != nil {
		logger.Error("failed to create orders.retry.1s consumer", zap.Error(err))
		os.Exit(1)
	}
	defer func() {
		closeConsumer(shutdownCtx, "orders.retry.1s", oneSecondConsumer.Close, logger)
	}()

	fiveSecondsConsumer, err := transport_kafka.NewConsumer(transport_kafka.ConsumerConfig{
		Brokers: config.KafkaCfg.Brokers,
		Topic:   "orders.retry.5s",
		GroupID: "order-sender-retry-5s",
	})
	if err != nil {
		logger.Error("failed to create orders.retry.5s consumer", zap.Error(err))
		os.Exit(1)
	}
	defer func() {
		closeConsumer(shutdownCtx, "orders.retry.5s", fiveSecondsConsumer.Close, logger)
	}()

	fifteenSecondsConsumer, err := transport_kafka.NewConsumer(transport_kafka.ConsumerConfig{
		Brokers: config.KafkaCfg.Brokers,
		Topic:   "orders.retry.15s",
		GroupID: "order-sender-retry-15s",
	})
	if err != nil {
		logger.Error("failed to create orders.retry.15s consumer", zap.Error(err))
		os.Exit(1)
	}
	defer func() {
		closeConsumer(shutdownCtx, "orders.retry.15s", fifteenSecondsConsumer.Close, logger)
	}()

	cancellerConsumer, err := transport_kafka.NewConsumer(transport_kafka.ConsumerConfig{
		Brokers: config.KafkaCfg.Brokers,
		Topic:   "orders.dlq",
		GroupID: "order-canceller-dlq",
	})
	if err != nil {
		logger.Error("failed to create orders.dlq consumer", zap.Error(err))
		os.Exit(1)
	}
	defer func() {
		closeConsumer(shutdownCtx, "orders.dlq", cancellerConsumer.Close, logger)
	}()

	writer, err := transport_kafka.NewSender(transport_kafka.SenderConfig{
		Brokers:      config.KafkaCfg.Brokers,
		MaxAttempts:  config.KafkaCfg.WriterMaxAttempts,
		WriteTimeout: config.KafkaCfg.WriterTimeout,
	})
	if err != nil {
		logger.Error("failed to create kafka writer", zap.Error(err))
		os.Exit(1)
	}
	defer func() {
		closeErr := writer.Close()
		if closeErr != nil {
			logger.Error("failed to close kafka writer", zap.Error(closeErr))
		}
	}()

	worker := service_worker.NewWorker(
		config.WorkerCfg,
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
	)
	worker.Start(ctx, logger)

	logger.Debug("creating services")
	basketService := service_basket.NewService(txManager, menuRepo, basketRepo)
	companiesService := service_companies.NewService(txManager, companiesRepo, ordersRepo)
	menuService := service_menu.NewService(txManager, menuRepo)
	ordersService := service_orders.NewService(txManager, ordersRepo, companiesRepo, writer, menuRepo, basketRepo)

	logger.Debug("creating handlers")
	httpValidator := validator.New()
	basketHandler := handler_basket.NewHandler(basketService, httpValidator)
	menuHandler := handler_menu.NewHandler(menuService, httpValidator)
	companiesHandler := handler_companies.NewHandler(companiesService, httpValidator)
	ordersHandler := handler_orders.NewHandler(ordersService, httpValidator)
	allRoutes := handlers.GetAllRoutes(basketHandler, menuHandler, companiesHandler, ordersHandler)

	routerV1 := core_server.NewApiVersionRouter(core_server.Version1)
	routerV1.ChainRoutes(allRoutes...)

	routers := []*core_server.Router{routerV1}

	server := core_server.NewServer(
		config.ServerCfg,
		routers,
		http_middleware.RequestId(),
		http_middleware.Logger(logger),
		http_middleware.Trace(),
		http_middleware.PanicRecoverer(),
	)
	logger.Debug("starting server")
	go func() {
		serveErr := server.ListenAndServe()
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("failed to start server", zap.Error(serveErr))
		}
	}()
	<-ctx.Done()
	logger.Debug("closing server")
	shutdownCtx, shutdownCancel = context.WithTimeout(parentCtx, config.ServerCfg.ShutdownTimeout)
	err = server.Shutdown(shutdownCtx)
	if err != nil {
		logger.Error("failed to shutdown server properly", zap.Error(err))
	} else {
		logger.Debug("server was closed properly")
	}
	err = postgres.PoolShutdown(shutdownCtx, pool)
	if err != nil {
		logger.Error("failed to shutdown pool properly", zap.Error(err))
	} else {
		logger.Debug("pool was closed properly")
	}
}

func closeConsumer(ctx context.Context, topic string, closeFn func() error, logger *core_logger.Logger) {
	closed := make(chan error, 1)
	go func() {
		closed <- closeFn()
	}()

	select {
	case err := <-closed:
		if err != nil {
			logger.Error("failed to close consumer", zap.String("topic", topic), zap.Error(err))
			return
		}
		logger.Debug("consumer was closed properly", zap.String("topic", topic))
	case <-ctx.Done():
		logger.Error("consumer shutdown timed out", zap.String("topic", topic), zap.Error(ctx.Err()))
	}
}
