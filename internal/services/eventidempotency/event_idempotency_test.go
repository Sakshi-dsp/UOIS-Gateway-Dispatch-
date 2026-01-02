package eventidempotency

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	args := m.Called(ctx, key, value, expiration)
	return args.Get(0).(*redis.BoolCmd)
}

func (m *MockRedisClient) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	args := m.Called(ctx, mock.Anything)
	return args.Get(0).(*redis.IntCmd)
}

func buildTestKey(eventID string) string {
	hash := sha256.Sum256([]byte(eventID))
	hashStr := hex.EncodeToString(hash[:])
	return fmt.Sprintf("event:idempotency:%s", hashStr)
}

func TestEventIdempotency_CheckAndStore(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)
	service := NewService(mockRedis, 5*time.Minute, logger)

	ctx := context.Background()
	eventID := "test-event-123"
	key := buildTestKey(eventID)

	// First check should return false (not processed) - SetNX returns true
	boolCmd1 := redis.NewBoolCmd(ctx)
	boolCmd1.SetVal(true)
	mockRedis.On("SetNX", ctx, key, "1", mock.AnythingOfType("time.Duration")).Return(boolCmd1).Once()

	processed, err := service.CheckAndStore(ctx, eventID)
	assert.NoError(t, err)
	assert.False(t, processed)

	// Second check should return true (already processed) - SetNX returns false
	boolCmd2 := redis.NewBoolCmd(ctx)
	boolCmd2.SetVal(false)
	mockRedis.On("SetNX", ctx, key, "1", mock.AnythingOfType("time.Duration")).Return(boolCmd2).Once()

	processed, err = service.CheckAndStore(ctx, eventID)
	assert.NoError(t, err)
	assert.True(t, processed)

	mockRedis.AssertExpectations(t)
}

func TestEventIdempotency_IsProcessed(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)
	service := NewService(mockRedis, 5*time.Minute, logger)

	ctx := context.Background()
	eventID := "test-event-456"
	key := buildTestKey(eventID)

	// Should not be processed initially - Exists returns 0
	intCmd1 := redis.NewIntCmd(ctx)
	intCmd1.SetVal(0)
	mockRedis.On("Exists", ctx, mock.Anything).Return(intCmd1).Once()

	processed, err := service.IsProcessed(ctx, eventID)
	assert.NoError(t, err)
	assert.False(t, processed)

	// Store it - SetNX returns true
	boolCmd := redis.NewBoolCmd(ctx)
	boolCmd.SetVal(true)
	mockRedis.On("SetNX", ctx, key, "1", mock.AnythingOfType("time.Duration")).Return(boolCmd).Once()

	_, err = service.CheckAndStore(ctx, eventID)
	assert.NoError(t, err)

	// Should be processed now - Exists returns 1
	intCmd2 := redis.NewIntCmd(ctx)
	intCmd2.SetVal(1)
	mockRedis.On("Exists", ctx, mock.Anything).Return(intCmd2).Once()

	processed, err = service.IsProcessed(ctx, eventID)
	assert.NoError(t, err)
	assert.True(t, processed)

	mockRedis.AssertExpectations(t)
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
