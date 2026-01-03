package callback

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"uois-gateway/internal/config"
	"uois-gateway/internal/services/audit"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockAuditService is a mock implementation of AuditService
type MockAuditService struct {
	mock.Mock
}

func (m *MockAuditService) LogCallbackDelivery(ctx context.Context, req *audit.CallbackDeliveryLogParams) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

// MockRedisClient is a mock implementation of RedisClient
type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) XAdd(ctx context.Context, a *redis.XAddArgs) *redis.StringCmd {
	args := m.Called(ctx, a)
	return args.Get(0).(*redis.StringCmd)
}

func (m *MockRedisClient) XInfoStream(ctx context.Context, stream string) *redis.XInfoStreamCmd {
	args := m.Called(ctx, stream)
	return args.Get(0).(*redis.XInfoStreamCmd)
}

func TestRetryService_SendCallbackWithRetry_SuccessOnFirstAttempt(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.CallbackConfig{
		HTTPTimeoutSeconds: 5,
		MaxConcurrent:      100,
		DLQEnabled:         false,
	}

	retryConfig := config.RetryConfig{
		CallbackMaxRetries: 3,
		CallbackBackoff:    []int{1, 2, 4},
	}

	mockSigner := new(MockSigner)
	mockSigner.On("SignRequest", mock.Anything, http.MethodPost, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return("Signature keyId=\"test\",signature=\"test\"", nil)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	callbackService := NewService(cfg, mockSigner, logger)
	mockAudit := new(MockAuditService)
	mockAudit.On("LogCallbackDelivery", mock.Anything, mock.Anything).Return(nil)

	retryService := NewRetryService(callbackService, retryConfig, cfg, nil, mockAudit, logger)
	callbackServiceWithRetry := NewServiceWithRetry(cfg, retryConfig, mockSigner, nil, mockAudit, logger)
	callbackServiceWithRetry.retryService = retryService

	payload := map[string]interface{}{"test": "data"}
	err := retryService.SendCallbackWithRetry(context.Background(), server.URL, payload, "req-123", 30)

	assert.NoError(t, err)
	mockSigner.AssertExpectations(t)
	mockAudit.AssertExpectations(t)
}

func TestRetryService_SendCallbackWithRetry_SuccessOnRetry(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.CallbackConfig{
		HTTPTimeoutSeconds: 5,
		MaxConcurrent:      100,
		DLQEnabled:         false,
	}

	retryConfig := config.RetryConfig{
		CallbackMaxRetries: 3,
		CallbackBackoff:    []int{1, 2, 4},
	}

	mockSigner := new(MockSigner)
	mockSigner.On("SignRequest", mock.Anything, http.MethodPost, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return("Signature keyId=\"test\",signature=\"test\"", nil).Times(2)

	attempt := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt == 1 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	callbackService := NewService(cfg, mockSigner, logger)
	mockAudit := new(MockAuditService)
	mockAudit.On("LogCallbackDelivery", mock.Anything, mock.Anything).Return(nil).Times(2)

	retryService := NewRetryService(callbackService, retryConfig, cfg, nil, mockAudit, logger)

	payload := map[string]interface{}{"test": "data"}
	err := retryService.SendCallbackWithRetry(context.Background(), server.URL, payload, "req-123", 30)

	assert.NoError(t, err)
	mockSigner.AssertExpectations(t)
	mockAudit.AssertExpectations(t)
}

func TestRetryService_SendCallbackWithRetry_AllRetriesFail(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.CallbackConfig{
		HTTPTimeoutSeconds: 5,
		MaxConcurrent:      100,
		DLQEnabled:         false,
	}

	retryConfig := config.RetryConfig{
		CallbackMaxRetries: 3,
		CallbackBackoff:    []int{1, 2, 4},
	}

	mockSigner := new(MockSigner)
	mockSigner.On("SignRequest", mock.Anything, http.MethodPost, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return("Signature keyId=\"test\",signature=\"test\"", nil).Times(3)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	callbackService := NewService(cfg, mockSigner, logger)
	mockAudit := new(MockAuditService)
	mockAudit.On("LogCallbackDelivery", mock.Anything, mock.Anything).Return(nil).Times(3)

	retryService := NewRetryService(callbackService, retryConfig, cfg, nil, mockAudit, logger)

	payload := map[string]interface{}{"test": "data"}
	err := retryService.SendCallbackWithRetry(context.Background(), server.URL, payload, "req-123", 30)

	assert.Error(t, err)
	mockSigner.AssertExpectations(t)
	mockAudit.AssertExpectations(t)
}

func TestRetryService_CalculateBackoff(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.CallbackConfig{DLQEnabled: false}
	retryConfig := config.RetryConfig{
		CallbackMaxRetries: 5,
		CallbackBackoff:    []int{1, 2, 4, 8, 15},
	}

	retryService := NewRetryService(nil, retryConfig, cfg, nil, nil, logger)

	// Test with configured backoff
	duration := retryService.calculateBackoff(1, 30)
	assert.Equal(t, 1*time.Second, duration)

	duration = retryService.calculateBackoff(2, 30)
	assert.Equal(t, 2*time.Second, duration)

	duration = retryService.calculateBackoff(3, 30)
	assert.Equal(t, 4*time.Second, duration)

	// Test TTL bounding
	duration = retryService.calculateBackoff(5, 5)
	assert.True(t, duration <= 5*time.Second)

	// Test fallback exponential backoff
	retryConfig2 := config.RetryConfig{
		CallbackMaxRetries: 5,
		CallbackBackoff:    []int{},
	}
	retryService2 := NewRetryService(nil, retryConfig2, cfg, nil, nil, logger)
	duration = retryService2.calculateBackoff(1, 30)
	assert.Equal(t, 2*time.Second, duration) // 2^1 = 2
}

func TestRetryService_SendToDLQ(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.CallbackConfig{
		DLQEnabled: true,
		DLQStream:  "dlq:callbacks",
	}

	retryConfig := config.RetryConfig{
		CallbackMaxRetries: 3,
		CallbackBackoff:    []int{1, 2, 4},
	}

	mockRedis := new(MockRedisClient)
	ctx := context.Background()

	// Mock XAdd operation
	stringCmd := redis.NewStringCmd(ctx)
	stringCmd.SetVal("1234567890-0")
	mockRedis.On("XAdd", ctx, mock.AnythingOfType("*redis.XAddArgs")).Return(stringCmd).Once()

	retryService := NewRetryService(nil, retryConfig, cfg, mockRedis, nil, logger)

	payload := map[string]interface{}{"test": "data"}
	err := retryService.sendToDLQ(ctx, "http://example.com/callback", payload, "req-123", assert.AnError)

	assert.NoError(t, err)
	mockRedis.AssertExpectations(t)
}

func TestRetryService_SendCallbackWithRetry_WithDLQ(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.CallbackConfig{
		HTTPTimeoutSeconds: 5,
		MaxConcurrent:      100,
		DLQEnabled:         true,
		DLQStream:          "dlq:callbacks:test",
	}

	retryConfig := config.RetryConfig{
		CallbackMaxRetries: 2,
		CallbackBackoff:    []int{1, 2},
	}

	mockSigner := new(MockSigner)
	mockSigner.On("SignRequest", mock.Anything, http.MethodPost, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return("Signature keyId=\"test\",signature=\"test\"", nil).Times(2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	callbackService := NewService(cfg, mockSigner, logger)
	mockAudit := new(MockAuditService)
	mockAudit.On("LogCallbackDelivery", mock.Anything, mock.Anything).Return(nil).Times(2)

	mockRedis := new(MockRedisClient)
	ctx := context.Background()

	// Mock XAdd operation for DLQ
	stringCmd := redis.NewStringCmd(ctx)
	stringCmd.SetVal("1234567890-0")
	mockRedis.On("XAdd", ctx, mock.AnythingOfType("*redis.XAddArgs")).Return(stringCmd).Once()

	retryService := NewRetryService(callbackService, retryConfig, cfg, mockRedis, mockAudit, logger)

	payload := map[string]interface{}{"test": "data"}
	err := retryService.SendCallbackWithRetry(ctx, server.URL, payload, "req-123", 30)

	assert.Error(t, err)
	mockSigner.AssertExpectations(t)
	mockAudit.AssertExpectations(t)
	mockRedis.AssertExpectations(t)
}

func TestRetryService_SendCallbackWithRetry_ContextCancellation(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.CallbackConfig{
		HTTPTimeoutSeconds: 5,
		MaxConcurrent:      100,
		DLQEnabled:         false,
	}

	retryConfig := config.RetryConfig{
		CallbackMaxRetries: 5,
		CallbackBackoff:    []int{10, 10, 10, 10, 10},
	}

	mockSigner := new(MockSigner)
	mockSigner.On("SignRequest", mock.Anything, http.MethodPost, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return("Signature keyId=\"test\",signature=\"test\"", nil).Once()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	callbackService := NewService(cfg, mockSigner, logger)
	mockAudit := new(MockAuditService)
	mockAudit.On("LogCallbackDelivery", mock.Anything, mock.Anything).Return(nil).Once()

	retryService := NewRetryService(callbackService, retryConfig, cfg, nil, mockAudit, logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	payload := map[string]interface{}{"test": "data"}
	err := retryService.SendCallbackWithRetry(ctx, server.URL, payload, "req-123", 30)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

