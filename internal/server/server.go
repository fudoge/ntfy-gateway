package server

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/fudoge/flux-to-ntfy/internal/config"
	"github.com/fudoge/flux-to-ntfy/internal/handler"
)

type Server struct {
	httpServer *http.Server
}

func New(cfg *config.Config) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:    cfg.BindAddress,
			Handler: handler.NewHandler(),
		},
	}
}

func (s *Server) Start() error {
	slog.Info("server is starting", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("server is shutting down")
	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		return err
	}
	slog.Info("server shutdown complete")
	return nil
}
