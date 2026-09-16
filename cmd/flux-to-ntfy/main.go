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

	"github.com/fudoge/flux-to-ntfy/internal/config"
	"github.com/fudoge/flux-to-ntfy/internal/handler"
	"github.com/fudoge/flux-to-ntfy/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With(
		slog.String("service", "flux-to-ntfy"),
	)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	httpHandler := handler.NewHandler(logger)
	s := server.New(cfg, logger, httpHandler)
	signalCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverErr := make(chan error, 1)

	go func() {
		serverErr <- s.Start()
	}()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
		return
	case <-signalCtx.Done():
		logger.Info("shutdown signal received")
	}

	stop()

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(), 10*time.Second,
	)
	defer cancel()

	if err := s.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}

	err = <-serverErr
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server stopped with error", "error", err)
	}
}
