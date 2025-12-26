package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"uois-gateway/internal/models"
	domainerrors "uois-gateway/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) AuthenticateClient(ctx context.Context, clientID, clientSecret, clientIP string) (*models.Client, error) {
	args := m.Called(ctx, clientID, clientSecret, clientIP)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Client), args.Error(1)
}

type MockRateLimitService struct {
	mock.Mock
}

func (m *MockRateLimitService) CheckRateLimit(ctx context.Context, clientID string) (allowed bool, remaining int64, resetAt time.Time, err error) {
	args := m.Called(ctx, clientID)
	return args.Bool(0), args.Get(1).(int64), args.Get(2).(time.Time), args.Error(3)
}

func (m *MockRateLimitService) GetRateLimitError(ctx context.Context, clientID string) error {
	args := m.Called(ctx, clientID)
	return args.Error(0)
}

func TestAuthMiddleware_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockAuth := new(MockAuthService)
	mockRateLimit := new(MockRateLimitService)
	logger := zap.NewNop()

	middleware := AuthMiddleware(mockAuth, mockRateLimit, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", nil)
	c.Request.SetBasicAuth("client-123", "secret-123")
	c.Request.RemoteAddr = "192.168.1.1:8080"

	client := &models.Client{
		ID:     "client-123",
		Status: models.ClientStatusActive,
	}

	mockAuth.On("AuthenticateClient", c.Request.Context(), "client-123", "secret-123", "192.168.1.1").Return(client, nil)
	mockRateLimit.On("CheckRateLimit", c.Request.Context(), "client-123").Return(true, int64(59), time.Now().Add(60*time.Second), nil)

	middleware(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockAuth.AssertExpectations(t)
	mockRateLimit.AssertExpectations(t)
}

func TestAuthMiddleware_MissingAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockAuth := new(MockAuthService)
	mockRateLimit := new(MockRateLimitService)
	logger := zap.NewNop()

	middleware := AuthMiddleware(mockAuth, mockRateLimit, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", nil)

	middleware(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockAuth.AssertNotCalled(t, "AuthenticateClient")
	mockRateLimit.AssertNotCalled(t, "CheckRateLimit")
}

func TestAuthMiddleware_InvalidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockAuth := new(MockAuthService)
	mockRateLimit := new(MockRateLimitService)
	logger := zap.NewNop()

	middleware := AuthMiddleware(mockAuth, mockRateLimit, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", nil)
	c.Request.SetBasicAuth("client-123", "wrong-secret")
	c.Request.RemoteAddr = "192.168.1.1:8080"

	authErr := domainerrors.NewDomainError(65002, "authentication failed", "invalid credentials")
	mockAuth.On("AuthenticateClient", c.Request.Context(), "client-123", "wrong-secret", "192.168.1.1").Return(nil, authErr)

	middleware(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockAuth.AssertExpectations(t)
	mockRateLimit.AssertNotCalled(t, "CheckRateLimit")
}

func TestAuthMiddleware_RateLimitExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockAuth := new(MockAuthService)
	mockRateLimit := new(MockRateLimitService)
	logger := zap.NewNop()

	middleware := AuthMiddleware(mockAuth, mockRateLimit, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", nil)
	c.Request.SetBasicAuth("client-123", "secret-123")
	c.Request.RemoteAddr = "192.168.1.1:8080"

	client := &models.Client{
		ID:     "client-123",
		Status: models.ClientStatusActive,
	}
	resetAt := time.Now().Add(60 * time.Second)
	rateLimitErr := domainerrors.NewDomainError(65012, "rate limit exceeded", "client exceeded rate limit")

	mockAuth.On("AuthenticateClient", c.Request.Context(), "client-123", "secret-123", "192.168.1.1").Return(client, nil)
	mockRateLimit.On("CheckRateLimit", c.Request.Context(), "client-123").Return(false, int64(0), resetAt, nil)
	mockRateLimit.On("GetRateLimitError", c.Request.Context(), "client-123").Return(rateLimitErr)

	middleware(c)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	mockAuth.AssertExpectations(t)
	mockRateLimit.AssertExpectations(t)
}

func TestAuthMiddleware_RateLimitRedisError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockAuth := new(MockAuthService)
	mockRateLimit := new(MockRateLimitService)
	logger := zap.NewNop()

	middleware := AuthMiddleware(mockAuth, mockRateLimit, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", nil)
	c.Request.SetBasicAuth("client-123", "secret-123")
	c.Request.RemoteAddr = "192.168.1.1:8080"

	client := &models.Client{
		ID:     "client-123",
		Status: models.ClientStatusActive,
	}
	redisErr := domainerrors.WrapDomainError(assert.AnError, 65011, "rate limiting unavailable", "redis error")

	mockAuth.On("AuthenticateClient", c.Request.Context(), "client-123", "secret-123", "192.168.1.1").Return(client, nil)
	mockRateLimit.On("CheckRateLimit", c.Request.Context(), "client-123").Return(false, int64(0), time.Time{}, redisErr)

	middleware(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	mockAuth.AssertExpectations(t)
	mockRateLimit.AssertExpectations(t)
}

func TestAuthMiddleware_SetsClientInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockAuth := new(MockAuthService)
	mockRateLimit := new(MockRateLimitService)
	logger := zap.NewNop()

	middleware := AuthMiddleware(mockAuth, mockRateLimit, logger)

	var capturedClient *models.Client
	handler := func(c *gin.Context) {
		capturedClient = GetClientFromContext(c)
		c.Status(http.StatusOK)
	}

	w := httptest.NewRecorder()
	engine := gin.New()
	engine.Use(middleware)
	engine.POST("/test", handler)

	req := httptest.NewRequest("POST", "/test", nil)
	req.SetBasicAuth("client-123", "secret-123")
	req.RemoteAddr = "192.168.1.1:8080"

	client := &models.Client{
		ID:     "client-123",
		Status: models.ClientStatusActive,
	}

	mockAuth.On("AuthenticateClient", req.Context(), "client-123", "secret-123", "192.168.1.1").Return(client, nil)
	mockRateLimit.On("CheckRateLimit", req.Context(), "client-123").Return(true, int64(59), time.Now().Add(60*time.Second), nil)

	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, capturedClient)
	assert.Equal(t, "client-123", capturedClient.ID)
	mockAuth.AssertExpectations(t)
	mockRateLimit.AssertExpectations(t)
}

func TestAuthMiddleware_SetsRateLimitHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockAuth := new(MockAuthService)
	mockRateLimit := new(MockRateLimitService)
	logger := zap.NewNop()

	middleware := AuthMiddleware(mockAuth, mockRateLimit, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", nil)
	c.Request.SetBasicAuth("client-123", "secret-123")
	c.Request.RemoteAddr = "192.168.1.1:8080"

	client := &models.Client{
		ID:     "client-123",
		Status: models.ClientStatusActive,
	}
	resetAt := time.Now().Add(60 * time.Second)

	mockAuth.On("AuthenticateClient", c.Request.Context(), "client-123", "secret-123", "192.168.1.1").Return(client, nil)
	mockRateLimit.On("CheckRateLimit", c.Request.Context(), "client-123").Return(true, int64(45), resetAt, nil)

	middleware(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "45", w.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
	mockAuth.AssertExpectations(t)
	mockRateLimit.AssertExpectations(t)
}

func TestAuthMiddleware_ErrorResponseSanitized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockAuth := new(MockAuthService)
	mockRateLimit := new(MockRateLimitService)
	logger := zap.NewNop()

	middleware := AuthMiddleware(mockAuth, mockRateLimit, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", nil)
	c.Request.SetBasicAuth("client-123", "wrong-secret")
	c.Request.RemoteAddr = "192.168.1.1:8080"

	authErr := domainerrors.NewDomainError(65002, "authentication failed", "invalid credentials")
	mockAuth.On("AuthenticateClient", c.Request.Context(), "client-123", "wrong-secret", "192.168.1.1").Return(nil, authErr)

	middleware(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "request rejected", response["error"])
	mockAuth.AssertExpectations(t)
}

func TestAuthMiddleware_TrustedProxyHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockAuth := new(MockAuthService)
	mockRateLimit := new(MockRateLimitService)
	logger := zap.NewNop()

	trustedProxy, err := NewTrustedProxyList([]string{"10.0.0.0/8", "172.16.0.0/12"})
	assert.NoError(t, err)

	config := AuthMiddlewareConfig{TrustedProxyChecker: trustedProxy}
	middleware := AuthMiddlewareWithConfig(mockAuth, mockRateLimit, logger, config)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", nil)
	c.Request.SetBasicAuth("client-123", "secret-123")
	c.Request.RemoteAddr = "10.0.0.1:54321"
	c.Request.Header.Set("X-Forwarded-For", "203.0.113.1")

	client := &models.Client{
		ID:     "client-123",
		Status: models.ClientStatusActive,
	}

	mockAuth.On("AuthenticateClient", c.Request.Context(), "client-123", "secret-123", "203.0.113.1").Return(client, nil)
	mockRateLimit.On("CheckRateLimit", c.Request.Context(), "client-123").Return(true, int64(59), time.Now().Add(60*time.Second), nil)

	middleware(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockAuth.AssertExpectations(t)
	mockRateLimit.AssertExpectations(t)
}

func TestAuthMiddleware_UntrustedProxyHeadersIgnored(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockAuth := new(MockAuthService)
	mockRateLimit := new(MockRateLimitService)
	logger := zap.NewNop()

	trustedProxy, err := NewTrustedProxyList([]string{"10.0.0.0/8"})
	assert.NoError(t, err)

	config := AuthMiddlewareConfig{TrustedProxyChecker: trustedProxy}
	middleware := AuthMiddlewareWithConfig(mockAuth, mockRateLimit, logger, config)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", nil)
	c.Request.SetBasicAuth("client-123", "secret-123")
	c.Request.RemoteAddr = "192.168.1.1:8080"
	c.Request.Header.Set("X-Forwarded-For", "127.0.0.1")

	client := &models.Client{
		ID:     "client-123",
		Status: models.ClientStatusActive,
	}

	mockAuth.On("AuthenticateClient", c.Request.Context(), "client-123", "secret-123", "192.168.1.1").Return(client, nil)
	mockRateLimit.On("CheckRateLimit", c.Request.Context(), "client-123").Return(true, int64(59), time.Now().Add(60*time.Second), nil)

	middleware(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockAuth.AssertExpectations(t)
	mockRateLimit.AssertExpectations(t)
}

func TestAuthMiddleware_BearerTokenOpaque(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockAuth := new(MockAuthService)
	mockRateLimit := new(MockRateLimitService)
	logger := zap.NewNop()

	middleware := AuthMiddleware(mockAuth, mockRateLimit, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer opaque-token-with:colons")
	c.Request.RemoteAddr = "192.168.1.1:8080"

	client := &models.Client{
		ID:     "opaque-token-with:colons",
		Status: models.ClientStatusActive,
	}

	mockAuth.On("AuthenticateClient", c.Request.Context(), "opaque-token-with:colons", "", "192.168.1.1").Return(client, nil)
	mockRateLimit.On("CheckRateLimit", c.Request.Context(), "opaque-token-with:colons").Return(true, int64(59), time.Now().Add(60*time.Second), nil)

	middleware(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockAuth.AssertExpectations(t)
	mockRateLimit.AssertExpectations(t)
}
