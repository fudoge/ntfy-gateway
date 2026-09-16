package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFile(t *testing.T) {
	t.Setenv("NTFY_TOKEN", "ntfy-token")
	t.Setenv("FLUX_SECRET", "flux-secret")

	path := writeConfig(t, `
ntfy:
  endpoint: https://ntfy.example.com
  token_env: NTFY_TOKEN
sources:
  flux-production:
    type: flux
    topic: production-alerts
    secret_env: FLUX_SECRET
`)

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if cfg.Server.Address != ":8080" {
		t.Errorf("Server.Address = %q, want %q", cfg.Server.Address, ":8080")
	}

	source := cfg.Sources["flux-production"]
	if source.Type != "flux" || source.Topic != "production-alerts" {
		t.Errorf("source = %#v", source)
	}
}

func TestLoadFileRejectsUnknownField(t *testing.T) {
	path := writeConfig(t, `
server:
  adress: :8080
ntfy:
  endpoint: https://ntfy.example.com
sources: {}
`)

	_, err := LoadFile(path)
	if err == nil || !strings.Contains(err.Error(), "field adress not found") {
		t.Fatalf("LoadFile() error = %v, want unknown field error", err)
	}
}

func TestLoadFileRejectsMissingSecret(t *testing.T) {
	t.Setenv("MISSING_SECRET", "")

	path := writeConfig(t, `
ntfy:
  endpoint: https://ntfy.example.com
sources:
  monitoring:
    type: generic
    topic: monitoring-alerts
    secret_env: MISSING_SECRET
`)

	_, err := LoadFile(path)
	if err == nil || !strings.Contains(err.Error(), "MISSING_SECRET") {
		t.Fatalf("LoadFile() error = %v, want missing secret error", err)
	}
}

func writeConfig(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
