package order_record

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"uois-gateway/internal/config"
	"uois-gateway/internal/handlers/ondc"
	"uois-gateway/pkg/errors"

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

func TestOrderRecordRepository_StoreOrderRecord_Success(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)
	
	cfg := config.Config{
		Redis: config.RedisConfig{
			KeyPrefix: "test-prefix",
		},
		TTL: config.TTLConfig{
			OrderMapping: 2592000, // 30 days
		},
	}

	repo := NewRepository(mockRedis, cfg, logger)

	record := &ondc.OrderRecord{
		SearchID:        "search-123",
		QuoteID:         "quote-456",
		DispatchOrderID: "dispatch-789",
		OrderID:         "order-abc",
		ClientID:        "client-1",
		TransactionID:   "txn-xyz",
		FulfillmentID:   "fulfill-1",
	}

	// StoreOrderRecord should store all keys (search_id, quote_id, order_id, transaction_id)
	// This is critical for lookups to work after StoreOrderRecord
	statusCmd := redis.NewStatusCmd(context.Background())
	statusCmd.SetVal("OK")
	mockRedis.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(statusCmd).Times(4)

	err := repo.StoreOrderRecord(context.Background(), record)

	assert.NoError(t, err)
	mockRedis.AssertExpectations(t)
}

func TestOrderRecordRepository_StoreOrderRecord_StoresAllKeys(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)
	
	cfg := config.Config{
		Redis: config.RedisConfig{
			KeyPrefix: "test-prefix",
		},
		TTL: config.TTLConfig{
			OrderMapping: 2592000,
		},
	}

	repo := NewRepository(mockRedis, cfg, logger)

	record := &ondc.OrderRecord{
		SearchID:        "search-123",
		QuoteID:         "quote-456",
		OrderID:         "order-abc",
		ClientID:        "client-1",
		TransactionID:   "txn-xyz",
		FulfillmentID:   "fulfill-1",
	}

	// Verify all keys are stored (critical for lookup by any identifier)
	var storedKeys []string
	statusCmd := redis.NewStatusCmd(context.Background())
	statusCmd.SetVal("OK")
	mockRedis.On("Set", mock.Anything, mock.MatchedBy(func(key string) bool {
		storedKeys = append(storedKeys, key)
		return true
	}), mock.Anything, mock.AnythingOfType("time.Duration")).Return(statusCmd).Times(4)

	err := repo.StoreOrderRecord(context.Background(), record)

	assert.NoError(t, err)
	
	// Verify all expected keys are stored
	assert.Contains(t, storedKeys, "test-prefix:order_record:search_id:search-123")
	assert.Contains(t, storedKeys, "test-prefix:order_record:quote_id:quote-456")
	assert.Contains(t, storedKeys, "test-prefix:order_record:order_id:client-1:order-abc")
	assert.Contains(t, storedKeys, "test-prefix:order_record:transaction_id:txn-xyz")
	mockRedis.AssertExpectations(t)
}

func TestOrderRecordRepository_GetOrderRecordBySearchID_Success(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.Config{
		Redis: config.RedisConfig{
			KeyPrefix: "test-prefix",
		},
		TTL: config.TTLConfig{
			OrderMapping: 2592000,
		},
	}

	repo := NewRepository(mockRedis, cfg, logger)

	expectedRecord := &ondc.OrderRecord{
		SearchID:        "search-123",
		QuoteID:         "quote-456",
		DispatchOrderID: "dispatch-789",
		OrderID:         "order-abc",
		ClientID:        "client-1",
		TransactionID:   "txn-xyz",
		FulfillmentID:   "fulfill-1",
	}

	recordJSON, _ := json.Marshal(expectedRecord)
	stringCmd := redis.NewStringCmd(context.Background())
	stringCmd.SetVal(string(recordJSON))
	mockRedis.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(stringCmd)

	record, err := repo.GetOrderRecordBySearchID(context.Background(), "search-123")

	assert.NoError(t, err)
	assert.NotNil(t, record)
	assert.Equal(t, "search-123", record.SearchID)
	assert.Equal(t, "quote-456", record.QuoteID)
	mockRedis.AssertExpectations(t)
}

func TestOrderRecordRepository_GetOrderRecordBySearchID_NotFound(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.Config{
		Redis: config.RedisConfig{
			KeyPrefix: "test-prefix",
		},
		TTL: config.TTLConfig{
			OrderMapping: 2592000,
		},
	}

	repo := NewRepository(mockRedis, cfg, logger)

	stringCmd := redis.NewStringCmd(context.Background())
	stringCmd.SetErr(redis.Nil)
	mockRedis.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(stringCmd)

	record, err := repo.GetOrderRecordBySearchID(context.Background(), "non-existent")

	assert.Error(t, err)
	assert.Nil(t, record)
	domainErr, ok := err.(*errors.DomainError)
	assert.True(t, ok)
	assert.Equal(t, 65006, domainErr.Code)
	mockRedis.AssertExpectations(t)
}

func TestOrderRecordRepository_GetOrderRecordByQuoteID_Success(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.Config{
		Redis: config.RedisConfig{
			KeyPrefix: "test-prefix",
		},
		TTL: config.TTLConfig{
			OrderMapping: 2592000,
		},
	}

	repo := NewRepository(mockRedis, cfg, logger)

	expectedRecord := &ondc.OrderRecord{
		SearchID:        "search-123",
		QuoteID:         "quote-456",
		DispatchOrderID: "dispatch-789",
		OrderID:         "order-abc",
		ClientID:        "client-1",
		TransactionID:   "txn-xyz",
		FulfillmentID:   "fulfill-1",
	}

	recordJSON, _ := json.Marshal(expectedRecord)
	stringCmd := redis.NewStringCmd(context.Background())
	stringCmd.SetVal(string(recordJSON))
	mockRedis.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(stringCmd)

	record, err := repo.GetOrderRecordByQuoteID(context.Background(), "quote-456")

	assert.NoError(t, err)
	assert.NotNil(t, record)
	assert.Equal(t, "quote-456", record.QuoteID)
	mockRedis.AssertExpectations(t)
}

func TestOrderRecordRepository_GetOrderRecordByOrderID_Success(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.Config{
		Redis: config.RedisConfig{
			KeyPrefix: "test-prefix",
		},
		TTL: config.TTLConfig{
			OrderMapping: 2592000,
		},
	}

	repo := NewRepository(mockRedis, cfg, logger)

	expectedRecord := &ondc.OrderRecord{
		SearchID:        "search-123",
		QuoteID:         "quote-456",
		DispatchOrderID: "dispatch-789",
		OrderID:         "order-abc",
		ClientID:        "client-1",
		TransactionID:   "txn-xyz",
		FulfillmentID:   "fulfill-1",
	}

	recordJSON, _ := json.Marshal(expectedRecord)
	stringCmd := redis.NewStringCmd(context.Background())
	stringCmd.SetVal(string(recordJSON))
	mockRedis.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(stringCmd)

	record, err := repo.GetOrderRecordByOrderID(context.Background(), "client-1", "order-abc")

	assert.NoError(t, err)
	assert.NotNil(t, record)
	assert.Equal(t, "order-abc", record.OrderID)
	assert.Equal(t, "client-1", record.ClientID)
	mockRedis.AssertExpectations(t)
}

func TestOrderRecordRepository_GetOrderRecordByTransactionID_Success(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.Config{
		Redis: config.RedisConfig{
			KeyPrefix: "test-prefix",
		},
		TTL: config.TTLConfig{
			OrderMapping: 2592000,
		},
	}

	repo := NewRepository(mockRedis, cfg, logger)

	expectedRecord := &ondc.OrderRecord{
		SearchID:        "search-123",
		QuoteID:         "quote-456",
		DispatchOrderID: "dispatch-789",
		OrderID:         "order-abc",
		ClientID:        "client-1",
		TransactionID:   "txn-xyz",
		FulfillmentID:   "fulfill-1",
	}

	recordJSON, _ := json.Marshal(expectedRecord)
	stringCmd := redis.NewStringCmd(context.Background())
	stringCmd.SetVal(string(recordJSON))
	mockRedis.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(stringCmd)

	record, err := repo.GetOrderRecordByTransactionID(context.Background(), "txn-xyz")

	assert.NoError(t, err)
	assert.NotNil(t, record)
	assert.Equal(t, "txn-xyz", record.TransactionID)
	mockRedis.AssertExpectations(t)
}

func TestOrderRecordRepository_UpdateOrderRecord_Success(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.Config{
		Redis: config.RedisConfig{
			KeyPrefix: "test-prefix",
		},
		TTL: config.TTLConfig{
			OrderMapping: 2592000,
		},
	}

	repo := NewRepository(mockRedis, cfg, logger)

	record := &ondc.OrderRecord{
		SearchID:        "search-123",
		QuoteID:         "quote-456",
		DispatchOrderID: "dispatch-789",
		OrderID:         "order-abc",
		ClientID:        "client-1",
		TransactionID:   "txn-xyz",
		FulfillmentID:   "fulfill-1",
	}

	statusCmd := redis.NewStatusCmd(context.Background())
	statusCmd.SetVal("OK")
	mockRedis.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(statusCmd).Times(4)

	err := repo.UpdateOrderRecord(context.Background(), record)

	assert.NoError(t, err)
	mockRedis.AssertExpectations(t)
}

func TestOrderRecordRepository_GetOrderRecord_InvalidJSON(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.Config{
		Redis: config.RedisConfig{
			KeyPrefix: "test-prefix",
		},
		TTL: config.TTLConfig{
			OrderMapping: 2592000,
		},
	}

	repo := NewRepository(mockRedis, cfg, logger)

	stringCmd := redis.NewStringCmd(context.Background())
	stringCmd.SetVal("invalid json")
	mockRedis.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(stringCmd)

	record, err := repo.GetOrderRecordBySearchID(context.Background(), "search-123")

	assert.Error(t, err)
	assert.Nil(t, record)
	mockRedis.AssertExpectations(t)
}
