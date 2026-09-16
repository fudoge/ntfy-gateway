package generic

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/fudoge/ntfy-gateway/internal/domain"
)

type Payload struct {
	Type      string            `json:"type"`
	Title     string            `json:"title"`
	Message   string            `json:"message"`
	Severity  string            `json:"severity"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata"`
}

type Decoder struct{}

func (Decoder) Decode(reader io.Reader) (domain.Event, error) {
	var payload Payload
	if err := json.NewDecoder(reader).Decode(&payload); err != nil {
		return domain.Event{}, fmt.Errorf("decode generic payload: %w", err)
	}

	return domain.Event{
		Type:      payload.Type,
		Title:     payload.Title,
		Message:   payload.Message,
		Severity:  payload.Severity,
		Timestamp: payload.Timestamp,
		Metadata:  payload.Metadata,
	}, nil
}
