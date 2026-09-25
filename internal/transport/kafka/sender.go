package transport_kafka

import (
	core_messaging "avitoKitchen/internal/core/messaging"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

const OrdersCreatedTopic = "orders.created"

type SenderConfig struct {
	Brokers      []string
	MaxAttempts  int
	WriteTimeout time.Duration
}

type Sender struct {
	writer *kafka.Writer
}

func NewSender(config SenderConfig) (*Sender, error) {
	if len(config.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are empty")
	}
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 3
	}
	if config.WriteTimeout <= 0 {
		config.WriteTimeout = 5 * time.Second
	}

	return &Sender{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(config.Brokers...),
			Balancer:               &kafka.Hash{},
			MaxAttempts:            config.MaxAttempts,
			BatchSize:              1,
			WriteTimeout:           config.WriteTimeout,
			RequiredAcks:           kafka.RequireAll,
			Async:                  false,
			AllowAutoTopicCreation: true,
		},
	}, nil
}

func (s *Sender) SendOrderCreated(ctx context.Context, orderID uuid.UUID) error {
	now := time.Now().UTC()
	event := core_messaging.OrderCreatedEvent{
		OrderID:   orderID,
		CreatedAt: now,
	}

	value, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal order created event: %w", err)
	}

	err = s.send(ctx, core_messaging.Message{
		Topic: OrdersCreatedTopic,
		Key:   []byte(orderID.String()),
		Value: value,
		Headers: []core_messaging.Header{
			{Key: "event_type", Value: []byte(OrdersCreatedTopic)},
		},
		Time: now,
	})
	if err != nil {
		return fmt.Errorf("send order created event: %w", err)
	}

	return nil
}

func (s *Sender) SendToNextTopic(ctx context.Context, nextTopic string, message core_messaging.Message) error {
	if nextTopic == "" {
		return fmt.Errorf("next kafka topic is empty")
	}

	message.Topic = nextTopic
	message.Time = time.Now().UTC()
	err := s.send(ctx, message)
	if err != nil {
		return fmt.Errorf("send message to topic %s: %w", nextTopic, err)
	}

	return nil
}

func (s *Sender) send(ctx context.Context, message core_messaging.Message) error {
	if message.Topic == "" {
		return fmt.Errorf("kafka topic is empty")
	}

	err := s.writer.WriteMessages(ctx, messageForWrite(message))
	if err != nil {
		return fmt.Errorf("write kafka message: %w", err)
	}

	return nil
}

func (s *Sender) Close() error {
	return s.writer.Close()
}
