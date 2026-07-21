package config

import "strings"

// Messaging backend identifiers.
const (
	BackendKafka = "kafka"
	BackendSQS   = "sqs"
)

// LoadBackend returns the active messaging backend, read from the
// MESSAGING_BACKEND environment variable. Defaults to "kafka".
// The value is lowercased and trimmed so "SQS", " sqs ", etc. all work.
//
//	MESSAGING_BACKEND — "kafka" | "sqs"  (default: kafka)
func LoadBackend() string {
	return strings.ToLower(strings.TrimSpace(env("MESSAGING_BACKEND", BackendKafka)))
}
