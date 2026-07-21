# WaterMill & Gin Reference Project

A reference Go application demonstrating how to serve the **same business logic over two transports simultaneously**:

- an **HTTP API** built with [Gin](https://github.com/gin-gonic/gin) — `POST /orders`
- a **message listener** built with [Watermill](https://github.com/ThreeDotsLabs/watermill) — Kafka topic or SQS queue, selected at runtime

Both the HTTP request and the incoming message decode the identical JSON payload and call `order.Service.Process`, so the business logic lives in exactly **one place**.

```
         POST /orders (Gin)          ┐
                                     ├─► order.Service.Process(ctx, source, Order)
   Kafka topic / SQS queue           ┘
   (Watermill — backend selectable)
```

---

## Project layout

```
.
├── config/
│   ├── http.go          HTTP server config  (HTTP_ADDR)
│   ├── kafka.go         Kafka config        (KAFKA_BROKERS, KAFKA_TOPIC, …)
│   ├── messaging.go     Backend selector    (MESSAGING_BACKEND)
│   └── sqs.go           SQS config          (AWS_REGION, SQS_QUEUE_NAME, …)
├── internal/
│   ├── api/
│   │   └── server.go    Gin router — decodes JSON, calls the shared handler
│   ├── messaging/
│   │   ├── consumer.go  Watermill router — decodes JSON, calls the shared handler
│   │   └── factory.go   Creates the right Subscriber/Publisher for the active backend
│   └── order/
│       └── order.go     Domain type + shared handler (Service.Process)
├── cmd/
│   └── publisher/
│       └── main.go      CLI helper — publishes a test Order to the active backend
├── main.go                    Wires everything together; graceful shutdown on SIGINT/SIGTERM
├── Dockerfile                 Multi-stage build; set HTTP_ADDR=:9090 (EXPOSE 9090)
├── docker-compose-kafka.yml   Kafka (KRaft, single-node)
└── docker-compose-sqs.yml     Floci — local SQS emulator (https://floci.io)
```

---

## Prerequisites

- Go 1.22+
- Docker (for local broker)

---

## Backend 1 — Kafka

### Start the broker

```sh
docker compose -f docker-compose-kafka.yml up -d
```

This starts a single-node Apache Kafka in KRaft mode (no ZooKeeper) on `localhost:9092`. Topics are auto-created on first use.

### Run the application

The docker compose script will build and start the container. You can log using the below command

```sh
docker logs -f watermill-gin
```

The app logs the selected backend and starts both transports:

```json
{"level":"INFO","msg":"messaging backend selected","backend":"kafka"}
{"level":"INFO","msg":"starting http server","addr":":8080"}
{"level":"INFO","msg":"starting consumer","source":"kafka","topic":"orders"}
```

### Test the HTTP path

```sh
curl -s -X POST localhost:9090/orders \
  -H 'Content-Type: application/json' \
  -d '{"id":"api-1","customer":"alice","amount":42.50,"currency":"USD"}'
```

Expected response:

```json
{"order_id":"api-1","status":"accepted","message":"order api-1 processed via api"}
```

### Test the Kafka path

```sh
go run ./cmd/publisher -id kafka-1 -customer bob -amount 99.99
```

Both calls produce an `order processed` log from the **same handler**, distinguished only by the `source` field:

```json
{"level":"INFO","msg":"order processed","source":"api",   "order_id":"api-1",   "customer":"alice","amount":42.5}
{"level":"INFO","msg":"order processed","source":"kafka",  "order_id":"kafka-1", "customer":"bob",  "amount":99.99}
```

### Test validation (applies equally to both transports)

```sh
curl -s -X POST localhost:9090/orders \
  -H 'Content-Type: application/json' \
  -d '{"id":"","customer":"x","amount":-1}'
```

```json
{"error":"invalid order: id is required"}
```

### Stop

```sh
docker compose -f docker-compose-kafka.yml down
```

---

## Backend 2 — SQS (via Floci)

[Floci](https://floci.io) is a local AWS cloud emulator. It runs the same API as Amazon SQS on `localhost:4566` so no real AWS account or credentials are needed.

### Start the full stack

`docker-compose-sqs.yml` starts both Floci **and** the application in the same network — no separate `go run .` needed. The app is built from the local `Dockerfile` and listens on port `9090`.

```sh
docker compose -f docker-compose-sqs.yml up -d
```

### Test the HTTP path

```sh
curl -s -X POST localhost:9090/orders \
  -H 'Content-Type: application/json' \
  -d '{"id":"api-2","customer":"carol","amount":75.00,"currency":"USD"}'
```

Expected response:

```json
{"order_id":"api-2","status":"accepted","message":"order api-2 processed via api"}
```

### Test the SQS path

```sh
MESSAGING_BACKEND=sqs \
AWS_ENDPOINT=http://localhost:4566 \
AWS_REGION=us-east-1 \
AWS_ACCESS_KEY_ID=test \
AWS_SECRET_ACCESS_KEY=test \
go run ./cmd/publisher -id sqs-1 -customer dave -amount 120.00
```

The app container log will show `"source":"sqs"`:

```json
{"level":"INFO","msg":"order processed","source":"sqs","order_id":"sqs-1","customer":"dave","amount":120}
```

### Stop

```sh
docker compose -f docker-compose-sqs.yml down
```

---

## Docker

The `Dockerfile` produces a minimal Alpine image via a two-stage build. It is used automatically by `docker-compose-sqs.yml`. To build and run standalone:

```sh
docker build -t watermill-gin .
docker run -p 9090:9090 \
  -e HTTP_ADDR=:9090 \
  -e MESSAGING_BACKEND=kafka \
  -e KAFKA_BROKERS=<host>:9092 \
  watermill-gin
```

> **Note:** The image `EXPOSE`s port `9090`. Always set `HTTP_ADDR=:9090` when running in Docker — the default `:8080` is for local `go run .` only.

---

## Against real AWS SQS

Create a standard SQS queue in your AWS account, then run:

```sh
MESSAGING_BACKEND=sqs \
SQS_QUEUE_NAME=orders \
AWS_REGION=us-east-1 \
go run .
```

Standard AWS credential resolution applies (env vars, `~/.aws/credentials`, IAM role, etc.). Do **not** set `AWS_ENDPOINT` — leave it unset and the SDK routes to real AWS.

---

## Configuration reference

### General

| Variable             | Default  | Description                          |
|----------------------|----------|--------------------------------------|
| `HTTP_ADDR`          | `:8080`  | Address the HTTP server listens on   |
| `MESSAGING_BACKEND`  | `kafka`  | Active message backend: `kafka` or `sqs` |

### Kafka (`MESSAGING_BACKEND=kafka`)

| Variable               | Default          | Description                           |
|------------------------|------------------|---------------------------------------|
| `KAFKA_BROKERS`        | `localhost:9092` | Comma-separated bootstrap broker list |
| `KAFKA_TOPIC`          | `orders`         | Topic to consume and publish to       |
| `KAFKA_CONSUMER_GROUP` | `orders-service` | Consumer group name                   |

### SQS (`MESSAGING_BACKEND=sqs`)

| Variable                | Default       | Description                                           |
|-------------------------|---------------|-------------------------------------------------------|
| `AWS_REGION`            | `us-east-1`            | AWS region                                            |
| `SQS_QUEUE_NAME`        | `orders`               | Queue name — created automatically if absent          |
| `AWS_ENDPOINT`          | `http://localhost:4566` | Endpoint override for Floci/LocalStack; unset for real AWS |
| `AWS_ACCESS_KEY_ID`     | *(unset)*     | AWS access key (use `test` for Floci)                 |
| `AWS_SECRET_ACCESS_KEY` | *(unset)*     | AWS secret key (use `test` for Floci)                 |

### Publisher CLI flags (`cmd/publisher`)

| Flag          | Default    | Description              |
|---------------|------------|--------------------------|
| `-id`         | `o-1`      | Order ID                 |
| `-customer`   | `alice`    | Customer name            |
| `-amount`     | `42.50`    | Order amount             |
| `-currency`   | `USD`      | Currency code            |

All connection settings are inherited from the same environment variables listed above.

---

## Order payload

Both the HTTP endpoint and the message listener accept the same JSON shape:

```json
{
  "id":       "order-123",
  "customer": "alice",
  "amount":   42.50,
  "currency": "USD"
}
```

Validation rules (enforced by `order.Service.Process` regardless of transport):

- `id` — required, non-empty
- `customer` — required, non-empty
- `amount` — must be greater than zero
