package cache

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestCacheService_GetSet(t *testing.T) {
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	defer redisClient.Close()

	ctx := context.Background()
	cache := NewService(redisClient, 5*time.Minute, nil)

	// Test Set
	key := "test:key"
	value := map[string]interface{}{
		"test": "value",
		"num":  123,
	}
	err := cache.Set(ctx, key, value)
	assert.NoError(t, err)

	// Test Get
	var result map[string]interface{}
	exists, err := cache.Get(ctx, key, &result)
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, "value", result["test"])
	assert.Equal(t, float64(123), result["num"])
}

func TestCacheService_Get_NotFound(t *testing.T) {
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	defer redisClient.Close()

	ctx := context.Background()
	cache := NewService(redisClient, 5*time.Minute, nil)

	var result map[string]interface{}
	exists, err := cache.Get(ctx, "nonexistent:key", &result)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestCacheService_Delete(t *testing.T) {
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	defer redisClient.Close()

	ctx := context.Background()
	cache := NewService(redisClient, 5*time.Minute, nil)

	key := "test:delete"
	value := map[string]string{"test": "value"}
	err := cache.Set(ctx, key, value)
	assert.NoError(t, err)

	// Verify it exists
	exists, _ := cache.Exists(ctx, key)
	assert.True(t, exists)

	// Delete it
	err = cache.Delete(ctx, key)
	assert.NoError(t, err)

	// Verify it's gone
	exists, _ = cache.Exists(ctx, key)
	assert.False(t, exists)
}

func TestCacheService_BuildKey(t *testing.T) {
	key := BuildKey("status", "client123", "order456")
	assert.Equal(t, "status:client123:order456", key)

	key = BuildKey("track", "order789")
	assert.Equal(t, "track:order789", key)
}

