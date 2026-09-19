package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fudoge/ntfy-gateway/internal/service"
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

func TestWebhookLogsResponse(t *testing.T) {
	tests := []struct {
		name          string
		contentType   string
		authorization string
		dispatchErr   error
		wantStatus    int
		wantLevel     string
		wantMessage   string
		wantError     bool
	}{
		{
			name:          "accepted",
			contentType:   "application/json",
			authorization: "Bearer webhook-secret",
			wantStatus:    http.StatusAccepted,
			wantLevel:     "INFO",
			wantMessage:   "webhook dispatched",
		},
		{
			name:          "unsupported content type",
			contentType:   "text/plain",
			authorization: "Bearer webhook-secret",
			wantStatus:    http.StatusUnsupportedMediaType,
			wantLevel:     "INFO",
			wantMessage:   "content type must be application/json",
		},
		{
			name:        "missing authorization",
			contentType: "application/json",
			wantStatus:  http.StatusUnauthorized,
			wantLevel:   "INFO",
			wantMessage: "unauthorized",
		},
		{
			name:          "unauthorized source",
			contentType:   "application/json",
			authorization: "Bearer webhook-secret",
			dispatchErr:   service.ErrUnauthorized,
			wantStatus:    http.StatusUnauthorized,
			wantLevel:     "INFO",
			wantMessage:   "unauthorized",
			wantError:     true,
		},
		{
			name:          "body too large",
			contentType:   "application/json",
			authorization: "Bearer webhook-secret",
			dispatchErr:   &http.MaxBytesError{Limit: maxWebhookBodySize},
			wantStatus:    http.StatusRequestEntityTooLarge,
			wantLevel:     "WARN",
			wantMessage:   "request body too large",
			wantError:     true,
		},
		{
			name:          "invalid payload",
			contentType:   "application/json",
			authorization: "Bearer webhook-secret",
			dispatchErr:   service.ErrInvalidPayload,
			wantStatus:    http.StatusBadRequest,
			wantLevel:     "WARN",
			wantMessage:   "invalid payload",
			wantError:     true,
		},
		{
			name:          "publish failure",
			contentType:   "application/json",
			authorization: "Bearer webhook-secret",
			dispatchErr:   service.ErrPublish,
			wantStatus:    http.StatusBadGateway,
			wantLevel:     "ERROR",
			wantMessage:   "failed to publish webhook",
			wantError:     true,
		},
		{
			name:          "unexpected failure",
			contentType:   "application/json",
			authorization: "Bearer webhook-secret",
			dispatchErr:   errors.New("unexpected failure"),
			wantStatus:    http.StatusInternalServerError,
			wantLevel:     "ERROR",
			wantMessage:   "failed to dispatch webhook",
			wantError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logOutput bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logOutput, nil))
			handler := NewHandler(&dispatcherStub{err: tt.dispatchErr}, logger)
			request := httptest.NewRequest(
				http.MethodPost,
				"/api/webhooks/flux-production",
				strings.NewReader(`{}`),
			)
			request.Header.Set("Content-Type", tt.contentType)
			if tt.authorization != "" {
				request.Header.Set("Authorization", tt.authorization)
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}

			var entry struct {
				Level     string `json:"level"`
				Message   string `json:"msg"`
				Component string `json:"component"`
				Request   struct {
					Method string `json:"method"`
					Path   string `json:"path"`
				} `json:"request"`
				Response struct {
					Status int `json:"status"`
				} `json:"response"`
				Error string `json:"error"`
			}
			if err := json.NewDecoder(&logOutput).Decode(&entry); err != nil {
				t.Fatalf("decode log: %v", err)
			}
			if entry.Level != tt.wantLevel {
				t.Errorf("log level = %q, want %q", entry.Level, tt.wantLevel)
			}
			if entry.Message != tt.wantMessage {
				t.Errorf("log message = %q, want %q", entry.Message, tt.wantMessage)
			}
			if entry.Component != "handler" {
				t.Errorf("log component = %q, want handler", entry.Component)
			}
			if entry.Request.Method != http.MethodPost {
				t.Errorf("request method = %q, want %q", entry.Request.Method, http.MethodPost)
			}
			if entry.Request.Path != "/api/webhooks/flux-production" {
				t.Errorf("request path = %q", entry.Request.Path)
			}
			if entry.Response.Status != tt.wantStatus {
				t.Errorf("response status = %d, want %d", entry.Response.Status, tt.wantStatus)
			}
			if (entry.Error != "") != tt.wantError {
				t.Errorf("error = %q, want present %t", entry.Error, tt.wantError)
			}
		})
	}
}
