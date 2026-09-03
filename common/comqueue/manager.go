package comqueue

import (
	"context"
)

type Consumer interface {
	FetchMessage(ctx context.Context) (msg Msg, err error)
	CommitMessage(ctx context.Context) error
}

type Msg struct {
	Raw []byte
	Key string
}

type Producer interface {
	Publish(ctx context.Context, subj string, msg []byte) error
}

type (
	ConsumerCreator func(topic []string, groupId string) (Consumer, error)
	ProducerCreator func(topic string) Producer
)
