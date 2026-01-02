package client

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"uois-gateway/internal/config"
	"uois-gateway/internal/models"
	domainErrors "uois-gateway/pkg/errors"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockClientRegistryRepository struct {
	mock.Mock
}

func (m *MockClientRegistryRepository) GetByClientID(ctx context.Context, clientID string) (*models.Client, error) {
	args := m.Called(ctx, clientID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Client), args.Error(1)
}

func (m *MockClientRegistryRepository) UpsertClient(ctx context.Context, client *models.Client) error {
	args := m.Called(ctx, client)
	return args.Error(0)
}

func (m *MockClientRegistryRepository) UpdateStatus(ctx context.Context, clientID, status string) error {
	args := m.Called(ctx, clientID, status)
	return args.Error(0)
}

type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	args := m.Called(ctx, key)
	cmd := redis.NewStringCmd(ctx)
	if args.Get(0) != nil {
		cmd.SetVal(args.String(0))
	} else if args.Error(1) == redis.Nil {
		cmd.SetErr(redis.Nil)
	} else {
		cmd.SetErr(args.Error(1))
	}
	return cmd
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	args := m.Called(ctx, key, value, expiration)
	cmd := redis.NewStatusCmd(ctx)
	if args.Error(0) != nil {
		cmd.SetErr(args.Error(0))
	} else {
		cmd.SetVal("OK")
	}
	return cmd
}

func (m *MockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	args := m.Called(ctx, mock.Anything)
	cmd := redis.NewIntCmd(ctx)
	if args.Error(0) != nil {
		cmd.SetErr(args.Error(0))
	} else {
		cmd.SetVal(1)
	}
	return cmd
}

func TestDBClientRegistry_GetByClientID_CacheHit(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.Config{
		TTL: config.TTLConfig{
			ClientRegistryCache: 300,
		},
	}

	mockRepo := new(MockClientRegistryRepository)
	mockRedis := new(MockRedisClient)

	registry := NewDBClientRegistry(mockRepo, mockRedis, cfg, logger)

	clientID := "client-123"
	client := &models.Client{
		ID:               clientID,
		ClientCode:       "ABC",
		ClientSecretHash: "hashed",
		Status:           models.ClientStatusActive,
	}

	clientJSON, _ := json.Marshal(client)
	mockRedis.On("Get", mock.Anything, "client:client-123").Return(string(clientJSON), nil)

	result, err := registry.GetByClientID(context.Background(), clientID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, clientID, result.ID)
	mockRepo.AssertNotCalled(t, "GetByClientID")
}

func TestDBClientRegistry_GetByClientID_CacheMiss(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.Config{
		TTL: config.TTLConfig{
			ClientRegistryCache: 300,
		},
	}

	mockRepo := new(MockClientRegistryRepository)
	mockRedis := new(MockRedisClient)

	registry := NewDBClientRegistry(mockRepo, mockRedis, cfg, logger)

	clientID := "client-123"
	client := &models.Client{
		ID:               clientID,
		ClientCode:       "ABC",
		ClientSecretHash: "hashed",
		Status:           models.ClientStatusActive,
	}

	mockRedis.On("Get", mock.Anything, "client:client-123").Return(nil, redis.Nil)
	mockRepo.On("GetByClientID", mock.Anything, clientID).Return(client, nil)
	mockRedis.On("Set", mock.Anything, "client:client-123", mock.Anything, 300*time.Second).Return(nil)

	result, err := registry.GetByClientID(context.Background(), clientID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, clientID, result.ID)
}

func TestDBClientRegistry_GetByClientID_NotFound(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.Config{
		TTL: config.TTLConfig{
			ClientRegistryCache: 300,
		},
	}

	mockRepo := new(MockClientRegistryRepository)
	mockRedis := new(MockRedisClient)

	registry := NewDBClientRegistry(mockRepo, mockRedis, cfg, logger)

	clientID := "client-123"
	domainErr := domainErrors.NewDomainError(65006, "client not found", "client_id not found")

	mockRedis.On("Get", mock.Anything, "client:client-123").Return(nil, redis.Nil)
	mockRepo.On("GetByClientID", mock.Anything, clientID).Return(nil, domainErr)

	result, err := registry.GetByClientID(context.Background(), clientID)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, domainErrors.IsDomainError(err))
}

func TestDBClientRegistry_UpsertClient_Success(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.Config{}

	mockRepo := new(MockClientRegistryRepository)
	mockRedis := new(MockRedisClient)

	registry := NewDBClientRegistry(mockRepo, mockRedis, cfg, logger)

	client := &models.Client{
		ID:               "client-123",
		ClientCode:       "ABC",
		ClientSecretHash: "hashed",
		Status:           models.ClientStatusActive,
	}

	mockRepo.On("UpsertClient", mock.Anything, client).Return(nil)
	mockRedis.On("Del", mock.Anything, "client:client-123").Return(nil)

	err := registry.UpsertClient(context.Background(), client)
	assert.NoError(t, err)
}

func TestDBClientRegistry_UpdateStatus_Success(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.Config{}

	mockRepo := new(MockClientRegistryRepository)
	mockRedis := new(MockRedisClient)

	registry := NewDBClientRegistry(mockRepo, mockRedis, cfg, logger)

	clientID := "client-123"
	status := models.ClientStatusSuspended

	mockRepo.On("UpdateStatus", mock.Anything, clientID, status).Return(nil)
	mockRedis.On("Del", mock.Anything, "client:client-123").Return(nil)

	err := registry.UpdateStatus(context.Background(), clientID, status)
	assert.NoError(t, err)
}
