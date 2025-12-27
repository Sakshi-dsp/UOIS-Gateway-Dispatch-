package idempotency

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"uois-gateway/internal/config"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	args := m.Called(ctx, key)
	return args.Get(0).(*redis.StringCmd)
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	args := m.Called(ctx, key, value, expiration)
	return args.Get(0).(*redis.StatusCmd)
}

func TestIdempotencyService_CheckIdempotency_NotFound(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.Config{
		Redis: config.RedisConfig{
			KeyPrefix: "test-prefix",
		},
		TTL: config.TTLConfig{
			IdempotencyKey: 86400,
		},
	}

	service := NewService(mockRedis, cfg, logger)

	// Mock Redis Get to return nil (key not found)
	stringCmd := redis.NewStringCmd(context.Background())
	stringCmd.SetErr(redis.Nil)
	mockRedis.On("Get", mock.Anything, "test-prefix:idempotency:test-key").Return(stringCmd)

	response, found, err := service.CheckIdempotency(context.Background(), "test-key")

	assert.NoError(t, err)
	assert.False(t, found)
	assert.Nil(t, response)
	mockRedis.AssertExpectations(t)
}

func TestIdempotencyService_CheckIdempotency_Found(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.Config{
		Redis: config.RedisConfig{
			KeyPrefix: "test-prefix",
		},
		TTL: config.TTLConfig{
			IdempotencyKey: 86400,
		},
	}

	service := NewService(mockRedis, cfg, logger)

	// Mock response as raw JSON bytes
	responseData := map[string]interface{}{
		"status": "ACK",
	}
	responseJSON, _ := json.Marshal(responseData)

	stringCmd := redis.NewStringCmd(context.Background())
	stringCmd.SetVal(string(responseJSON))
	mockRedis.On("Get", mock.Anything, "test-prefix:idempotency:test-key").Return(stringCmd)

	response, found, err := service.CheckIdempotency(context.Background(), "test-key")

	assert.NoError(t, err)
	assert.True(t, found)
	assert.NotNil(t, response)
	assert.Equal(t, responseJSON, response)
	mockRedis.AssertExpectations(t)
}

func TestIdempotencyService_CheckIdempotency_RedisError(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.Config{
		Redis: config.RedisConfig{
			KeyPrefix: "test-prefix",
		},
		TTL: config.TTLConfig{
			IdempotencyKey: 86400,
		},
	}

	service := NewService(mockRedis, cfg, logger)

	stringCmd := redis.NewStringCmd(context.Background())
	stringCmd.SetErr(redis.ErrClosed)
	mockRedis.On("Get", mock.Anything, "test-prefix:idempotency:test-key").Return(stringCmd)

	response, found, err := service.CheckIdempotency(context.Background(), "test-key")

	assert.Error(t, err)
	assert.False(t, found)
	assert.Nil(t, response)
	mockRedis.AssertExpectations(t)
}

func TestIdempotencyService_StoreIdempotency_Success(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.Config{
		Redis: config.RedisConfig{
			KeyPrefix: "test-prefix",
		},
		TTL: config.TTLConfig{
			IdempotencyKey: 86400,
		},
	}

	service := NewService(mockRedis, cfg, logger)

	responseData := map[string]interface{}{
		"status": "ACK",
	}
	responseJSON, _ := json.Marshal(responseData)

	statusCmd := redis.NewStatusCmd(context.Background())
	statusCmd.SetVal("OK")
	mockRedis.On("Set", mock.Anything, "test-prefix:idempotency:test-key", responseJSON, mock.AnythingOfType("time.Duration")).Return(statusCmd)

	err := service.StoreIdempotency(context.Background(), "test-key", responseJSON, 1*time.Hour)

	assert.NoError(t, err)
	mockRedis.AssertExpectations(t)
}

func TestIdempotencyService_StoreIdempotency_RedisError(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.Config{
		Redis: config.RedisConfig{
			KeyPrefix: "test-prefix",
		},
		TTL: config.TTLConfig{
			IdempotencyKey: 86400,
		},
	}

	service := NewService(mockRedis, cfg, logger)

	responseData := map[string]interface{}{
		"status": "ACK",
	}
	responseJSON, _ := json.Marshal(responseData)

	statusCmd := redis.NewStatusCmd(context.Background())
	statusCmd.SetErr(redis.ErrClosed)
	mockRedis.On("Set", mock.Anything, "test-prefix:idempotency:test-key", responseJSON, mock.AnythingOfType("time.Duration")).Return(statusCmd)

	err := service.StoreIdempotency(context.Background(), "test-key", responseJSON, 1*time.Hour)

	assert.Error(t, err)
	mockRedis.AssertExpectations(t)
}
