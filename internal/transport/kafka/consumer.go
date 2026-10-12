package transport_kafka

import (
	"context"
	"fmt"
	core_messaging "foodDelivery/internal/core/messaging"
	"time"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
}

type ConsumerConfig struct {
	Brokers []string
	Topic   string
	GroupID string
}

func NewConsumer(config ConsumerConfig) (*Consumer, error) {
	readerConfig := kafka.ReaderConfig{
		Brokers:                config.Brokers,
		Topic:                  config.Topic,
		GroupID:                config.GroupID,
		StartOffset:            kafka.FirstOffset,
		MinBytes:               1,
		MaxBytes:               10 << 20,
		MaxWait:                time.Second,
		CommitInterval:         0,
		WatchPartitionChanges:  true,
		PartitionWatchInterval: 5 * time.Second,
	}
	err := readerConfig.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid kafka reader config: %w", err)
	}

	return &Consumer{
		reader: kafka.NewReader(readerConfig),
	}, nil
}

func (c *Consumer) ReadMessage(ctx context.Context) (core_messaging.Message, error) {
	message, err := c.reader.FetchMessage(ctx)
	if err != nil {
		return core_messaging.Message{}, fmt.Errorf("fetch kafka message: %w", err)
	}

	return messageFromKafka(message), nil
}

func (c *Consumer) CommitMessage(ctx context.Context, message core_messaging.Message) error {
	err := c.reader.CommitMessages(ctx, kafka.Message{
		Topic:     message.Topic,
		Partition: message.Partition,
		Offset:    message.Offset,
	})
	if err != nil {
		return fmt.Errorf("commit kafka message: %w", err)
	}

	return nil
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
