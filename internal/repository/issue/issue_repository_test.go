package issue

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"uois-gateway/internal/config"
	"uois-gateway/internal/models"
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

func TestIssueRepository_StoreIssue_Success(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.Config{
		Redis: config.RedisConfig{
			KeyPrefix: "test-prefix",
		},
		TTL: config.TTLConfig{
			IssueStorage: 2592000, // 30 days
		},
	}

	repo := NewRepository(mockRedis, cfg, logger)

	issue := &models.Issue{
		IssueID:       "issue-123",
		TransactionID: "txn-456",
		OrderID:       "order-789",
		IssueType:     models.IssueTypeIssue,
		Status:        models.IssueStatusOpen,
		Category:      "ORDER",
		Description:   "Test issue",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	statusCmd := redis.NewStatusCmd(context.Background())
	statusCmd.SetVal("OK")
	mockRedis.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(statusCmd)

	err := repo.StoreIssue(context.Background(), issue)
	assert.NoError(t, err)
	mockRedis.AssertExpectations(t)
}

func TestIssueRepository_GetIssue_Success(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.Config{
		Redis: config.RedisConfig{
			KeyPrefix: "test-prefix",
		},
		TTL: config.TTLConfig{
			IssueStorage: 2592000,
		},
	}

	repo := NewRepository(mockRedis, cfg, logger)

	issue := &models.Issue{
		IssueID:       "issue-123",
		TransactionID: "txn-456",
		OrderID:       "order-789",
		IssueType:     models.IssueTypeIssue,
		Status:        models.IssueStatusOpen,
		Category:      "ORDER",
		Description:   "Test issue",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	issueJSON, _ := json.Marshal(issue)
	stringCmd := redis.NewStringCmd(context.Background())
	stringCmd.SetVal(string(issueJSON))
	mockRedis.On("Get", mock.Anything, mock.Anything).Return(stringCmd)

	result, err := repo.GetIssue(context.Background(), "issue-123")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, issue.IssueID, result.IssueID)
	mockRedis.AssertExpectations(t)
}

func TestIssueRepository_GetIssue_NotFound(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.Config{
		Redis: config.RedisConfig{
			KeyPrefix: "test-prefix",
		},
		TTL: config.TTLConfig{
			IssueStorage: 2592000,
		},
	}

	repo := NewRepository(mockRedis, cfg, logger)

	stringCmd := redis.NewStringCmd(context.Background())
	stringCmd.SetErr(redis.Nil)
	mockRedis.On("Get", mock.Anything, mock.Anything).Return(stringCmd)

	result, err := repo.GetIssue(context.Background(), "issue-123")
	assert.Error(t, err)
	assert.True(t, errors.IsDomainError(err))
	assert.Nil(t, result)
	mockRedis.AssertExpectations(t)
}

func TestIssueRepository_StoreZendeskReference_Success(t *testing.T) {
	logger := zap.NewNop()
	mockRedis := new(MockRedisClient)

	cfg := config.Config{
		Redis: config.RedisConfig{
			KeyPrefix: "test-prefix",
		},
		TTL: config.TTLConfig{
			IssueStorage: 2592000,
		},
	}

	repo := NewRepository(mockRedis, cfg, logger)

	statusCmd := redis.NewStatusCmd(context.Background())
	statusCmd.SetVal("OK")
	mockRedis.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(statusCmd)

	err := repo.StoreZendeskReference(context.Background(), "ticket-123", "issue-456")
	assert.NoError(t, err)
	mockRedis.AssertExpectations(t)
}
