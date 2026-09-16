package config

import (
	"errors"
	"os"
)

type Config struct {
	BindAddress string
	Endpoint    string
}

func Load() (*Config, error) {
	endpoint := os.Getenv("NTFY_ENDPOINT")

	if len(endpoint) == 0 {
		return nil, errors.New("no endpoint given")
	}

	c := &Config{
		BindAddress: ":8080",
		Endpoint:    endpoint,
	}

	return c, nil
}
