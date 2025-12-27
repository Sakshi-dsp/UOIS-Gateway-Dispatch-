package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Client wraps Redis client for use by services
type Client struct {
	rdb  *redis.Client
	logger *zap.Logger
}

// NewClient creates a new Redis client
func NewClient(addr string, password string, db int, logger *zap.Logger) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Client{
		rdb:    rdb,
		logger: logger,
	}, nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.rdb.Close()
}

// GetClient returns the underlying Redis client
func (c *Client) GetClient() *redis.Client {
	return c.rdb
}

// Incr increments a key (for rate limiting)
func (c *Client) Incr(ctx context.Context, key string) *redis.IntCmd {
	return c.rdb.Incr(ctx, key)
}

// Expire sets expiration on a key
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	return c.rdb.Expire(ctx, key, expiration)
}

// TTL returns time to live for a key
func (c *Client) TTL(ctx context.Context, key string) *redis.DurationCmd {
	return c.rdb.TTL(ctx, key)
}

