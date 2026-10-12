package transport_kafka

import (
	core_messaging "foodDelivery/internal/core/messaging"

	"github.com/segmentio/kafka-go"
)

func messageFromKafka(message kafka.Message) core_messaging.Message {
	headers := make([]core_messaging.Header, len(message.Headers))
	for i := range message.Headers {
		headers[i] = core_messaging.Header{
			Key:   message.Headers[i].Key,
			Value: message.Headers[i].Value,
		}
	}

	return core_messaging.Message{
		Topic:     message.Topic,
		Partition: message.Partition,
		Offset:    message.Offset,
		Key:       message.Key,
		Value:     message.Value,
		Headers:   headers,
		Time:      message.Time,
	}
}

func messageForWrite(message core_messaging.Message) kafka.Message {
	headers := make([]kafka.Header, len(message.Headers))
	for i := range message.Headers {
		headers[i] = kafka.Header{
			Key:   message.Headers[i].Key,
			Value: message.Headers[i].Value,
		}
	}

	return kafka.Message{
		Topic:   message.Topic,
		Key:     message.Key,
		Value:   message.Value,
		Headers: headers,
		Time:    message.Time,
	}
}
