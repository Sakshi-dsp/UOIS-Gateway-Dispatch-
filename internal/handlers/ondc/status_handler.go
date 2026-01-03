package ondc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"uois-gateway/internal/models"
	"uois-gateway/internal/services/audit"
	"uois-gateway/internal/utils"
	"uois-gateway/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// StatusHandler handles /status ONDC requests
type StatusHandler struct {
	callbackService      CallbackService
	idempotencyService    IdempotencyService
	orderServiceClient    OrderServiceClient
	orderRecordService    OrderRecordService
	billingStorageService BillingStorageService
	auditService          AuditService
	cacheService          CacheService
	bppID                 string // BPP ID (ONDC-registered Seller NP identity)
	bppURI                string // BPP URI
	logger                *zap.Logger
}

// NewStatusHandler creates a new status handler
func NewStatusHandler(
	callbackService CallbackService,
	idempotencyService IdempotencyService,
	orderServiceClient OrderServiceClient,
	orderRecordService OrderRecordService,
	billingStorageService BillingStorageService,
	auditService AuditService,
	cacheService CacheService,
	bppID string,
	bppURI string,
	logger *zap.Logger,
) *StatusHandler {
	return &StatusHandler{
		callbackService:       callbackService,
		idempotencyService:     idempotencyService,
		orderServiceClient:     orderServiceClient,
		orderRecordService:     orderRecordService,
		billingStorageService:  billingStorageService,
		auditService:           auditService,
		cacheService:           cacheService,
		bppID:                  bppID,
		bppURI:                 bppURI,
		logger:                 logger,
	}
}

// HandleStatus handles POST /status requests
func (h *StatusHandler) HandleStatus(c *gin.Context) {
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

	// Check idempotency
	idempotencyKey := h.buildIdempotencyKey(req.Context.TransactionID, req.Context.MessageID)
	if existingResponseBytes, exists, err := h.idempotencyService.CheckIdempotency(ctx, idempotencyKey); err == nil && exists {
		var existingResponse models.ONDCACKResponse
		if err := json.Unmarshal(existingResponseBytes, &existingResponse); err == nil {
			h.respondACK(c, existingResponse)
			return
		}
	}

	// Extract order.id (ONDC) from request (echoed from /on_confirm - seller-generated)
	orderID, err := h.extractOrderID(&req)
	if err != nil {
		h.respondNACK(c, errors.NewDomainError(65001, "invalid request", err.Error()))
		return
	}

	// Get client ID from context
	client, _ := c.Get("client")
	var clientID string
	if cl, ok := client.(*models.Client); ok {
		clientID = cl.ID
	}

	// Look up order record using (client_id + order.id (ONDC))
	// Per ID Domain Isolation Law: Extract order.id (ONDC), lookup order record, retrieve dispatch_order_id
	orderRecord, err := h.orderRecordService.GetOrderRecordByOrderID(ctx, clientID, orderID)
	if err != nil || orderRecord == nil {
		h.logger.Error("order record not found", zap.Error(err), zap.String("trace_id", traceID), zap.String("client_id", clientID), zap.String("order.id", orderID))
		h.respondNACK(c, errors.NewDomainError(65006, "order not found", "order_id not found"))
		return
	}

	// Retrieve dispatch_order_id from order record (internal-only, for Order Service calls)
	dispatchOrderID := orderRecord.DispatchOrderID
	if dispatchOrderID == "" {
		h.logger.Error("dispatch_order_id not found in order record", zap.String("trace_id", traceID), zap.String("order.id", orderID))
		h.respondNACK(c, errors.NewDomainError(65006, "order not found", "order not confirmed"))
		return
	}

	// Check cache first
	cacheKey := fmt.Sprintf("status:%s:%s", clientID, orderID)
	var orderStatus *OrderStatus
	cached := false

	if h.cacheService != nil {
		var cachedStatus OrderStatus
		if found, err := h.cacheService.Get(ctx, cacheKey, &cachedStatus); err == nil && found {
			orderStatus = &cachedStatus
			cached = true
			h.logger.Debug("status retrieved from cache", zap.String("trace_id", traceID), zap.String("order.id", orderID))
		}
	}

	// If not cached, call Order Service
	if !cached {
		var err error
		orderStatus, err = h.orderServiceClient.GetOrder(ctx, dispatchOrderID)
		if err != nil {
			h.logger.Error("failed to get order status", zap.Error(err), zap.String("trace_id", traceID), zap.String("dispatch_order_id", dispatchOrderID))
			domainErr, ok := err.(*errors.DomainError)
			if ok {
				h.respondNACK(c, domainErr)
			} else {
				h.respondNACK(c, errors.NewDomainError(65020, "internal error", "failed to get order status"))
			}
			return
		}

		// Cache the result
		if h.cacheService != nil {
			_ = h.cacheService.Set(ctx, cacheKey, orderStatus)
		}
	}

	// Compose response
	response := h.composeStatusResponse(&req, orderStatus)

	// Store idempotency (marshal to preserve byte-exactness for ONDC signatures)
	responseBytes, _ := json.Marshal(response)
	_ = h.idempotencyService.StoreIdempotency(ctx, idempotencyKey, responseBytes, 24*time.Hour)

	// Log request/response audit
	h.logRequestResponse(ctx, &req, response, nil, orderRecord, clientID, traceID)

	// Send callback asynchronously (pass orderRecord for stable fulfillment.id)
	go h.sendStatusCallback(ctx, &req, orderStatus, orderRecord, traceID)

	// Return ACK
	h.respondACK(c, response)
}

func (h *StatusHandler) extractOrderID(req *models.ONDCRequest) (string, error) {
	order, ok := req.Message["order"].(map[string]interface{})
	if !ok {
		return "", errors.NewDomainError(65001, "invalid request", "missing order")
	}

	orderID, ok := order["id"].(string)
	if !ok || orderID == "" {
		return "", errors.NewDomainError(65001, "invalid request", "missing order.id")
	}
	return orderID, nil
}

func (h *StatusHandler) composeStatusResponse(req *models.ONDCRequest, orderStatus *OrderStatus) models.ONDCACKResponse {
	return models.ONDCACKResponse{
		Message: models.ONDCACKMessage{
			Ack: models.ONDCACKStatus{
				Status: "ACK",
			},
		},
	}
}

func (h *StatusHandler) sendStatusCallback(ctx context.Context, req *models.ONDCRequest, orderStatus *OrderStatus, orderRecord *OrderRecord, traceID string) {
	callbackURL := req.Context.BapURI + "/on_status"
	callbackPayload := h.buildOnStatusCallback(ctx, req, orderStatus, orderRecord)

	if err := h.callbackService.SendCallback(ctx, callbackURL, callbackPayload); err != nil {
		h.logger.Error("failed to send /on_status callback", zap.Error(err), zap.String("trace_id", traceID), zap.String("callback_url", callbackURL))
		h.logCallbackDelivery(ctx, req.Context.TransactionID, callbackURL, 1, "failed", err.Error())
	} else {
		h.logCallbackDelivery(ctx, req.Context.TransactionID, callbackURL, 1, "success", "")
	}
}

func (h *StatusHandler) buildOnStatusCallback(ctx context.Context, req *models.ONDCRequest, orderStatus *OrderStatus, orderRecord *OrderRecord) models.ONDCResponse {
	// Regenerate callback context (ONDC protocol requirement)
	callbackCtx := req.Context
	callbackCtx.MessageID = uuid.New().String()
	callbackCtx.Timestamp = time.Now().UTC()
	callbackCtx.BppID = h.bppID
	callbackCtx.BppURI = h.bppURI

	// Retrieve order.id from orderRecord (stable identifier, seller-generated)
	orderID := orderRecord.OrderID
	if orderID == "" {
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

	// Use stable fulfillment ID from orderRecord (set in /init, reused across all callbacks)
	fulfillmentID := orderRecord.FulfillmentID
	if fulfillmentID == "" {
		// Fallback: should not happen in normal flow, but provide default
		fulfillmentID = "F1"
	}

	// Map Order Service state to ONDC order state
	// ONDC order states: INITIATED, IN_PROGRESS, COMPLETED, CANCELLED
	ondcOrderState := "IN_PROGRESS"
	switch orderStatus.State {
	case "CONFIRMED", "RIDER_ASSIGNED", "PICKED_UP", "IN_TRANSIT":
		ondcOrderState = "IN_PROGRESS"
	case "DELIVERED", "COMPLETED":
		ondcOrderState = "COMPLETED"
	case "CANCELLED":
		ondcOrderState = "CANCELLED"
	default:
		ondcOrderState = "IN_PROGRESS"
	}

	// Map Order Service fulfillment state to ONDC fulfillment state code
	fulfillmentStateCode := orderStatus.Fulfillment.State
	if fulfillmentStateCode == "" {
		fulfillmentStateCode = "IN_TRANSIT"
	}

	// Build ONDC-compliant structure: order.fulfillments[] array
	fulfillment := map[string]interface{}{
		"id": fulfillmentID, // Stable fulfillment ID (reused from /init)
		"state": map[string]interface{}{
			"descriptor": map[string]interface{}{
				"code": fulfillmentStateCode,
			},
		},
	}

	// Add rider info if available
	if orderStatus.RiderID != "" {
		fulfillment["agent"] = map[string]interface{}{
			"id": orderStatus.RiderID,
		}
	}

	// Note: POP/POD removed - ONDC uses documents/tags/images, not location.descriptor.name
	// For certification, omit POP/POD or implement via domain-specific extensions

	// Retrieve billing: first from request, then from Redis (stored during /init)
	billing := h.getBilling(ctx, req)

	orderMap := map[string]interface{}{
		"id":           orderID,
		"state":        ondcOrderState,
		"fulfillments": []map[string]interface{}{fulfillment},
	}

	// Add billing if available (ONDC requirement)
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

// getBilling retrieves billing information: first from request, then from Redis
func (h *StatusHandler) getBilling(ctx context.Context, req *models.ONDCRequest) map[string]interface{} {
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

func (h *StatusHandler) buildIdempotencyKey(transactionID, messageID string) string {
	return "status:" + transactionID + ":" + messageID
}

func (h *StatusHandler) respondACK(c *gin.Context, response interface{}) {
	c.JSON(http.StatusOK, response)
}

func (h *StatusHandler) respondNACK(c *gin.Context, err *errors.DomainError) {
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
		h.logRequestResponse(ctx, req, response, nil, nil, clientID, traceID)
	}

	c.JSON(errors.GetHTTPStatus(err), response)
}

func (h *StatusHandler) logRequestResponse(ctx context.Context, req *models.ONDCRequest, ackResponse interface{}, callbackPayload interface{}, orderRecord *OrderRecord, clientID, traceID string) {
	if h.auditService == nil {
		return
	}

	reqPayload := h.toMap(req)
	ackPayload := h.toMap(ackResponse)
	callbackPayloadMap := h.toMap(callbackPayload)

	var orderID, dispatchOrderID string
	if orderRecord != nil {
		orderID = orderRecord.OrderID
		dispatchOrderID = orderRecord.DispatchOrderID
	}

	params := &audit.RequestResponseLogParams{
		TransactionID:   req.Context.TransactionID,
		MessageID:       req.Context.MessageID,
		Action:          "status",
		RequestPayload:  reqPayload,
		ACKPayload:      ackPayload,
		CallbackPayload: callbackPayloadMap,
		TraceID:         traceID,
		ClientID:        clientID,
		OrderID:         orderID,
		DispatchOrderID: dispatchOrderID,
	}

	_ = h.auditService.LogRequestResponse(ctx, params)
}

func (h *StatusHandler) logCallbackDelivery(ctx context.Context, transactionID, callbackURL string, attemptNo int, status, errorMsg string) {
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

func (h *StatusHandler) toMap(v interface{}) map[string]interface{} {
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
