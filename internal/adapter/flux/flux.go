package flux

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/fudoge/ntfy-gateway/internal/domain"
)

type Payload struct {
	InvolvedObject      KubernetesObject  `json:"involvedObject"`
	Metadata            map[string]string `json:"metadata"`
	Severity            string            `json:"severity"`
	Reason              string            `json:"reason"`
	Message             string            `json:"message"`
	ReportingController string            `json:"reportingController"`
	ReportingInstance   string            `json:"reportingInstance"`
	Timestamp           time.Time         `json:"timestamp"`
}

type Decoder struct{}

func (Decoder) Decode(reader io.Reader) (domain.Event, error) {
	var payload Payload
	if err := json.NewDecoder(reader).Decode(&payload); err != nil {
		return domain.Event{}, fmt.Errorf("decode flux payload: %w", err)
	}

	return payload.Event(""), nil
}

type KubernetesObject struct {
	APIVersion      string `json:"apiVersion"`
	Kind            string `json:"kind"`
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	UID             string `json:"uid"`
	ResourceVersion string `json:"resourceVersion"`
}

func (p Payload) Event(sourceID string) domain.Event {
	return domain.Event{
		SourceID:  sourceID,
		Type:      p.Reason,
		Title:     p.InvolvedObject.Kind + "/" + p.InvolvedObject.Name,
		Message:   p.Message,
		Severity:  p.Severity,
		Timestamp: p.Timestamp,
		Metadata:  p.Metadata,
	}
}
