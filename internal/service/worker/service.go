package service_worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	core_config "foodDelivery/internal/core/config"
	"foodDelivery/internal/core/domain"
	core_errors "foodDelivery/internal/core/errors"
	core_logger "foodDelivery/internal/core/logger"
	core_messaging "foodDelivery/internal/core/messaging"
	transport_dto "foodDelivery/internal/transport/http/company"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type Worker struct {
	config                 *core_config.WorkerConfig
	txManager              txManager
	ordersRepo             ordersRepo
	companiesRepo          companiesRepo
	menuRepo               menuRepo
	createdConsumer        consumer
	oneSecondConsumer      consumer
	fiveSecondsConsumer    consumer
	fifteenSecondsConsumer consumer
	cancellerConsumer      consumer
	writer                 writer
}

func NewWorker(
	config *core_config.WorkerConfig,
	txManager txManager,
	ordersRepo ordersRepo,
	companiesRepo companiesRepo,
	menuRepo menuRepo,
	createdConsumer consumer,
	oneSecondConsumer consumer,
	fiveSecondsConsumer consumer,
	fifteenSecondsConsumer consumer,
	cancellerConsumer consumer,
	writer writer,
) *Worker {
	return &Worker{
		config:                 config,
		txManager:              txManager,
		ordersRepo:             ordersRepo,
		companiesRepo:          companiesRepo,
		menuRepo:               menuRepo,
		createdConsumer:        createdConsumer,
		oneSecondConsumer:      oneSecondConsumer,
		fiveSecondsConsumer:    fiveSecondsConsumer,
		fifteenSecondsConsumer: fifteenSecondsConsumer,
		cancellerConsumer:      cancellerConsumer,
		writer:                 writer,
	}
}

var topicMapper = map[string]string{
	"orders.created":   "orders.retry.1s",
	"orders.retry.1s":  "orders.retry.5s",
	"orders.retry.5s":  "orders.retry.15s",
	"orders.retry.15s": "orders.dlq",
}

var retryDelayByTopic = map[string]time.Duration{
	"orders.retry.1s":  time.Second,
	"orders.retry.5s":  5 * time.Second,
	"orders.retry.15s": 15 * time.Second,
}

const ordersDLQTopic = "orders.dlq"

func (w *Worker) Start(ctx context.Context, logger *core_logger.Logger) {
	go w.consume(ctx, w.createdConsumer, logger.With(zap.String("topic", "orders.created")))
	go w.consume(ctx, w.oneSecondConsumer, logger.With(zap.String("topic", "orders.retry.1s")))
	go w.consume(ctx, w.fiveSecondsConsumer, logger.With(zap.String("topic", "orders.retry.5s")))
	go w.consume(ctx, w.fifteenSecondsConsumer, logger.With(zap.String("topic", "orders.retry.15s")))
	go w.cancelConsume(ctx, w.cancellerConsumer, logger.With(zap.String("topic", "orders.dlq")))
}

func (w *Worker) cancelConsume(ctx context.Context, canceller consumer, logger *core_logger.Logger) {
	for {
		message, err := canceller.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Error("failed to read message", zap.Error(err))
			continue
		}
		var event core_messaging.OrderCreatedEvent
		err = json.Unmarshal(message.Value, &event)
		if err != nil {
			logger.Error(
				"discarding invalid event from dlq topic",
				zap.Int("partition", message.Partition),
				zap.Int64("offset", message.Offset),
				zap.ByteString("key", message.Key),
				zap.Int("value_size", len(message.Value)),
				zap.Error(err),
			)
			if !w.commitMessage(ctx, canceller, message, logger) {
				return
			}
			continue
		}
		for {
			err = w.txManager.WithinTx(ctx, func(txCtx context.Context) error {
				err = w.ordersRepo.CancelOrder(txCtx, event.OrderID, event.RetryReason)
				if err != nil {
					return err
				}
				err = w.ordersRepo.RestoreOrderProducts(txCtx, event.OrderID)
				if err != nil {
					return err
				}
				err = w.companiesRepo.AddCompanyOrderStatusHistory(
					txCtx,
					event.OrderID,
					domain.OrderStatusRejected,
				)
				if err != nil {
					return err
				}
				return nil
			})
			if err == nil {
				logger.Debug("order was cancelled", zap.String("orderId", event.OrderID.String()))
				break
			} else if errors.Is(err, core_errors.ErrNotFound) {
				logger.Debug("no order in db", zap.String("orderId", event.OrderID.String()))
				break
			}
			logger.Error("failed to cancel order", zap.Error(err))
			err = waitUntil(ctx, time.Now().Add(w.config.RetryOperationDelay))
			if err != nil {
				logger.Error("failed to retry operation delay", zap.Error(err))
				return
			}
		}
		if !w.commitMessage(ctx, canceller, message, logger) {
			return
		}
	}
}

func (w *Worker) consume(ctx context.Context, consumer consumer, logger *core_logger.Logger) {
	for {
		message, err := consumer.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Error("failed to read message", zap.Error(err))
			continue
		}

		var event core_messaging.OrderCreatedEvent
		err = json.Unmarshal(message.Value, &event)
		if err != nil {
			logger.Error(
				"discarding invalid order event",
				zap.Int("partition", message.Partition),
				zap.Int64("offset", message.Offset),
				zap.ByteString("key", message.Key),
				zap.Int("value_size", len(message.Value)),
				zap.Error(err),
			)
			if !w.commitMessage(ctx, consumer, message, logger) {
				return
			}
			continue
		}

		retryAt := event.RetryAt
		if retryAt.IsZero() {
			if delay, ok := retryDelayByTopic[message.Topic]; ok {
				retryAt = message.Time.Add(delay)
			}
		}
		err = waitUntil(ctx, retryAt)
		if err != nil {
			return
		}

		if event.Order == nil {
			order, getOrderErr := w.ordersRepo.GetOrderForSend(ctx, event.OrderID)
			if getOrderErr != nil {
				logger.Error(fmt.Sprintf("failed to get order with id: %s", event.OrderID.String()), zap.Error(getOrderErr))
				w.scheduleRetry(
					ctx,
					consumer,
					message,
					&event,
					topicMapper[message.Topic],
					core_messaging.RetryReasonOrderNotFound,
					logger,
				)
				continue
			}
			event.Order = &order
		}

		body := transport_dto.OrderDtoFromDomain(*event.Order)
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			logger.Error("failed to marshal order to company", zap.Error(err))
			continue
		}
		ctxWithTimeoutForRequest, cancel := context.WithTimeout(ctx, w.config.CompanyRequestTimeout)
		req, err := http.NewRequestWithContext(
			ctxWithTimeoutForRequest,
			http.MethodPost,
			event.Order.CompanyUrl+"/orders",
			bytes.NewBuffer(bodyJSON),
		)
		if err != nil {
			cancel()
			logger.Error("failed to create order request to company", zap.Error(err))
			w.scheduleRetry(
				ctx,
				consumer,
				message,
				&event,
				ordersDLQTopic,
				core_messaging.RetryReasonInvalidCompanyURL,
				logger,
			)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		response, err := http.DefaultClient.Do(req)
		if err != nil {
			cancel()
			logger.Error("failed to send order to company", zap.Error(err))
			w.scheduleRetry(
				ctx,
				consumer,
				message,
				&event,
				topicMapper[message.Topic],
				core_messaging.RetryReasonCompanyUnavailable,
				logger,
			)
			continue
		}
		closeErr := response.Body.Close()
		if closeErr != nil {
			logger.Warn("failed to close company response body", zap.Error(closeErr))
		}
		cancel()

		switch {
		case response.StatusCode == http.StatusOK || response.StatusCode == http.StatusCreated:
			if !w.commitMessage(ctx, consumer, message, logger) {
				return
			}
		case response.StatusCode >= 500 &&
			response.StatusCode < 600:
			logger.Error(
				"company server error",
				zap.Int("status_code", response.StatusCode),
			)
			w.scheduleRetry(
				ctx,
				consumer,
				message,
				&event,
				topicMapper[message.Topic],
				core_messaging.RetryReasonCompanyUnavailable,
				logger,
			)
		default:
			logger.Error(
				"company rejected order",
				zap.Int("status_code", response.StatusCode),
			)
			w.scheduleRetry(
				ctx,
				consumer,
				message,
				&event,
				ordersDLQTopic,
				core_messaging.RetryReasonCompanyRejected,
				logger,
			)
		}
	}
}

func (w *Worker) commitMessage(ctx context.Context, consumer consumer, message core_messaging.Message, logger *core_logger.Logger) bool {
	for {
		err := consumer.CommitMessage(ctx, message)
		if err == nil {
			return true
		}

		logger.Error(
			"failed to commit message",
			zap.Int("partition", message.Partition),
			zap.Int64("offset", message.Offset),
			zap.Error(err),
		)
		if waitUntil(ctx, time.Now().Add(w.config.RetryOperationDelay)) != nil {
			return false
		}
	}
}

func waitUntil(ctx context.Context, deadline time.Time) error {
	if deadline.IsZero() {
		return nil
	}

	delay := time.Until(deadline)
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Worker) scheduleRetry(
	ctx context.Context,
	consumer consumer,
	message core_messaging.Message,
	event *core_messaging.OrderCreatedEvent,
	nextTopic string,
	reason core_messaging.Reason,
	logger *core_logger.Logger,
) {
	if nextTopic == "" {
		logger.Error(
			"retry topic is empty",
			zap.String("source_topic", message.Topic),
		)
		return
	}

	event.RetryReason = reason
	if delay, ok := retryDelayByTopic[nextTopic]; ok {
		event.RetryAt = time.Now().UTC().Add(delay)
	} else {
		event.RetryAt = time.Time{}
	}

	messageValue, err := json.Marshal(event)
	if err != nil {
		logger.Error("failed to marshal retry event", zap.Error(err))
		return
	}
	message.Value = messageValue
	w.forwardMessage(ctx, consumer, message, nextTopic, logger)
}

func (w *Worker) forwardMessage(
	ctx context.Context,
	consumer consumer,
	message core_messaging.Message,
	nextTopic string,
	logger *core_logger.Logger,
) {
	for {
		err := w.writer.SendToNextTopic(ctx, nextTopic, message)
		if err == nil {
			break
		}

		logger.Error("failed to send message to next retry topic", zap.String("topic", nextTopic), zap.Error(err))
		if waitUntil(ctx, time.Now().Add(w.config.RetryOperationDelay)) != nil {
			return
		}
	}

	for {
		err := consumer.CommitMessage(ctx, message)
		if err == nil {
			return
		}

		logger.Error("failed to commit retried message", zap.Error(err))
		if waitUntil(ctx, time.Now().Add(w.config.RetryOperationDelay)) != nil {
			return
		}
	}
}
