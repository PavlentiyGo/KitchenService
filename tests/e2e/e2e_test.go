package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	core_config "foodDelivery/internal/core/config"
	"foodDelivery/internal/core/domain"
	http_middleware "foodDelivery/internal/core/http/middleware"
	core_server "foodDelivery/internal/core/http/server"
	core_logger "foodDelivery/internal/core/logger"
	"foodDelivery/internal/handlers"
	handler_basket "foodDelivery/internal/handlers/basket"
	handler_companies "foodDelivery/internal/handlers/companies"
	"foodDelivery/internal/handlers/dto"
	handler_menu "foodDelivery/internal/handlers/menu"
	handler_orders "foodDelivery/internal/handlers/orders"
	"foodDelivery/internal/repository/postgres"
	repository_basket "foodDelivery/internal/repository/postgres/basket"
	repository_companies "foodDelivery/internal/repository/postgres/companies"
	repository_menu "foodDelivery/internal/repository/postgres/menu"
	repository_orders "foodDelivery/internal/repository/postgres/orders"
	service_basket "foodDelivery/internal/service/basket"
	service_companies "foodDelivery/internal/service/companies"
	service_menu "foodDelivery/internal/service/menu"
	service_orders "foodDelivery/internal/service/orders"
	service_worker "foodDelivery/internal/service/worker"
	transport_dto "foodDelivery/internal/transport/http/company"
	transport_kafka "foodDelivery/internal/transport/kafka"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/go-playground/validator"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	segmentio_kafka "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcontainerkafka "github.com/testcontainers/testcontainers-go/modules/kafka"
	tcontainerpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"go.uber.org/zap"
)

const (
	demoCompanyId        = "11111111-1111-4111-8111-111111111111"
	demoUserId           = "22222222-2222-4222-8222-222222222222"
	demoProductId        = "33333333-3333-4333-8333-000000000001"
	demoPartnerProductId = "pizza-margherita"
)

var orderTopics = []string{
	"orders.created",
	"orders.retry.1s",
	"orders.retry.5s",
	"orders.retry.15s",
	"orders.dlq",
}

type testDeps struct {
	router         http.Handler
	logger         *core_logger.Logger
	receivedOrders <-chan transport_dto.OrderDto
}

func createKafkaTopics(ctx context.Context, broker string, topics []string) (resultErr error) {
	connection, err := segmentio_kafka.DialContext(ctx, "tcp", broker)
	if err != nil {
		return fmt.Errorf("connect to kafka broker: %w", err)
	}

	controller, err := connection.Controller()
	closeErr := connection.Close()
	if err != nil {
		return fmt.Errorf("get kafka controller: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("close kafka broker connection: %w", closeErr)
	}

	controllerAddress := net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port))
	controllerConnection, err := segmentio_kafka.DialContext(ctx, "tcp", controllerAddress)
	if err != nil {
		return fmt.Errorf("connect to kafka controller: %w", err)
	}
	defer func() {
		resultErr = errors.Join(resultErr, controllerConnection.Close())
	}()

	topicConfigs := make([]segmentio_kafka.TopicConfig, 0, len(topics))
	for _, topic := range topics {
		topicConfigs = append(topicConfigs, segmentio_kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		})
	}

	if err = controllerConnection.CreateTopics(topicConfigs...); err != nil {
		return fmt.Errorf("create kafka topics: %w", err)
	}

	return nil
}

func newKafkaConsumer(
	t *testing.T,
	brokers []string,
	topic string,
	groupSuffix string,
) *transport_kafka.Consumer {
	t.Helper()

	consumer, err := transport_kafka.NewConsumer(transport_kafka.ConsumerConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: "e2e-" + topic + "-" + groupSuffix,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, consumer.Close())
	})

	return consumer
}

func makeRequest(
	handler http.Handler,
	method string,
	target string,
	body []byte,
	headers map[string]string,
	logger *core_logger.Logger,
) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	for header, value := range headers {
		req.Header.Set(header, value)
	}

	ctx := core_logger.CtxWithLogger(req.Context(), logger)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req.WithContext(ctx))

	return recorder
}

func setupEnvironment(t *testing.T) *testDeps {
	t.Helper()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	const (
		postgresDb       = "test_db"
		postgresUser     = "test_user"
		postgresPassword = "test_password"
	)
	postgresContainer, err := tcontainerpostgres.Run(
		t.Context(),
		"postgres:16-alpine",
		tcontainerpostgres.WithDatabase(postgresDb),
		tcontainerpostgres.WithUsername(postgresUser),
		tcontainerpostgres.WithPassword(postgresPassword),
		tcontainerpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, postgresContainer)

	kafkaContainer, err := tcontainerkafka.Run(
		t.Context(),
		"confluentinc/confluent-local:7.5.0",
		tcontainerkafka.WithClusterID("food-delivery-e2e"),
	)
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, kafkaContainer)

	brokers, err := kafkaContainer.Brokers(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, brokers)
	require.NoError(t, createKafkaTopics(t.Context(), brokers[0], orderTopics))

	host, err := postgresContainer.Host(t.Context())
	require.NoError(t, err)

	port, err := postgresContainer.MappedPort(t.Context(), "5432/tcp")
	require.NoError(t, err)

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		postgresUser,
		postgresPassword,
		host,
		port.Port(),
		postgresDb,
	)

	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	migrationsPath := filepath.Join(filepath.Dir(filename), "../../migrations")

	migrator, err := migrate.New("file://"+migrationsPath, dsn)
	require.NoError(t, err)

	err = migrator.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		require.NoError(t, err)
	}

	sourceErr, databaseErr := migrator.Close()
	require.NoError(t, sourceErr)
	require.NoError(t, databaseErr)

	config := &core_config.Config{
		LoggerCfg: &core_config.LoggerConfig{
			Level:  "DEBUG",
			Folder: t.TempDir(),
		},
		WorkerCfg: &core_config.WorkerConfig{
			CompanyRequestTimeout: 2 * time.Second,
			RetryOperationDelay:   10 * time.Millisecond,
		},
		ServerCfg: &core_config.ServerConfig{
			ShutdownTimeout: 2 * time.Second,
		},
		KafkaCfg: &core_config.KafkaConfig{
			Brokers:           brokers,
			WriterMaxAttempts: 3,
			WriterTimeout:     5 * time.Second,
		},
		DbConfig: &core_config.DatabaseConfig{
			User:     postgresUser,
			Password: postgresPassword,
			Name:     postgresDb,
			Host:     host,
			Port:     port.Port(),
			Timeout:  5 * time.Second,
			SSLMode:  "disable",
		},
	}

	logger := &core_logger.Logger{Logger: zap.NewNop()}

	pool, err := postgres.NewPool(t.Context(), config.DbConfig)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	txManager := postgres.NewTxManager(pool, config.DbConfig.Timeout)
	basketRepo := repository_basket.NewRepository(txManager)
	companiesRepo := repository_companies.NewRepository(txManager)
	menuRepo := repository_menu.NewRepository(txManager)
	ordersRepo := repository_orders.NewRepository(txManager)

	sender, err := transport_kafka.NewSender(transport_kafka.SenderConfig{
		Brokers:      config.KafkaCfg.Brokers,
		MaxAttempts:  config.KafkaCfg.WriterMaxAttempts,
		WriteTimeout: config.KafkaCfg.WriterTimeout,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, sender.Close())
	})

	groupSuffix := uuid.NewString()
	createdConsumer := newKafkaConsumer(t, brokers, "orders.created", groupSuffix)
	oneSecondConsumer := newKafkaConsumer(t, brokers, "orders.retry.1s", groupSuffix)
	fiveSecondsConsumer := newKafkaConsumer(t, brokers, "orders.retry.5s", groupSuffix)
	fifteenSecondsConsumer := newKafkaConsumer(t, brokers, "orders.retry.15s", groupSuffix)
	cancellerConsumer := newKafkaConsumer(t, brokers, "orders.dlq", groupSuffix)

	basketService := service_basket.NewService(txManager, menuRepo, basketRepo)
	companiesService := service_companies.NewService(txManager, companiesRepo, ordersRepo)
	menuService := service_menu.NewService(txManager, menuRepo)
	ordersService := service_orders.NewService(
		txManager,
		ordersRepo,
		companiesRepo,
		sender,
		menuRepo,
		basketRepo,
	)

	httpValidator := validator.New()
	basketHandler := handler_basket.NewHandler(basketService, httpValidator)
	companiesHandler := handler_companies.NewHandler(companiesService, httpValidator)
	menuHandler := handler_menu.NewHandler(menuService, httpValidator)
	ordersHandler := handler_orders.NewHandler(ordersService, httpValidator)

	routerV1 := core_server.NewApiVersionRouter(core_server.Version1)
	routerV1.ChainRoutes(
		handlers.GetAllRoutes(
			basketHandler,
			menuHandler,
			companiesHandler,
			ordersHandler,
		)...,
	)

	server := core_server.NewServer(
		config.ServerCfg,
		[]*core_server.Router{routerV1},
		http_middleware.RequestId(),
		http_middleware.Logger(logger),
		http_middleware.Trace(),
		http_middleware.PanicRecoverer(),
	)

	apiServer := httptest.NewServer(server.Handler)
	t.Cleanup(apiServer.Close)

	receivedOrders := make(chan transport_dto.OrderDto, 1)
	companyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/orders" {
			http.NotFound(w, r)
			return
		}

		var order transport_dto.OrderDto
		if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
			http.Error(w, "invalid order", http.StatusBadRequest)
			return
		}

		select {
		case receivedOrders <- order:
		default:
		}

		statusBody, err := json.Marshal(dto.ChangeOrderStatusRequest{
			Status: string(domain.OrderStatusAccepted),
		})
		if err != nil {
			http.Error(w, "failed to marshal status", http.StatusInternalServerError)
			return
		}

		request, err := http.NewRequestWithContext(
			r.Context(),
			http.MethodPatch,
			fmt.Sprintf(
				"%s/api/v1/company/orders/%s/status",
				apiServer.URL,
				order.OrderId,
			),
			bytes.NewReader(statusBody),
		)
		if err != nil {
			http.Error(w, "failed to create status request", http.StatusInternalServerError)
			return
		}
		request.Header.Set("Authorization", demoCompanyId)
		request.Header.Set("Content-Type", "application/json")

		response, err := apiServer.Client().Do(request)
		if err != nil {
			http.Error(w, "failed to change order status", http.StatusBadGateway)
			return
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			http.Error(w, "status callback was rejected", http.StatusBadGateway)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(companyServer.Close)

	commandTag, err := pool.Exec(
		t.Context(),
		"UPDATE companies SET company_source_url = $1 WHERE id = $2",
		companyServer.URL,
		uuid.MustParse(demoCompanyId),
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), commandTag.RowsAffected())

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
		sender,
	)
	worker.Start(t.Context(), logger)

	return &testDeps{
		router:         server.Handler,
		logger:         logger,
		receivedOrders: receivedOrders,
	}
}

func waitForOrderStatus(
	ctx context.Context,
	deps *testDeps,
	orderId string,
	status domain.OrderStatus,
	timeout time.Duration,
) (dto.OrderDto, error) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	var lastStatus string
	for {
		response := makeRequest(
			deps.router,
			http.MethodGet,
			"/api/v1/orders/"+orderId,
			nil,
			map[string]string{"Authorization": demoUserId},
			deps.logger,
		)
		if response.Code == http.StatusOK {
			var order dto.OrderDto
			if err := json.NewDecoder(response.Body).Decode(&order); err == nil {
				lastStatus = order.Status
				if order.Status == string(status) {
					return order, nil
				}
			}
		}

		select {
		case <-ctx.Done():
			return dto.OrderDto{}, ctx.Err()
		case <-deadline.C:
			return dto.OrderDto{}, fmt.Errorf(
				"order %s did not reach status %s, last status: %s",
				orderId,
				status,
				lastStatus,
			)
		case <-ticker.C:
		}
	}
}

func TestE2E_OrderAcceptedByCompany(t *testing.T) {
	deps := setupEnvironment(t)

	response := makeRequest(
		deps.router,
		http.MethodGet,
		"/api/v1/companies",
		nil,
		nil,
		deps.logger,
	)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	var companiesResponse dto.GetCompaniesResponse
	err := json.NewDecoder(response.Body).Decode(&companiesResponse)
	require.NoError(t, err)
	require.NotEmpty(t, companiesResponse.Companies)

	var company dto.CompanyDto
	for _, currentCompany := range companiesResponse.Companies {
		if currentCompany.Id == demoCompanyId {
			company = currentCompany
			break
		}
	}
	require.Equal(t, demoCompanyId, company.Id)
	require.True(t, company.IsActive)

	response = makeRequest(
		deps.router,
		http.MethodGet,
		"/api/v1/companies/"+company.Id+"/products",
		nil,
		nil,
		deps.logger,
	)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	var productsResponse dto.GetCompanyProductsResponse
	err = json.NewDecoder(response.Body).Decode(&productsResponse)
	require.NoError(t, err)
	require.NotEmpty(t, productsResponse.Products)

	var menuProduct dto.ProductDto
	for _, product := range productsResponse.Products {
		if product.Id == demoProductId {
			menuProduct = product
			break
		}
	}
	require.Equal(t, demoProductId, menuProduct.Id)
	require.True(t, menuProduct.IsAvailable)

	response = makeRequest(
		deps.router,
		http.MethodGet,
		"/api/v1/products/"+menuProduct.Id,
		nil,
		nil,
		deps.logger,
	)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	var productResponse dto.ProductDto
	err = json.NewDecoder(response.Body).Decode(&productResponse)
	require.NoError(t, err)
	require.Equal(t, menuProduct, productResponse)

	userHeaders := map[string]string{
		"Authorization": demoUserId,
		"Content-Type":  "application/json",
	}
	putBasketBody, err := json.Marshal(dto.PutBasketProductRequest{Count: 1})
	require.NoError(t, err)

	response = makeRequest(
		deps.router,
		http.MethodPut,
		fmt.Sprintf(
			"/api/v1/companies/%s/basket/items/%s",
			company.Id,
			productResponse.Id,
		),
		putBasketBody,
		userHeaders,
		deps.logger,
	)
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())

	response = makeRequest(
		deps.router,
		http.MethodGet,
		"/api/v1/companies/"+company.Id+"/basket",
		nil,
		userHeaders,
		deps.logger,
	)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	var basketResponse dto.UserBasketDto
	err = json.NewDecoder(response.Body).Decode(&basketResponse)
	require.NoError(t, err)
	require.Len(t, basketResponse.Products, 1)
	require.Equal(t, productResponse.Id, basketResponse.Products[0].Id)
	require.Equal(t, 1, basketResponse.Products[0].ChosenCount)
	require.Equal(t, productResponse.Price, basketResponse.FullPrice)

	createOrderBody, err := json.Marshal(dto.CreateOrderRequest{
		Products: []dto.ProductPriceRequest{
			{
				Id:    productResponse.Id,
				Price: productResponse.Price,
			},
		},
	})
	require.NoError(t, err)

	response = makeRequest(
		deps.router,
		http.MethodPost,
		"/api/v1/companies/"+company.Id+"/orders",
		createOrderBody,
		userHeaders,
		deps.logger,
	)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	var receivedOrder transport_dto.OrderDto
	select {
	case receivedOrder = <-deps.receivedOrders:
	case <-time.After(10 * time.Second):
		require.FailNow(t, "company did not receive the order")
	}

	require.NotEmpty(t, receivedOrder.OrderId)
	require.Len(t, receivedOrder.Products, 1)
	require.Equal(t, demoPartnerProductId, receivedOrder.Products[0].ProductId)
	require.Equal(t, 1, receivedOrder.Products[0].Count)

	order, err := waitForOrderStatus(
		t.Context(),
		deps,
		receivedOrder.OrderId,
		domain.OrderStatusAccepted,
		10*time.Second,
	)
	require.NoError(t, err)
	require.Equal(t, receivedOrder.OrderId, order.Id)
	require.Equal(t, company.Id, order.CompanyId)
	require.Equal(t, string(domain.OrderStatusAccepted), order.Status)
	require.Len(t, order.Products, 1)
	require.Equal(t, productResponse.Id, order.Products[0].Id)
	require.Equal(t, productResponse.Name, order.Products[0].Name)
	require.Equal(t, productResponse.Price, order.Products[0].Price)
	require.Equal(t, 1, order.Products[0].Count)
	require.Equal(t, productResponse.Price, order.FullPrice)
}
