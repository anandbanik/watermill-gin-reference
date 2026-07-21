// Command watermill-gin is a reference application that serves the same order
// handler over two transports:
//
//   - an HTTP API (Gin)                  POST /orders
//   - a message listener (Watermill)     Kafka topic or SQS queue, depending on
//     the MESSAGING_BACKEND env var
//
// Both decode the identical JSON payload and call order.Service.Process, so the
// business logic lives in exactly one place.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anandbanik/watermill-gin/config"
	"github.com/anandbanik/watermill-gin/internal/api"
	"github.com/anandbanik/watermill-gin/internal/messaging"
	"github.com/anandbanik/watermill-gin/internal/order"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	httpCfg := config.LoadHTTP()
	backend := config.LoadBackend()

	logger.Info("messaging backend selected", "backend", backend)

	// The single shared handler — same code regardless of which transport fires.
	svc := order.NewService(logger)

	// --- Message listener (Watermill: Kafka or SQS) ---
	sub, topic, err := messaging.NewSubscriber(backend, logger)
	if err != nil {
		logger.Error("failed to create subscriber", "backend", backend, "error", err)
		os.Exit(1)
	}

	consumer, err := messaging.NewConsumer(backend, topic, sub, svc, logger)
	if err != nil {
		logger.Error("failed to create consumer", "error", err)
		os.Exit(1)
	}
	defer consumer.Close()

	// --- HTTP API (Gin) ---
	httpServer := &http.Server{
		Addr:    httpCfg.Addr,
		Handler: api.NewRouter(svc),
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 2)

	go func() {
		logger.Info("starting http server", "addr", httpCfg.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	go func() {
		if err := consumer.Run(ctx); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		logger.Error("component failed", "error", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("http shutdown error", "error", err)
	}
	logger.Info("stopped")
}
