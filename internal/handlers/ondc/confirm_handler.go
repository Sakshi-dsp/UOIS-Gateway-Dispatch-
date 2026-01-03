package ondc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"uois-gateway/internal/models"
	"uois-gateway/internal/services/audit"
	"uois-gateway/internal/utils"
	"uois-gateway/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ConfirmHandler handles /confirm ONDC requests
type ConfirmHandler struct {
	eventPublisher        EventPublisher
	eventConsumer         EventConsumer
	callbackService       CallbackService
	idempotencyService    IdempotencyService
	orderServiceClient    OrderServiceClient
	orderRecordService    OrderRecordService
	billingStorageService BillingStorageService
	auditService          AuditService
	bppID                 string // BPP ID (ONDC-registered Seller NP identity)
	bppURI                string // BPP URI
	logger                *zap.Logger
}

// NewConfirmHandler creates a new confirm handler
func NewConfirmHandler(
	eventPublisher EventPublisher,
	eventConsumer EventConsumer,
	callbackService CallbackService,
	idempotencyService IdempotencyService,
	orderServiceClient OrderServiceClient,
	orderRecordService OrderRecordService,
	billingStorageService BillingStorageService,
	auditService AuditService,
	bppID string,
	bppURI string,
	logger *zap.Logger,
) *ConfirmHandler {
	return &ConfirmHandler{
		eventPublisher:        eventPublisher,
		eventConsumer:         eventConsumer,
		callbackService:       callbackService,
		idempotencyService:    idempotencyService,
		orderServiceClient:    orderServiceClient,
		orderRecordService:    orderRecordService,
		billingStorageService: billingStorageService,
		auditService:          auditService,
		bppID:                 bppID,
		bppURI:                bppURI,
		logger:                logger,
	}
}

// HandleConfirm handles POST /confirm requests
func (h *ConfirmHandler) HandleConfirm(c *gin.Context) {
	ctx := c.Request.Context()

	// Extract and ensure traceparent for distributed tracing
	traceparent := utils.EnsureTraceparent(c.GetHeader("traceparent"))
	traceID := utils.ExtractTraceID(traceparent)

	// Parse request
	var req models.ONDCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request", zap.Error(err), zap.String("trace_id", traceID))
		h.respondNACK(c, errors.NewDomainError(65001, "invalid request", err.Error()))
		return
	}

	// Store request in context for error responses
	c.Set("ondc_request", &req)

	// Validate context
	if err := req.Context.Validate(); err != nil {
		h.logger.Error("invalid context", zap.Error(err), zap.String("trace_id", traceID))
		h.respondNACK(c, errors.NewDomainError(65001, "invalid context", err.Error()))
		return
	}

	// Validate payment type (Dispatch does not support COD)
	order, _ := req.Message["order"].(map[string]interface{})
	paymentInfo, _ := order["payment"].(map[string]interface{})
	if err := utils.ValidatePaymentType(paymentInfo); err != nil {
		h.logger.Warn("payment type validation failed", zap.Error(err), zap.String("trace_id", traceID))
		h.respondNACK(c, err)
		return
	}

	// Check idempotency
	idempotencyKey := h.buildIdempotencyKey(req.Context.TransactionID, req.Context.MessageID)
	if existingResponseBytes, exists, err := h.idempotencyService.CheckIdempotency(ctx, idempotencyKey); err == nil && exists {
		var existingResponse models.ONDCACKResponse
		if err := json.Unmarshal(existingResponseBytes, &existingResponse); err == nil {
			h.respondACK(c, existingResponse)
			return
		}
	}

	// Extract quote_id and order.id from request
	// ONDC spec: Buyer DOES send order.id in /confirm, Seller must echo the same order.id in /on_confirm
	quoteID, orderID, paymentInfo, err := h.extractConfirmData(&req)
	if err != nil {
		h.respondNACK(c, errors.NewDomainError(65001, "invalid request", err.Error()))
		return
	}

	// Validate quote_id was previously received from Order Service
	orderRecord, err := h.orderRecordService.GetOrderRecordByQuoteID(ctx, quoteID)
	if err != nil || orderRecord == nil {
		h.logger.Error("quote_id not found or was not issued by Order Service", zap.Error(err), zap.String("trace_id", traceID), zap.String("quote_id", quoteID))
		h.respondNACK(c, errors.NewDomainError(65005, "quote invalid", "quote_id not found or invalid"))
		return
	}

	// Validate quote_id TTL via Order Service
	valid, err := h.orderServiceClient.ValidateQuoteIDTTL(ctx, quoteID)
	if err != nil {
		h.logger.Error("failed to validate quote_id TTL", zap.Error(err), zap.String("trace_id", traceID), zap.String("quote_id", quoteID))
		domainErr, ok := err.(*errors.DomainError)
		if ok {
			h.respondNACK(c, domainErr)
		} else {
			h.respondNACK(c, errors.NewDomainError(65020, "internal error", "failed to validate quote_id"))
		}
		return
	}
	if !valid {
		h.respondNACK(c, errors.NewDomainError(65004, "quote expired", "quote_id TTL expired"))
		return
	}

	// Get client ID from context
	client, _ := c.Get("client")
	var clientID string
	if cl, ok := client.(*models.Client); ok {
		clientID = cl.ID
	}

	// Publish CONFIRM_REQUESTED event
	confirmEvent := h.buildConfirmRequestedEvent(quoteID, clientID, paymentInfo, traceparent)
	if err := h.eventPublisher.PublishEvent(ctx, "stream.uois.confirm_requested", confirmEvent); err != nil {
		h.logger.Error("failed to publish CONFIRM_REQUESTED event", zap.Error(err), zap.String("trace_id", traceID), zap.String("quote_id", quoteID))
		h.respondNACK(c, errors.NewDomainError(65020, "internal error", "failed to publish event"))
		return
	}

	// Consume ORDER_CONFIRMED or ORDER_CONFIRM_FAILED event
	// NOTE: Confirm events MUST be keyed by quote_id.
	// Changing this requires protocol-level coordination with Order Service.
	// If Order Service switches to order_id later, this will break silently.
	ttlDuration, domainErr := h.parseTTL(req.Context.TTL)
	if domainErr != nil {
		h.respondNACK(c, domainErr)
		return
	}
	var orderEvent interface{}

	// Try ORDER_CONFIRMED first
	orderEvent, err = h.eventConsumer.ConsumeEvent(ctx, "stream.uois.order_confirmed", "uois-gateway-consumers", quoteID, ttlDuration)
	if err != nil {
		// Fallback to ORDER_CONFIRM_FAILED stream
		orderEvent, err = h.eventConsumer.ConsumeEvent(ctx, "stream.uois.order_confirm_failed", "uois-gateway-consumers", quoteID, ttlDuration)
		if err != nil {
			h.logger.Error("failed to consume order event from both streams", zap.Error(err), zap.String("trace_id", traceID), zap.String("quote_id", quoteID))
			h.respondNACK(c, errors.NewDomainError(65020, "internal error", "failed to consume event"))
			return
		}
	}

	// Update order record with dispatch_order_id and order.id if ORDER_CONFIRMED
	// ONDC spec: Use order.id from buyer request, validate it matches stored transaction
	if orderConfirmed, ok := orderEvent.(*models.OrderConfirmedEvent); ok {
		// Validate order.id matches transaction (if order.id was provided in request)
		if orderID != "" {
			// Verify order.id belongs to this transaction
			if orderRecord.TransactionID != req.Context.TransactionID {
				h.logger.Error("order.id transaction mismatch", zap.String("trace_id", traceID), zap.String("order.id", orderID), zap.String("expected_transaction_id", orderRecord.TransactionID), zap.String("received_transaction_id", req.Context.TransactionID))
				h.respondNACK(c, errors.NewDomainError(65001, "invalid request", "order.id does not match transaction"))
				return
			}
		} else {
			// If buyer didn't provide order.id, generate one (fallback for backward compatibility)
			orderID = uuid.New().String()
			h.logger.Warn("buyer did not provide order.id in /confirm, generating new one", zap.String("trace_id", traceID), zap.String("quote_id", quoteID))
		}

		orderRecord.DispatchOrderID = orderConfirmed.DispatchOrderID
		orderRecord.OrderID = orderID // Use buyer-provided order.id (or generated fallback)
		orderRecord.ClientID = clientID
		if err := h.orderRecordService.UpdateOrderRecord(ctx, orderRecord); err != nil {
			h.logger.Warn("failed to update order record with dispatch_order_id and order.id", zap.Error(err), zap.String("trace_id", traceID), zap.String("quote_id", quoteID), zap.String("dispatch_order_id", orderConfirmed.DispatchOrderID), zap.String("order.id", orderID))
		}
	}

	// Compose response
	response := h.composeConfirmResponse(&req, orderEvent)

	// Store idempotency (marshal to preserve byte-exactness for ONDC signatures)
	responseBytes, _ := json.Marshal(response)
	_ = h.idempotencyService.StoreIdempotency(ctx, idempotencyKey, responseBytes, 24*time.Hour)

	// Extract IDs for audit logging
	auditQuoteID := orderRecord.QuoteID
	auditOrderID := orderRecord.OrderID
	auditDispatchOrderID := orderRecord.DispatchOrderID

	// Log request/response audit
	h.logRequestResponse(ctx, &req, response, nil, "", auditQuoteID, auditOrderID, auditDispatchOrderID, clientID, traceID)

	// Send callback asynchronously (pass orderRecord for order.id retrieval)
	go h.sendConfirmCallback(ctx, &req, orderEvent, orderRecord, traceID)

	// Return ACK
	h.respondACK(c, response)
}

func (h *ConfirmHandler) extractConfirmData(req *models.ONDCRequest) (string, string, map[string]interface{}, error) {
	order, ok := req.Message["order"].(map[string]interface{})
	if !ok {
		return "", "", nil, errors.NewDomainError(65001, "invalid request", "missing order")
	}

	// Extract quote_id (echoed from /on_init)
	quote, ok := order["quote"].(map[string]interface{})
	if !ok {
		return "", "", nil, errors.NewDomainError(65001, "invalid request", "missing quote")
	}

	quoteID, ok := quote["id"].(string)
	if !ok || quoteID == "" {
		return "", "", nil, errors.NewDomainError(65001, "invalid request", "missing quote.id (quote_id)")
	}

	// Extract order.id (ONDC spec: Buyer DOES send order.id in /confirm)
	// Seller must echo the same order.id in /on_confirm, do NOT regenerate
	var orderID string
	if id, ok := order["id"].(string); ok && id != "" {
		orderID = id
	}

	// Extract payment info (optional)
	paymentInfo, _ := order["payment"].(map[string]interface{})

	return quoteID, orderID, paymentInfo, nil
}

func (h *ConfirmHandler) buildConfirmRequestedEvent(quoteID, clientID string, paymentInfo map[string]interface{}, traceparent string) *models.ConfirmRequestedEvent {
	traceparent = utils.EnsureTraceparent(traceparent)

	return &models.ConfirmRequestedEvent{
		BaseEvent: models.BaseEvent{
			EventType:   "CONFIRM_REQUESTED",
			EventID:     uuid.New().String(),
			Traceparent: traceparent,
			Timestamp:   time.Now(),
		},
		QuoteID:     quoteID,
		ClientID:    clientID,
		PaymentInfo: paymentInfo,
	}
}

func (h *ConfirmHandler) parseTTL(ttl string) (time.Duration, *errors.DomainError) {
	if ttl == "" {
		// Default fallback for missing TTL (explicit, not silent)
		return 30 * time.Second, nil
	}

	if !strings.HasPrefix(ttl, "PT") {
		return 0, errors.NewDomainError(65001, "invalid context", "invalid ttl format (expected ISO8601 duration)")
	}

	ttlStr := strings.TrimPrefix(ttl, "PT")
	var hours, minutes, seconds int

	if idx := strings.Index(ttlStr, "H"); idx != -1 {
		if _, err := fmt.Sscanf(ttlStr[:idx+1], "%dH", &hours); err != nil {
			return 0, errors.NewDomainError(65001, "invalid context", fmt.Sprintf("invalid hours format: %v", err))
		}
		ttlStr = ttlStr[idx+1:]
	}

	if idx := strings.Index(ttlStr, "M"); idx != -1 {
		if _, err := fmt.Sscanf(ttlStr[:idx+1], "%dM", &minutes); err != nil {
			return 0, errors.NewDomainError(65001, "invalid context", fmt.Sprintf("invalid minutes format: %v", err))
		}
		ttlStr = ttlStr[idx+1:]
	}

	if idx := strings.Index(ttlStr, "S"); idx != -1 {
		if _, err := fmt.Sscanf(ttlStr[:idx+1], "%dS", &seconds); err != nil {
			return 0, errors.NewDomainError(65001, "invalid context", fmt.Sprintf("invalid seconds format: %v", err))
		}
	}

	// Convert to time.Duration
	// PT0S is valid ISO8601 (immediate timeout) - allow zero duration
	duration := time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second

	return duration, nil
}

func (h *ConfirmHandler) composeConfirmResponse(req *models.ONDCRequest, orderEvent interface{}) models.ONDCACKResponse {
	return models.ONDCACKResponse{
		Message: models.ONDCACKMessage{
			Ack: models.ONDCACKStatus{
				Status: "ACK",
			},
		},
	}
}

func (h *ConfirmHandler) sendConfirmCallback(ctx context.Context, req *models.ONDCRequest, orderEvent interface{}, orderRecord *OrderRecord, traceID string) {
	callbackURL := req.Context.BapURI + "/on_confirm"
	callbackPayload := h.buildOnConfirmCallback(ctx, req, orderEvent, orderRecord)

	if err := h.callbackService.SendCallback(ctx, callbackURL, callbackPayload); err != nil {
		h.logger.Error("failed to send /on_confirm callback", zap.Error(err), zap.String("trace_id", traceID), zap.String("callback_url", callbackURL))
		h.logCallbackDelivery(ctx, req.Context.TransactionID, callbackURL, 1, "failed", err.Error())
	} else {
		h.logCallbackDelivery(ctx, req.Context.TransactionID, callbackURL, 1, "success", "")
	}
}

func (h *ConfirmHandler) buildOnConfirmCallback(ctx context.Context, req *models.ONDCRequest, orderEvent interface{}, orderRecord *OrderRecord) models.ONDCResponse {
	// Regenerate callback context (ONDC protocol requirement)
	callbackCtx := req.Context
	callbackCtx.MessageID = uuid.New().String()
	callbackCtx.Timestamp = time.Now().UTC()
	callbackCtx.BppID = h.bppID
	callbackCtx.BppURI = h.bppURI

	orderConfirmed, ok := orderEvent.(*models.OrderConfirmedEvent)
	if ok {
		// Retrieve seller-generated order.id from order record
		orderID := orderRecord.OrderID
		if orderID == "" {
			// Note: traceID not available in this callback context
			h.logger.Error("order.id not found in order record")
			return models.ONDCResponse{
				Context: callbackCtx,
				Error: &models.ONDCError{
					Type:    "CONTEXT_ERROR",
					Code:    "65020",
					Message: map[string]string{"en": "internal error"},
				},
			}
		}

		// Success case: ORDER_CONFIRMED
		// Send seller-generated order.id (ONDC) in callback, NOT dispatch_order_id (internal-only)
		// Use fulfillment ID from orderRecord if available (stable per order, set in /init)
		fulfillmentID := orderRecord.FulfillmentID
		if fulfillmentID == "" {
			// Fallback: generate new fulfillment ID if not stored (should not happen in normal flow)
			fulfillmentID = "F1"
		}

		// Build ONDC-compliant structure: order.fulfillments[] array (not singular fulfillment)
		fulfillment := map[string]interface{}{
			"id": fulfillmentID, // Stable fulfillment ID (reused from /init)
			"state": map[string]interface{}{
				"descriptor": map[string]interface{}{
					"code": "RIDER_ASSIGNED",
				},
			},
		}

		// Add rider info if assigned
		if orderConfirmed.RiderID != "" {
			fulfillment["agent"] = map[string]interface{}{
				"id": orderConfirmed.RiderID,
			}
		}

		// Retrieve billing: first from request, then from Redis (stored during /init)
		billing := h.getBilling(ctx, req)

		orderMap := map[string]interface{}{
			"id":           orderID, // Buyer-provided order.id (echoed back)
			"state":        "CONFIRMED",
			"fulfillments": []map[string]interface{}{fulfillment}, // ONDC requires fulfillments[] array
		}

		// Add billing if available (ONDC requirement: billing should be same as in /init)
		if billing != nil {
			orderMap["billing"] = billing
		}

		message := map[string]interface{}{
			"order": orderMap,
		}

		return models.ONDCResponse{
			Context: callbackCtx,
			Message: message,
		}
	}

	orderConfirmFailed, ok := orderEvent.(*models.OrderConfirmFailedEvent)
	if ok {
		// Error case: ORDER_CONFIRM_FAILED
		return models.ONDCResponse{
			Context: callbackCtx,
			Error: &models.ONDCError{
				Type:    "CONTEXT_ERROR",
				Code:    "65005",
				Message: map[string]string{"en": orderConfirmFailed.Reason},
			},
		}
	}

	// Unknown event type
	return models.ONDCResponse{
		Context: callbackCtx,
		Error: &models.ONDCError{
			Type:    "CONTEXT_ERROR",
			Code:    "65020",
			Message: map[string]string{"en": "internal error"},
		},
	}
}

// getBilling retrieves billing information: first from request, then from Redis
func (h *ConfirmHandler) getBilling(ctx context.Context, req *models.ONDCRequest) map[string]interface{} {
	// First: Check if billing exists in the current request
	order, ok := req.Message["order"].(map[string]interface{})
	if ok {
		if billing, ok := order["billing"].(map[string]interface{}); ok && billing != nil {
			return billing
		}
	}

	// Second: Retrieve from Redis using transaction_id (if stored during /init)
	if h.billingStorageService != nil {
		billing, err := h.billingStorageService.GetBilling(ctx, req.Context.TransactionID)
		if err == nil && billing != nil {
			return billing
		}
		// Non-fatal: log but don't fail if billing retrieval fails
		if err != nil {
			h.logger.Debug("failed to retrieve billing from storage", zap.Error(err), zap.String("transaction_id", req.Context.TransactionID))
		}
	}

	return nil
}

func (h *ConfirmHandler) buildIdempotencyKey(transactionID, messageID string) string {
	return "confirm:" + transactionID + ":" + messageID
}

func (h *ConfirmHandler) respondACK(c *gin.Context, response interface{}) {
	c.JSON(http.StatusOK, response)
}

func (h *ConfirmHandler) respondNACK(c *gin.Context, err *errors.DomainError) {
	ctx := c.Request.Context()
	traceID := utils.ExtractTraceID(utils.EnsureTraceparent(c.GetHeader("traceparent")))

	var ondcCtx models.ONDCContext
	var req *models.ONDCRequest
	if reqVal, ok := c.Get("ondc_request"); ok {
		if ondcReq, ok := reqVal.(*models.ONDCRequest); ok {
			ondcCtx = ondcReq.Context
			req = ondcReq
		}
	}

	response := models.ONDCResponse{
		Context: ondcCtx,
		Error: &models.ONDCError{
			Type:    "CONTEXT_ERROR",
			Code:    fmt.Sprintf("%d", err.Code),
			Message: map[string]string{"en": err.Message},
		},
	}

	// Log request/response audit
	if req != nil {
		client, _ := c.Get("client")
		var clientID string
		if cl, ok := client.(*models.Client); ok {
			clientID = cl.ID
		}
		h.logRequestResponse(ctx, req, response, nil, "", "", "", "", clientID, traceID)
	}

	c.JSON(errors.GetHTTPStatus(err), response)
}

func (h *ConfirmHandler) logRequestResponse(ctx context.Context, req *models.ONDCRequest, ackResponse interface{}, callbackPayload interface{}, searchID, quoteID, orderID, dispatchOrderID, clientID, traceID string) {
	if h.auditService == nil {
		return
	}

	reqPayload := h.toMap(req)
	ackPayload := h.toMap(ackResponse)
	callbackPayloadMap := h.toMap(callbackPayload)

	params := &audit.RequestResponseLogParams{
		TransactionID:   req.Context.TransactionID,
		MessageID:       req.Context.MessageID,
		Action:          "confirm",
		RequestPayload:  reqPayload,
		ACKPayload:      ackPayload,
		CallbackPayload: callbackPayloadMap,
		TraceID:         traceID,
		ClientID:        clientID,
		SearchID:        searchID,
		QuoteID:         quoteID,
		OrderID:         orderID,
		DispatchOrderID: dispatchOrderID,
	}

	_ = h.auditService.LogRequestResponse(ctx, params)
}

func (h *ConfirmHandler) logCallbackDelivery(ctx context.Context, transactionID, callbackURL string, attemptNo int, status, errorMsg string) {
	if h.auditService == nil {
		return
	}

	params := &audit.CallbackDeliveryLogParams{
		RequestID:   transactionID,
		CallbackURL: callbackURL,
		AttemptNo:   attemptNo,
		Status:      status,
		Error:       errorMsg,
	}

	_ = h.auditService.LogCallbackDelivery(ctx, params)
}

func (h *ConfirmHandler) toMap(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}
