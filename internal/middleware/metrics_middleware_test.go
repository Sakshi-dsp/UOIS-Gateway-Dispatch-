package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockMetricsService struct {
	mock.Mock
}

func (m *MockMetricsService) RecordRequest(endpoint, clientID, protocol, status string) {
	m.Called(endpoint, clientID, protocol, status)
}

func (m *MockMetricsService) RecordRequestDuration(endpoint, status string, duration time.Duration) {
	m.Called(endpoint, status, duration)
}

func TestMetricsMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		endpoint       string
		statusCode     int
		expectedStatus string
	}{
		{
			name:           "successful request",
			endpoint:       "/ondc/search",
			statusCode:     200,
			expectedStatus: "success",
		},
		{
			name:           "error request",
			endpoint:       "/ondc/search",
			statusCode:     400,
			expectedStatus: "client_error",
		},
		{
			name:           "server error",
			endpoint:       "/ondc/search",
			statusCode:     500,
			expectedStatus: "server_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMetrics := new(MockMetricsService)
			logger := zap.NewNop()

			mockMetrics.On("RecordRequest", tt.endpoint, "", "http", tt.expectedStatus).Once()
			mockMetrics.On("RecordRequestDuration", tt.endpoint, tt.expectedStatus, mock.AnythingOfType("time.Duration")).Once()

			router := gin.New()
			router.Use(MetricsMiddleware(mockMetrics, logger))
			router.POST(tt.endpoint, func(c *gin.Context) {
				c.Status(tt.statusCode)
			})

			req := httptest.NewRequest("POST", tt.endpoint, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.statusCode, w.Code)
			mockMetrics.AssertExpectations(t)
		})
	}
}

func TestMetricsMiddlewareExtractsEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockMetrics := new(MockMetricsService)
	logger := zap.NewNop()

	mockMetrics.On("RecordRequest", "/ondc/search", "", "http", "success").Once()
	mockMetrics.On("RecordRequestDuration", "/ondc/search", "success", mock.AnythingOfType("time.Duration")).Once()

	router := gin.New()
	router.Use(MetricsMiddleware(mockMetrics, logger))
	router.POST("/ondc/search", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/ondc/search", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockMetrics.AssertExpectations(t)
}

func TestMetricsMiddlewareRecordsDuration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockMetrics := new(MockMetricsService)
	logger := zap.NewNop()

	var recordedDuration time.Duration
	mockMetrics.On("RecordRequest", "/ondc/search", "", "http", "success").Once()
	mockMetrics.On("RecordRequestDuration", "/ondc/search", "success", mock.AnythingOfType("time.Duration")).
		Run(func(args mock.Arguments) {
			recordedDuration = args.Get(2).(time.Duration)
		}).Once()

	router := gin.New()
	router.Use(MetricsMiddleware(mockMetrics, logger))
	router.POST("/ondc/search", func(c *gin.Context) {
		time.Sleep(10 * time.Millisecond)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/ondc/search", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.GreaterOrEqual(t, recordedDuration, 10*time.Millisecond)
	mockMetrics.AssertExpectations(t)
}
