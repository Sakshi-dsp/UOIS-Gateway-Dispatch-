package igm

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"uois-gateway/internal/models"
	"uois-gateway/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestIssueStatusHandler_HandleIssueStatus_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockIssueRepository)
	mockCallback := new(MockCallbackService)
	mockIdempotency := new(MockIdempotencyService)
	mockGRO := new(MockGROService)

	logger := zap.NewNop()
	handler := NewIssueStatusHandler(mockRepo, mockCallback, mockIdempotency, mockGRO, "bpp-1", "https://bpp.example.com", logger)

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

	reqBody := map[string]interface{}{
		"context": map[string]interface{}{
			"domain":         "nic2004:60232",
			"action":         "issue_status",
			"bap_id":         "buyer-1",
			"bap_uri":        "https://buyer.example.com",
			"transaction_id": "txn-123",
			"message_id":     "msg-456",
			"timestamp":      time.Now().Format(time.RFC3339),
		},
		"message": map[string]interface{}{
			"issue_id": "issue-123",
		},
	}

	mockIdempotency.On("CheckIdempotency", mock.Anything, mock.Anything).Return([]byte{}, false, nil)
	mockIdempotency.On("StoreIdempotency", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("GetIssue", mock.Anything, "issue-123").Return(issue, nil)
	mockGRO.On("GetGRODetails", mock.Anything, models.IssueTypeIssue).Return(&models.GRO{Level: "L1"}, nil)
	mockCallback.On("SendCallback", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/issue_status", nil)
	c.Set("client", &models.Client{ID: "client-1"})

	c.Request.Header.Set("Content-Type", "application/json")
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request.Body = http.MaxBytesReader(w, httptest.NewRequest("POST", "/issue_status", bytes.NewReader(bodyBytes)).Body, 1048576)

	handler.HandleIssueStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
	mockCallback.AssertExpectations(t)
	mockIdempotency.AssertExpectations(t)
}

func TestIssueStatusHandler_HandleIssueStatus_IssueNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockIssueRepository)
	mockCallback := new(MockCallbackService)
	mockIdempotency := new(MockIdempotencyService)
	mockGRO := new(MockGROService)

	logger := zap.NewNop()
	handler := NewIssueStatusHandler(mockRepo, mockCallback, mockIdempotency, mockGRO, "bpp-1", "https://bpp.example.com", logger)

	reqBody := map[string]interface{}{
		"context": map[string]interface{}{
			"domain":         "nic2004:60232",
			"action":         "issue_status",
			"bap_id":         "buyer-1",
			"bap_uri":        "https://buyer.example.com",
			"transaction_id": "txn-123",
			"message_id":     "msg-456",
			"timestamp":      time.Now().Format(time.RFC3339),
		},
		"message": map[string]interface{}{
			"issue_id": "issue-999",
		},
	}

	mockIdempotency.On("CheckIdempotency", mock.Anything, mock.Anything).Return([]byte{}, false, nil)
	mockRepo.On("GetIssue", mock.Anything, "issue-999").Return(nil, errors.NewDomainError(65006, "issue not found", "issue_id not found"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/issue_status", nil)

	c.Request.Header.Set("Content-Type", "application/json")
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request.Body = http.MaxBytesReader(w, httptest.NewRequest("POST", "/issue_status", bytes.NewReader(bodyBytes)).Body, 1048576)

	handler.HandleIssueStatus(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}
