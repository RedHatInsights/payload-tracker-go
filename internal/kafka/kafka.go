package kafka

import (
	"context"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"

	config "github.com/redhatinsights/payload-tracker-go/internal/config"
	l "github.com/redhatinsights/payload-tracker-go/internal/logging"
)

// NewConsumer Creates brand new consumer instance based on topic
func NewConsumer(ctx context.Context, config *config.TrackerConfig, topic string) (*kafka.Consumer, error) {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        config.KafkaConfig.KafkaBootstrapServers,
		"group.id":                 config.KafkaConfig.KafkaGroupID,
		"auto.offset.reset":        config.KafkaConfig.KafkaAutoOffsetReset,
		"auto.commit.interval.ms":  config.KafkaConfig.KafkaAutoCommitInterval,
		"go.logs.channel.enable":   true,
		"allow.auto.create.topics": true,
	})

	if err != nil {
		return nil, err
	}

	err = consumer.SubscribeTopics([]string{topic}, nil)

	if err != nil {
		return nil, err
	}

	l.Log.Info("Connected to Kafka")

	return consumer, nil
}

// NewConsumerEventLoop creates a new consumer event loop based on the information passed with it
func NewConsumerEventLoop(
	ctx context.Context,
	consumer *kafka.Consumer,
) {
	for {
		msg, err := consumer.ReadMessage(10 * time.Second) // TODO: configurable

		if err != nil {
			l.Log.Fatal("Consumer error: %v (%v)\n", err, msg)
			break
		} else {
			l.Log.Info("message %s = %s\n", string(msg.Key), string(msg.Value))
		}
		// TODO: Add Handler
	}

	consumer.Close()
}