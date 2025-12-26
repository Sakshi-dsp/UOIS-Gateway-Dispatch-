package auth

import (
	"context"
	"fmt"
	"time"

	"uois-gateway/internal/config"
	"uois-gateway/pkg/errors"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisClient interface {
	Incr(ctx context.Context, key string) *redis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	TTL(ctx context.Context, key string) *redis.DurationCmd
}

type RateLimitService struct {
	redis  RedisClient
	config config.RateLimitConfig
	logger *zap.Logger
}

func NewRateLimitService(redis RedisClient, cfg config.RateLimitConfig, logger *zap.Logger) *RateLimitService {
	return &RateLimitService{
		redis:  redis,
		config: cfg,
		logger: logger,
	}
}

func (s *RateLimitService) CheckRateLimit(ctx context.Context, clientID string) (allowed bool, remaining int64, resetAt time.Time, err error) {
	if !s.config.Enabled {
		return true, -1, time.Time{}, nil
	}

	// Key format: {prefix}:{clientID}
	// Future extensibility: Consider {prefix}:{clientID}:{endpoint} or {prefix}:{clientID}:{ip}
	// for per-endpoint or per-IP rate limiting
	key := fmt.Sprintf("%s:%s", s.config.RedisKeyPrefix, clientID)
	windowDuration := time.Duration(s.config.WindowSeconds) * time.Second

	countCmd := s.redis.Incr(ctx, key)
	if err := countCmd.Err(); err != nil {
		return false, 0, time.Time{},
			errors.WrapDomainError(err, 65011, "rate limiting unavailable", "redis error")
	}

	currentCount := countCmd.Val()

	if currentCount == 1 {
		if err := s.redis.Expire(ctx, key, windowDuration).Err(); err != nil {
			s.logger.Warn("failed to set expire on rate limit key", zap.Error(err))
		}
	}

	ttlCmd := s.redis.TTL(ctx, key)
	ttlDuration := ttlCmd.Val()
	if ttlDuration < 0 {
		ttlDuration = windowDuration
	}

	burstLimit := int64(s.config.Burst)
	// Note: RequestsPerMinute is used as "requests per window" (WindowSeconds)
	// Consider renaming to RequestsPerWindow for clarity if config is refactored
	limit := int64(s.config.RequestsPerMinute)

	if currentCount > burstLimit {
		resetAt = time.Now().Add(ttlDuration)
		// TODO: Add metrics counter: rate_limit_exceeded_total{reason="burst", client_id=clientID}
		return false, 0, resetAt, nil
	}

	if currentCount > limit {
		resetAt = time.Now().Add(ttlDuration)
		// TODO: Add metrics counter: rate_limit_exceeded_total{reason="steady_state", client_id=clientID}
		return false, 0, resetAt, nil
	}

	remaining = limit - currentCount
	if remaining < 0 {
		remaining = 0
	}

	resetAt = time.Now().Add(ttlDuration)
	return true, remaining, resetAt, nil
}

func (s *RateLimitService) GetRateLimitError(ctx context.Context, clientID string) error {
	return errors.NewDomainError(65012, "rate limit exceeded", fmt.Sprintf("client %s has exceeded rate limit", clientID))
}
