package ondc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"uois-gateway/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestTrackHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	callbackService := new(mockCallbackService)
	idempotencyService := new(mockIdempotencyService)
	orderServiceClient := new(mockOrderServiceClient)
	orderRecordService := new(mockOrderRecordService)
	auditService := new(mockAuditService)
	cacheService := new(mockCacheService)

	handler := NewTrackHandler(callbackService, idempotencyService, orderServiceClient, orderRecordService, auditService, cacheService, "test-bpp-id", "https://bpp.example.com", logger)

	clientOrderID := uuid.New().String()
	dispatchOrderID := uuid.New().String()
	transactionID := uuid.New().String()
	messageID := uuid.New().String()

	idempotencyService.On("CheckIdempotency", mock.Anything, mock.AnythingOfType("string")).Return(nil, false, nil)
	idempotencyService.On("StoreIdempotency", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

	// Mock audit service (optional, handler logs request/response)
	auditService.On("LogRequestResponse", mock.Anything, mock.Anything).Return(nil).Maybe()

	// Mock cache miss
	cacheService.On("Get", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(false, nil)
	cacheService.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil).Maybe()

	fulfillmentID := uuid.New().String()
	orderRecord := &OrderRecord{
		DispatchOrderID: dispatchOrderID,
		OrderID:         clientOrderID, // ONDC order.id (seller-generated)
		ClientID:        "test-client",
		FulfillmentID:   fulfillmentID, // Stable fulfillment ID (set in /init, reused in /track)
	}
	orderRecordService.On("GetOrderRecordByOrderID", mock.Anything, "test-client", clientOrderID).Return(orderRecord, nil)

	orderTracking := &OrderTracking{
		DispatchOrderID: dispatchOrderID,
		CurrentLocation: Location{Lat: 12.9716, Lng: 77.5946},
		ETA:             time.Now().Add(30 * time.Minute),
		TrackingURL:     "https://track.example.com/order/123",
		Timeline: []OrderTimelineEvent{
			{Timestamp: time.Now(), Event: "ORDER_CONFIRMED", State: "CONFIRMED"},
		},
	}
	orderServiceClient.On("GetOrderTracking", mock.Anything, dispatchOrderID).Return(orderTracking, nil)

	// ONDC v1.2.0: /track is SYNC only, no callback
	// callbackService should NOT be called

	requestBody := map[string]interface{}{
		"context": map[string]interface{}{
			"domain":         "nic2004:52110",
			"action":         "track",
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
	req := httptest.NewRequest(http.MethodPost, "/track", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("client", &models.Client{ID: "test-client", ClientCode: "test-client"})

	handler.HandleTrack(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// ONDC v1.2.0: /track returns SYNC response with tracking data (not just ACK)
	var response models.ONDCResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response.Message)

	// Verify response contains order.fulfillments[] with tracking data
	order, ok := response.Message["order"].(map[string]interface{})
	assert.True(t, ok, "response should contain order")

	fulfillments, ok := order["fulfillments"].([]interface{})
	assert.True(t, ok, "order should contain fulfillments array")
	assert.Greater(t, len(fulfillments), 0, "fulfillments array should not be empty")

	fulfillment, ok := fulfillments[0].(map[string]interface{})
	assert.True(t, ok, "first fulfillment should be a map")
	assert.Equal(t, fulfillmentID, fulfillment["id"], "fulfillment.id should match")
	assert.Equal(t, true, fulfillment["tracking"], "tracking should be enabled")
	assert.Contains(t, fulfillment, "tracking_url", "fulfillment should contain tracking_url")

	idempotencyService.AssertExpectations(t)
	orderServiceClient.AssertExpectations(t)
	orderRecordService.AssertExpectations(t)
}
