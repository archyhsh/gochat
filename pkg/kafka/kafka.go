package kafka

import (
	"context"
	"github.com/IBM/sarama"
)

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

func (p *Producer) Send(key, value []byte) error {
	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.ByteEncoder(key),
		Value: sarama.ByteEncoder(value),
	}
	_, _, err := p.producer.SendMessage(msg)
	return err
}

func (p *Producer) SendToTopic(topic string, key, value []byte) error {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.ByteEncoder(key),
		Value: sarama.ByteEncoder(value),
	}
	_, _, err := p.producer.SendMessage(msg)
	return err
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
func (h *consumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }

func (h *consumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		if err := h.handler.Handle(session.Context(), message); err != nil {
			// 记录错误但继续处理
			continue
		}
		session.MarkMessage(message, "")
	}
	return nil
}
