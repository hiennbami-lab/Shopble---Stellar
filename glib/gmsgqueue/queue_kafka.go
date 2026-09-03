package gmsgqueue

import (
	"context"
	"fmt"
	"shopble/common/comqueue"

	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	reader     *kafka.Reader
	currentMsg *kafka.Message
}

func NewkafkaConsumer(topics []string, groupId string) (consumer comqueue.Consumer, err error) {
	kafkaReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{"localhost:9092"},
		GroupTopics: topics,
		GroupID:     groupId,
		MaxBytes:    10e6,
	})
	consumer = &KafkaConsumer{reader: kafkaReader}
	return
}

func (c *KafkaConsumer) FetchMessage(ctx context.Context) (msg comqueue.Msg, err error) {
	fetchedMsg, err := c.reader.FetchMessage(ctx)
	if err != nil {
		err = fmt.Errorf("fetch msg failed | err=%s", err.Error())
		return
	}
	c.currentMsg = &fetchedMsg
	msg = comqueue.Msg{
		Raw: c.currentMsg.Value,
		Key: string(c.currentMsg.Key),
	}
	return
}

func (c KafkaConsumer) CommitMessage(ctx context.Context) error {
	defer func() { c.currentMsg = nil }()
	err := c.reader.CommitMessages(ctx, *c.currentMsg)
	if err != nil {
		err = fmt.Errorf("ack msg failed | err=%s", err.Error())
		return err
	}
	return nil
}

type KafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(topic string) comqueue.Producer {
	kafkaWriter := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{"localhost:9092"},
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	})
	return &KafkaProducer{writer: kafkaWriter}
}

func (p KafkaProducer) Publish(ctx context.Context, key string, msg []byte) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: msg,
	})
}
