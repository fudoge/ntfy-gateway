package flux

import (
	"strings"
	"testing"
	"time"
)

func TestDecoderDecode(t *testing.T) {
	const body = `{
  "involvedObject": {
    "apiVersion": "kustomize.toolkit.fluxcd.io/v1",
    "kind": "Kustomization",
    "name": "apps",
    "namespace": "flux-system",
    "uid": "example-uid",
    "resourceVersion": "42"
  },
  "metadata": {"revision": "main@sha1:abc123"},
  "severity": "info",
  "reason": "ReconciliationSucceeded",
  "message": "Applied revision: main@sha1:abc123",
  "reportingController": "kustomize-controller",
  "reportingInstance": "kustomize-controller-example",
  "timestamp": "2026-09-17T12:00:00Z"
}`

	event, err := (Decoder{}).Decode(strings.NewReader(body))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if event.Type != "ReconciliationSucceeded" {
		t.Errorf("Type = %q", event.Type)
	}
	if event.Title != "Kustomization/apps" {
		t.Errorf("Title = %q", event.Title)
	}
	if event.Severity != "info" {
		t.Errorf("Severity = %q", event.Severity)
	}
	if event.Timestamp != time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC) {
		t.Errorf("Timestamp = %v", event.Timestamp)
	}
	if event.Metadata["revision"] != "main@sha1:abc123" {
		t.Errorf("Metadata = %#v", event.Metadata)
	}
}
