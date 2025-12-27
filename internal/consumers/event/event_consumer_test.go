package event

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"uois-gateway/internal/config"
	"uois-gateway/pkg/errors"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) XReadGroup(ctx context.Context, args *redis.XReadGroupArgs) *redis.XStreamSliceCmd {
	mockArgs := m.Called(ctx, args)
	return mockArgs.Get(0).(*redis.XStreamSliceCmd)
}

func (m *MockRedisClient) XAck(ctx context.Context, stream, group, id string) *redis.IntCmd {
	mockArgs := m.Called(ctx, stream, group, id)
	return mockArgs.Get(0).(*redis.IntCmd)
}

func TestEventConsumer_ConsumeEvent_Success(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.StreamsConfig{
		ConsumerID: "test-consumer-1",
	}

	consumer := NewConsumer(mockRedis, cfg, logger)

	// Create mock event data with matching business ID (search_id)
	eventData := map[string]interface{}{
		"event_type": "QUOTE_COMPUTED",
		"search_id":  "search-123",
	}
	eventJSON, _ := json.Marshal(eventData)

	// Create mock Redis stream message
	streamSliceCmd := redis.NewXStreamSliceCmd(context.Background())
	streams := []redis.XStream{
		{
			Stream: "test-stream",
			Messages: []redis.XMessage{
				{
					ID: "1234567890-0",
					Values: map[string]interface{}{
						"data": string(eventJSON),
					},
				},
			},
		},
	}
	streamSliceCmd.SetVal(streams)
	mockRedis.On("XReadGroup", mock.Anything, mock.Anything).Return(streamSliceCmd)

	intCmd := redis.NewIntCmd(context.Background())
	intCmd.SetVal(1)
	mockRedis.On("XAck", mock.Anything, "test-stream", "test-group", "1234567890-0").Return(intCmd)

	event, err := consumer.ConsumeEvent(context.Background(), "test-stream", "test-group", "search-123", 5*time.Second)

	assert.NoError(t, err)
	assert.NotNil(t, event)
	mockRedis.AssertExpectations(t)
}

func TestEventConsumer_ConsumeEvent_NoEvents(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.StreamsConfig{
		ConsumerID: "test-consumer-1",
	}

	consumer := NewConsumer(mockRedis, cfg, logger)

	// Mock Redis returning nil (no events)
	streamSliceCmd := redis.NewXStreamSliceCmd(context.Background())
	streamSliceCmd.SetErr(redis.Nil)
	mockRedis.On("XReadGroup", mock.Anything, mock.Anything).Return(streamSliceCmd)

	event, err := consumer.ConsumeEvent(context.Background(), "test-stream", "test-group", "correlation-123", 5*time.Second)

	assert.NoError(t, err)
	assert.Nil(t, event)
	mockRedis.AssertExpectations(t)
}

func TestEventConsumer_ConsumeEvent_RedisError(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.StreamsConfig{
		ConsumerID: "test-consumer-1",
	}

	consumer := NewConsumer(mockRedis, cfg, logger)

	streamSliceCmd := redis.NewXStreamSliceCmd(context.Background())
	streamSliceCmd.SetErr(redis.ErrClosed)
	mockRedis.On("XReadGroup", mock.Anything, mock.Anything).Return(streamSliceCmd)

	event, err := consumer.ConsumeEvent(context.Background(), "test-stream", "test-group", "correlation-123", 5*time.Second)

	assert.Error(t, err)
	assert.Nil(t, event)
	domainErr, ok := err.(*errors.DomainError)
	assert.True(t, ok)
	assert.Equal(t, 65011, domainErr.Code)
	mockRedis.AssertExpectations(t)
}

func TestEventConsumer_ConsumeEvent_InvalidEventData(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.StreamsConfig{
		ConsumerID: "test-consumer-1",
	}

	consumer := NewConsumer(mockRedis, cfg, logger)

	// Create mock Redis stream message with invalid data type
	streamSliceCmd := redis.NewXStreamSliceCmd(context.Background())
	streams := []redis.XStream{
		{
			Stream: "test-stream",
			Messages: []redis.XMessage{
				{
					ID: "1234567890-0",
					Values: map[string]interface{}{
						"data": 12345, // Invalid: not a string
					},
				},
			},
		},
	}
	streamSliceCmd.SetVal(streams)
	mockRedis.On("XReadGroup", mock.Anything, mock.Anything).Return(streamSliceCmd)

	event, err := consumer.ConsumeEvent(context.Background(), "test-stream", "test-group", "correlation-123", 5*time.Second)

	assert.Error(t, err)
	assert.Nil(t, event)
	domainErr, ok := err.(*errors.DomainError)
	assert.True(t, ok)
	assert.Equal(t, 65020, domainErr.Code)
	mockRedis.AssertExpectations(t)
}

func TestEventConsumer_ConsumeEvent_InvalidJSON(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.StreamsConfig{
		ConsumerID: "test-consumer-1",
	}

	consumer := NewConsumer(mockRedis, cfg, logger)

	// Create mock Redis stream message with invalid JSON
	streamSliceCmd := redis.NewXStreamSliceCmd(context.Background())
	streams := []redis.XStream{
		{
			Stream: "test-stream",
			Messages: []redis.XMessage{
				{
					ID: "1234567890-0",
					Values: map[string]interface{}{
						"data": "invalid json {",
					},
				},
			},
		},
	}
	streamSliceCmd.SetVal(streams)
	mockRedis.On("XReadGroup", mock.Anything, mock.Anything).Return(streamSliceCmd)

	// Note: XAck won't be called if JSON unmarshal fails before ACK
	// The consumer returns error before ACK, so we don't expect XAck call

	event, err := consumer.ConsumeEvent(context.Background(), "test-stream", "test-group", "correlation-123", 5*time.Second)

	assert.Error(t, err)
	assert.Nil(t, event)
	mockRedis.AssertExpectations(t)
}

func TestEventConsumer_ConsumeEvent_EmptyStream(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.StreamsConfig{
		ConsumerID: "test-consumer-1",
	}

	consumer := NewConsumer(mockRedis, cfg, logger)

	// Create mock Redis stream with no messages
	streamSliceCmd := redis.NewXStreamSliceCmd(context.Background())
	streams := []redis.XStream{
		{
			Stream:   "test-stream",
			Messages: []redis.XMessage{},
		},
	}
	streamSliceCmd.SetVal(streams)
	mockRedis.On("XReadGroup", mock.Anything, mock.Anything).Return(streamSliceCmd)

	event, err := consumer.ConsumeEvent(context.Background(), "test-stream", "test-group", "correlation-123", 5*time.Second)

	assert.NoError(t, err)
	assert.Nil(t, event)
	mockRedis.AssertExpectations(t)
}

func TestEventConsumer_ConsumeEvent_CorrelationMismatch(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.StreamsConfig{
		ConsumerID: "test-consumer-1",
	}

	consumer := NewConsumer(mockRedis, cfg, logger)

	// Create mock event data with different business ID
	eventData := map[string]interface{}{
		"event_type": "QUOTE_COMPUTED",
		"search_id":  "search-456", // Different from expected
	}
	eventJSON, _ := json.Marshal(eventData)

	// Create mock Redis stream message
	streamSliceCmd := redis.NewXStreamSliceCmd(context.Background())
	streams := []redis.XStream{
		{
			Stream: "test-stream",
			Messages: []redis.XMessage{
				{
					ID: "1234567890-0",
					Values: map[string]interface{}{
						"data": string(eventJSON),
					},
				},
			},
		},
	}
	streamSliceCmd.SetVal(streams)
	mockRedis.On("XReadGroup", mock.Anything, mock.Anything).Return(streamSliceCmd)

	// XAck should NOT be called when business ID doesn't match
	event, err := consumer.ConsumeEvent(context.Background(), "test-stream", "test-group", "search-123", 5*time.Second)

	assert.NoError(t, err)
	assert.Nil(t, event) // Event skipped due to business ID mismatch
	mockRedis.AssertExpectations(t)
}

func TestEventConsumer_ConsumeEvent_QuoteIDCorrelation(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.StreamsConfig{
		ConsumerID: "test-consumer-1",
	}

	consumer := NewConsumer(mockRedis, cfg, logger)

	// Create mock event data with quote_id as business ID
	eventData := map[string]interface{}{
		"event_type": "ORDER_CONFIRMED",
		"quote_id":   "quote-123",
	}
	eventJSON, _ := json.Marshal(eventData)

	// Create mock Redis stream message
	streamSliceCmd := redis.NewXStreamSliceCmd(context.Background())
	streams := []redis.XStream{
		{
			Stream: "test-stream",
			Messages: []redis.XMessage{
				{
					ID: "1234567890-0",
					Values: map[string]interface{}{
						"data": string(eventJSON),
					},
				},
			},
		},
	}
	streamSliceCmd.SetVal(streams)
	mockRedis.On("XReadGroup", mock.Anything, mock.Anything).Return(streamSliceCmd)

	intCmd := redis.NewIntCmd(context.Background())
	intCmd.SetVal(1)
	mockRedis.On("XAck", mock.Anything, "test-stream", "test-group", "1234567890-0").Return(intCmd)

	event, err := consumer.ConsumeEvent(context.Background(), "test-stream", "test-group", "quote-123", 5*time.Second)

	assert.NoError(t, err)
	assert.NotNil(t, event)
	mockRedis.AssertExpectations(t)
}
