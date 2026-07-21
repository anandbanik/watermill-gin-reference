// Package config centralises application configuration loaded from environment
// variables.
package config

import (
	"os"
	"strings"
)

// KafkaConfig holds all connection parameters for the Kafka broker.
type KafkaConfig struct {
	// Brokers is the list of bootstrap broker addresses.
	Brokers []string
	// Topic is the Kafka topic both the consumer and the test publisher use.
	Topic string
	// ConsumerGroup identifies this service's consumer group.
	ConsumerGroup string
}

// LoadKafka reads Kafka connection settings from environment variables,
// falling back to sensible local-dev defaults when they are absent.
//
//	KAFKA_BROKERS         — comma-separated list (default: localhost:9092)
//	KAFKA_TOPIC           — topic name           (default: orders)
//	KAFKA_CONSUMER_GROUP  — consumer group name  (default: orders-service)
func LoadKafka() KafkaConfig {
	return KafkaConfig{
		Brokers:       strings.Split(env("KAFKA_BROKERS", "localhost:9092"), ","),
		Topic:         env("KAFKA_TOPIC", "orders"),
		ConsumerGroup: env("KAFKA_CONSUMER_GROUP", "orders-service"),
	}
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
