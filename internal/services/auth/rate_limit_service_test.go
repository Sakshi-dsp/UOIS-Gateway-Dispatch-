package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"uois-gateway/internal/config"
	domainerrors "uois-gateway/pkg/errors"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) Incr(ctx context.Context, key string) *redis.IntCmd {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return redis.NewIntResult(0, errors.New("mock error"))
	}
	return args.Get(0).(*redis.IntCmd)
}

func (m *MockRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	args := m.Called(ctx, key, expiration)
	if args.Get(0) == nil {
		return redis.NewBoolResult(false, errors.New("mock error"))
	}
	return args.Get(0).(*redis.BoolCmd)
}

func (m *MockRedisClient) TTL(ctx context.Context, key string) *redis.DurationCmd {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return redis.NewDurationResult(0, errors.New("mock error"))
	}
	return args.Get(0).(*redis.DurationCmd)
}

func TestRateLimitService_CheckRateLimit_Success_FirstRequest(t *testing.T) {
	mockRedis := new(MockRedisClient)
	logger := zap.NewNop()
	cfg := config.RateLimitConfig{
		Enabled:           true,
		RedisKeyPrefix:    "rate_limit:uois",
		RequestsPerMinute: 60,
		Burst:             10,
		WindowSeconds:     60,
	}

	service := NewRateLimitService(mockRedis, cfg, logger)
	ctx := context.Background()
	clientID := "test-client-123"

	incrCmd := redis.NewIntResult(1, nil)
	expireCmd := redis.NewBoolResult(true, nil)
	ttlCmd := redis.NewDurationResult(60*time.Second, nil)
	mockRedis.On("Incr", ctx, "rate_limit:uois:test-client-123").Return(incrCmd)
	mockRedis.On("Expire", ctx, "rate_limit:uois:test-client-123", time.Duration(60)*time.Second).Return(expireCmd)
	mockRedis.On("TTL", ctx, "rate_limit:uois:test-client-123").Return(ttlCmd)

	allowed, remaining, resetAt, err := service.CheckRateLimit(ctx, clientID)

	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, int64(59), remaining)
	assert.NotZero(t, resetAt)
	mockRedis.AssertExpectations(t)
}

func TestRateLimitService_CheckRateLimit_Success_SubsequentRequest(t *testing.T) {
	mockRedis := new(MockRedisClient)
	logger := zap.NewNop()
	cfg := config.RateLimitConfig{
		Enabled:           true,
		RedisKeyPrefix:    "rate_limit:uois",
		RequestsPerMinute: 60,
		Burst:             10,
		WindowSeconds:     60,
	}

	service := NewRateLimitService(mockRedis, cfg, logger)
	ctx := context.Background()
	clientID := "test-client-123"

	incrCmd := redis.NewIntResult(5, nil)
	ttlCmd := redis.NewDurationResult(45*time.Second, nil)
	mockRedis.On("Incr", ctx, "rate_limit:uois:test-client-123").Return(incrCmd)
	mockRedis.On("TTL", ctx, "rate_limit:uois:test-client-123").Return(ttlCmd)

	allowed, remaining, resetAt, err := service.CheckRateLimit(ctx, clientID)

	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, int64(55), remaining)
	assert.NotZero(t, resetAt)
	mockRedis.AssertNotCalled(t, "Expire")
	mockRedis.AssertExpectations(t)
}

func TestRateLimitService_CheckRateLimit_Exceeded(t *testing.T) {
	mockRedis := new(MockRedisClient)
	logger := zap.NewNop()
	cfg := config.RateLimitConfig{
		Enabled:           true,
		RedisKeyPrefix:    "rate_limit:uois",
		RequestsPerMinute: 60,
		Burst:             10,
		WindowSeconds:     60,
	}

	service := NewRateLimitService(mockRedis, cfg, logger)
	ctx := context.Background()
	clientID := "test-client-123"

	incrCmd := redis.NewIntResult(61, nil)
	ttlCmd := redis.NewDurationResult(30*time.Second, nil)
	mockRedis.On("Incr", ctx, "rate_limit:uois:test-client-123").Return(incrCmd)
	mockRedis.On("TTL", ctx, "rate_limit:uois:test-client-123").Return(ttlCmd)

	allowed, remaining, resetAt, err := service.CheckRateLimit(ctx, clientID)

	assert.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, int64(0), remaining)
	assert.NotZero(t, resetAt)
	mockRedis.AssertNotCalled(t, "Expire")
	mockRedis.AssertExpectations(t)
}

func TestRateLimitService_CheckRateLimit_Disabled(t *testing.T) {
	mockRedis := new(MockRedisClient)
	logger := zap.NewNop()
	cfg := config.RateLimitConfig{
		Enabled: false,
	}

	service := NewRateLimitService(mockRedis, cfg, logger)
	ctx := context.Background()
	clientID := "test-client-123"

	allowed, remaining, resetAt, err := service.CheckRateLimit(ctx, clientID)

	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, int64(-1), remaining)
	assert.Zero(t, resetAt)
	mockRedis.AssertNotCalled(t, "Incr")
}

func TestRateLimitService_CheckRateLimit_RedisError(t *testing.T) {
	mockRedis := new(MockRedisClient)
	logger := zap.NewNop()
	cfg := config.RateLimitConfig{
		Enabled:           true,
		RedisKeyPrefix:    "rate_limit:uois",
		RequestsPerMinute: 60,
		Burst:             10,
		WindowSeconds:     60,
	}

	service := NewRateLimitService(mockRedis, cfg, logger)
	ctx := context.Background()
	clientID := "test-client-123"

	redisErr := redis.NewIntResult(0, errors.New("redis connection error"))
	mockRedis.On("Incr", ctx, "rate_limit:uois:test-client-123").Return(redisErr)

	allowed, remaining, resetAt, err := service.CheckRateLimit(ctx, clientID)

	assert.Error(t, err)
	assert.False(t, allowed)
	assert.Equal(t, int64(0), remaining)
	assert.Zero(t, resetAt)
	assert.True(t, domainerrors.IsDomainError(err))
	domainErr, ok := err.(*domainerrors.DomainError)
	assert.True(t, ok)
	assert.Equal(t, 65011, domainErr.Code)
	assert.Equal(t, 503, domainerrors.GetHTTPStatus(err))
	mockRedis.AssertExpectations(t)
}

func TestRateLimitService_CheckRateLimit_BurstLimit(t *testing.T) {
	mockRedis := new(MockRedisClient)
	logger := zap.NewNop()
	cfg := config.RateLimitConfig{
		Enabled:           true,
		RedisKeyPrefix:    "rate_limit:uois",
		RequestsPerMinute: 60,
		Burst:             10,
		WindowSeconds:     60,
	}

	service := NewRateLimitService(mockRedis, cfg, logger)
	ctx := context.Background()
	clientID := "test-client-123"

	incrCmd := redis.NewIntResult(11, nil)
	ttlCmd := redis.NewDurationResult(50*time.Second, nil)
	mockRedis.On("Incr", ctx, "rate_limit:uois:test-client-123").Return(incrCmd)
	mockRedis.On("TTL", ctx, "rate_limit:uois:test-client-123").Return(ttlCmd)

	allowed, remaining, resetAt, err := service.CheckRateLimit(ctx, clientID)

	assert.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, int64(0), remaining)
	assert.NotZero(t, resetAt)
	mockRedis.AssertNotCalled(t, "Expire")
	mockRedis.AssertExpectations(t)
}

func TestRateLimitService_GetRateLimitError(t *testing.T) {
	mockRedis := new(MockRedisClient)
	logger := zap.NewNop()
	cfg := config.RateLimitConfig{
		Enabled:           true,
		RedisKeyPrefix:    "rate_limit:uois",
		RequestsPerMinute: 60,
		Burst:             10,
		WindowSeconds:     60,
	}

	service := NewRateLimitService(mockRedis, cfg, logger)
	ctx := context.Background()
	clientID := "test-client-123"

	rateLimitErr := service.GetRateLimitError(ctx, clientID)
	assert.NotNil(t, rateLimitErr)
	assert.True(t, domainerrors.IsDomainError(rateLimitErr))
	domainErr, ok := rateLimitErr.(*domainerrors.DomainError)
	assert.True(t, ok)
	assert.Equal(t, 65012, domainErr.Code)
	assert.Equal(t, 429, domainerrors.GetHTTPStatus(rateLimitErr))
}
