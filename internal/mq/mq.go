package mq

import "context"

// Message represents a broker-agnostic payload delivered to subscribers.
type Message struct {
	ID         string
	Data       []byte
	Attributes map[string]string
}

// Handler processes a message. Return an error to signal a retry/nack.
type Handler func(ctx context.Context, msg Message) error

// Backend defines the broker-agnostic operations used by the app.
type Backend interface {
	Publish(ctx context.Context, channel string, data []byte, attrs map[string]string) (string, error)
	Subscribe(ctx context.Context, channel string, handler Handler) error
	Close() error
}

// MQ wraps a backend with a stable API.
type MQ struct {
	backend Backend
}

// New constructs an MQ wrapper for the provided backend.
func New(backend Backend) *MQ {
	return &MQ{backend: backend}
}

// Publish sends a message to the named channel.
func (m *MQ) Publish(ctx context.Context, channel string, data []byte, attrs map[string]string) (string, error) {
	return m.backend.Publish(ctx, channel, data, attrs)
}

// Subscribe consumes messages from the named channel.
func (m *MQ) Subscribe(ctx context.Context, channel string, handler Handler) error {
	return m.backend.Subscribe(ctx, channel, handler)
}

// Close closes the underlying backend.
func (m *MQ) Close() error {
	return m.backend.Close()
}
