# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & run

```sh
go build ./...          # build all packages
go vet ./...            # static analysis
go run .                # run locally (HTTP on :9090, Kafka backend by default)
go run ./cmd/publisher  # send a test order to the active backend
```

Docker (both compose files build the app from `Dockerfile` and start it on port `9090`):

```sh
docker compose -f docker-compose-kafka.yml up -d   # Kafka + app
docker compose -f docker-compose-sqs.yml up -d     # Floci (SQS) + app
```

There are no tests yet.

## Architecture: shared handler pattern

The central design constraint is that **one Go function handles orders from both transports**. `internal/order/order.go` contains `Service.Process(ctx, source, Order) (Result, error)`. The `source` string (`"api"`, `"kafka"`, `"sqs"`) is passed only for logging — the logic is identical.

Both entry points decode the same JSON payload and call `Service.Process`:

- `internal/api/server.go` — Gin HTTP handler, `POST /orders`
- `internal/messaging/consumer.go` — Watermill router, any message broker

## Backend selection

`MESSAGING_BACKEND=kafka|sqs` (default `kafka`) controls which broker is used. The value is normalised with `strings.ToLower` + `strings.TrimSpace` before the switch. An unknown value produces a hard error at startup rather than silently falling back to Kafka.

`config/messaging.go` → `internal/messaging/factory.go` is the selection path. `NewSubscriber(backend, logger)` and `NewPublisher(backend, logger)` return the right Watermill implementation and the topic/queue name to use. Only the config for the active backend is loaded — the inactive one is never touched.

## Config package

All env-var loading lives in `config/`. Each file owns one concern:

| File | Env vars read |
|------|--------------|
| `http.go` | `HTTP_ADDR` (default `:9090`) |
| `messaging.go` | `MESSAGING_BACKEND` |
| `kafka.go` | `KAFKA_BROKERS`, `KAFKA_TOPIC`, `KAFKA_CONSUMER_GROUP` |
| `sqs.go` | `AWS_REGION`, `SQS_QUEUE_NAME`, `AWS_ENDPOINT` |

The unexported `env(key, fallback)` helper is defined once in `config/kafka.go` and shared across the package.

## Watermill wiring

`internal/messaging/consumer.go` is transport-agnostic. `NewConsumer(source, topic, sub, svc, logger)` accepts any `message.Subscriber`. The router calls `sub.Subscribe(ctx, topic)` — for Kafka `topic` is the topic name, for SQS it is the queue name (resolved to a URL by watermill-aws's default resolver). Bad payloads are acked and dropped (logged as errors); processing errors return a non-nil error which nacks the message for redelivery.

## Ports

- `:9090` — HTTP API (both local `go run .` and Docker)
- `:9092` — Kafka broker (external / local)
- `:9094` — Kafka internal listener (used by the app container in compose)
- `:4566` — Floci SQS emulator
