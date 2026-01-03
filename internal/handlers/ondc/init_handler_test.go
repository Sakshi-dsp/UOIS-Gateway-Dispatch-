package ondc

import (
	"bytes"
	"context"
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

type mockOrderServiceClient struct {
	mock.Mock
}

func (m *mockOrderServiceClient) ValidateSearchIDTTL(ctx context.Context, searchID string) (bool, error) {
	args := m.Called(ctx, searchID)
	return args.Bool(0), args.Error(1)
}

func (m *mockOrderServiceClient) ValidateQuoteIDTTL(ctx context.Context, quoteID string) (bool, error) {
	args := m.Called(ctx, quoteID)
	return args.Bool(0), args.Error(1)
}

func (m *mockOrderServiceClient) GetOrder(ctx context.Context, dispatchOrderID string) (*OrderStatus, error) {
	args := m.Called(ctx, dispatchOrderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*OrderStatus), args.Error(1)
}

func (m *mockOrderServiceClient) GetOrderTracking(ctx context.Context, dispatchOrderID string) (*OrderTracking, error) {
	args := m.Called(ctx, dispatchOrderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*OrderTracking), args.Error(1)
}

func (m *mockOrderServiceClient) CancelOrder(ctx context.Context, dispatchOrderID string, reason string) error {
	args := m.Called(ctx, dispatchOrderID, reason)
	return args.Error(0)
}

func (m *mockOrderServiceClient) UpdateOrder(ctx context.Context, dispatchOrderID string, updates map[string]interface{}) error {
	args := m.Called(ctx, dispatchOrderID, updates)
	return args.Error(0)
}

func (m *mockOrderServiceClient) InitiateRTO(ctx context.Context, dispatchOrderID string) error {
	args := m.Called(ctx, dispatchOrderID)
	return args.Error(0)
}

type mockBillingStorageService struct {
	mock.Mock
}

func (m *mockBillingStorageService) StoreBilling(ctx context.Context, transactionID string, billing map[string]interface{}) error {
	args := m.Called(ctx, transactionID, billing)
	return args.Error(0)
}

func (m *mockBillingStorageService) GetBilling(ctx context.Context, transactionID string) (map[string]interface{}, error) {
	args := m.Called(ctx, transactionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *mockBillingStorageService) DeleteBilling(ctx context.Context, transactionID string) error {
	args := m.Called(ctx, transactionID)
	return args.Error(0)
}

type mockFulfillmentContactsStorageService struct {
	mock.Mock
}

func (m *mockFulfillmentContactsStorageService) StoreFulfillmentContacts(ctx context.Context, transactionID string, contacts map[string]interface{}) error {
	args := m.Called(ctx, transactionID, contacts)
	return args.Error(0)
}

func (m *mockFulfillmentContactsStorageService) GetFulfillmentContacts(ctx context.Context, transactionID string) (map[string]interface{}, error) {
	args := m.Called(ctx, transactionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *mockFulfillmentContactsStorageService) DeleteFulfillmentContacts(ctx context.Context, transactionID string) error {
	args := m.Called(ctx, transactionID)
	return args.Error(0)
}

type mockOrderRecordService struct {
	mock.Mock
}

func (m *mockOrderRecordService) StoreOrderRecord(ctx context.Context, record *OrderRecord) error {
	args := m.Called(ctx, record)
	return args.Error(0)
}

func (m *mockOrderRecordService) GetOrderRecordBySearchID(ctx context.Context, searchID string) (*OrderRecord, error) {
	args := m.Called(ctx, searchID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*OrderRecord), args.Error(1)
}

func (m *mockOrderRecordService) GetOrderRecordByQuoteID(ctx context.Context, quoteID string) (*OrderRecord, error) {
	args := m.Called(ctx, quoteID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*OrderRecord), args.Error(1)
}

func (m *mockOrderRecordService) GetOrderRecordByOrderID(ctx context.Context, clientID, orderID string) (*OrderRecord, error) {
	args := m.Called(ctx, clientID, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*OrderRecord), args.Error(1)
}

func (m *mockOrderRecordService) GetOrderRecordByTransactionID(ctx context.Context, transactionID string) (*OrderRecord, error) {
	args := m.Called(ctx, transactionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*OrderRecord), args.Error(1)
}

func (m *mockOrderRecordService) UpdateOrderRecord(ctx context.Context, record *OrderRecord) error {
	args := m.Called(ctx, record)
	return args.Error(0)
}

func TestInitHandler_Success(t *testing.T) {
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
	handler := NewInitHandler(eventPublisher, eventConsumer, callbackService, idempotencyService, orderServiceClient, orderRecordService, billingStorageService, fulfillmentContactsStorageService, auditService, "P1", "test-bpp-id", "https://bpp.example.com", logger)

	searchID := uuid.New().String()
	transactionID := uuid.New().String()
	messageID := uuid.New().String()
	quoteID := uuid.New().String()

	// Mock order record service - get order record by transaction_id
	orderRecord := &OrderRecord{
		SearchID:      searchID,
		TransactionID: transactionID,
	}
	orderRecordService.On("GetOrderRecordByTransactionID", mock.Anything, transactionID).Return(orderRecord, nil)

	// Mock Order Service validation
	orderServiceClient.On("ValidateSearchIDTTL", mock.Anything, searchID).Return(true, nil)

	// Mock idempotency check (no existing request)
	idempotencyService.On("CheckIdempotency", mock.Anything, mock.AnythingOfType("string")).Return(nil, false, nil)
	idempotencyService.On("StoreIdempotency", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

	// Mock successful event publishing
	eventPublisher.On("PublishEvent", mock.Anything, "stream.uois.init_requested", mock.AnythingOfType("*models.InitRequestedEvent")).Return(nil)

	// Mock successful event consumption (QUOTE_CREATED)
	quoteCreatedEvent := &models.QuoteCreatedEvent{
		BaseEvent: models.BaseEvent{
			EventType:   "QUOTE_CREATED",
			EventID:     uuid.New().String(),
			Traceparent: "00-12345678901234567890123456789012-1234567890123456-01",
			Timestamp:   time.Now(),
		},
		SearchID: searchID,
		QuoteID:  quoteID,
		Price:    models.Price{Value: 60.0, Currency: "INR"},
		Breakup: []models.BreakupItem{
			{
				ItemID:    "delivery_charge",
				TitleType: "delivery",
				Price:     models.Price{Value: 50.85, Currency: "INR"},
			},
			{
				ItemID:    "tax",
				TitleType: "tax",
				Price:     models.Price{Value: 9.15, Currency: "INR"},
			},
		},
		TTL:                         "PT15M",
		DistanceOriginToDestination: 5.5,
		ETAOrigin:                   timePtr(time.Now().Add(10 * time.Minute)),
		ETADestination:              timePtr(time.Now().Add(30 * time.Minute)),
	}
	eventConsumer.On("ConsumeEvent", mock.Anything, "stream.uois.quote_created", "uois-gateway-consumers", searchID, mock.AnythingOfType("time.Duration")).Return(quoteCreatedEvent, nil)

	// Mock order record update - store quote_id and fulfillment_id alongside search_id
	orderRecordService.On("UpdateOrderRecord", mock.Anything, mock.MatchedBy(func(record *OrderRecord) bool {
		return record.SearchID == searchID && record.QuoteID == quoteID && record.TransactionID == transactionID && record.FulfillmentID != ""
	})).Return(nil)

	// Mock successful callback
	callbackService.On("SendCallback", mock.Anything, mock.MatchedBy(func(url string) bool {
		return strings.HasSuffix(url, "/on_init")
	}), mock.Anything).Return(nil).Maybe()

	auditService.On("LogRequestResponse", mock.Anything, mock.Anything).Return(nil)
	auditService.On("LogCallbackDelivery", mock.Anything, mock.Anything).Return(nil).Maybe()

	// Create request
	requestBody := map[string]interface{}{
		"context": map[string]interface{}{
			"domain":         "nic2004:52110",
			"action":         "init",
			"transaction_id": transactionID,
			"message_id":     messageID,
			"timestamp":      time.Now().Format(time.RFC3339),
			"ttl":            "PT30S",
			"bap_uri":        "https://buyer.example.com",
		},
		"message": map[string]interface{}{
			"order": map[string]interface{}{
				"provider": map[string]interface{}{
					"id": "P1", // catalog provider.id (stable identifier)
				},
				"items": []map[string]interface{}{
					{
						"id": "item-1",
					},
				},
				"fulfillment": map[string]interface{}{
					"start": map[string]interface{}{
						"location": map[string]interface{}{
							"gps": "12.9716,77.5946",
							"address": map[string]interface{}{
								"name": "Pickup Address",
							},
						},
					},
					"end": map[string]interface{}{
						"location": map[string]interface{}{
							"gps": "12.9352,77.6245",
							"address": map[string]interface{}{
								"name": "Drop Address",
							},
						},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/init", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("client", &models.Client{ID: "test-client", ClientCode: "test-client"})

	handler.HandleInit(c)

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

func TestInitHandler_InvalidSearchID(t *testing.T) {
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
	handler := NewInitHandler(eventPublisher, eventConsumer, callbackService, idempotencyService, orderServiceClient, orderRecordService, billingStorageService, fulfillmentContactsStorageService, auditService, "P1", "test-bpp-id", "https://bpp.example.com", logger)

	searchID := uuid.New().String()
	transactionID := uuid.New().String()
	messageID := uuid.New().String()

	// Mock idempotency check
	idempotencyService.On("CheckIdempotency", mock.Anything, mock.AnythingOfType("string")).Return(nil, false, nil)

	// Mock order record service - get order record by transaction_id
	orderRecord := &OrderRecord{
		SearchID:      searchID,
		TransactionID: transactionID,
	}
	orderRecordService.On("GetOrderRecordByTransactionID", mock.Anything, transactionID).Return(orderRecord, nil)

	// Mock Order Service validation failure (search_id expired)
	orderServiceClient.On("ValidateSearchIDTTL", mock.Anything, searchID).Return(false, errors.NewDomainError(65004, "quote expired", "search_id TTL expired"))

	auditService.On("LogRequestResponse", mock.Anything, mock.Anything).Return(nil)

	// Create request
	requestBody := map[string]interface{}{
		"context": map[string]interface{}{
			"domain":         "nic2004:52110",
			"action":         "init",
			"transaction_id": transactionID,
			"message_id":     messageID,
			"timestamp":      time.Now().Format(time.RFC3339),
			"ttl":            "PT30S",
			"bap_uri":        "https://buyer.example.com",
		},
		"message": map[string]interface{}{
			"order": map[string]interface{}{
				"provider": map[string]interface{}{
					"id": "P1", // catalog provider.id (stable identifier)
				},
				"fulfillment": map[string]interface{}{
					"start": map[string]interface{}{
						"location": map[string]interface{}{
							"gps": "12.9716,77.5946",
						},
					},
					"end": map[string]interface{}{
						"location": map[string]interface{}{
							"gps": "12.9352,77.6245",
						},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/init", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("client", &models.Client{ID: "test-client", ClientCode: "test-client"})

	handler.HandleInit(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	orderServiceClient.AssertExpectations(t)
}

func TestInitHandler_QuoteInvalidated(t *testing.T) {
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
	handler := NewInitHandler(eventPublisher, eventConsumer, callbackService, idempotencyService, orderServiceClient, orderRecordService, billingStorageService, fulfillmentContactsStorageService, auditService, "P1", "test-bpp-id", "https://bpp.example.com", logger)

	searchID := uuid.New().String()
	transactionID := uuid.New().String()
	messageID := uuid.New().String()

	// Mock order record service - get order record by transaction_id
	orderRecord := &OrderRecord{
		SearchID:      searchID,
		TransactionID: transactionID,
	}
	orderRecordService.On("GetOrderRecordByTransactionID", mock.Anything, transactionID).Return(orderRecord, nil)

	// Mock Order Service validation
	orderServiceClient.On("ValidateSearchIDTTL", mock.Anything, searchID).Return(true, nil)

	idempotencyService.On("CheckIdempotency", mock.Anything, mock.AnythingOfType("string")).Return(nil, false, nil)
	idempotencyService.On("StoreIdempotency", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

	eventPublisher.On("PublishEvent", mock.Anything, "stream.uois.init_requested", mock.AnythingOfType("*models.InitRequestedEvent")).Return(nil)

	// Mock QUOTE_CREATED consumption failure (will try QUOTE_INVALIDATED next)
	eventConsumer.On("ConsumeEvent", mock.Anything, "stream.uois.quote_created", "uois-gateway-consumers", searchID, mock.AnythingOfType("time.Duration")).Return(nil, errors.NewDomainError(65020, "event not found", "no quote created"))

	// Mock QUOTE_INVALIDATED event consumption
	quoteInvalidatedEvent := &models.QuoteInvalidatedEvent{
		BaseEvent: models.BaseEvent{
			EventType:   "QUOTE_INVALIDATED",
			EventID:     uuid.New().String(),
			Traceparent: "00-12345678901234567890123456789012-1234567890123456-01",
			Timestamp:   time.Now(),
		},
		SearchID:         searchID,
		QuoteID:          uuid.New().String(),
		Error:            "QUOTE_INVALIDATED",
		Message:          "Quote invalidated",
		RequiresResearch: true,
	}
	eventConsumer.On("ConsumeEvent", mock.Anything, "stream.uois.quote_invalidated", "uois-gateway-consumers", searchID, mock.AnythingOfType("time.Duration")).Return(quoteInvalidatedEvent, nil)

	callbackService.On("SendCallback", mock.Anything, mock.MatchedBy(func(url string) bool {
		return strings.HasSuffix(url, "/on_init")
	}), mock.Anything).Return(nil).Maybe()

	auditService.On("LogRequestResponse", mock.Anything, mock.Anything).Return(nil)
	auditService.On("LogCallbackDelivery", mock.Anything, mock.Anything).Return(nil).Maybe()

	requestBody := map[string]interface{}{
		"context": map[string]interface{}{
			"domain":         "nic2004:52110",
			"action":         "init",
			"transaction_id": transactionID,
			"message_id":     messageID,
			"timestamp":      time.Now().Format(time.RFC3339),
			"ttl":            "PT30S",
			"bap_uri":        "https://buyer.example.com",
		},
		"message": map[string]interface{}{
			"order": map[string]interface{}{
				"provider": map[string]interface{}{
					"id": "P1", // catalog provider.id (stable identifier)
				},
				"fulfillment": map[string]interface{}{
					"start": map[string]interface{}{
						"location": map[string]interface{}{
							"gps": "12.9716,77.5946",
						},
					},
					"end": map[string]interface{}{
						"location": map[string]interface{}{
							"gps": "12.9352,77.6245",
						},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/init", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("client", &models.Client{ID: "test-client", ClientCode: "test-client"})

	handler.HandleInit(c)

	assert.Equal(t, http.StatusOK, w.Code)

	time.Sleep(100 * time.Millisecond)

	eventConsumer.AssertExpectations(t)
	callbackService.AssertExpectations(t)
}
