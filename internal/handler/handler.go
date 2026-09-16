package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"

	"github.com/fudoge/ntfy-gateway/internal/service"
)

const maxWebhookBodySize = 1 << 20

type Dispatcher interface {
	Dispatch(ctx context.Context, sourceID, credential string, reader io.Reader) error
}

func NewHandler(gateway Dispatcher, logger *slog.Logger) *http.ServeMux {
	logger = logger.With(slog.String("component", "handler"))
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		health(logger, w, r)
	})

	mux.HandleFunc("POST /api/webhooks/{source}", func(w http.ResponseWriter, r *http.Request) {
		handleWebhook(gateway, logger, w, r)
	})

	return mux
}

func handleWebhook(gateway Dispatcher, logger *slog.Logger, w http.ResponseWriter, r *http.Request) {
	if !isJSON(r.Header.Get("Content-Type")) {
		http.Error(w, "content type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	credential, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxWebhookBodySize)
	sourceID := r.PathValue("source")

	err := gateway.Dispatch(r.Context(), sourceID, credential, r.Body)
	if err == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	var maxBytesError *http.MaxBytesError
	switch {
	case errors.Is(err, service.ErrUnauthorized):
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	case errors.As(err, &maxBytesError):
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
	case errors.Is(err, service.ErrInvalidPayload):
		http.Error(w, "invalid payload", http.StatusBadRequest)
	case errors.Is(err, service.ErrPublish):
		logger.Error("failed to publish webhook", "source", sourceID, "error", err)
		http.Error(w, "upstream notification service failed", http.StatusBadGateway)
	default:
		logger.Error("failed to dispatch webhook", "source", sourceID, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func isJSON(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "application/json"
}

func bearerToken(authorization string) (string, bool) {
	parts := strings.Fields(authorization)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}

	return parts[1], true
}

func health(logger *slog.Logger, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprintln(w, "ok"); err != nil {
		logger.Error("failed to write health response", "error", err)
	}
}
