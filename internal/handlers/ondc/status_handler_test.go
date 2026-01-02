package ondc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"uois-gateway/internal/models"
	"uois-gateway/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestStatusHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	callbackService := new(mockCallbackService)
	idempotencyService := new(mockIdempotencyService)
	orderServiceClient := new(mockOrderServiceClient)
	orderRecordService := new(mockOrderRecordService)
	auditService := new(mockAuditService)
	cacheService := new(mockCacheService)

	handler := NewStatusHandler(callbackService, idempotencyService, orderServiceClient, orderRecordService, auditService, cacheService, "test-bpp-id", "https://bpp.example.com", logger)

	clientOrderID := uuid.New().String()
	dispatchOrderID := uuid.New().String()
	transactionID := uuid.New().String()
	messageID := uuid.New().String()
	riderID := uuid.New().String()

	idempotencyService.On("CheckIdempotency", mock.Anything, mock.AnythingOfType("string")).Return(nil, false, nil)
	idempotencyService.On("StoreIdempotency", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

	// Mock audit service (optional, handler logs request/response and callbacks)
	auditService.On("LogRequestResponse", mock.Anything, mock.Anything).Return(nil).Maybe()
	auditService.On("LogCallbackDelivery", mock.Anything, mock.Anything).Return(nil).Maybe()

	// Mock cache miss
	cacheService.On("Get", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(false, nil)
	cacheService.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil).Maybe()

	// Mock order record lookup by client_id + order.id (ONDC)
	fulfillmentID := uuid.New().String()
	orderRecord := &OrderRecord{
		DispatchOrderID: dispatchOrderID,
		OrderID:         clientOrderID, // ONDC order.id (seller-generated)
		ClientID:        "test-client",
		FulfillmentID:   fulfillmentID, // Stable fulfillment ID (set in /init, reused in /status)
	}
	orderRecordService.On("GetOrderRecordByOrderID", mock.Anything, "test-client", clientOrderID).Return(orderRecord, nil)

	// Mock Order Service GetOrder
	orderStatus := &OrderStatus{
		DispatchOrderID: dispatchOrderID,
		State:           "CONFIRMED",
		RiderID:         riderID,
		Timeline: []OrderTimelineEvent{
			{Timestamp: time.Now(), Event: "ORDER_CONFIRMED", State: "CONFIRMED"},
		},
		Fulfillment: FulfillmentStatus{
			State: "PENDING",
		},
	}
	orderServiceClient.On("GetOrder", mock.Anything, dispatchOrderID).Return(orderStatus, nil)

	callbackService.On("SendCallback", mock.Anything, mock.MatchedBy(func(url string) bool {
		return strings.HasSuffix(url, "/on_status")
	}), mock.Anything).Return(nil).Maybe()

	requestBody := map[string]interface{}{
		"context": map[string]interface{}{
			"domain":         "nic2004:52110",
			"action":         "status",
			"transaction_id": transactionID,
			"message_id":     messageID,
			"timestamp":      time.Now().Format(time.RFC3339),
			"ttl":            "PT30S",
			"bap_uri":        "https://buyer.example.com",
		},
		"message": map[string]interface{}{
			"order": map[string]interface{}{
				"id": clientOrderID,
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("client", &models.Client{ID: "test-client", ClientCode: "test-client"})

	handler.HandleStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.ONDCACKResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ACK", response.Message.Ack.Status)

	time.Sleep(100 * time.Millisecond)

	callbackService.AssertExpectations(t)
	idempotencyService.AssertExpectations(t)
	orderServiceClient.AssertExpectations(t)
	orderRecordService.AssertExpectations(t)
}

func TestStatusHandler_OrderNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	callbackService := new(mockCallbackService)
	idempotencyService := new(mockIdempotencyService)
	orderServiceClient := new(mockOrderServiceClient)
	orderRecordService := new(mockOrderRecordService)
	auditService := new(mockAuditService)
	cacheService := new(mockCacheService)

	handler := NewStatusHandler(callbackService, idempotencyService, orderServiceClient, orderRecordService, auditService, cacheService, "test-bpp-id", "https://bpp.example.com", logger)

	clientOrderID := uuid.New().String()
	transactionID := uuid.New().String()
	messageID := uuid.New().String()

	idempotencyService.On("CheckIdempotency", mock.Anything, mock.AnythingOfType("string")).Return(nil, false, nil)

	// Mock audit service (optional, handler logs request/response)
	auditService.On("LogRequestResponse", mock.Anything, mock.Anything).Return(nil).Maybe()

	// Mock order record lookup failure
	orderRecordService.On("GetOrderRecordByOrderID", mock.Anything, "test-client", clientOrderID).Return(nil, errors.NewDomainError(65006, "order not found", "order_id not found"))

	requestBody := map[string]interface{}{
		"context": map[string]interface{}{
			"domain":         "nic2004:52110",
			"action":         "status",
			"transaction_id": transactionID,
			"message_id":     messageID,
			"timestamp":      time.Now().Format(time.RFC3339),
			"ttl":            "PT30S",
			"bap_uri":        "https://buyer.example.com",
		},
		"message": map[string]interface{}{
			"order": map[string]interface{}{
				"id": clientOrderID,
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("client", &models.Client{ID: "test-client", ClientCode: "test-client"})

	handler.HandleStatus(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	orderRecordService.AssertExpectations(t)
}
