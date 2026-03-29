package messaging

import "context"

// FailureStore defines the behavior for storing messages that failed to be sent.
type FailureStore interface {
	Save(ctx context.Context, topic string, key, value []byte, err error) error
}

// Producer defines the basic behavior of a message producer.
type Producer interface {
	Send(ctx context.Context, key, value []byte) error
	SendToTopic(ctx context.Context, topic string, key, value []byte) error
}
