package gmsgqueue

import (
	"context"
	"fmt"
	"shopble/common/comqueue"

	"github.com/nats-io/nats.go"
)

type (
	JsOption interface {
		Apply(*JetstreamMeta)
	}
	JsOptionSetter func(*JetstreamMeta)
)

func (o JsOptionSetter) Apply(c *JetstreamMeta) {
	o(c)
}

type JetstreamMeta struct {
	streamName string
}

type JetstreamConsumer struct {
	*JetstreamMeta

	sub        *nats.Subscription
	currentMsg *nats.Msg
}

func JsWithStreamName(streamName string) JsOptionSetter {
	return func(c *JetstreamMeta) {
		c.streamName = streamName
	}
}

func NewJetstreamConsumerWithOpts(optSetter ...JsOption) func(topics []string, durable string) (comqueue.Consumer, error) {
	return func(topics []string, durable string) (comqueue.Consumer, error) {
		return NewJetstreamConsumer(topics, durable, optSetter...)
	}
}

func NewJetstreamConsumer(topics []string, durable string, optSetter ...JsOption) (consumer comqueue.Consumer, err error) {
	jsConsumer := &JetstreamConsumer{}
	for _, opt := range optSetter {
		opt.Apply(jsConsumer.JetstreamMeta)
	}
	pullSub, err := GetJsCtx().PullSubscribe(topics[0], durable,
		nats.BindStream(jsConsumer.streamName),
		nats.AckExplicit())
	if err != nil {
		err = fmt.Errorf("init consumer failed | err=%s", err.Error())
		return
	}
	jsConsumer.sub = pullSub
	consumer = jsConsumer
	return
}

func (c *JetstreamConsumer) FetchMessage(ctx context.Context) (msg comqueue.Msg, err error) {
	fetchedMsg, err := c.sub.Fetch(1)
	if err != nil {
		err = fmt.Errorf("fetch msg failed | err=%s", err.Error())
		return
	}
	c.currentMsg = fetchedMsg[0]
	msg = comqueue.Msg{
		Raw: c.currentMsg.Data,
		Key: c.currentMsg.Header.Get("msgKey"),
	}
	return
}

func (c JetstreamConsumer) CommitMessage(ctx context.Context) error {
	defer func() { c.currentMsg = nil }()
	if err := c.currentMsg.AckSync(); err != nil {
		err = fmt.Errorf("ack msg failed | err=%s", err.Error())
		return err
	}
	return nil
}

type JetstreamProducer struct {
	*JetstreamMeta

	subject string
}

func NewJetstreamProducer(topic string, optSetter ...JsOption) comqueue.Producer {
	jsProducer := &JetstreamProducer{subject: topic}
	for _, opt := range optSetter {
		opt.Apply(jsProducer.JetstreamMeta)
	}
	return jsProducer
}

func (p JetstreamProducer) Publish(ctx context.Context, key string, msg []byte) error {
	_, err := GetJsCtx().PublishMsg(&nats.Msg{
		Subject: p.subject,
		Data:    msg,
		Header: nats.Header{
			"msgKey": []string{key},
		},
	},
		nats.ExpectStream(p.streamName),
	)
	return err
}
