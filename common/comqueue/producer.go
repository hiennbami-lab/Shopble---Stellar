package comqueue

import (
	"context"
	"encoding/json"
)

type OurProducer struct {
	producer Producer
}

func (o *OurProducer) EncodeMsg(msg any) ([]byte, error) {
	return json.Marshal(msg)
}

func NewOurProducer(producer Producer) *OurProducer {
	return &OurProducer{
		producer: producer,
	}
}

func (p OurProducer) Publish(ctx context.Context, key string, msg []byte) error {
	return p.producer.Publish(ctx, key, msg)
}
