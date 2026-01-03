package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisClient interface for Redis operations
type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
}

// Service provides caching functionality using Redis
type Service struct {
	redis  RedisClient
	logger *zap.Logger
	ttl    time.Duration
}

// NewService creates a new cache service
func NewService(redis RedisClient, ttl time.Duration, logger *zap.Logger) *Service {
	return &Service{
		redis:  redis,
		logger: logger,
		ttl:    ttl,
	}
}

// Get retrieves a value from cache
func (s *Service) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	val, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("cache get error: %w", err)
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return false, fmt.Errorf("cache unmarshal error: %w", err)
	}

	return true, nil
}

// Set stores a value in cache
func (s *Service) Set(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache marshal error: %w", err)
	}

	if err := s.redis.Set(ctx, key, data, s.ttl).Err(); err != nil {
		return fmt.Errorf("cache set error: %w", err)
	}

	return nil
}

// Delete removes a value from cache
func (s *Service) Delete(ctx context.Context, key string) error {
	return s.redis.Del(ctx, key).Err()
}

// Exists checks if a key exists in cache
func (s *Service) Exists(ctx context.Context, key string) (bool, error) {
	count, err := s.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// BuildKey builds a cache key with prefix
func BuildKey(prefix string, parts ...string) string {
	key := prefix
	for _, part := range parts {
		key += ":" + part
	}
	return key
}
