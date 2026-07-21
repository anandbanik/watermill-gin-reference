package messaging

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	wmsqs "github.com/ThreeDotsLabs/watermill-aws/sqs"
	wmkafka "github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/IBM/sarama"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/anandbanik/watermill-gin/config"
)

// NewSubscriber creates the Watermill subscriber appropriate for the configured
// backend and returns it alongside the topic/queue name to pass to NewConsumer.
func NewSubscriber(backend string, logger *slog.Logger) (message.Subscriber, string, error) {
	switch backend {
	case config.BackendSQS:
		cfg := config.LoadSQS()
		if cfg.QueueName == "" {
			return nil, "", fmt.Errorf("SQS_QUEUE_NAME must be set when MESSAGING_BACKEND=sqs")
		}
		sub, err := newSQSSubscriber(cfg, logger)
		return sub, cfg.QueueName, err
	case config.BackendKafka:
		cfg := config.LoadKafka()
		sub, err := newKafkaSubscriber(cfg, logger)
		return sub, cfg.Topic, err
	default:
		return nil, "", fmt.Errorf("unknown MESSAGING_BACKEND %q: must be %q or %q", backend, config.BackendKafka, config.BackendSQS)
	}
}

// NewPublisher creates the Watermill publisher appropriate for the configured
// backend and returns it alongside the destination topic/queue name to publish to.
func NewPublisher(backend string, logger *slog.Logger) (message.Publisher, string, error) {
	switch backend {
	case config.BackendSQS:
		cfg := config.LoadSQS()
		if cfg.QueueName == "" {
			return nil, "", fmt.Errorf("SQS_QUEUE_NAME must be set when MESSAGING_BACKEND=sqs")
		}
		pub, err := newSQSPublisher(cfg, logger)
		return pub, cfg.QueueName, err
	case config.BackendKafka:
		cfg := config.LoadKafka()
		pub, err := newKafkaPublisher(cfg, logger)
		return pub, cfg.Topic, err
	default:
		return nil, "", fmt.Errorf("unknown MESSAGING_BACKEND %q: must be %q or %q", backend, config.BackendKafka, config.BackendSQS)
	}
}

// --- Kafka ---

func newKafkaSubscriber(cfg config.KafkaConfig, logger *slog.Logger) (message.Subscriber, error) {
	saramaCfg := wmkafka.DefaultSaramaSubscriberConfig()
	saramaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest

	return wmkafka.NewSubscriber(
		wmkafka.SubscriberConfig{
			Brokers:               cfg.Brokers,
			Unmarshaler:           wmkafka.DefaultMarshaler{},
			OverwriteSaramaConfig: saramaCfg,
			ConsumerGroup:         cfg.ConsumerGroup,
		},
		watermill.NewSlogLogger(logger),
	)
}

func newKafkaPublisher(cfg config.KafkaConfig, logger *slog.Logger) (message.Publisher, error) {
	return wmkafka.NewPublisher(
		wmkafka.PublisherConfig{
			Brokers:   cfg.Brokers,
			Marshaler: wmkafka.DefaultMarshaler{},
		},
		watermill.NewSlogLogger(logger),
	)
}

// --- SQS ---

func newSQSSubscriber(cfg config.SQSConfig, logger *slog.Logger) (message.Subscriber, error) {
	awsCfg, err := loadAWSConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	return wmsqs.NewSubscriber(
		wmsqs.SubscriberConfig{AWSConfig: awsCfg},
		watermill.NewSlogLogger(logger),
	)
}

func newSQSPublisher(cfg config.SQSConfig, logger *slog.Logger) (message.Publisher, error) {
	awsCfg, err := loadAWSConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	return wmsqs.NewPublisher(
		wmsqs.PublisherConfig{AWSConfig: awsCfg},
		watermill.NewSlogLogger(logger),
	)
}

// loadAWSConfig builds an aws.Config from SQSConfig. When Endpoint is set the
// AWS SDK routes all requests to that address — useful for LocalStack.
func loadAWSConfig(cfg config.SQSConfig) (aws.Config, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
	}
	if cfg.Endpoint != "" {
		opts = append(opts, awsconfig.WithBaseEndpoint(cfg.Endpoint))
	}
	return awsconfig.LoadDefaultConfig(context.Background(), opts...)
}
