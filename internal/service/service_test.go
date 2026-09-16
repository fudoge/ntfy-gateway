package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/fudoge/ntfy-gateway/internal/domain"
)

type decoderStub struct {
	event domain.Event
	err   error
}

func (d decoderStub) Decode(io.Reader) (domain.Event, error) {
	return d.event, d.err
}

type publisherStub struct {
	topic string
	event domain.Event
	err   error
}

func (p *publisherStub) Publish(_ context.Context, topic string, event domain.Event) error {
	p.topic = topic
	p.event = event
	return p.err
}

func TestGatewayDispatch(t *testing.T) {
	publisher := &publisherStub{}
	gateway := NewGateway(
		map[string]Source{
			"flux-production": {
				Type:   "flux",
				Topic:  "production-alerts",
				Secret: "webhook-secret",
			},
		},
		map[string]Decoder{
			"flux": decoderStub{event: domain.Event{Message: "deployed"}},
		},
		publisher,
	)

	err := gateway.Dispatch(
		context.Background(),
		"flux-production",
		"webhook-secret",
		strings.NewReader(`{}`),
	)
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}

	if publisher.topic != "production-alerts" {
		t.Errorf("published topic = %q", publisher.topic)
	}
	if publisher.event.SourceID != "flux-production" {
		t.Errorf("published SourceID = %q", publisher.event.SourceID)
	}
	if publisher.event.Message != "deployed" {
		t.Errorf("published Message = %q", publisher.event.Message)
	}
}

func TestGatewayDispatchRejectsInvalidCredential(t *testing.T) {
	gateway := NewGateway(
		map[string]Source{
			"flux-production": {Secret: "webhook-secret"},
		},
		nil,
		&publisherStub{},
	)

	err := gateway.Dispatch(
		context.Background(),
		"flux-production",
		"wrong-secret",
		strings.NewReader(`{}`),
	)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Dispatch() error = %v, want ErrUnauthorized", err)
	}
}

func TestGatewayDispatchRejectsUnknownSource(t *testing.T) {
	gateway := NewGateway(nil, nil, &publisherStub{})

	err := gateway.Dispatch(
		context.Background(),
		"unknown",
		"webhook-secret",
		strings.NewReader(`{}`),
	)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Dispatch() error = %v, want ErrUnauthorized", err)
	}
}
