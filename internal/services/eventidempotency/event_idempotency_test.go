package eventidempotency

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestEventIdempotency_CheckAndStore(t *testing.T) {
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	defer redisClient.Close()

	ctx := context.Background()
	service := NewService(redisClient, 5*time.Minute, nil)

	eventID := "test-event-123"

	// First check should return false (not processed)
	processed, err := service.CheckAndStore(ctx, eventID)
	assert.NoError(t, err)
	assert.False(t, processed)

	// Second check should return true (already processed)
	processed, err = service.CheckAndStore(ctx, eventID)
	assert.NoError(t, err)
	assert.True(t, processed)
}

func TestEventIdempotency_IsProcessed(t *testing.T) {
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	defer redisClient.Close()

	ctx := context.Background()
	service := NewService(redisClient, 5*time.Minute, nil)

	eventID := "test-event-456"

	// Should not be processed initially
	processed, err := service.IsProcessed(ctx, eventID)
	assert.NoError(t, err)
	assert.False(t, processed)

	// Store it
	_, err = service.CheckAndStore(ctx, eventID)
	assert.NoError(t, err)

	// Should be processed now
	processed, err = service.IsProcessed(ctx, eventID)
	assert.NoError(t, err)
	assert.True(t, processed)
}

func TestGenerateEventID(t *testing.T) {
	eventID1 := GenerateEventID("stream1", "data1")
	eventID2 := GenerateEventID("stream1", "data1")
	eventID3 := GenerateEventID("stream1", "data2")

	// Same stream and data should generate same ID
	assert.Equal(t, eventID1, eventID2)

	// Different data should generate different ID
	assert.NotEqual(t, eventID1, eventID3)
}

