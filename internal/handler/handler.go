package handler

import (
	"fmt"
	"log/slog"
	"net/http"
)

func NewHandler() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", health)

	return mux
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprintln(w, "ok"); err != nil {
		slog.Error("failed to write health response", "error", err)
	}
}
