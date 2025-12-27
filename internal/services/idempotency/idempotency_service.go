package idempotency

import (
	"context"
	"fmt"
	"time"

	"uois-gateway/internal/config"
	"uois-gateway/pkg/errors"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisClient interface for Redis operations
type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
}

// Service handles request idempotency checks and storage
type Service struct {
	redis  RedisClient
	config config.Config
	logger *zap.Logger
}

// NewService creates a new idempotency service
func NewService(rdb RedisClient, cfg config.Config, logger *zap.Logger) *Service {
	return &Service{
		redis:  rdb,
		config: cfg,
		logger: logger,
	}
}

// buildKey constructs the Redis key with prefix for idempotency
func (s *Service) buildKey(key string) string {
	return fmt.Sprintf("%s:idempotency:%s", s.config.Redis.KeyPrefix, key)
}

// CheckIdempotency checks if a request with the given key was already processed
// Returns raw JSON bytes to preserve ONDC signature byte-exactness
func (s *Service) CheckIdempotency(ctx context.Context, key string) ([]byte, bool, error) {
	redisKey := s.buildKey(key)
	val, err := s.redis.Get(ctx, redisKey).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, errors.WrapDomainError(err, 65011, "idempotency check failed", "redis error")
	}

	return []byte(val), true, nil
}

// StoreIdempotency stores raw response bytes for idempotency checking
// Accepts raw JSON bytes to preserve ONDC signature byte-exactness
func (s *Service) StoreIdempotency(ctx context.Context, key string, responseBytes []byte, ttl time.Duration) error {
	redisKey := s.buildKey(key)
	if err := s.redis.Set(ctx, redisKey, responseBytes, ttl).Err(); err != nil {
		return errors.WrapDomainError(err, 65011, "idempotency storage failed", "redis error")
	}

	return nil
}
