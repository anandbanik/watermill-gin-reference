package config

// SQSConfig holds all connection parameters for Amazon SQS.
type SQSConfig struct {
	// Region is the AWS region (e.g. "us-east-1").
	Region string
	// QueueName is the SQS queue name. The Watermill subscriber resolves it to a
	// full URL automatically. For LocalStack the queue is created if it is absent.
	QueueName string
	// Endpoint overrides the AWS service endpoint. Set this to the LocalStack
	// URL (e.g. "http://localhost:4566") for local development; leave blank for
	// real AWS.
	Endpoint string
}

// LoadSQS reads SQS connection settings from environment variables,
// falling back to sensible local-dev defaults when they are absent.
//
//	AWS_REGION      — AWS region        (default: us-east-1)
//	SQS_QUEUE_NAME  — queue name        (required when MESSAGING_BACKEND=sqs)
//	AWS_ENDPOINT    — endpoint override  (optional, for LocalStack)
func LoadSQS() SQSConfig {
	return SQSConfig{
		Region:    env("AWS_REGION", "us-east-1"),
		QueueName: env("SQS_QUEUE_NAME", "orders"),
		Endpoint:  env("AWS_ENDPOINT", "http://localhost:4566"),
	}
}
