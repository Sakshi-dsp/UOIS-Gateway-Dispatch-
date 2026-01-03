package cache

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func (m *MockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	args := m.Called(ctx, keys)
	return args.Get(0).(*redis.IntCmd)
}

func (m *MockRedisClient) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	args := m.Called(ctx, keys)
	return args.Get(0).(*redis.IntCmd)
}

func TestCacheService_GetSet(t *testing.T) {
	ctx := context.Background()
	mockRedis := new(MockRedisClient)
	cache := NewService(mockRedis, 5*time.Minute, nil)

	key := "test:key"
	value := map[string]interface{}{
		"test": "value",
		"num":  123,
	}

	// Mock Set operation
	valueJSON, _ := json.Marshal(value)
	statusCmd := redis.NewStatusCmd(ctx)
	statusCmd.SetVal("OK")
	mockRedis.On("Set", ctx, key, valueJSON, 5*time.Minute).Return(statusCmd).Once()

	err := cache.Set(ctx, key, value)
	assert.NoError(t, err)

	// Mock Get operation
	stringCmd := redis.NewStringCmd(ctx)
	stringCmd.SetVal(string(valueJSON))
	mockRedis.On("Get", ctx, key).Return(stringCmd).Once()

	var result map[string]interface{}
	exists, err := cache.Get(ctx, key, &result)
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, "value", result["test"])
	assert.Equal(t, float64(123), result["num"])

	mockRedis.AssertExpectations(t)
}

func TestCacheService_Get_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRedis := new(MockRedisClient)
	cache := NewService(mockRedis, 5*time.Minute, nil)

	key := "nonexistent:key"

	// Mock Get operation returning redis.Nil
	stringCmd := redis.NewStringCmd(ctx)
	stringCmd.SetErr(redis.Nil)
	mockRedis.On("Get", ctx, key).Return(stringCmd).Once()

	var result map[string]interface{}
	exists, err := cache.Get(ctx, key, &result)
	assert.NoError(t, err)
	assert.False(t, exists)

	mockRedis.AssertExpectations(t)
}

func TestCacheService_Delete(t *testing.T) {
	ctx := context.Background()
	mockRedis := new(MockRedisClient)
	cache := NewService(mockRedis, 5*time.Minute, nil)

	key := "test:delete"
	value := map[string]string{"test": "value"}

	// Mock Set operation
	valueJSON, _ := json.Marshal(value)
	statusCmd := redis.NewStatusCmd(ctx)
	statusCmd.SetVal("OK")
	mockRedis.On("Set", ctx, key, valueJSON, 5*time.Minute).Return(statusCmd).Once()

	err := cache.Set(ctx, key, value)
	assert.NoError(t, err)

	// Mock Exists operation - key exists
	intCmd1 := redis.NewIntCmd(ctx)
	intCmd1.SetVal(1)
	mockRedis.On("Exists", ctx, []string{key}).Return(intCmd1).Once()

	exists, err := cache.Exists(ctx, key)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Mock Delete operation
	intCmd2 := redis.NewIntCmd(ctx)
	intCmd2.SetVal(1)
	mockRedis.On("Del", ctx, []string{key}).Return(intCmd2).Once()

	err = cache.Delete(ctx, key)
	assert.NoError(t, err)

	// Mock Exists operation - key doesn't exist
	intCmd3 := redis.NewIntCmd(ctx)
	intCmd3.SetVal(0)
	mockRedis.On("Exists", ctx, []string{key}).Return(intCmd3).Once()

	exists, err = cache.Exists(ctx, key)
	assert.NoError(t, err)
	assert.False(t, exists)

	mockRedis.AssertExpectations(t)
}

func TestCacheService_BuildKey(t *testing.T) {
	key := BuildKey("status", "client123", "order456")
	assert.Equal(t, "status:client123:order456", key)

	key = BuildKey("track", "order789")
	assert.Equal(t, "track:order789", key)
}
