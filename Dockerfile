# syntax=docker/dockerfile:1
FROM golang:1.26 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o watermill-gin .

FROM alpine:3.24

COPY --from=builder /app/watermill-gin /watermill-gin

EXPOSE 9090

ENTRYPOINT ["/watermill-gin"]
