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

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestUpdateHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	callbackService := new(mockCallbackService)
	idempotencyService := new(mockIdempotencyService)
	orderServiceClient := new(mockOrderServiceClient)
	orderRecordService := new(mockOrderRecordService)
	auditService := new(mockAuditService)

	handler := NewUpdateHandler(callbackService, idempotencyService, orderServiceClient, orderRecordService, auditService, "test-bpp-id", "https://bpp.example.com", logger)

	clientOrderID := uuid.New().String()
	dispatchOrderID := uuid.New().String()
	transactionID := uuid.New().String()
	messageID := uuid.New().String()

	idempotencyService.On("CheckIdempotency", mock.Anything, mock.AnythingOfType("string")).Return(nil, false, nil)
	idempotencyService.On("StoreIdempotency", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

	// Mock audit service (optional, handler logs request/response and callbacks)
	auditService.On("LogRequestResponse", mock.Anything, mock.Anything).Return(nil).Maybe()
	auditService.On("LogCallbackDelivery", mock.Anything, mock.Anything).Return(nil).Maybe()

	fulfillmentID := uuid.New().String()
	orderRecord := &OrderRecord{
		DispatchOrderID: dispatchOrderID,
		OrderID:         clientOrderID, // ONDC order.id (seller-generated)
		ClientID:        "test-client",
		FulfillmentID:   fulfillmentID, // Stable fulfillment ID (set in /init, reused in /update)
	}
	orderRecordService.On("GetOrderRecordByOrderID", mock.Anything, "test-client", clientOrderID).Return(orderRecord, nil)

	orderServiceClient.On("UpdateOrder", mock.Anything, dispatchOrderID, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	callbackService.On("SendCallback", mock.Anything, mock.MatchedBy(func(url string) bool {
		return strings.HasSuffix(url, "/on_update")
	}), mock.Anything).Return(nil).Maybe()

	requestBody := map[string]interface{}{
		"context": map[string]interface{}{
			"domain":         "nic2004:52110",
			"action":         "update",
			"transaction_id": transactionID,
			"message_id":     messageID,
			"timestamp":      time.Now().Format(time.RFC3339),
			"ttl":            "PT30S",
			"bap_uri":        "https://buyer.example.com",
		},
		"message": map[string]interface{}{
			"order": map[string]interface{}{
				"id": clientOrderID,
				"fulfillments": []map[string]interface{}{
					{
						"tracking": map[string]interface{}{
							"url": "https://track.example.com/123",
						},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("client", &models.Client{ID: "test-client", ClientCode: "test-client"})

	handler.HandleUpdate(c)

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
