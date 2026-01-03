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

func TestConfirmHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	eventPublisher := new(mockEventPublisher)
	eventConsumer := new(mockEventConsumer)
	callbackService := new(mockCallbackService)
	idempotencyService := new(mockIdempotencyService)
	orderServiceClient := new(mockOrderServiceClient)
	orderRecordService := new(mockOrderRecordService)
	billingStorageService := new(mockBillingStorageService)
	fulfillmentContactsStorageService := new(mockFulfillmentContactsStorageService)
	auditService := new(mockAuditService)

	handler := NewConfirmHandler(eventPublisher, eventConsumer, callbackService, idempotencyService, orderServiceClient, orderRecordService, billingStorageService, fulfillmentContactsStorageService, auditService, "test-bpp-id", "https://bpp.example.com", logger)

	quoteID := uuid.New().String()
	transactionID := uuid.New().String()
	messageID := uuid.New().String()
	clientOrderID := uuid.New().String()
	dispatchOrderID := uuid.New().String()
	riderID := uuid.New().String()

	// Mock idempotency check
	idempotencyService.On("CheckIdempotency", mock.Anything, mock.AnythingOfType("string")).Return(nil, false, nil)
	idempotencyService.On("StoreIdempotency", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

	// Mock billing storage (billing retrieval is optional, may return nil)
	billingStorageService.On("GetBilling", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil).Maybe()
	// Mock fulfillment contacts storage (contacts retrieval is optional, may return nil)
	fulfillmentContactsStorageService.On("GetFulfillmentContacts", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil).Maybe()

	// Mock Order Service validation (quote_id TTL check)
	orderServiceClient.On("ValidateQuoteIDTTL", mock.Anything, quoteID).Return(true, nil)

	// Mock successful event publishing
	eventPublisher.On("PublishEvent", mock.Anything, "stream.uois.confirm_requested", mock.AnythingOfType("*models.ConfirmRequestedEvent")).Return(nil)

	// Mock successful event consumption (ORDER_CONFIRMED)
	orderConfirmedEvent := &models.OrderConfirmedEvent{
		BaseEvent: models.BaseEvent{
			EventType:   "ORDER_CONFIRMED",
			EventID:     uuid.New().String(),
			Traceparent: "00-12345678901234567890123456789012-1234567890123456-01",
			Timestamp:   time.Now(),
		},
		QuoteID:         quoteID,
		DispatchOrderID: dispatchOrderID,
		RiderID:         riderID,
	}
	eventConsumer.On("ConsumeEvent", mock.Anything, "stream.uois.order_confirmed", "uois-gateway-consumers", quoteID, mock.AnythingOfType("time.Duration")).Return(orderConfirmedEvent, nil)

	// Mock order record service - get order record by quote_id
	orderRecord := &OrderRecord{
		SearchID:      uuid.New().String(),
		QuoteID:       quoteID,
		TransactionID: transactionID, // Required for validation
	}
	orderRecordService.On("GetOrderRecordByQuoteID", mock.Anything, quoteID).Return(orderRecord, nil)

	// Mock order record update - store dispatch_order_id and buyer-provided order.id
	// ONDC spec: Buyer provides order.id in /confirm, Seller echoes it back
	orderRecordService.On("UpdateOrderRecord", mock.Anything, mock.MatchedBy(func(record *OrderRecord) bool {
		return record.SearchID == orderRecord.SearchID &&
			record.QuoteID == quoteID &&
			record.DispatchOrderID == dispatchOrderID &&
			record.OrderID == clientOrderID && // Buyer-provided order.id (echoed back)
			record.ClientID == "test-client"
	})).Return(nil)

	auditService.On("LogRequestResponse", mock.Anything, mock.Anything).Return(nil).Maybe()
	auditService.On("LogCallbackDelivery", mock.Anything, mock.Anything).Return(nil).Maybe()

	// Mock successful callback
	callbackService.On("SendCallback", mock.Anything, mock.MatchedBy(func(url string) bool {
		return strings.HasSuffix(url, "/on_confirm")
	}), mock.Anything).Return(nil).Maybe()

	// Create request
	requestBody := map[string]interface{}{
		"context": map[string]interface{}{
			"domain":         "nic2004:52110",
			"action":         "confirm",
			"transaction_id": transactionID,
			"message_id":     messageID,
			"timestamp":      time.Now().Format(time.RFC3339),
			"ttl":            "PT30S",
			"bap_uri":        "https://buyer.example.com",
		},
		"message": map[string]interface{}{
			"order": map[string]interface{}{
				"id": clientOrderID,
				"quote": map[string]interface{}{
					"id": quoteID,
				},
				"payment": map[string]interface{}{
					"@ondc/org/settlement_details": []map[string]interface{}{
						{
							"settlement_counterparty": "buyer",
							"settlement_phase":        "sale-amount",
						},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/confirm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("client", &models.Client{ID: "test-client", ClientCode: "test-client"})

	handler.HandleConfirm(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.ONDCACKResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ACK", response.Message.Ack.Status)

	time.Sleep(100 * time.Millisecond)

	eventPublisher.AssertExpectations(t)
	eventConsumer.AssertExpectations(t)
	callbackService.AssertExpectations(t)
	idempotencyService.AssertExpectations(t)
	orderServiceClient.AssertExpectations(t)
	orderRecordService.AssertExpectations(t)
}

func TestConfirmHandler_InvalidQuoteID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	eventPublisher := new(mockEventPublisher)
	eventConsumer := new(mockEventConsumer)
	callbackService := new(mockCallbackService)
	idempotencyService := new(mockIdempotencyService)
	orderServiceClient := new(mockOrderServiceClient)
	orderRecordService := new(mockOrderRecordService)
	billingStorageService := new(mockBillingStorageService)
	fulfillmentContactsStorageService := new(mockFulfillmentContactsStorageService)

	auditService := new(mockAuditService)
	handler := NewConfirmHandler(eventPublisher, eventConsumer, callbackService, idempotencyService, orderServiceClient, orderRecordService, billingStorageService, fulfillmentContactsStorageService, auditService, "test-bpp-id", "https://bpp.example.com", logger)

	transactionID := uuid.New().String()
	messageID := uuid.New().String()

	idempotencyService.On("CheckIdempotency", mock.Anything, mock.AnythingOfType("string")).Return(nil, false, nil)
	auditService.On("LogRequestResponse", mock.Anything, mock.Anything).Return(nil)

	// Note: ValidateQuoteIDTTL should not be called when quote_id is missing
	// The handler should return error during extraction

	requestBody := map[string]interface{}{
		"context": map[string]interface{}{
			"domain":         "nic2004:52110",
			"action":         "confirm",
			"transaction_id": transactionID,
			"message_id":     messageID,
			"timestamp":      time.Now().Format(time.RFC3339),
			"ttl":            "PT30S",
			"bap_uri":        "https://buyer.example.com",
		},
		"message": map[string]interface{}{
			"order": map[string]interface{}{
				"quote": map[string]interface{}{},
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/confirm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("client", &models.Client{ID: "test-client", ClientCode: "test-client"})

	handler.HandleConfirm(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	orderServiceClient.AssertExpectations(t)
}

func TestConfirmHandler_OrderConfirmFailed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	eventPublisher := new(mockEventPublisher)
	eventConsumer := new(mockEventConsumer)
	callbackService := new(mockCallbackService)
	idempotencyService := new(mockIdempotencyService)
	orderServiceClient := new(mockOrderServiceClient)
	orderRecordService := new(mockOrderRecordService)
	billingStorageService := new(mockBillingStorageService)
	fulfillmentContactsStorageService := new(mockFulfillmentContactsStorageService)

	auditService := new(mockAuditService)
	handler := NewConfirmHandler(eventPublisher, eventConsumer, callbackService, idempotencyService, orderServiceClient, orderRecordService, billingStorageService, fulfillmentContactsStorageService, auditService, "test-bpp-id", "https://bpp.example.com", logger)

	quoteID := uuid.New().String()
	transactionID := uuid.New().String()
	messageID := uuid.New().String()
	clientOrderID := uuid.New().String()

	idempotencyService.On("CheckIdempotency", mock.Anything, mock.AnythingOfType("string")).Return(nil, false, nil)
	idempotencyService.On("StoreIdempotency", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

	orderServiceClient.On("ValidateQuoteIDTTL", mock.Anything, quoteID).Return(true, nil)

	// Mock order record service - get order record by quote_id
	orderRecord := &OrderRecord{
		SearchID: uuid.New().String(),
		QuoteID:  quoteID,
	}
	orderRecordService.On("GetOrderRecordByQuoteID", mock.Anything, quoteID).Return(orderRecord, nil)

	eventPublisher.On("PublishEvent", mock.Anything, "stream.uois.confirm_requested", mock.AnythingOfType("*models.ConfirmRequestedEvent")).Return(nil)

	// Mock ORDER_CONFIRM_FAILED event consumption
	orderConfirmFailedEvent := &models.OrderConfirmFailedEvent{
		BaseEvent: models.BaseEvent{
			EventType:   "ORDER_CONFIRM_FAILED",
			EventID:     uuid.New().String(),
			Traceparent: "00-12345678901234567890123456789012-1234567890123456-01",
			Timestamp:   time.Now(),
		},
		QuoteID: quoteID,
		Reason:  "Order confirmation failed",
	}

	// Mock ORDER_CONFIRMED consumption failure (will try ORDER_CONFIRM_FAILED next)
	eventConsumer.On("ConsumeEvent", mock.Anything, "stream.uois.order_confirmed", "uois-gateway-consumers", quoteID, mock.AnythingOfType("time.Duration")).Return(nil, errors.NewDomainError(65020, "event not found", "no order confirmed"))

	eventConsumer.On("ConsumeEvent", mock.Anything, "stream.uois.order_confirm_failed", "uois-gateway-consumers", quoteID, mock.AnythingOfType("time.Duration")).Return(orderConfirmFailedEvent, nil)

	callbackService.On("SendCallback", mock.Anything, mock.MatchedBy(func(url string) bool {
		return strings.HasSuffix(url, "/on_confirm")
	}), mock.Anything).Return(nil).Maybe()

	auditService.On("LogRequestResponse", mock.Anything, mock.Anything).Return(nil)
	auditService.On("LogCallbackDelivery", mock.Anything, mock.Anything).Return(nil).Maybe()

	requestBody := map[string]interface{}{
		"context": map[string]interface{}{
			"domain":         "nic2004:52110",
			"action":         "confirm",
			"transaction_id": transactionID,
			"message_id":     messageID,
			"timestamp":      time.Now().Format(time.RFC3339),
			"ttl":            "PT30S",
			"bap_uri":        "https://buyer.example.com",
		},
		"message": map[string]interface{}{
			"order": map[string]interface{}{
				"id": clientOrderID,
				"quote": map[string]interface{}{
					"id": quoteID,
				},
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/confirm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("client", &models.Client{ID: "test-client", ClientCode: "test-client"})

	handler.HandleConfirm(c)

	assert.Equal(t, http.StatusOK, w.Code)

	time.Sleep(100 * time.Millisecond)

	eventConsumer.AssertExpectations(t)
	callbackService.AssertExpectations(t)
}
