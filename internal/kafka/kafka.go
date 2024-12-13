package kafka

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"gorm.io/gorm"

	config "github.com/redhatinsights/payload-tracker-go/internal/config"
	"github.com/redhatinsights/payload-tracker-go/internal/endpoints"
	l "github.com/redhatinsights/payload-tracker-go/internal/logging"
)

// NewConsumer Creates brand new consumer instance based on topic
func NewConsumer(ctx context.Context, config *config.TrackerConfig, topic string) (*kafka.Consumer, error) {
	var configMap kafka.ConfigMap

	if config.KafkaConfig.SASLMechanism != "" {
		configMap = kafka.ConfigMap{
			"bootstrap.servers":        config.KafkaConfig.KafkaBootstrapServers,
			"group.id":                 config.KafkaConfig.KafkaGroupID,
			"security.protocol":        config.KafkaConfig.Protocol,
			"sasl.mechanism":           config.KafkaConfig.SASLMechanism,
			"ssl.ca.location":          config.KafkaConfig.KafkaCA,
			"sasl.username":            config.KafkaConfig.KafkaUsername,
			"sasl.password":            config.KafkaConfig.KafkaPassword,
			"go.logs.channel.enable":   true,
			"allow.auto.create.topics": true,
		}
	} else {
		configMap = kafka.ConfigMap{
			"bootstrap.servers":        config.KafkaConfig.KafkaBootstrapServers,
			"group.id":                 config.KafkaConfig.KafkaGroupID,
			"auto.offset.reset":        config.KafkaConfig.KafkaAutoOffsetReset,
			"auto.commit.interval.ms":  config.KafkaConfig.KafkaAutoCommitInterval,
			"go.logs.channel.enable":   true,
			"allow.auto.create.topics": true,
		}
	}

	consumer, err := kafka.NewConsumer(&configMap)

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
	cfg *config.TrackerConfig,
	consumer *kafka.Consumer,
	db *gorm.DB,
) {

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	handler := &handler{
		db: db,
	}

	run := true

	for run {
		select {
		case sig := <-sigchan:
			l.Log.Infof("Caught Signal %v: terminating\n", sig)
			run = false
		default:

			event := consumer.Poll(100)
			if event == nil {
				continue
			}

			switch e := event.(type) {
			case *kafka.Message:
				endpoints.IncConsumedMessages()
				handler.onMessage(ctx, e, cfg)
			case kafka.Error:
				endpoints.IncConsumeErrors()
				l.Log.Errorf("Consumer error: %v (%v)\n", e.Code(), e)
			case kafka.OffsetsCommitted:
				if e.Error != nil {
					l.Log.Errorf("Unable to commit offset: %#v", e)
					break
				}

				l.Log.Tracef("Ignored OffsetsComitted: %#v", e)

			default:
				l.Log.Infof("Ignored Kafka event: %#v\n", e)
			}

		}
	}

	consumer.Close()
}
