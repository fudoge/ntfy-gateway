package service

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"

	"github.com/fudoge/ntfy-gateway/internal/domain"
)

var (
	ErrUnauthorized    = errors.New("unauthorized source")
	ErrDecoderNotFound = errors.New("decoder not found")
	ErrInvalidPayload  = errors.New("invalid payload")
	ErrPublish         = errors.New("publish notification")
)

type Decoder interface {
	Decode(io.Reader) (domain.Event, error)
}

type Publisher interface {
	Publish(context.Context, string, domain.Event) error
}

type Source struct {
	Type   string
	Topic  string
	Secret string
}

type Gateway struct {
	sources   map[string]Source
	decoders  map[string]Decoder
	publisher Publisher
}

func NewGateway(
	sources map[string]Source,
	decoders map[string]Decoder,
	publisher Publisher,
) *Gateway {
	return &Gateway{
		sources:   sources,
		decoders:  decoders,
		publisher: publisher,
	}
}

func (g *Gateway) Dispatch(
	ctx context.Context,
	sourceID string,
	credential string,
	reader io.Reader,
) error {
	source, ok := g.sources[sourceID]
	if !ok || !validCredential(source.Secret, credential) {
		return ErrUnauthorized
	}

	decoder, ok := g.decoders[source.Type]
	if !ok {
		return fmt.Errorf("%w: %q", ErrDecoderNotFound, source.Type)
	}

	event, err := decoder.Decode(reader)
	if err != nil {
		return fmt.Errorf("%w for source %q: %w", ErrInvalidPayload, sourceID, err)
	}

	event.SourceID = sourceID
	if err := g.publisher.Publish(ctx, source.Topic, event); err != nil {
		return fmt.Errorf("%w for source %q: %w", ErrPublish, sourceID, err)
	}

	return nil
}

func validCredential(expected, actual string) bool {
	if expected == "" || actual == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}
