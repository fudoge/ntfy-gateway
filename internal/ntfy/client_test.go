package ntfy

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/fudoge/ntfy-gateway/internal/domain"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestClientPublish(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
			if request.Method != http.MethodPost {
				t.Errorf("method = %q", request.Method)
			}
			if request.URL.String() != "https://ntfy.example.com/production-alerts" {
				t.Errorf("URL = %q", request.URL.String())
			}
			if request.Header.Get("Authorization") != "Bearer ntfy-token" {
				t.Errorf("Authorization = %q", request.Header.Get("Authorization"))
			}
			if request.Header.Get("X-Title") != "Kustomization/apps" {
				t.Errorf("X-Title = %q", request.Header.Get("X-Title"))
			}

			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Errorf("read request body: %v", err)
			}
			if string(body) != "deployment completed" {
				t.Errorf("body = %q", body)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(strings.NewReader(`{}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	client := NewClient("https://ntfy.example.com", "ntfy-token", httpClient)
	err := client.Publish(context.Background(), "production-alerts", domain.Event{
		Title:   "Kustomization/apps",
		Message: "deployment completed",
	})
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
}
