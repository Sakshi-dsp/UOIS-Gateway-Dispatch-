package ondc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"uois-gateway/internal/models"
	"uois-gateway/internal/utils"
	"uois-gateway/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// InitHandler handles /init ONDC requests
type InitHandler struct {
	eventPublisher     EventPublisher
	eventConsumer      EventConsumer
	callbackService    CallbackService
	idempotencyService IdempotencyService
	orderServiceClient OrderServiceClient
	orderRecordService OrderRecordService
	providerID         string // Stable provider identifier (e.g., "P1")
	bppID              string // BPP ID (ONDC-registered Seller NP identity)
	bppURI             string // BPP URI
	logger             *zap.Logger
}

// NewInitHandler creates a new init handler
func NewInitHandler(
	eventPublisher EventPublisher,
	eventConsumer EventConsumer,
	callbackService CallbackService,
	idempotencyService IdempotencyService,
	orderServiceClient OrderServiceClient,
	orderRecordService OrderRecordService,
	providerID string,
	bppID string,
	bppURI string,
	logger *zap.Logger,
) *InitHandler {
	return &InitHandler{
		eventPublisher:     eventPublisher,
		eventConsumer:      eventConsumer,
		callbackService:    callbackService,
		idempotencyService: idempotencyService,
		orderServiceClient: orderServiceClient,
		orderRecordService: orderRecordService,
		providerID:         providerID,
		bppID:              bppID,
		bppURI:             bppURI,
		logger:             logger,
	}
}

// HandleInit handles POST /init requests
func (h *InitHandler) HandleInit(c *gin.Context) {
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

	// Validate provider.id matches catalog provider (protocol requirement)
	// provider.id is stable identifier (e.g., "P1"), NOT search_id, NOT bppID
	if err := h.validateProviderID(&req); err != nil {
		h.respondNACK(c, err.(*errors.DomainError))
		return
	}

	// Lookup order record using transaction_id (internal correlation mechanism)
	// IMPORTANT: transaction_id is the primary flow key (identifies the business flow).
	// message_id is used ONLY for idempotency (deduplication of protocol messages).
	// Per ID Domain Isolation Law: search_id is internal-only, retrieved from order record, NOT from ONDC payload
	orderRecord, err := h.orderRecordService.GetOrderRecordByTransactionID(ctx, req.Context.TransactionID)
	if err != nil {
		h.logger.Error("order record not found for transaction_id", zap.Error(err), zap.String("trace_id", traceID), zap.String("transaction_id", req.Context.TransactionID))
		h.respondNACK(c, errors.NewDomainError(65001, "invalid request", "order record not found"))
		return
	}
	if orderRecord == nil {
		h.logger.Error("order record not found for transaction_id", zap.String("trace_id", traceID), zap.String("transaction_id", req.Context.TransactionID))
		h.respondNACK(c, errors.NewDomainError(65001, "invalid request", "order record not found"))
		return
	}

	// Retrieve internal search_id from order record (internal-only, never in ONDC payload)
	searchID := orderRecord.SearchID
	if searchID == "" {
		h.logger.Error("search_id not found in order record", zap.String("trace_id", traceID), zap.String("transaction_id", req.Context.TransactionID))
		h.respondNACK(c, errors.NewDomainError(65001, "invalid request", "search_id not found"))
		return
	}

	// Validate search_id TTL via Order Service
	// TODO: Architecturally, search TTL validation should belong to Quote/Location service layer,
	// not Order Service. This is acceptable for now but should be refactored later.
	valid, err := h.orderServiceClient.ValidateSearchIDTTL(ctx, searchID)
	if err != nil {
		h.logger.Error("failed to validate search_id TTL", zap.Error(err), zap.String("trace_id", traceID), zap.String("search_id", searchID))
		domainErr, ok := err.(*errors.DomainError)
		if ok {
			h.respondNACK(c, domainErr)
		} else {
			h.respondNACK(c, errors.NewDomainError(65020, "internal error", "failed to validate search_id"))
		}
		return
	}
	if !valid {
		h.respondNACK(c, errors.NewDomainError(65004, "quote expired", "search_id TTL expired"))
		return
	}

	// Extract coordinates and addresses
	originLat, originLng, destLat, destLng, originAddr, destAddr, packageInfo, err := h.extractInitData(&req)
	if err != nil {
		h.respondNACK(c, errors.NewDomainError(65001, "invalid request", err.Error()))
		return
	}

	// Publish INIT_REQUESTED event
	initEvent := h.buildInitRequestedEvent(searchID, originLat, originLng, destLat, destLng, originAddr, destAddr, packageInfo, traceparent)
	if err := h.eventPublisher.PublishEvent(ctx, "stream.uois.init_requested", initEvent); err != nil {
		h.logger.Error("failed to publish INIT_REQUESTED event", zap.Error(err), zap.String("trace_id", traceID), zap.String("search_id", searchID))
		h.respondNACK(c, errors.NewDomainError(65020, "internal error", "failed to publish event"))
		return
	}

	// Parse and validate TTL
	ttlDuration, domainErr := h.parseTTL(req.Context.TTL)
	if domainErr != nil {
		h.respondNACK(c, domainErr)
		return
	}

	// Consume QUOTE_CREATED or QUOTE_INVALIDATED event
	var quoteEvent interface{}

	// Try QUOTE_CREATED first
	quoteEvent, err = h.eventConsumer.ConsumeEvent(ctx, "stream.uois.quote_created", "uois-gateway-consumers", searchID, ttlDuration)
	if err != nil {
		// Fallback to QUOTE_INVALIDATED stream
		quoteEvent, err = h.eventConsumer.ConsumeEvent(ctx, "stream.uois.quote_invalidated", "uois-gateway-consumers", searchID, ttlDuration)
		if err != nil {
			h.logger.Error("failed to consume quote event from both streams", zap.Error(err), zap.String("trace_id", traceID), zap.String("search_id", searchID))
			h.respondNACK(c, errors.NewDomainError(65020, "internal error", "failed to consume event"))
			return
		}
	}

	// Update order record with quote_id and fulfillment_id if QUOTE_CREATED
	// Store quote_id and fulfillment_id alongside search_id on the same order record (no mapping, just storage)
	// FulfillmentID must be stable per order and reused in /confirm
	var fulfillmentID string
	if quoteCreated, ok := quoteEvent.(*models.QuoteCreatedEvent); ok {
		orderRecord.QuoteID = quoteCreated.QuoteID
		// Generate stable fulfillment ID (reused in /confirm)
		if orderRecord.FulfillmentID == "" {
			fulfillmentID = uuid.New().String()
			orderRecord.FulfillmentID = fulfillmentID
		} else {
			fulfillmentID = orderRecord.FulfillmentID
		}
		if err := h.orderRecordService.UpdateOrderRecord(ctx, orderRecord); err != nil {
			h.logger.Warn("failed to update order record with quote_id and fulfillment_id", zap.Error(err), zap.String("trace_id", traceID), zap.String("search_id", searchID), zap.String("quote_id", quoteCreated.QuoteID))
		}
	}

	// Compose response
	response := h.composeInitResponse(&req, quoteEvent)

	// Store idempotency (marshal to preserve byte-exactness for ONDC signatures)
	responseBytes, _ := json.Marshal(response)
	_ = h.idempotencyService.StoreIdempotency(ctx, idempotencyKey, responseBytes, 24*time.Hour)

	// Send callback asynchronously (pass fulfillmentID for stable reuse)
	go h.sendInitCallback(ctx, &req, quoteEvent, fulfillmentID, traceID)

	// Return ACK
	h.respondACK(c, response)
}

func (h *InitHandler) validateProviderID(req *models.ONDCRequest) error {
	order, ok := req.Message["order"].(map[string]interface{})
	if !ok {
		return errors.NewDomainError(65001, "invalid request", "missing order")
	}

	provider, ok := order["provider"].(map[string]interface{})
	if !ok {
		return errors.NewDomainError(65001, "invalid request", "missing provider")
	}

	providerID, ok := provider["id"].(string)
	if !ok || providerID == "" {
		return errors.NewDomainError(65001, "invalid request", "missing provider.id")
	}

	if providerID != h.providerID {
		return errors.NewDomainError(65001, "invalid request", "provider.id does not match catalog provider")
	}

	return nil
}

func (h *InitHandler) extractInitData(req *models.ONDCRequest) (float64, float64, float64, float64, map[string]interface{}, map[string]interface{}, map[string]interface{}, error) {
	order, ok := req.Message["order"].(map[string]interface{})
	if !ok {
		return 0, 0, 0, 0, nil, nil, nil, errors.NewDomainError(65001, "invalid request", "missing order")
	}

	fulfillment, ok := order["fulfillment"].(map[string]interface{})
	if !ok {
		return 0, 0, 0, 0, nil, nil, nil, errors.NewDomainError(65001, "invalid request", "missing fulfillment")
	}

	start, ok := fulfillment["start"].(map[string]interface{})
	if !ok {
		return 0, 0, 0, 0, nil, nil, nil, errors.NewDomainError(65001, "invalid request", "missing start location")
	}

	end, ok := fulfillment["end"].(map[string]interface{})
	if !ok {
		return 0, 0, 0, 0, nil, nil, nil, errors.NewDomainError(65001, "invalid request", "missing end location")
	}

	startLoc, ok := start["location"].(map[string]interface{})
	if !ok {
		return 0, 0, 0, 0, nil, nil, nil, errors.NewDomainError(65001, "invalid request", "missing start location")
	}

	endLoc, ok := end["location"].(map[string]interface{})
	if !ok {
		return 0, 0, 0, 0, nil, nil, nil, errors.NewDomainError(65001, "invalid request", "missing end location")
	}

	startGPS, ok := startLoc["gps"].(string)
	if !ok {
		return 0, 0, 0, 0, nil, nil, nil, errors.NewDomainError(65001, "invalid request", "missing start GPS")
	}

	endGPS, ok := endLoc["gps"].(string)
	if !ok {
		return 0, 0, 0, 0, nil, nil, nil, errors.NewDomainError(65001, "invalid request", "missing end GPS")
	}

	originLat, originLng, err := h.parseGPS(startGPS)
	if err != nil {
		return 0, 0, 0, 0, nil, nil, nil, err
	}

	destLat, destLng, err := h.parseGPS(endGPS)
	if err != nil {
		return 0, 0, 0, 0, nil, nil, nil, err
	}

	// Extract addresses (optional)
	originAddr, _ := startLoc["address"].(map[string]interface{})
	destAddr, _ := endLoc["address"].(map[string]interface{})

	// Extract package info from items (optional)
	var packageInfo map[string]interface{}
	if items, ok := order["items"].([]interface{}); ok && len(items) > 0 {
		if item, ok := items[0].(map[string]interface{}); ok {
			packageInfo = item
		}
	}

	return originLat, originLng, destLat, destLng, originAddr, destAddr, packageInfo, nil
}

func (h *InitHandler) extractItemsFromRequest(req *models.ONDCRequest) []map[string]interface{} {
	order, ok := req.Message["order"].(map[string]interface{})
	if !ok {
		return []map[string]interface{}{}
	}

	items, ok := order["items"].([]interface{})
	if !ok || len(items) == 0 {
		return []map[string]interface{}{}
	}

	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if itemMap, ok := item.(map[string]interface{}); ok {
			result = append(result, itemMap)
		}
	}

	return result
}

func (h *InitHandler) buildItemsWithFulfillment(items []map[string]interface{}, fulfillmentID string, quoteCreated *models.QuoteCreatedEvent) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		itemCopy := make(map[string]interface{})
		for k, v := range item {
			itemCopy[k] = v
		}
		itemCopy["fulfillment"] = map[string]interface{}{
			"id": fulfillmentID,
			"time": map[string]interface{}{
				"duration": map[string]interface{}{
					"to_pickup": h.formatDuration(quoteCreated.ETAOrigin, quoteCreated.Timestamp),
					"to_drop":   h.formatDuration(quoteCreated.ETADestination, quoteCreated.Timestamp),
					"unit":      "ISO8601",
				},
			},
		}
		result = append(result, itemCopy)
	}

	return result
}

func (h *InitHandler) parseGPS(gps string) (float64, float64, error) {
	parts := strings.Split(gps, ",")
	if len(parts) != 2 {
		return 0, 0, errors.NewDomainError(65001, "invalid GPS format", "expected lat,lng")
	}

	var lat, lng float64
	if _, err := fmt.Sscanf(parts[0], "%f", &lat); err != nil {
		return 0, 0, errors.NewDomainError(65001, "invalid latitude", err.Error())
	}
	if _, err := fmt.Sscanf(parts[1], "%f", &lng); err != nil {
		return 0, 0, errors.NewDomainError(65001, "invalid longitude", err.Error())
	}

	return lat, lng, nil
}

func (h *InitHandler) buildInitRequestedEvent(searchID string, originLat, originLng, destLat, destLng float64, originAddr, destAddr, packageInfo map[string]interface{}, traceparent string) *models.InitRequestedEvent {
	traceparent = utils.EnsureTraceparent(traceparent)

	return &models.InitRequestedEvent{
		BaseEvent: models.BaseEvent{
			EventType:   "INIT_REQUESTED",
			EventID:     uuid.New().String(),
			Traceparent: traceparent,
			Timestamp:   time.Now(),
		},
		SearchID:           searchID,
		OriginLat:          originLat,
		OriginLng:          originLng,
		OriginAddress:      originAddr,
		DestinationLat:     destLat,
		DestinationLng:     destLng,
		DestinationAddress: destAddr,
		PackageInfo:        packageInfo,
	}
}

func (h *InitHandler) parseTTL(ttl string) (time.Duration, *errors.DomainError) {
	if ttl == "" {
		return 0, errors.NewDomainError(65001, "invalid context", "ttl is required")
	}

	if !strings.HasPrefix(ttl, "PT") {
		return 0, errors.NewDomainError(65001, "invalid context", "invalid ttl format (expected ISO8601 duration)")
	}

	ttlStr := strings.TrimPrefix(ttl, "PT")
	var hours, minutes, seconds int

	if idx := strings.Index(ttlStr, "H"); idx != -1 {
		fmt.Sscanf(ttlStr[:idx+1], "%dH", &hours)
		ttlStr = ttlStr[idx+1:]
	}

	if idx := strings.Index(ttlStr, "M"); idx != -1 {
		fmt.Sscanf(ttlStr[:idx+1], "%dM", &minutes)
		ttlStr = ttlStr[idx+1:]
	}

	if idx := strings.Index(ttlStr, "S"); idx != -1 {
		fmt.Sscanf(ttlStr[:idx+1], "%dS", &seconds)
	}

	// Convert to time.Duration
	// PT0S is valid ISO8601 (immediate timeout) - allow zero duration
	duration := time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second

	return duration, nil
}

func (h *InitHandler) composeInitResponse(req *models.ONDCRequest, quoteEvent interface{}) models.ONDCACKResponse {
	return models.ONDCACKResponse{
		Message: models.ONDCACKMessage{
			Ack: models.ONDCACKStatus{
				Status: "ACK",
			},
		},
	}
}

func (h *InitHandler) sendInitCallback(ctx context.Context, req *models.ONDCRequest, quoteEvent interface{}, fulfillmentID string, traceID string) {
	callbackURL := req.Context.BapURI + "/on_init"
	callbackPayload := h.buildOnInitCallback(req, quoteEvent, fulfillmentID)

	if err := h.callbackService.SendCallback(ctx, callbackURL, callbackPayload); err != nil {
		h.logger.Error("failed to send /on_init callback", zap.Error(err), zap.String("trace_id", traceID), zap.String("callback_url", callbackURL))
	}
}

func (h *InitHandler) buildOnInitCallback(req *models.ONDCRequest, quoteEvent interface{}, fulfillmentID string) models.ONDCResponse {
	// Regenerate callback context (ONDC protocol requirement)
	callbackCtx := req.Context
	callbackCtx.MessageID = uuid.New().String()
	callbackCtx.Timestamp = time.Now().UTC()
	callbackCtx.BppID = h.bppID
	callbackCtx.BppURI = h.bppURI

	quoteCreated, ok := quoteEvent.(*models.QuoteCreatedEvent)
	if ok {
		// Extract items from request (no hardcoding)
		items := h.extractItemsFromRequest(req)

		// Use stable fulfillment ID (generated in HandleInit, stored in orderRecord, reused in /confirm)
		// If not provided, generate new one (fallback for QUOTE_INVALIDATED case)
		if fulfillmentID == "" {
			fulfillmentID = uuid.New().String()
		}

		// Success case: QUOTE_CREATED
		message := map[string]interface{}{
			"order": map[string]interface{}{
				"provider": map[string]interface{}{
					"id": h.bppID,
				},
				"quote": map[string]interface{}{
					"id": quoteCreated.QuoteID,
					"price": map[string]interface{}{
						"value":    quoteCreated.Price.Value,
						"currency": quoteCreated.Price.Currency,
					},
					"ttl": quoteCreated.TTL,
				},
				"items": h.buildItemsWithFulfillment(items, fulfillmentID, quoteCreated),
			},
		}

		return models.ONDCResponse{
			Context: callbackCtx,
			Message: message,
		}
	}

	quoteInvalidated, ok := quoteEvent.(*models.QuoteInvalidatedEvent)
	if ok {
		// Error case: QUOTE_INVALIDATED
		return models.ONDCResponse{
			Context: callbackCtx,
			Error: &models.ONDCError{
				Type:    "CONTEXT_ERROR",
				Code:    "65005",
				Message: map[string]string{"en": quoteInvalidated.Message},
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

func (h *InitHandler) formatDuration(t *time.Time, baseTime time.Time) string {
	if t == nil {
		return ""
	}
	duration := t.Sub(baseTime)
	if duration < 0 {
		return "PT0S"
	}
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("PT%dH%dM%dS", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("PT%dM%dS", minutes, seconds)
	}
	return fmt.Sprintf("PT%dS", seconds)
}

func (h *InitHandler) buildIdempotencyKey(transactionID, messageID string) string {
	return "init:" + transactionID + ":" + messageID
}

func (h *InitHandler) respondACK(c *gin.Context, response interface{}) {
	c.JSON(http.StatusOK, response)
}

func (h *InitHandler) respondNACK(c *gin.Context, err *errors.DomainError) {
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
