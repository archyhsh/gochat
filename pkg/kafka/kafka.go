package kafka

import (
	"context"

	"github.com/IBM/sarama"
)

// Producer Kafka 生产者
type Producer struct {
	producer sarama.SyncProducer
	topic    string
}

// NewProducer 创建 Kafka 生产者
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

// Send 发送消息
func (p *Producer) Send(key, value []byte) error {
	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.ByteEncoder(key),
		Value: sarama.ByteEncoder(value),
	}

	_, _, err := p.producer.SendMessage(msg)
	return err
}

// SendToTopic 发送消息到指定 topic
func (p *Producer) SendToTopic(topic string, key, value []byte) error {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.ByteEncoder(key),
		Value: sarama.ByteEncoder(value),
	}

	_, _, err := p.producer.SendMessage(msg)
	return err
}

// Close 关闭生产者
func (p *Producer) Close() error {
	return p.producer.Close()
}

// Consumer Kafka 消费者
type Consumer struct {
	consumer sarama.ConsumerGroup
	topics   []string
	handler  ConsumerHandler
}

// ConsumerHandler 消息处理接口
type ConsumerHandler interface {
	Handle(ctx context.Context, message *sarama.ConsumerMessage) error
}

// NewConsumer 创建 Kafka 消费者
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

// Start 启动消费者
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

// Close 关闭消费者
func (c *Consumer) Close() error {
	return c.consumer.Close()
}

// consumerGroupHandler 消费者组处理器
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
