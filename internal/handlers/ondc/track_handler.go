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
	"go.uber.org/zap"
)

// TrackHandler handles /track ONDC requests
type TrackHandler struct {
	callbackService    CallbackService
	idempotencyService IdempotencyService
	orderServiceClient OrderServiceClient
	orderRecordService OrderRecordService
	auditService       AuditService
	bppID              string // BPP ID (ONDC-registered Seller NP identity)
	bppURI             string // BPP URI
	logger             *zap.Logger
}

// NewTrackHandler creates a new track handler
func NewTrackHandler(
	callbackService CallbackService,
	idempotencyService IdempotencyService,
	orderServiceClient OrderServiceClient,
	orderRecordService OrderRecordService,
	auditService AuditService,
	bppID string,
	bppURI string,
	logger *zap.Logger,
) *TrackHandler {
	return &TrackHandler{
		callbackService:    callbackService,
		idempotencyService: idempotencyService,
		orderServiceClient: orderServiceClient,
		orderRecordService: orderRecordService,
		auditService:       auditService,
		bppID:              bppID,
		bppURI:             bppURI,
		logger:             logger,
	}
}

// HandleTrack handles POST /track requests
func (h *TrackHandler) HandleTrack(c *gin.Context) {
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
		var existingResponse models.ONDCResponse
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

	orderTracking, err := h.orderServiceClient.GetOrderTracking(ctx, dispatchOrderID)
	if err != nil {
		h.logger.Error("failed to get order tracking", zap.Error(err), zap.String("trace_id", traceID), zap.String("dispatch_order_id", dispatchOrderID))
		domainErr, ok := err.(*errors.DomainError)
		if ok {
			h.respondNACK(c, domainErr)
		} else {
			h.respondNACK(c, errors.NewDomainError(65020, "internal error", "failed to get order tracking"))
		}
		return
	}

	// ONDC v1.2.0: /track is polling-based, SYNC response only
	// callback_url was removed - updates are through polling only
	// DO NOT send /on_track callback
	response := h.composeTrackResponse(&req, orderTracking, orderRecord)

	responseBytes, _ := json.Marshal(response)
	_ = h.idempotencyService.StoreIdempotency(ctx, idempotencyKey, responseBytes, 24*time.Hour)

	// Log request/response audit (no callback for /track)
	h.logRequestResponse(ctx, &req, response, nil, orderRecord, clientID, traceID)

	h.respondACK(c, response)
}

func (h *TrackHandler) extractOrderID(req *models.ONDCRequest) (string, error) {
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

func (h *TrackHandler) composeTrackResponse(req *models.ONDCRequest, orderTracking *OrderTracking, orderRecord *OrderRecord) models.ONDCResponse {
	// ONDC v1.2.0: /track returns SYNC response with tracking data
	// No callback - polling-based only

	// Retrieve order.id and fulfillment.id from orderRecord (stable identifiers)
	orderID := orderRecord.OrderID
	if orderID == "" {
		h.logger.Error("order.id not found in order record")
		return models.ONDCResponse{
			Context: req.Context,
			Error: &models.ONDCError{
				Type:    "CONTEXT_ERROR",
				Code:    "65020",
				Message: map[string]string{"en": "internal error"},
			},
		}
	}

	// Use stable fulfillment ID from orderRecord (set in /init, reused across all responses)
	fulfillmentID := orderRecord.FulfillmentID
	if fulfillmentID == "" {
		fulfillmentID = "F1" // Fallback
	}

	// Map tracking state to ONDC fulfillment state code
	stateCode := "IN_TRANSIT"
	if len(orderTracking.Timeline) > 0 {
		latestEvent := orderTracking.Timeline[len(orderTracking.Timeline)-1]
		switch latestEvent.State {
		case "CONFIRMED", "RIDER_ASSIGNED":
			stateCode = "RIDER_ASSIGNED"
		case "PICKED_UP":
			stateCode = "PICKED_UP"
		case "IN_TRANSIT":
			stateCode = "IN_TRANSIT"
		case "DELIVERED":
			stateCode = "DELIVERED"
		default:
			stateCode = "IN_TRANSIT"
		}
	}

	// Build ONDC-compliant structure: order.fulfillments[] with tracking
	fulfillment := map[string]interface{}{
		"id":           fulfillmentID,
		"tracking":     true,
		"tracking_url": orderTracking.TrackingURL,
		"state": map[string]interface{}{
			"descriptor": map[string]interface{}{
				"code": stateCode,
			},
		},
	}

	// Add location if available
	if orderTracking.CurrentLocation.Lat != 0 && orderTracking.CurrentLocation.Lng != 0 {
		fulfillment["location"] = map[string]interface{}{
			"gps": fmt.Sprintf("%f,%f", orderTracking.CurrentLocation.Lat, orderTracking.CurrentLocation.Lng),
		}
	}

	message := map[string]interface{}{
		"order": map[string]interface{}{
			"id":           orderID,
			"fulfillments": []map[string]interface{}{fulfillment},
		},
	}

	return models.ONDCResponse{
		Context: req.Context,
		Message: message,
	}
}

// NOTE: /on_track callback removed per ONDC v1.2.0
// /track is polling-based only - callback_url was removed
// Updates are through polling only

func (h *TrackHandler) buildIdempotencyKey(transactionID, messageID string) string {
	return "track:" + transactionID + ":" + messageID
}

func (h *TrackHandler) respondACK(c *gin.Context, response interface{}) {
	// ONDC v1.2.0: /track returns SYNC response with tracking data
	c.JSON(http.StatusOK, response)
}

func (h *TrackHandler) respondNACK(c *gin.Context, err *errors.DomainError) {
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

func (h *TrackHandler) logRequestResponse(ctx context.Context, req *models.ONDCRequest, ackResponse interface{}, callbackPayload interface{}, orderRecord *OrderRecord, clientID, traceID string) {
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
		Action:          "track",
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

func (h *TrackHandler) toMap(v interface{}) map[string]interface{} {
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
