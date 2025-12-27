package redis

import (
	"context"
	"encoding/json"

	"uois-gateway/pkg/errors"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// StreamClient interface for Redis stream operations
type StreamClient interface {
	XAdd(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd
}

// EventPublisher publishes events to Redis streams
type EventPublisher struct {
	redis  StreamClient
	logger *zap.Logger
}

// NewEventPublisher creates a new event publisher
func NewEventPublisher(rdb StreamClient, logger *zap.Logger) *EventPublisher {
	return &EventPublisher{
		redis:  rdb,
		logger: logger,
	}
}

// PublishEvent publishes an event to a Redis stream
func (p *EventPublisher) PublishEvent(ctx context.Context, stream string, event interface{}) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return errors.WrapDomainError(err, 65020, "event serialization failed", "failed to marshal event")
	}

	args := &redis.XAddArgs{
		Stream: stream,
		Values: map[string]interface{}{
			"data": string(eventJSON),
		},
	}

	if err := p.redis.XAdd(ctx, args).Err(); err != nil {
		return errors.WrapDomainError(err, 65011, "event publication failed", "redis error")
	}

	return nil
}
