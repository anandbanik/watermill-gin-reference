// Package messaging holds the event-driven transport layer. It is intentionally
// decoupled from any specific broker: the Consumer works with any Watermill
// message.Subscriber, whether backed by Kafka, SQS, or anything else.
package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/anandbanik/watermill-gin/internal/order"
)

// Consumer subscribes to a topic/queue and routes each message to the shared
// order service via a Watermill router.
type Consumer struct {
	router     *message.Router
	subscriber message.Subscriber
	topic      string
	source     string // passed to order.Service.Process for observability
	svc        *order.Service
	logger     *slog.Logger
}

// NewConsumer wires a Watermill router around the provided subscriber.
//
//   - source identifies the transport ("kafka", "sqs") and is forwarded to the
//     order handler for observability — the handler logic is identical either way.
//   - topic is the Kafka topic name or the SQS queue URL, depending on backend.
func NewConsumer(source, topic string, sub message.Subscriber, svc *order.Service, logger *slog.Logger) (*Consumer, error) {
	wmLogger := watermill.NewSlogLogger(logger)

	router, err := message.NewRouter(message.RouterConfig{}, wmLogger)
	if err != nil {
		return nil, fmt.Errorf("create router: %w", err)
	}
	router.AddMiddleware(middleware.Recoverer)

	c := &Consumer{
		router:     router,
		subscriber: sub,
		topic:      topic,
		source:     source,
		svc:        svc,
		logger:     logger,
	}

	router.AddNoPublisherHandler(
		"orders-consumer",
		topic,
		sub,
		c.handle,
	)

	return c, nil
}

// handle decodes the JSON payload and calls the shared handler.
func (c *Consumer) handle(msg *message.Message) error {
	var o order.Order
	if err := json.Unmarshal(msg.Payload, &o); err != nil {
		// Bad payloads are acked (dropped) so they don't block the queue/partition.
		c.logger.Error("failed to decode message payload; dropping",
			"source", c.source, "uuid", msg.UUID, "error", err)
		return nil
	}

	// Same handler the HTTP API invokes.
	if _, err := c.svc.Process(msg.Context(), c.source, o); err != nil {
		// Returning an error nacks the message; the broker will redeliver it.
		return fmt.Errorf("process order %s: %w", o.ID, err)
	}
	return nil
}

// Run starts the router and blocks until ctx is cancelled.
func (c *Consumer) Run(ctx context.Context) error {
	c.logger.Info("starting consumer", "source", c.source, "topic", c.topic)
	return c.router.Run(ctx)
}

// Close releases the underlying subscriber.
func (c *Consumer) Close() error {
	return c.subscriber.Close()
}
