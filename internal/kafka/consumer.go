package kafka

import (
	"context"
	"encoding/json"
	"log"

	"go-project/internal/domain"

	kafkago "github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafkago.Reader
}

func NewConsumer(brokers []string, topic, groupID string) *Consumer {
	return &Consumer{
		reader: kafkago.NewReader(kafkago.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		}),
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

func (c *Consumer) Run(ctx context.Context) {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("kafka fetch error: %v", err)
			continue
		}

		var event domain.TransactionEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("kafka unmarshal error: %v", err)
			c.reader.CommitMessages(ctx, msg)
			continue
		}
		log.Printf("notification sent: transaction=%d from=%d to=%d amount=%d status=%s",
			event.TransactionID, event.FromUserID, event.ToUserID, event.Amount, event.Status)
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("kafka commit error: %v", err)
		}
	}
}
