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
	sourceID := r.PathValue("source")
	logger = logger.With(
		slog.Group(
			"request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		),
	)

	if !isJSON(r.Header.Get("Content-Type")) {
		msg := "content type must be application/json"
		logger.Info(
			msg,
			slog.Group("response", slog.Int("status", http.StatusUnsupportedMediaType)),
		)
		http.Error(w, msg, http.StatusUnsupportedMediaType)
		return
	}

	credential, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		w.Header().Set("WWW-Authenticate", "Bearer")
		msg := "unauthorized"
		logger.Info(
			msg,
			slog.Group("response", slog.Int("status", http.StatusUnauthorized)),
		)
		http.Error(w, msg, http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxWebhookBodySize)

	err := gateway.Dispatch(r.Context(), sourceID, credential, r.Body)
	if err == nil {
		logger.Info(
			"webhook dispatched",
			slog.Group("response", slog.Int("status", http.StatusAccepted)),
		)
		w.WriteHeader(http.StatusAccepted)
		return
	}

	var maxBytesError *http.MaxBytesError
	switch {
	case errors.Is(err, service.ErrUnauthorized):
		w.Header().Set("WWW-Authenticate", "Bearer")
		msg := "unauthorized"
		logger.Info(
			msg,
			slog.Group("response", slog.Int("status", http.StatusUnauthorized)),
			slog.Any("error", err),
		)
		http.Error(w, msg, http.StatusUnauthorized)
	case errors.As(err, &maxBytesError):
		msg := "request body too large"
		logger.Warn(
			msg,
			slog.Group("response", slog.Int("status", http.StatusRequestEntityTooLarge)),
			slog.Any("error", err),
		)
		http.Error(w, msg, http.StatusRequestEntityTooLarge)
	case errors.Is(err, service.ErrInvalidPayload):
		msg := "invalid payload"
		logger.Warn(
			msg,
			slog.Group("response", slog.Int("status", http.StatusBadRequest)),
			slog.Any("error", err),
		)
		http.Error(w, msg, http.StatusBadRequest)
	case errors.Is(err, service.ErrPublish):
		logger.Error(
			"failed to publish webhook",
			slog.Group("response", slog.Int("status", http.StatusBadGateway)),
			slog.Any("error", err),
		)
		http.Error(w, "upstream notification service failed", http.StatusBadGateway)
	default:
		logger.Error(
			"failed to dispatch webhook",
			slog.Group("response", slog.Int("status", http.StatusInternalServerError)),
			slog.Any("error", err),
		)
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
