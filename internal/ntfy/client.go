package ntfy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/fudoge/ntfy-gateway/internal/domain"
)

type Client struct {
	endpoint   string
	token      string
	httpClient *http.Client
}

func NewClient(endpoint, token string, httpClient *http.Client) *Client {
	return &Client{
		endpoint:   strings.TrimRight(endpoint, "/"),
		token:      token,
		httpClient: httpClient,
	}
}

func (c *Client) Publish(ctx context.Context, topic string, event domain.Event) error {
	requestURL := c.endpoint + "/" + url.PathEscape(topic)
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		requestURL,
		strings.NewReader(event.Message),
	)
	if err != nil {
		return fmt.Errorf("create ntfy request: %w", err)
	}

	request.Header.Set("Content-Type", "text/plain; charset=utf-8")
	request.Header.Set("X-Title", event.Title)
	if tag := severityTag(event.Severity); tag != "" {
		request.Header.Set("X-Tags", tag)
	}
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send ntfy request: %w", err)
	}

	_, discardErr := io.Copy(io.Discard, response.Body)
	closeErr := response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("ntfy returned status %s", response.Status)
	}
	if discardErr != nil {
		return fmt.Errorf("read ntfy response: %w", discardErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close ntfy response: %w", closeErr)
	}

	return nil
}

func severityTag(severity string) string {
	switch strings.ToLower(severity) {
	case "critical", "fatal":
		return "rotating_light"
	case "error":
		return "x"
	case "warning", "warn":
		return "warning"
	case "success":
		return "white_check_mark"
	case "info":
		return "information_source"
	default:
		return ""
	}
}
