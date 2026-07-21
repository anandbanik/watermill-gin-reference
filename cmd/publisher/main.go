// Command publisher sends a single order JSON payload to the configured
// messaging backend (Kafka or SQS) so you can exercise the listener path
// without going through the HTTP API.
//
// The backend is selected by MESSAGING_BACKEND, with all connection details
// read from the same environment variables the main application uses.
//
// Usage:
//
//	go run ./cmd/publisher -id o-1 -customer alice -amount 42.50
package main

import (
	"encoding/json"
	"flag"
	"log"
	"log/slog"
	"os"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/anandbanik/watermill-gin/config"
	"github.com/anandbanik/watermill-gin/internal/messaging"
	"github.com/anandbanik/watermill-gin/internal/order"
)

func main() {
	id := flag.String("id", "o-1", "order id")
	customer := flag.String("customer", "alice", "customer name")
	amount := flag.Float64("amount", 42.50, "order amount")
	currency := flag.String("currency", "USD", "currency")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	backend := config.LoadBackend()

	pub, destination, err := messaging.NewPublisher(backend, logger)
	if err != nil {
		log.Fatalf("create publisher (%s): %v", backend, err)
	}
	defer pub.Close()

	o := order.Order{ID: *id, Customer: *customer, Amount: *amount, Currency: *currency}
	payload, err := json.Marshal(o)
	if err != nil {
		log.Fatalf("marshal order: %v", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)
	if err := pub.Publish(destination, msg); err != nil {
		log.Fatalf("publish: %v", err)
	}
	log.Printf("published order %s via %s to %q", o.ID, backend, destination)
}
