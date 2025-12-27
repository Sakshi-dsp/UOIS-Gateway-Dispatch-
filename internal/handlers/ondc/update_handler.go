package ondc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"uois-gateway/internal/models"
	"uois-gateway/internal/utils"
	"uois-gateway/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// UpdateHandler handles /update ONDC requests
// NOTE: For Seller NP (P2P - Logistics), /update is rarely needed for certification.
// Most logistics flows use /confirm, /status, /track, /rto, and seller-initiated /on_update.
// This handler is provided for completeness but may not be required for certification.
type UpdateHandler struct {
	callbackService    CallbackService
	idempotencyService IdempotencyService
	orderServiceClient OrderServiceClient
	orderRecordService OrderRecordService
	bppID              string // BPP ID (ONDC-registered Seller NP identity)
	bppURI             string // BPP URI
	logger             *zap.Logger
}

// NewUpdateHandler creates a new update handler
func NewUpdateHandler(
	callbackService CallbackService,
	idempotencyService IdempotencyService,
	orderServiceClient OrderServiceClient,
	orderRecordService OrderRecordService,
	bppID string,
	bppURI string,
	logger *zap.Logger,
) *UpdateHandler {
	return &UpdateHandler{
		callbackService:    callbackService,
		idempotencyService: idempotencyService,
		orderServiceClient: orderServiceClient,
		orderRecordService: orderRecordService,
		bppID:              bppID,
		bppURI:             bppURI,
		logger:             logger,
	}
}

// HandleUpdate handles POST /update requests
func (h *UpdateHandler) HandleUpdate(c *gin.Context) {
	ctx := c.Request.Context()

	// Extract and ensure traceparent for distributed tracing
	traceparent := utils.EnsureTraceparent(c.GetHeader("traceparent"))
	traceID := utils.ExtractTraceID(traceparent)

	var req models.ONDCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request", zap.Error(err), zap.String("trace_id", traceID))
		h.respondNACK(c, errors.NewDomainError(65001, "invalid request", err.Error()))
		return
	}

	c.Set("ondc_request", &req)

	if err := req.Context.Validate(); err != nil {
		h.logger.Error("invalid context", zap.Error(err), zap.String("trace_id", traceID))
		h.respondNACK(c, errors.NewDomainError(65001, "invalid context", err.Error()))
		return
	}

	idempotencyKey := h.buildIdempotencyKey(req.Context.TransactionID, req.Context.MessageID)
	if existingResponseBytes, exists, err := h.idempotencyService.CheckIdempotency(ctx, idempotencyKey); err == nil && exists {
		var existingResponse models.ONDCACKResponse
		if err := json.Unmarshal(existingResponseBytes, &existingResponse); err == nil {
			h.respondACK(c, existingResponse)
			return
		}
	}

	// Extract order.id (ONDC) from request (echoed from /on_confirm - seller-generated)
	orderID, updates, err := h.extractUpdateData(&req)
	if err != nil {
		h.respondNACK(c, errors.NewDomainError(65001, "invalid request", err.Error()))
		return
	}

	client, _ := c.Get("client")
	var clientID string
	if cl, ok := client.(*models.Client); ok {
		clientID = cl.ID
	}

	// Look up order record using (client_id + order.id (ONDC))
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

	if err := h.orderServiceClient.UpdateOrder(ctx, dispatchOrderID, updates); err != nil {
		h.logger.Error("failed to update order", zap.Error(err), zap.String("trace_id", traceID), zap.String("dispatch_order_id", dispatchOrderID))
		domainErr, ok := err.(*errors.DomainError)
		if ok {
			h.respondNACK(c, domainErr)
		} else {
			h.respondNACK(c, errors.NewDomainError(65020, "internal error", "failed to update order"))
		}
		return
	}

	response := h.composeUpdateResponse(&req)

	responseBytes, _ := json.Marshal(response)
	_ = h.idempotencyService.StoreIdempotency(ctx, idempotencyKey, responseBytes, 24*time.Hour)

	go h.sendUpdateCallback(ctx, &req, updates, orderRecord, traceID)

	h.respondACK(c, response)
}

func (h *UpdateHandler) extractUpdateData(req *models.ONDCRequest) (string, map[string]interface{}, error) {
	order, ok := req.Message["order"].(map[string]interface{})
	if !ok {
		return "", nil, errors.NewDomainError(65001, "invalid request", "missing order")
	}

	orderID, ok := order["id"].(string)
	if !ok || orderID == "" {
		return "", nil, errors.NewDomainError(65001, "invalid request", "missing order.id")
	}

	updates := make(map[string]interface{})
	hasAllowedUpdate := false

	// ONDC /update is strictly for: PCC/DCC, Authorization updates, Reverse QC, Fulfillment updates
	// Validate only allowed update fields

	// Extract fulfillments array (allowed: Fulfillment updates)
	if fulfillments, ok := order["fulfillments"].([]interface{}); ok && len(fulfillments) > 0 {
		updates["fulfillments"] = fulfillments
		hasAllowedUpdate = true
	}

	// Extract payment info (allowed: PCC/DCC, Authorization updates)
	if payment, ok := order["payment"].(map[string]interface{}); ok && len(payment) > 0 {
		// Validate payment update contains allowed fields only
		allowedPaymentFields := map[string]bool{
			"@ondc/org/settlement_details": true, // PCC/DCC
			"@ondc/org/settlement_basis":   true,
			"@ondc/org/settlement_window":  true,
			"@ondc/org/authorization":      true, // Authorization updates
			"status":                       true,
			"type":                         true,
		}

		validatedPayment := make(map[string]interface{})
		for key, value := range payment {
			if allowedPaymentFields[key] {
				validatedPayment[key] = value
				hasAllowedUpdate = true
			} else {
				// Note: traceID not available in this validation context
				h.logger.Warn("rejecting unknown payment field in /update", zap.String("field", key))
			}
		}

		if len(validatedPayment) > 0 {
			updates["payment"] = validatedPayment
		}
	}

	// Reject if no allowed updates provided
	if !hasAllowedUpdate {
		return "", nil, errors.NewDomainError(65001, "invalid request", "/update must contain allowed fields: fulfillments, payment (PCC/DCC/Authorization)")
	}

	return orderID, updates, nil
}

func (h *UpdateHandler) composeUpdateResponse(req *models.ONDCRequest) models.ONDCACKResponse {
	return models.ONDCACKResponse{
		Message: models.ONDCACKMessage{
			Ack: models.ONDCACKStatus{
				Status: "ACK",
			},
		},
	}
}

func (h *UpdateHandler) sendUpdateCallback(ctx context.Context, req *models.ONDCRequest, updates map[string]interface{}, orderRecord *OrderRecord, traceID string) {
	callbackURL := req.Context.BapURI + "/on_update"
	callbackPayload := h.buildOnUpdateCallback(req, updates, orderRecord)

	if err := h.callbackService.SendCallback(ctx, callbackURL, callbackPayload); err != nil {
		h.logger.Error("failed to send /on_update callback", zap.Error(err), zap.String("trace_id", traceID), zap.String("callback_url", callbackURL))
	}
}

func (h *UpdateHandler) buildOnUpdateCallback(req *models.ONDCRequest, updates map[string]interface{}, orderRecord *OrderRecord) models.ONDCResponse {
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

	// Build ONDC-compliant structure: order.fulfillments[] array
	message := map[string]interface{}{
		"order": map[string]interface{}{
			"id": orderID,
		},
	}

	// Process fulfillments from updates, enforcing stable fulfillment.id
	if fulfillments, ok := updates["fulfillments"].([]interface{}); ok && len(fulfillments) > 0 {
		// Take first fulfillment and enforce stable ID
		fulfillment, ok := fulfillments[0].(map[string]interface{})
		if ok {
			// Override fulfillment.id with stable ID from orderRecord
			fulfillment["id"] = fulfillmentID
			message["order"].(map[string]interface{})["fulfillments"] = []map[string]interface{}{fulfillment}
		}
	} else {
		// If no fulfillments in update, create minimal fulfillment with stable ID
		message["order"].(map[string]interface{})["fulfillments"] = []map[string]interface{}{
			{
				"id": fulfillmentID,
			},
		}
	}

	// Add payment info if provided
	if payment, ok := updates["payment"].(map[string]interface{}); ok {
		message["order"].(map[string]interface{})["payment"] = payment
	}

	return models.ONDCResponse{
		Context: callbackCtx,
		Message: message,
	}
}

func (h *UpdateHandler) buildIdempotencyKey(transactionID, messageID string) string {
	return "update:" + transactionID + ":" + messageID
}

func (h *UpdateHandler) respondACK(c *gin.Context, response interface{}) {
	c.JSON(http.StatusOK, response)
}

func (h *UpdateHandler) respondNACK(c *gin.Context, err *errors.DomainError) {
	var ctx models.ONDCContext
	if req, ok := c.Get("ondc_request"); ok {
		if ondcReq, ok := req.(*models.ONDCRequest); ok {
			ctx = ondcReq.Context
		}
	}

	c.JSON(errors.GetHTTPStatus(err), models.ONDCResponse{
		Context: ctx,
		Error: &models.ONDCError{
			Type:    "CONTEXT_ERROR",
			Code:    fmt.Sprintf("%d", err.Code),
			Message: map[string]string{"en": err.Message},
		},
	})
}
