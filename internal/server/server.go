package server

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/fudoge/ntfy-gateway/internal/config"
)

type Server struct {
	logger     *slog.Logger
	httpServer *http.Server
}

func New(cfg *config.Config, logger *slog.Logger, handler http.Handler) *Server {
	return &Server{
		logger: logger.With(slog.String("component", "server")),
		httpServer: &http.Server{
			Addr:    cfg.BindAddress,
			Handler: handler,
		},
	}
}

func (s *Server) Start() error {
	s.logger.Info("server is starting", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("server is shutting down")
	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		return err
	}
	s.logger.Info("server shutdown complete")
	return nil
}
