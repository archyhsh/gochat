package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"go.opentelemetry.io/otel"
)

type ProducerHeaderCarrier struct {
	Headers *[]sarama.RecordHeader
}

func (c *ProducerHeaderCarrier) Get(key string) string {
	for _, h := range *c.Headers {
		if string(h.Key) == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c *ProducerHeaderCarrier) Set(key string, value string) {
	*c.Headers = append(*c.Headers, sarama.RecordHeader{
		Key:   []byte(key),
		Value: []byte(value),
	})
}

func (c *ProducerHeaderCarrier) Keys() []string {
	keys := make([]string, len(*c.Headers))
	for i, h := range *c.Headers {
		keys[i] = string(h.Key)
	}
	return keys
}

type ConsumerHeaderCarrier struct {
	Headers []*sarama.RecordHeader
}

func (c *ConsumerHeaderCarrier) Get(key string) string {
	for _, h := range c.Headers {
		if string(h.Key) == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c *ConsumerHeaderCarrier) Set(key string, value string) {
	// Not usually used for extraction
}

func (c *ConsumerHeaderCarrier) Keys() []string {
	keys := make([]string, len(c.Headers))
	for i, h := range c.Headers {
		keys[i] = string(h.Key)
	}
	return keys
}

type Producer struct {
	producer sarama.SyncProducer
	topic    string
}

func NewProducer(brokers []string, topic string) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 3
	config.Producer.Return.Successes = true
	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}
	return &Producer{
		producer: producer,
		topic:    topic,
	}, nil
}

func (p *Producer) Send(ctx context.Context, key, value []byte) error {
	return p.SafeSendToTopic(ctx, p.topic, key, value)
}

func (p *Producer) SendToTopic(ctx context.Context, topic string, key, value []byte) error {
	return p.SafeSendToTopic(ctx, topic, key, value)
}

func (p *Producer) SafeSendToTopic(ctx context.Context, topic string, key, value []byte) error {
	var headers []sarama.RecordHeader
	carrier := &ProducerHeaderCarrier{Headers: &headers}
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	msg := &sarama.ProducerMessage{
		Topic:   topic,
		Key:     sarama.ByteEncoder(key),
		Value:   sarama.ByteEncoder(value),
		Headers: headers,
	}

	var lastErr error
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		_, _, lastErr = p.producer.SendMessage(msg)
		if lastErr == nil {
			return nil
		}

		if i < maxRetries-1 {
			backoff := time.Duration(i+1) * 200 * time.Millisecond
			time.Sleep(backoff)
		}
	}
	return fmt.Errorf("failed to send message to kafka after %d retries: %v", maxRetries, lastErr)
}

func (p *Producer) Close() error {
	return p.producer.Close()
}

type Consumer struct {
	consumer sarama.ConsumerGroup
	topics   []string
	handler  ConsumerHandler
}

type ConsumerHandler interface {
	Handle(ctx context.Context, message *sarama.ConsumerMessage) error
}

func NewConsumer(brokers []string, groupID string, topics []string, handler ConsumerHandler) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	consumer, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, err
	}
	return &Consumer{
		consumer: consumer,
		topics:   topics,
		handler:  handler,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	handler := &consumerGroupHandler{handler: c.handler}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := c.consumer.Consume(ctx, c.topics, handler); err != nil {
				return err
			}
		}
	}
}

func (c *Consumer) Close() error {
	return c.consumer.Close()
}

type consumerGroupHandler struct {
	handler ConsumerHandler
}

func (h *consumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *consumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		// Extract tracing context from Kafka headers
		carrier := &ConsumerHeaderCarrier{Headers: message.Headers}
		ctx := otel.GetTextMapPropagator().Extract(session.Context(), carrier)

		if err := h.handler.Handle(ctx, message); err != nil {
			// Log error but continue
			continue
		}
		session.MarkMessage(message, "")
	}
	return nil
}
