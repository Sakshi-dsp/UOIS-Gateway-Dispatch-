package redis

import (
	"context"
	"encoding/json"
	"testing"

	"uois-gateway/pkg/errors"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) XAdd(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd {
	mockArgs := m.Called(ctx, args)
	return mockArgs.Get(0).(*redis.StringCmd)
}

func TestEventPublisher_PublishEvent_Success(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	publisher := NewEventPublisher(mockRedis, logger)

	event := map[string]interface{}{
		"event_type": "SEARCH_REQUESTED",
		"search_id":  "search-123",
	}

	stringCmd := redis.NewStringCmd(context.Background())
	stringCmd.SetVal("1234567890-0")
	mockRedis.On("XAdd", mock.Anything, mock.MatchedBy(func(args *redis.XAddArgs) bool {
		return args.Stream == "test-stream"
	})).Return(stringCmd)

	err := publisher.PublishEvent(context.Background(), "test-stream", event)

	assert.NoError(t, err)
	mockRedis.AssertExpectations(t)
}

func TestEventPublisher_PublishEvent_SerializationFailure(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	publisher := NewEventPublisher(mockRedis, logger)

	// Create a payload that cannot be serialized
	type Circular struct {
		Self *Circular
	}
	circular := &Circular{}
	circular.Self = circular

	err := publisher.PublishEvent(context.Background(), "test-stream", circular)

	assert.Error(t, err)
	domainErr, ok := err.(*errors.DomainError)
	assert.True(t, ok)
	assert.Equal(t, 65020, domainErr.Code)
}

func TestEventPublisher_PublishEvent_RedisError(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	publisher := NewEventPublisher(mockRedis, logger)

	event := map[string]interface{}{
		"event_type": "SEARCH_REQUESTED",
		"search_id":  "search-123",
	}

	stringCmd := redis.NewStringCmd(context.Background())
	stringCmd.SetErr(redis.ErrClosed)
	mockRedis.On("XAdd", mock.Anything, mock.Anything).Return(stringCmd)

	err := publisher.PublishEvent(context.Background(), "test-stream", event)

	assert.Error(t, err)
	domainErr, ok := err.(*errors.DomainError)
	assert.True(t, ok)
	assert.Equal(t, 65011, domainErr.Code)
	mockRedis.AssertExpectations(t)
}

func TestEventPublisher_PublishEvent_ValidEventStructure(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	publisher := NewEventPublisher(mockRedis, logger)

	event := map[string]interface{}{
		"event_type":  "SEARCH_REQUESTED",
		"search_id":   "search-123",
		"traceparent": "00-4bf92f3577b34da6a811ce9a-1234567890abcdef-01",
	}

	var capturedArgs *redis.XAddArgs
	stringCmd := redis.NewStringCmd(context.Background())
	stringCmd.SetVal("1234567890-0")
	mockRedis.On("XAdd", mock.Anything, mock.MatchedBy(func(args *redis.XAddArgs) bool {
		capturedArgs = args
		return args.Stream == "test-stream"
	})).Return(stringCmd)

	err := publisher.PublishEvent(context.Background(), "test-stream", event)

	assert.NoError(t, err)
	assert.NotNil(t, capturedArgs)
	assert.Equal(t, "test-stream", capturedArgs.Stream)

	// Verify event data is stored correctly
	if capturedArgs.Values != nil {
		valuesMap, ok := capturedArgs.Values.(map[string]interface{})
		assert.True(t, ok)
		dataVal, exists := valuesMap["data"]
		assert.True(t, exists)
		if dataStr, ok := dataVal.(string); ok {
			var decodedEvent map[string]interface{}
			err = json.Unmarshal([]byte(dataStr), &decodedEvent)
			assert.NoError(t, err)
			assert.Equal(t, "search-123", decodedEvent["search_id"])
		}
	}
	mockRedis.AssertExpectations(t)
}
