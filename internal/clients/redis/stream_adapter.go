package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// StreamConsumerAdapter adapts redis.Client to match StreamConsumerClient interface
type StreamConsumerAdapter struct {
	client *redis.Client
}

// NewStreamConsumerAdapter creates a new adapter
func NewStreamConsumerAdapter(client *redis.Client) *StreamConsumerAdapter {
	return &StreamConsumerAdapter{client: client}
}

// XReadGroup implements StreamConsumerClient interface
func (a *StreamConsumerAdapter) XReadGroup(ctx context.Context, args *redis.XReadGroupArgs) *redis.XStreamSliceCmd {
	return a.client.XReadGroup(ctx, args)
}

// XAck implements StreamConsumerClient interface (adapts variadic to single ID)
func (a *StreamConsumerAdapter) XAck(ctx context.Context, stream, group, id string) *redis.IntCmd {
	return a.client.XAck(ctx, stream, group, id)
}

// ConsumerGroupAdapter adapts redis.Client to match ConsumerGroupClient interface
type ConsumerGroupAdapter struct {
	client *redis.Client
}

// NewConsumerGroupAdapter creates a new adapter
func NewConsumerGroupAdapter(client *redis.Client) *ConsumerGroupAdapter {
	return &ConsumerGroupAdapter{client: client}
}

// XGroupCreate implements ConsumerGroupClient interface
func (a *ConsumerGroupAdapter) XGroupCreate(ctx context.Context, stream, group, start string, mkStream bool) *redis.StatusCmd {
	if mkStream {
		return a.client.XGroupCreateMkStream(ctx, stream, group, start)
	}
	return a.client.XGroupCreate(ctx, stream, group, start)
}
