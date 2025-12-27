package igm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"uois-gateway/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockIssueRepository struct {
	mock.Mock
}

func (m *MockIssueRepository) StoreIssue(ctx context.Context, issue *models.Issue) error {
	args := m.Called(ctx, issue)
	return args.Error(0)
}

func (m *MockIssueRepository) GetIssue(ctx context.Context, issueID string) (*models.Issue, error) {
	args := m.Called(ctx, issueID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Issue), args.Error(1)
}

func (m *MockIssueRepository) StoreZendeskReference(ctx context.Context, zendeskTicketID, issueID string) error {
	args := m.Called(ctx, zendeskTicketID, issueID)
	return args.Error(0)
}

func (m *MockIssueRepository) GetIssueIDByZendeskTicket(ctx context.Context, zendeskTicketID string) (string, error) {
	args := m.Called(ctx, zendeskTicketID)
	return args.String(0), args.Error(1)
}

type MockCallbackService struct {
	mock.Mock
}

func (m *MockCallbackService) SendCallback(ctx context.Context, callbackURL string, payload interface{}) error {
	args := m.Called(ctx, callbackURL, payload)
	return args.Error(0)
}

type MockIdempotencyService struct {
	mock.Mock
}

func (m *MockIdempotencyService) CheckIdempotency(ctx context.Context, key string) ([]byte, bool, error) {
	args := m.Called(ctx, key)
	return args.Get(0).([]byte), args.Bool(1), args.Error(2)
}

func (m *MockIdempotencyService) StoreIdempotency(ctx context.Context, key string, responseBytes []byte, ttl time.Duration) error {
	args := m.Called(ctx, key, responseBytes, ttl)
	return args.Error(0)
}

type MockGROService struct {
	mock.Mock
}

func (m *MockGROService) GetGRODetails(ctx context.Context, issueType models.IssueType) (*models.GRO, error) {
	args := m.Called(ctx, issueType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.GRO), args.Error(1)
}

func TestIssueHandler_HandleIssue_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockIssueRepository)
	mockCallback := new(MockCallbackService)
	mockIdempotency := new(MockIdempotencyService)
	mockGRO := new(MockGROService)

	logger := zap.NewNop()
	handler := NewIssueHandler(mockRepo, mockCallback, mockIdempotency, mockGRO, "bpp-1", "https://bpp.example.com", logger)

	reqBody := map[string]interface{}{
		"context": map[string]interface{}{
			"domain":      "nic2004:60232",
			"action":      "issue",
			"bap_id":      "buyer-1",
			"bap_uri":     "https://buyer.example.com",
			"transaction_id": "txn-123",
			"message_id":  "msg-456",
			"timestamp":   time.Now().Format(time.RFC3339),
		},
		"message": map[string]interface{}{
			"issue": map[string]interface{}{
				"id":          "issue-123",
				"issue_type":  "ISSUE",
				"category":    "ORDER",
				"description": "Test issue",
				"order_details": map[string]interface{}{
					"order_id": "order-789",
				},
			},
		},
	}

	mockIdempotency.On("CheckIdempotency", mock.Anything, mock.Anything).Return([]byte{}, false, nil)
	mockIdempotency.On("StoreIdempotency", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("StoreIssue", mock.Anything, mock.Anything).Return(nil)
	mockGRO.On("GetGRODetails", mock.Anything, models.IssueTypeIssue).Return(&models.GRO{Level: "L1"}, nil)
	mockCallback.On("SendCallback", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/issue", nil)
	c.Set("client", &models.Client{ID: "client-1"})

	c.Request.Header.Set("Content-Type", "application/json")
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request.Body = http.NoBody
	c.Request.Body = http.MaxBytesReader(w, httptest.NewRequest("POST", "/issue", nil).Body, 1048576)

	handler.HandleIssue(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
	mockCallback.AssertExpectations(t)
	mockIdempotency.AssertExpectations(t)
}

func TestIssueHandler_HandleIssue_InvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockIssueRepository)
	mockCallback := new(MockCallbackService)
	mockIdempotency := new(MockIdempotencyService)
	mockGRO := new(MockGROService)

	logger := zap.NewNop()
	handler := NewIssueHandler(mockRepo, mockCallback, mockIdempotency, mockGRO, "bpp-1", "https://bpp.example.com", logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/issue", nil)

	handler.HandleIssue(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

