package handler

import (
	"fmt"
	"log/slog"
	"net/http"
)

func NewHandler(logger *slog.Logger) *http.ServeMux {
	logger = logger.With(slog.String("component", "handler"))
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		health(logger, w, r)
	})

	return mux
}

func health(logger *slog.Logger, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprintln(w, "ok"); err != nil {
		logger.Error("failed to write health response", "error", err)
	}
}
