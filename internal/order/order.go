// Package order holds the domain type and the business logic that is shared
// between the HTTP API (Gin) and the Kafka listener (Watermill). Neither
// transport knows about the other; both simply decode an Order and call
// Service.Process.
package order

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// Order is the JSON payload accepted by both the API and the Kafka listener.
type Order struct {
	ID       string  `json:"id"`
	Customer string  `json:"customer"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// Validate returns an error if the payload is missing required fields.
func (o Order) Validate() error {
	switch {
	case o.ID == "":
		return errors.New("id is required")
	case o.Customer == "":
		return errors.New("customer is required")
	case o.Amount <= 0:
		return errors.New("amount must be greater than zero")
	}
	return nil
}

// Result is what Process returns to the caller.
type Result struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Service contains the business logic. This is the single code path that both
// the API request and the Kafka message flow through.
type Service struct {
	logger *slog.Logger
}

// NewService builds a Service.
func NewService(logger *slog.Logger) *Service {
	return &Service{logger: logger}
}

// Process is the shared handler. Both the Gin handler and the Watermill
// consumer decode a JSON payload into an Order and hand it to this method, so
// there is exactly one implementation of the business rules.
//
// The `source` argument identifies where the call originated ("api" or
// "kafka") purely for observability — the logic itself is identical either way.
func (s *Service) Process(ctx context.Context, source string, o Order) (Result, error) {
	if err := o.Validate(); err != nil {
		s.logger.WarnContext(ctx, "order rejected",
			"source", source, "order_id", o.ID, "error", err)
		return Result{}, fmt.Errorf("invalid order: %w", err)
	}

	// Stand-in for real work (persist, charge, publish downstream events, ...).
	s.logger.InfoContext(ctx, "order processed",
		"source", source,
		"order_id", o.ID,
		"customer", o.Customer,
		"amount", o.Amount,
		"currency", o.Currency,
	)

	return Result{
		OrderID: o.ID,
		Status:  "accepted",
		Message: fmt.Sprintf("order %s processed via %s", o.ID, source),
	}, nil
}
