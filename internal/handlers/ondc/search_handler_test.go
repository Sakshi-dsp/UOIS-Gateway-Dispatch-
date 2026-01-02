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
	"uois-gateway/internal/services/audit"
	"uois-gateway/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// Mock dependencies
type mockEventPublisher struct {
	mock.Mock
}

func (m *mockEventPublisher) PublishEvent(ctx context.Context, stream string, event interface{}) error {
	args := m.Called(ctx, stream, event)
	return args.Error(0)
}

type mockEventConsumer struct {
	mock.Mock
}

func (m *mockEventConsumer) ConsumeEvent(ctx context.Context, stream, consumerGroup, correlationID string, timeout time.Duration) (interface{}, error) {
	args := m.Called(ctx, stream, consumerGroup, correlationID, timeout)
	return args.Get(0), args.Error(1)
}

type mockCallbackService struct {
	mock.Mock
}

func (m *mockCallbackService) SendCallback(ctx context.Context, callbackURL string, payload interface{}) error {
	args := m.Called(ctx, callbackURL, payload)
	return args.Error(0)
}

type mockIdempotencyService struct {
	mock.Mock
}

func (m *mockIdempotencyService) CheckIdempotency(ctx context.Context, key string) ([]byte, bool, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).([]byte), args.Bool(1), args.Error(2)
}

func (m *mockIdempotencyService) StoreIdempotency(ctx context.Context, key string, responseBytes []byte, ttl time.Duration) error {
	args := m.Called(ctx, key, responseBytes, ttl)
	return args.Error(0)
}

type mockAuditService struct {
	mock.Mock
}

func (m *mockAuditService) LogRequestResponse(ctx context.Context, req *audit.RequestResponseLogParams) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *mockAuditService) LogCallbackDelivery(ctx context.Context, req *audit.CallbackDeliveryLogParams) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

type mockCacheService struct {
	mock.Mock
}

func (m *mockCacheService) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	args := m.Called(ctx, key, dest)
	return args.Bool(0), args.Error(1)
}

func (m *mockCacheService) Set(ctx context.Context, key string, value interface{}) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *mockCacheService) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func TestSearchHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	eventPublisher := new(mockEventPublisher)
	eventConsumer := new(mockEventConsumer)
	callbackService := new(mockCallbackService)
	idempotencyService := new(mockIdempotencyService)

	orderRecordService := new(mockOrderRecordService)
	auditService := new(mockAuditService)
	handler := NewSearchHandler(eventPublisher, eventConsumer, callbackService, idempotencyService, orderRecordService, auditService, "P1", "bpp.example.com", "https://bpp.example.com", "Test BPP", "https://bpp.example.com/terms", logger)

	transactionID := uuid.New().String()
	messageID := uuid.New().String()

	// Mock successful event publishing
	eventPublisher.On("PublishEvent", mock.Anything, "stream.location.search", mock.AnythingOfType("*models.SearchRequestedEvent")).Return(nil)

	// Mock successful event consumption (accept any searchID since handler generates it)
	quoteComputedEvent := &models.QuoteComputedEvent{
		BaseEvent: models.BaseEvent{
			EventType:   "QUOTE_COMPUTED",
			EventID:     uuid.New().String(),
			Traceparent: "00-12345678901234567890123456789012-1234567890123456-01",
			Timestamp:   time.Now(),
		},
		SearchID:                    uuid.New().String(), // Will be matched by mock
		Serviceable:                 true,
		Price:                       models.Price{Value: 60.0, Currency: "INR"},
		TTL:                         "PT10M",
		DistanceOriginToDestination: 5.5,
		ETAOrigin:                   timePtr(time.Now().Add(10 * time.Minute)),
		ETADestination:              timePtr(time.Now().Add(30 * time.Minute)),
	}
	eventConsumer.On("ConsumeEvent", mock.Anything, "quote:computed", "uois-gateway-consumers", mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(quoteComputedEvent, nil)

	// Mock successful callback (called asynchronously, so we'll wait for it)
	callbackService.On("SendCallback", mock.Anything, mock.MatchedBy(func(url string) bool {
		return strings.HasSuffix(url, "/on_search")
	}), mock.Anything).Return(nil).Maybe() // Maybe() allows the call to happen asynchronously

	// Mock idempotency check (no existing request)
	idempotencyService.On("CheckIdempotency", mock.Anything, mock.AnythingOfType("string")).Return(nil, false, nil)
	idempotencyService.On("StoreIdempotency", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

	// Mock successful order record storage
	orderRecordService.On("StoreOrderRecord", mock.Anything, mock.AnythingOfType("*ondc.OrderRecord")).Return(nil)

	auditService.On("LogRequestResponse", mock.Anything, mock.Anything).Return(nil)
	auditService.On("LogCallbackDelivery", mock.Anything, mock.Anything).Return(nil).Maybe()

	// Create request
	requestBody := map[string]interface{}{
		"context": map[string]interface{}{
			"domain":         "nic2004:52110",
			"action":         "search",
			"transaction_id": transactionID,
			"message_id":     messageID,
			"timestamp":      time.Now().Format(time.RFC3339),
			"ttl":            "PT30S",
			"bap_uri":        "https://buyer.example.com",
		},
		"message": map[string]interface{}{
			"intent": map[string]interface{}{
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
	req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("client", &models.Client{ID: "test-client", ClientCode: "test-client"})

	// Execute handler
	handler.HandleSearch(c)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.ONDCACKResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ACK", response.Message.Ack.Status)

	// Wait a bit for async callback (callback is sent in goroutine)
	time.Sleep(100 * time.Millisecond)

	eventPublisher.AssertExpectations(t)
	eventConsumer.AssertExpectations(t)
	callbackService.AssertExpectations(t)
	idempotencyService.AssertExpectations(t)
	orderRecordService.AssertExpectations(t)
}

func TestSearchHandler_InvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	eventPublisher := new(mockEventPublisher)
	eventConsumer := new(mockEventConsumer)
	callbackService := new(mockCallbackService)
	idempotencyService := new(mockIdempotencyService)

	orderRecordService := new(mockOrderRecordService)
	auditService := new(mockAuditService)
	handler := NewSearchHandler(eventPublisher, eventConsumer, callbackService, idempotencyService, orderRecordService, auditService, "P1", "bpp.example.com", "https://bpp.example.com", "Test BPP", "https://bpp.example.com/terms", logger)

	auditService.On("LogRequestResponse", mock.Anything, mock.Anything).Return(nil)

	// Create invalid request (missing context)
	requestBody := map[string]interface{}{
		"message": map[string]interface{}{},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("client", &models.Client{ID: "test-client", ClientCode: "test-client"})

	// Execute handler
	handler.HandleSearch(c)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ONDCResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response.Error)
}

func TestSearchHandler_EventPublishFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	eventPublisher := new(mockEventPublisher)
	eventConsumer := new(mockEventConsumer)
	callbackService := new(mockCallbackService)
	idempotencyService := new(mockIdempotencyService)

	orderRecordService := new(mockOrderRecordService)
	auditService := new(mockAuditService)
	handler := NewSearchHandler(eventPublisher, eventConsumer, callbackService, idempotencyService, orderRecordService, auditService, "P1", "bpp.example.com", "https://bpp.example.com", "Test BPP", "https://bpp.example.com/terms", logger)

	transactionID := uuid.New().String()
	messageID := uuid.New().String()

	// Mock idempotency check (no existing request)
	idempotencyService.On("CheckIdempotency", mock.Anything, mock.AnythingOfType("string")).Return(nil, false, nil)

	// Mock successful order record storage
	orderRecordService.On("StoreOrderRecord", mock.Anything, mock.AnythingOfType("*ondc.OrderRecord")).Return(nil)

	auditService.On("LogRequestResponse", mock.Anything, mock.Anything).Return(nil)

	// Mock event publishing failure
	eventPublisher.On("PublishEvent", mock.Anything, "stream.location.search", mock.AnythingOfType("*models.SearchRequestedEvent")).Return(errors.NewDomainError(65020, "internal error", "failed to publish event"))

	// Create request
	requestBody := map[string]interface{}{
		"context": map[string]interface{}{
			"domain":         "nic2004:52110",
			"action":         "search",
			"transaction_id": transactionID,
			"message_id":     messageID,
			"timestamp":      time.Now().Format(time.RFC3339),
			"ttl":            "PT30S",
			"bap_uri":        "https://buyer.example.com",
		},
		"message": map[string]interface{}{
			"intent": map[string]interface{}{
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
	req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("client", &models.Client{ID: "test-client", ClientCode: "test-client"})

	// Execute handler
	handler.HandleSearch(c)

	// Assertions
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	eventPublisher.AssertExpectations(t)
}

func TestSearchHandler_CallbackContextRegeneration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	eventPublisher := new(mockEventPublisher)
	eventConsumer := new(mockEventConsumer)
	callbackService := new(mockCallbackService)
	idempotencyService := new(mockIdempotencyService)

	orderRecordService := new(mockOrderRecordService)
	auditService := new(mockAuditService)
	handler := NewSearchHandler(eventPublisher, eventConsumer, callbackService, idempotencyService, orderRecordService, auditService, "P1", "bpp.example.com", "https://bpp.example.com", "Test BPP", "https://bpp.example.com/terms", logger)

	transactionID := uuid.New().String()
	originalMessageID := uuid.New().String()
	originalTimestamp := time.Now().Add(-5 * time.Minute)

	// Mock successful event publishing
	eventPublisher.On("PublishEvent", mock.Anything, "stream.location.search", mock.AnythingOfType("*models.SearchRequestedEvent")).Return(nil)

	// Mock successful event consumption
	quoteComputedEvent := &models.QuoteComputedEvent{
		BaseEvent: models.BaseEvent{
			EventType:   "QUOTE_COMPUTED",
			EventID:     uuid.New().String(),
			Traceparent: "00-12345678901234567890123456789012-1234567890123456-01",
			Timestamp:   time.Now(),
		},
		SearchID:                    uuid.New().String(),
		Serviceable:                 true,
		Price:                       models.Price{Value: 60.0, Currency: "INR"},
		TTL:                         "PT10M",
		DistanceOriginToDestination: 5.5,
		ETAOrigin:                   timePtr(time.Now().Add(10 * time.Minute)),
		ETADestination:              timePtr(time.Now().Add(30 * time.Minute)),
	}
	eventConsumer.On("ConsumeEvent", mock.Anything, "quote:computed", "uois-gateway-consumers", mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(quoteComputedEvent, nil)

	// Mock successful order record storage
	orderRecordService.On("StoreOrderRecord", mock.Anything, mock.AnythingOfType("*ondc.OrderRecord")).Return(nil)

	auditService.On("LogRequestResponse", mock.Anything, mock.Anything).Return(nil)
	auditService.On("LogCallbackDelivery", mock.Anything, mock.Anything).Return(nil).Maybe()

	// Capture callback payload to verify context regeneration
	var capturedCallbackPayload models.ONDCResponse
	callbackService.On("SendCallback", mock.Anything, mock.AnythingOfType("string"), mock.MatchedBy(func(payload interface{}) bool {
		if resp, ok := payload.(models.ONDCResponse); ok {
			capturedCallbackPayload = resp
			return true
		}
		return false
	})).Return(nil).Maybe()

	idempotencyService.On("CheckIdempotency", mock.Anything, mock.AnythingOfType("string")).Return(nil, false, nil)
	idempotencyService.On("StoreIdempotency", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

	// Create request with original message_id and timestamp
	requestBody := map[string]interface{}{
		"context": map[string]interface{}{
			"domain":         "nic2004:52110",
			"action":         "search",
			"transaction_id": transactionID,
			"message_id":     originalMessageID,
			"timestamp":      originalTimestamp.Format(time.RFC3339),
			"ttl":            "PT30S",
			"bap_uri":        "https://buyer.example.com",
		},
		"message": map[string]interface{}{
			"intent": map[string]interface{}{
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
	req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("client", &models.Client{ID: "test-client", ClientCode: "test-client"})

	handler.HandleSearch(c)

	// Wait for async callback
	time.Sleep(100 * time.Millisecond)

	// Verify callback context regeneration
	assert.NotEqual(t, originalMessageID, capturedCallbackPayload.Context.MessageID, "callback must have new message_id")
	assert.Equal(t, transactionID, capturedCallbackPayload.Context.TransactionID, "callback must preserve transaction_id")
	assert.True(t, capturedCallbackPayload.Context.Timestamp.After(originalTimestamp), "callback must have new timestamp")

	// Verify callback structure: search_id must be in providers[].id, not items[].id
	if catalog, ok := capturedCallbackPayload.Message["catalog"].(map[string]interface{}); ok {
		if providers, ok := catalog["providers"].([]interface{}); ok && len(providers) > 0 {
			if provider, ok := providers[0].(map[string]interface{}); ok {
				providerID, exists := provider["id"]
				assert.True(t, exists, "provider.id must exist")
				assert.NotEmpty(t, providerID, "provider.id must not be empty")
				// Verify items don't have id field
				if items, ok := provider["items"].([]interface{}); ok && len(items) > 0 {
					if item, ok := items[0].(map[string]interface{}); ok {
						_, hasID := item["id"]
						assert.False(t, hasID, "items[].id must not exist - search_id should be in provider.id")
					}
				}
			}
		}
	}
}

func TestSearchHandler_TTLParsing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	orderRecordService := new(mockOrderRecordService)
	auditService := new(mockAuditService)
	handler := NewSearchHandler(nil, nil, nil, nil, orderRecordService, auditService, "P1", "bpp.example.com", "https://bpp.example.com", "Test BPP", "https://bpp.example.com/terms", logger)

	tests := []struct {
		name        string
		ttl         string
		expected    time.Duration
		expectError bool
	}{
		{"PT30S", "PT30S", 30 * time.Second, false},
		{"PT15M", "PT15M", 15 * time.Minute, false},
		{"PT1H30M", "PT1H30M", 90 * time.Minute, false},
		{"PT2H15M30S", "PT2H15M30S", 2*time.Hour + 15*time.Minute + 30*time.Second, false},
		{"Empty", "", 0, true},
		{"Invalid", "INVALID", 0, true},
		{"NoPrefix", "30S", 0, true},
		{"NoComponents", "PT", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handler.parseTTL(tt.ttl)
			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, time.Duration(0), result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSearchHandler_StoreOrderRecordFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	eventPublisher := new(mockEventPublisher)
	eventConsumer := new(mockEventConsumer)
	callbackService := new(mockCallbackService)
	idempotencyService := new(mockIdempotencyService)
	orderRecordService := new(mockOrderRecordService)

	auditService := new(mockAuditService)
	handler := NewSearchHandler(eventPublisher, eventConsumer, callbackService, idempotencyService, orderRecordService, auditService, "P1", "bpp.example.com", "https://bpp.example.com", "Test BPP", "https://bpp.example.com/terms", logger)

	transactionID := uuid.New().String()
	messageID := uuid.New().String()

	// Mock idempotency check (no existing request)
	idempotencyService.On("CheckIdempotency", mock.Anything, mock.AnythingOfType("string")).Return(nil, false, nil)

	// Mock order record storage failure
	orderRecordService.On("StoreOrderRecord", mock.Anything, mock.AnythingOfType("*ondc.OrderRecord")).Return(errors.NewDomainError(65020, "internal error", "storage failure"))

	auditService.On("LogRequestResponse", mock.Anything, mock.Anything).Return(nil)

	// Create request
	requestBody := map[string]interface{}{
		"context": map[string]interface{}{
			"domain":         "nic2004:52110",
			"action":         "search",
			"transaction_id": transactionID,
			"message_id":     messageID,
			"timestamp":      time.Now().Format(time.RFC3339),
			"ttl":            "PT30S",
			"bap_uri":        "https://buyer.example.com",
		},
		"message": map[string]interface{}{
			"intent": map[string]interface{}{
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
	req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("client", &models.Client{ID: "test-client", ClientCode: "test-client"})

	// Execute handler
	handler.HandleSearch(c)

	// Assertions
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response models.ONDCResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response.Error)

	orderRecordService.AssertExpectations(t)
}

func TestSearchHandler_InvalidTTL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	eventPublisher := new(mockEventPublisher)
	eventConsumer := new(mockEventConsumer)
	callbackService := new(mockCallbackService)
	idempotencyService := new(mockIdempotencyService)
	orderRecordService := new(mockOrderRecordService)

	auditService := new(mockAuditService)
	handler := NewSearchHandler(eventPublisher, eventConsumer, callbackService, idempotencyService, orderRecordService, auditService, "P1", "bpp.example.com", "https://bpp.example.com", "Test BPP", "https://bpp.example.com/terms", logger)

	transactionID := uuid.New().String()
	messageID := uuid.New().String()

	// Mock audit service for error response logging
	auditService.On("LogRequestResponse", mock.Anything, mock.Anything).Return(nil)

	// Mock idempotency check (no existing request)
	idempotencyService.On("CheckIdempotency", mock.Anything, mock.AnythingOfType("string")).Return(nil, false, nil)

	// Mock successful order record storage (will be called before TTL validation)
	orderRecordService.On("StoreOrderRecord", mock.Anything, mock.AnythingOfType("*ondc.OrderRecord")).Return(nil).Maybe()

	// Mock successful event publishing (will be called before TTL validation)
	eventPublisher.On("PublishEvent", mock.Anything, "stream.location.search", mock.AnythingOfType("*models.SearchRequestedEvent")).Return(nil).Maybe()

	// Create request with invalid TTL
	requestBody := map[string]interface{}{
		"context": map[string]interface{}{
			"domain":         "nic2004:52110",
			"action":         "search",
			"transaction_id": transactionID,
			"message_id":     messageID,
			"timestamp":      time.Now().Format(time.RFC3339),
			"ttl":            "INVALID",
			"bap_uri":        "https://buyer.example.com",
		},
		"message": map[string]interface{}{
			"intent": map[string]interface{}{
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
	req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("client", &models.Client{ID: "test-client", ClientCode: "test-client"})

	// Execute handler
	handler.HandleSearch(c)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ONDCResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response.Error)
}

func TestSearchHandler_EmptyTTL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	eventPublisher := new(mockEventPublisher)
	eventConsumer := new(mockEventConsumer)
	callbackService := new(mockCallbackService)
	idempotencyService := new(mockIdempotencyService)
	orderRecordService := new(mockOrderRecordService)

	auditService := new(mockAuditService)
	handler := NewSearchHandler(eventPublisher, eventConsumer, callbackService, idempotencyService, orderRecordService, auditService, "P1", "bpp.example.com", "https://bpp.example.com", "Test BPP", "https://bpp.example.com/terms", logger)

	transactionID := uuid.New().String()
	messageID := uuid.New().String()

	// Mock idempotency check (no existing request)
	idempotencyService.On("CheckIdempotency", mock.Anything, mock.AnythingOfType("string")).Return(nil, false, nil)

	// Mock successful order record storage (will be called before TTL validation)
	orderRecordService.On("StoreOrderRecord", mock.Anything, mock.AnythingOfType("*ondc.OrderRecord")).Return(nil).Maybe()

	// Mock successful event publishing (will be called before TTL validation)
	eventPublisher.On("PublishEvent", mock.Anything, "stream.location.search", mock.AnythingOfType("*models.SearchRequestedEvent")).Return(nil).Maybe()

	// Mock idempotency storage (handler stores idempotency before callback, even if callback fails)
	idempotencyService.On("StoreIdempotency", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil).Maybe()

	// Mock audit service (optional, handler logs request/response)
	auditService.On("LogRequestResponse", mock.Anything, mock.Anything).Return(nil).Maybe()

	// Create request with empty TTL
	requestBody := map[string]interface{}{
		"context": map[string]interface{}{
			"domain":         "nic2004:52110",
			"action":         "search",
			"transaction_id": transactionID,
			"message_id":     messageID,
			"timestamp":      time.Now().Format(time.RFC3339),
			"ttl":            "",
			"bap_uri":        "https://buyer.example.com",
		},
		"message": map[string]interface{}{
			"intent": map[string]interface{}{
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
	req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("client", &models.Client{ID: "test-client", ClientCode: "test-client"})

	// Execute handler
	handler.HandleSearch(c)

	// Assertions
	// Note: Empty TTL is valid in ONDC (optional field)
	// TTL validation only happens in callback handler, not main handler
	// Main handler returns ACK, callback will fail silently if TTL is invalid
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.ONDCACKResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ACK", response.Message.Ack.Status)
}

func timePtr(t time.Time) *time.Time {
	return &t
}
