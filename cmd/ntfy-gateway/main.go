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

	"github.com/fudoge/ntfy-gateway/internal/adapter/flux"
	"github.com/fudoge/ntfy-gateway/internal/adapter/generic"
	"github.com/fudoge/ntfy-gateway/internal/config"
	"github.com/fudoge/ntfy-gateway/internal/handler"
	"github.com/fudoge/ntfy-gateway/internal/ntfy"
	"github.com/fudoge/ntfy-gateway/internal/server"
	"github.com/fudoge/ntfy-gateway/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With(
		slog.String("service", "ntfy-gateway"),
	)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	decoders := map[string]service.Decoder{
		"flux":    flux.Decoder{},
		"generic": generic.Decoder{},
	}

	sources := make(map[string]service.Source, len(cfg.Sources))
	for sourceID, sourceConfig := range cfg.Sources {
		sources[sourceID] = service.Source{
			Type:   sourceConfig.Type,
			Topic:  sourceConfig.Topic,
			Secret: os.Getenv(sourceConfig.SecretEnv),
		}
	}

	ntfyToken := ""
	if cfg.Ntfy.TokenEnv != "" {
		ntfyToken = os.Getenv(cfg.Ntfy.TokenEnv)
	}

	publisher := ntfy.NewClient(
		cfg.Ntfy.Endpoint,
		ntfyToken,
		&http.Client{Timeout: 10 * time.Second},
	)
	gateway := service.NewGateway(sources, decoders, publisher)
	httpHandler := handler.NewHandler(gateway, logger)
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
