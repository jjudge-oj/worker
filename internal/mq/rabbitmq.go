package mq

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/jjudge-oj/worker/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQClient wraps a RabbitMQ connection/channel pair.
type RabbitMQClient struct {
	conn            *amqp.Connection
	channel         *amqp.Channel
	queueDurable    bool
	queueAutoDelete bool
	prefetchCount   int
}

// NewRabbitMQClient constructs a RabbitMQ client from config.
func NewRabbitMQClient(cfg *config.RabbitMQConfig) (*RabbitMQClient, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, errors.New("rabbitmq url is required")
	}

	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	if cfg.PrefetchCount > 0 {
		if err := ch.Qos(cfg.PrefetchCount, 0, false); err != nil {
			_ = ch.Close()
			_ = conn.Close()
			return nil, err
		}
	}

	return &RabbitMQClient{
		conn:            conn,
		channel:         ch,
		queueDurable:    cfg.QueueDurable,
		queueAutoDelete: cfg.QueueAutoDelete,
		prefetchCount:   cfg.PrefetchCount,
	}, nil
}

// Publish sends a message to the named queue.
func (r *RabbitMQClient) Publish(ctx context.Context, channel string, data []byte, attrs map[string]string) (string, error) {
	if strings.TrimSpace(channel) == "" {
		return "", errors.New("rabbitmq channel is required")
	}

	if _, err := r.declareQueue(channel); err != nil {
		return "", err
	}

	headers := amqp.Table{}
	for key, value := range attrs {
		headers[key] = value
	}

	messageID := newMessageID()
	err := r.channel.PublishWithContext(ctx, "", channel, false, false, amqp.Publishing{
		ContentType: "application/octet-stream",
		MessageId:   messageID,
		Headers:     headers,
		Body:        data,
	})
	if err != nil {
		return "", err
	}
	return messageID, nil
}

// Subscribe consumes messages from the named queue.
func (r *RabbitMQClient) Subscribe(ctx context.Context, channel string, handler Handler) error {
	if strings.TrimSpace(channel) == "" {
		return errors.New("rabbitmq channel is required")
	}

	if _, err := r.declareQueue(channel); err != nil {
		return err
	}

	consumerTag := fmt.Sprintf("consumer-%s", newMessageID())
	deliveries, err := r.channel.Consume(channel, consumerTag, false, false, false, false, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = r.channel.Cancel(consumerTag, false)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case delivery, ok := <-deliveries:
			if !ok {
				return errors.New("rabbitmq delivery channel closed")
			}
			message := Message{
				ID:         delivery.MessageId,
				Data:       delivery.Body,
				Attributes: headersToAttributes(delivery.Headers),
			}
			if err := handler(ctx, message); err != nil {
				_ = delivery.Nack(false, true)
				continue
			}
			_ = delivery.Ack(false)
		}
	}
}

// Close closes the underlying channel and connection.
func (r *RabbitMQClient) Close() error {
	if r.channel != nil {
		_ = r.channel.Close()
	}
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

func (r *RabbitMQClient) declareQueue(name string) (amqp.Queue, error) {
	return r.channel.QueueDeclare(
		name,
		r.queueDurable,
		r.queueAutoDelete,
		false,
		false,
		nil,
	)
}

func headersToAttributes(headers amqp.Table) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	attrs := make(map[string]string, len(headers))
	for key, value := range headers {
		switch typed := value.(type) {
		case string:
			attrs[key] = typed
		case []byte:
			attrs[key] = string(typed)
		default:
			attrs[key] = fmt.Sprint(value)
		}
	}
	return attrs
}

func newMessageID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(buf[:])
}
