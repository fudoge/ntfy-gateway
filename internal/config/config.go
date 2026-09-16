package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	defaultPath = "config.yaml"
	pathEnv     = "CONFIG_PATH"
)

var (
	sourceNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)
	topicPattern      = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
)

type Config struct {
	Server  ServerConfig            `yaml:"server"`
	Ntfy    NtfyConfig              `yaml:"ntfy"`
	Sources map[string]SourceConfig `yaml:"sources"`
}

type ServerConfig struct {
	Address string `yaml:"address"`
}

type NtfyConfig struct {
	Endpoint string `yaml:"endpoint"`
	TokenEnv string `yaml:"token_env"`
}

type SourceConfig struct {
	Type      string `yaml:"type"`
	Topic     string `yaml:"topic"`
	SecretEnv string `yaml:"secret_env"`
}

func Load() (*Config, error) {
	path := os.Getenv(pathEnv)
	if path == "" {
		path = defaultPath
	}

	return LoadFile(path)
}

func LoadFile(path string) (*Config, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	cfg := &Config{
		Server: ServerConfig{Address: ":8080"},
	}

	decoder := yaml.NewDecoder(bytes.NewReader(contents))
	decoder.KnownFields(true)

	if err := decoder.Decode(cfg); err != nil {
		return nil, fmt.Errorf("decode config %q: %w", path, err)
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err != nil {
			return nil, fmt.Errorf("decode config %q: %w", path, err)
		}
		return nil, fmt.Errorf("decode config %q: multiple YAML documents are not allowed", path)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config %q: %w", path, err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.Server.Address) == "" {
		return errors.New("server.address is required")
	}

	if err := validateEndpoint(c.Ntfy.Endpoint); err != nil {
		return err
	}

	if err := validateEnvReference("ntfy.token_env", c.Ntfy.TokenEnv, false); err != nil {
		return err
	}

	if len(c.Sources) == 0 {
		return errors.New("at least one source is required")
	}

	for name, source := range c.Sources {
		if err := validateSource(name, source); err != nil {
			return err
		}
	}

	return nil
}

func validateEndpoint(rawURL string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.New("ntfy.endpoint must be a valid HTTP(S) URL")
	}
	return nil
}

func validateSource(name string, source SourceConfig) error {
	if !sourceNamePattern.MatchString(name) {
		return fmt.Errorf("source %q: name must contain only letters, numbers, hyphens, and underscores", name)
	}

	switch source.Type {
	case "flux", "generic":
	default:
		return fmt.Errorf("source %q: unsupported type %q", name, source.Type)
	}

	if !topicPattern.MatchString(source.Topic) {
		return fmt.Errorf("source %q: topic must contain 1-64 letters, numbers, hyphens, or underscores", name)
	}

	if err := validateEnvReference("secret_env", source.SecretEnv, true); err != nil {
		return fmt.Errorf("source %q: %w", name, err)
	}

	return nil
}

func validateEnvReference(field, name string, required bool) error {
	if name == "" {
		if required {
			return fmt.Errorf("%s is required", field)
		}
		return nil
	}

	if value, ok := os.LookupEnv(name); !ok || value == "" {
		return fmt.Errorf("%s references unset or empty environment variable %q", field, name)
	}

	return nil
}
