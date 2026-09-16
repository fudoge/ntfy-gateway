package handler

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type dispatcherStub struct {
	sourceID   string
	credential string
	err        error
}

func (d *dispatcherStub) Dispatch(
	_ context.Context,
	sourceID string,
	credential string,
	_ io.Reader,
) error {
	d.sourceID = sourceID
	d.credential = credential
	return d.err
}

func TestWebhook(t *testing.T) {
	dispatcher := &dispatcherStub{}
	handler := NewHandler(dispatcher, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/webhooks/flux-production",
		strings.NewReader(`{"message":"deployed"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer webhook-secret")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
	if dispatcher.sourceID != "flux-production" {
		t.Errorf("sourceID = %q", dispatcher.sourceID)
	}
	if dispatcher.credential != "webhook-secret" {
		t.Errorf("credential = %q", dispatcher.credential)
	}
}

func TestWebhookRequiresBearerToken(t *testing.T) {
	handler := NewHandler(&dispatcherStub{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/webhooks/flux-production",
		strings.NewReader(`{}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}
