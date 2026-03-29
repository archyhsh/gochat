package messaging

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
)

type ReliableProducer struct {
	producer Producer
	store    FailureStore
	topic    string
}

func NewReliableProducer(p Producer, store FailureStore, defaultTopic string) *ReliableProducer {
	return &ReliableProducer{
		producer: p,
		store:    store,
		topic:    defaultTopic,
	}
}

func (rp *ReliableProducer) Send(ctx context.Context, key, value []byte) error {
	return rp.SendToTopic(ctx, rp.topic, key, value)
}

func (rp *ReliableProducer) SendToTopic(ctx context.Context, topic string, key, value []byte) error {
	err := rp.producer.SendToTopic(ctx, topic, key, value)
	if err != nil {
		// If primary sending (including its internal retries) fails, save to DLQ
		logx.WithContext(ctx).Errorf("[ReliableProducer] Failed to send to topic %s, saving to DLQ: %v", topic, err)
		if rp.store != nil {
			saveErr := rp.store.Save(ctx, topic, key, value, err)
			if saveErr != nil {
				logx.WithContext(ctx).Errorf("[ReliableProducer] CRITICAL: Failed to save to DLQ: %v", saveErr)
			}
		}
		return err
	}
	return nil
}
