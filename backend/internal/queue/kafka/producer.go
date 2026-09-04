package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ecommerce-backend/internal/domain"
	"ecommerce-backend/pkg/metrics"
	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(broker string, topic string) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(broker),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		Async:        true, // Asynchronous non-blocking batch production
		BatchTimeout: 5 * time.Millisecond,
		BatchSize:    200,
		Compression:  kafka.Snappy,
		RequiredAcks: kafka.RequireOne,
		Completion: func(messages []kafka.Message, err error) {
			if err != nil {
				fmt.Printf("[KAFKA PRODUCER ERROR] Failed to flush messages: %v\n", err)
			}
		},
	}

	return &Producer{writer: writer}
}

func (p *Producer) PublishOrder(ctx context.Context, event domain.OrderPlacedEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal order event: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(event.OrderID),
		Value: payload,
		Time:  time.Now(),
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to write message to kafka: %w", err)
	}

	metrics.KafkaOrdersProducedTotal.Inc()
	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
