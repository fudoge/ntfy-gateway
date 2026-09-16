package generic

import (
	"strings"
	"testing"
)

func TestDecoderDecode(t *testing.T) {
	event, err := (Decoder{}).Decode(strings.NewReader(`{
  "type": "deployment",
  "title": "Production deployment",
  "message": "Version 1.2.3 deployed",
  "severity": "info",
  "timestamp": "2026-09-17T12:00:00Z",
  "metadata": {"version": "1.2.3"}
}`))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if event.Title != "Production deployment" {
		t.Errorf("Title = %q", event.Title)
	}
	if event.Message != "Version 1.2.3 deployed" {
		t.Errorf("Message = %q", event.Message)
	}
	if event.Metadata["version"] != "1.2.3" {
		t.Errorf("Metadata = %#v", event.Metadata)
	}
}
