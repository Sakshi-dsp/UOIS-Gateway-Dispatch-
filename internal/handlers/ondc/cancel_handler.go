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

// CancelHandler handles /cancel ONDC requests
type CancelHandler struct {
	callbackService    CallbackService
	idempotencyService IdempotencyService
	orderServiceClient OrderServiceClient
	orderRecordService OrderRecordService
	bppID              string // BPP ID (ONDC-registered Seller NP identity)
	bppURI             string // BPP URI
	logger             *zap.Logger
}

// NewCancelHandler creates a new cancel handler
func NewCancelHandler(
	callbackService CallbackService,
	idempotencyService IdempotencyService,
	orderServiceClient OrderServiceClient,
	orderRecordService OrderRecordService,
	bppID string,
	bppURI string,
	logger *zap.Logger,
) *CancelHandler {
	return &CancelHandler{
		callbackService:    callbackService,
		idempotencyService: idempotencyService,
		orderServiceClient: orderServiceClient,
		orderRecordService: orderRecordService,
		bppID:              bppID,
		bppURI:             bppURI,
		logger:             logger,
	}
}

// HandleCancel handles POST /cancel requests
func (h *CancelHandler) HandleCancel(c *gin.Context) {
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
	orderID, reason, err := h.extractCancelData(&req)
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

	if err := h.orderServiceClient.CancelOrder(ctx, dispatchOrderID, reason); err != nil {
		h.logger.Error("failed to cancel order", zap.Error(err), zap.String("trace_id", traceID), zap.String("dispatch_order_id", dispatchOrderID))
		domainErr, ok := err.(*errors.DomainError)
		if ok {
			h.respondNACK(c, domainErr)
		} else {
			h.respondNACK(c, errors.NewDomainError(65020, "internal error", "failed to cancel order"))
		}
		return
	}

	response := h.composeCancelResponse(&req)

	responseBytes, _ := json.Marshal(response)
	_ = h.idempotencyService.StoreIdempotency(ctx, idempotencyKey, responseBytes, 24*time.Hour)

	go h.sendCancelCallback(ctx, &req, orderRecord, traceID)

	h.respondACK(c, response)
}

func (h *CancelHandler) extractCancelData(req *models.ONDCRequest) (string, string, error) {
	order, ok := req.Message["order"].(map[string]interface{})
	if !ok {
		return "", "", errors.NewDomainError(65001, "invalid request", "missing order")
	}

	orderID, ok := order["id"].(string)
	if !ok || orderID == "" {
		return "", "", errors.NewDomainError(65001, "invalid request", "missing order.id")
	}

	// Extract cancellation_reason_id (ONDC reason code, not free text)
	reason := "001" // Default: "Buyer cancelled"
	if reasonID, ok := req.Message["cancellation_reason_id"].(string); ok && reasonID != "" {
		reason = reasonID
	}

	return orderID, reason, nil
}

func (h *CancelHandler) composeCancelResponse(req *models.ONDCRequest) models.ONDCACKResponse {
	return models.ONDCACKResponse{
		Message: models.ONDCACKMessage{
			Ack: models.ONDCACKStatus{
				Status: "ACK",
			},
		},
	}
}

func (h *CancelHandler) sendCancelCallback(ctx context.Context, req *models.ONDCRequest, orderRecord *OrderRecord, traceID string) {
	callbackURL := req.Context.BapURI + "/on_cancel"
	callbackPayload := h.buildOnCancelCallback(req, orderRecord)

	if err := h.callbackService.SendCallback(ctx, callbackURL, callbackPayload); err != nil {
		h.logger.Error("failed to send /on_cancel callback", zap.Error(err), zap.String("trace_id", traceID), zap.String("callback_url", callbackURL))
	}
}

func (h *CancelHandler) buildOnCancelCallback(req *models.ONDCRequest, orderRecord *OrderRecord) models.ONDCResponse {
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

	// Build ONDC-compliant structure: order.fulfillments[] array with cancellation state
	fulfillment := map[string]interface{}{
		"id": fulfillmentID, // Stable fulfillment ID (reused from /init)
		"state": map[string]interface{}{
			"descriptor": map[string]interface{}{
				"code": "CANCELLED", // Cancellation state in fulfillment
			},
		},
	}

	message := map[string]interface{}{
		"order": map[string]interface{}{
			"id":           orderID,
			"state":        "CANCELLED", // Order-level cancellation state
			"fulfillments": []map[string]interface{}{fulfillment},
		},
	}

	return models.ONDCResponse{
		Context: callbackCtx,
		Message: message,
	}
}

func (h *CancelHandler) buildIdempotencyKey(transactionID, messageID string) string {
	return "cancel:" + transactionID + ":" + messageID
}

func (h *CancelHandler) respondACK(c *gin.Context, response interface{}) {
	c.JSON(http.StatusOK, response)
}

func (h *CancelHandler) respondNACK(c *gin.Context, err *errors.DomainError) {
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
