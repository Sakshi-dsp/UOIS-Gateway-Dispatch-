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

// SearchHandler handles /search ONDC requests
type SearchHandler struct {
	eventPublisher     EventPublisher
	eventConsumer      EventConsumer
	callbackService    CallbackService
	idempotencyService IdempotencyService
	orderRecordService OrderRecordService
	auditService       AuditService
	providerID         string // Stable provider identifier (e.g., "P1")
	bppID              string // BPP ID (ONDC-registered Seller NP identity)
	bppURI             string // BPP URI
	bppName            string // BPP display name
	bppTermsURL        string // Static terms URL
	logger             *zap.Logger
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(
	eventPublisher EventPublisher,
	eventConsumer EventConsumer,
	callbackService CallbackService,
	idempotencyService IdempotencyService,
	orderRecordService OrderRecordService,
	auditService AuditService,
	providerID string,
	bppID string,
	bppURI string,
	bppName string,
	bppTermsURL string,
	logger *zap.Logger,
) *SearchHandler {
	return &SearchHandler{
		eventPublisher:     eventPublisher,
		eventConsumer:      eventConsumer,
		callbackService:    callbackService,
		idempotencyService: idempotencyService,
		orderRecordService: orderRecordService,
		auditService:       auditService,
		providerID:         providerID,
		bppID:              bppID,
		bppURI:             bppURI,
		bppName:            bppName,
		bppTermsURL:        bppTermsURL,
		logger:             logger,
	}
}

// HandleSearch handles POST /search requests
func (h *SearchHandler) HandleSearch(c *gin.Context) {
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
	intent, _ := req.Message["intent"].(map[string]interface{})
	paymentInfo, _ := intent["payment"].(map[string]interface{})
	if err := utils.ValidatePaymentType(paymentInfo); err != nil {
		h.logger.Warn("payment type validation failed", zap.Error(err), zap.String("trace_id", traceID))
		h.respondNACK(c, err)
		return
	}

	// Validate delivery category (Dispatch only supports Immediate Delivery or Standard Delivery with immediate subcategory)
	categoryID := utils.ExtractCategoryID(intent)
	timeDuration := utils.ExtractTimeDuration(intent)
	if err := utils.ValidateDeliveryCategory(categoryID, timeDuration); err != nil {
		h.logger.Warn("delivery category validation failed", zap.Error(err), zap.String("trace_id", traceID), zap.String("category_id", categoryID), zap.String("time_duration", timeDuration))
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

	// Generate search_id
	searchID := uuid.New().String()

	// Get client ID from context
	client, _ := c.Get("client")
	var clientID string
	if cl, ok := client.(*models.Client); ok {
		clientID = cl.ID
	}

	// Store order record with search_id, transaction_id, and message_id
	// IMPORTANT: transaction_id is the primary flow key for /init correlation lookup.
	// message_id is used ONLY for idempotency (deduplication of protocol messages).
	// Per ID Domain Isolation Law: /init handler uses transaction_id + message_id to lookup order record.
	orderRecord := &OrderRecord{
		SearchID:      searchID,
		ClientID:      clientID,
		TransactionID: req.Context.TransactionID,
		MessageID:     req.Context.MessageID,
	}
	if err := h.orderRecordService.StoreOrderRecord(ctx, orderRecord); err != nil {
		h.logger.Error("failed to store order record with search_id", zap.Error(err), zap.String("trace_id", traceID), zap.String("search_id", searchID))
		h.respondNACK(c, errors.NewDomainError(65020, "internal error", "failed to store order record"))
		return
	}

	// Extract coordinates from request
	originLat, originLng, destLat, destLng, err := h.extractCoordinates(&req)
	if err != nil {
		h.respondNACK(c, errors.NewDomainError(65001, "invalid coordinates", err.Error()))
		return
	}

	// Publish SEARCH_REQUESTED event
	searchEvent := h.buildSearchRequestedEvent(searchID, originLat, originLng, destLat, destLng, traceparent)
	if err := h.eventPublisher.PublishEvent(ctx, "stream.location.search", searchEvent); err != nil {
		h.logger.Error("failed to publish SEARCH_REQUESTED event", zap.Error(err), zap.String("trace_id", traceID), zap.String("search_id", searchID))
		h.respondNACK(c, errors.NewDomainError(65020, "internal error", "failed to publish event"))
		return
	}

	// Compose ACK response (immediate, does not depend on quote event)
	response := h.composeSearchResponse()

	// Store idempotency (marshal to preserve byte-exactness for ONDC signatures)
	responseBytes, _ := json.Marshal(response)
	_ = h.idempotencyService.StoreIdempotency(ctx, idempotencyKey, responseBytes, 24*time.Hour)

	// Log request/response audit
	h.logRequestResponse(ctx, &req, response, nil, searchID, clientID, traceID)

	// Send callback asynchronously (consumes QUOTE_COMPUTED event inside)
	go h.sendSearchCallback(ctx, &req, searchID, traceID)

	// Return ACK immediately
	h.respondACK(c, response)
}

func (h *SearchHandler) extractCoordinates(req *models.ONDCRequest) (float64, float64, float64, float64, error) {
	intent, ok := req.Message["intent"].(map[string]interface{})
	if !ok {
		return 0, 0, 0, 0, errors.NewDomainError(65001, "invalid request", "missing intent")
	}

	fulfillment, ok := intent["fulfillment"].(map[string]interface{})
	if !ok {
		return 0, 0, 0, 0, errors.NewDomainError(65001, "invalid request", "missing fulfillment")
	}

	start, ok := fulfillment["start"].(map[string]interface{})
	if !ok {
		return 0, 0, 0, 0, errors.NewDomainError(65001, "invalid request", "missing start location")
	}

	end, ok := fulfillment["end"].(map[string]interface{})
	if !ok {
		return 0, 0, 0, 0, errors.NewDomainError(65001, "invalid request", "missing end location")
	}

	startLoc, ok := start["location"].(map[string]interface{})
	if !ok {
		return 0, 0, 0, 0, errors.NewDomainError(65001, "invalid request", "missing start location")
	}

	endLoc, ok := end["location"].(map[string]interface{})
	if !ok {
		return 0, 0, 0, 0, errors.NewDomainError(65001, "invalid request", "missing end location")
	}

	startGPS, ok := startLoc["gps"].(string)
	if !ok {
		return 0, 0, 0, 0, errors.NewDomainError(65001, "invalid request", "missing start GPS")
	}

	endGPS, ok := endLoc["gps"].(string)
	if !ok {
		return 0, 0, 0, 0, errors.NewDomainError(65001, "invalid request", "missing end GPS")
	}

	originLat, originLng, err := h.parseGPS(startGPS)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	destLat, destLng, err := h.parseGPS(endGPS)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	return originLat, originLng, destLat, destLng, nil
}

func (h *SearchHandler) parseGPS(gps string) (float64, float64, error) {
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

	if lat < -90 || lat > 90 {
		return 0, 0, errors.NewDomainError(65001, "invalid latitude", "latitude must be between -90 and 90")
	}
	if lng < -180 || lng > 180 {
		return 0, 0, errors.NewDomainError(65001, "invalid longitude", "longitude must be between -180 and 180")
	}

	return lat, lng, nil
}

func (h *SearchHandler) buildSearchRequestedEvent(searchID string, originLat, originLng, destLat, destLng float64, traceparent string) *models.SearchRequestedEvent {
	traceparent = utils.EnsureTraceparent(traceparent)

	return &models.SearchRequestedEvent{
		BaseEvent: models.BaseEvent{
			EventType:   "SEARCH_REQUESTED",
			EventID:     uuid.New().String(),
			Traceparent: traceparent,
			Timestamp:   time.Now(),
		},
		SearchID:       searchID,
		OriginLat:      originLat,
		OriginLng:      originLng,
		DestinationLat: destLat,
		DestinationLng: destLng,
	}
}

func (h *SearchHandler) parseTTL(ttl string) (time.Duration, error) {
	if ttl == "" {
		return 0, errors.NewDomainError(65001, "invalid ttl", "ttl is required")
	}

	// Parse ISO8601 duration format (PT30S, PT15M, PT1H30M, etc.)
	// Remove "PT" prefix
	if !strings.HasPrefix(ttl, "PT") {
		return 0, errors.NewDomainError(65001, "invalid ttl", "ttl must be ISO8601 format (PT30S, PT15M, etc.)")
	}

	ttlStr := strings.TrimPrefix(ttl, "PT")
	var hours, minutes, seconds int

	// Parse hours (H suffix)
	if idx := strings.Index(ttlStr, "H"); idx != -1 {
		if _, err := fmt.Sscanf(ttlStr[:idx+1], "%dH", &hours); err != nil {
			return 0, errors.NewDomainError(65001, "invalid ttl", fmt.Sprintf("invalid hours format: %v", err))
		}
		ttlStr = ttlStr[idx+1:]
	}

	// Parse minutes (M suffix)
	if idx := strings.Index(ttlStr, "M"); idx != -1 {
		if _, err := fmt.Sscanf(ttlStr[:idx+1], "%dM", &minutes); err != nil {
			return 0, errors.NewDomainError(65001, "invalid ttl", fmt.Sprintf("invalid minutes format: %v", err))
		}
		ttlStr = ttlStr[idx+1:]
	}

	// Parse seconds (S suffix)
	if idx := strings.Index(ttlStr, "S"); idx != -1 {
		if _, err := fmt.Sscanf(ttlStr[:idx+1], "%dS", &seconds); err != nil {
			return 0, errors.NewDomainError(65001, "invalid ttl", fmt.Sprintf("invalid seconds format: %v", err))
		}
		ttlStr = ttlStr[idx+1:]
	}

	// Validate that at least one component (H, M, or S) was provided
	// "PT" with no components is invalid ISO8601
	// If ttlStr is empty after removing "PT", it means input was exactly "PT"
	// If ttlStr is not empty but we didn't parse anything, it means invalid format
	originalTTLStr := strings.TrimPrefix(ttl, "PT")
	if originalTTLStr == "" || (hours == 0 && minutes == 0 && seconds == 0 && originalTTLStr != "") {
		return 0, errors.NewDomainError(65001, "invalid ttl", "ttl must contain at least one time component (H, M, or S)")
	}

	// Convert to time.Duration
	// PT0S is valid ISO8601 (immediate timeout) - allow zero duration
	duration := time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second
	return duration, nil
}

func (h *SearchHandler) composeSearchResponse() models.ONDCACKResponse {
	// ACK response is intentionally minimal per ONDC spec.
	// No message payload required for /search ACK.
	// This minimal response is stored for idempotency replay.
	return models.ONDCACKResponse{
		Message: models.ONDCACKMessage{
			Ack: models.ONDCACKStatus{
				Status: "ACK",
			},
		},
	}
}

func (h *SearchHandler) sendSearchCallback(ctx context.Context, req *models.ONDCRequest, searchID string, traceID string) {
	// Parse TTL for event consumption timeout
	ttlDuration, err := h.parseTTL(req.Context.TTL)
	if err != nil {
		h.logger.Error("failed to parse TTL for callback", zap.Error(err), zap.String("trace_id", traceID), zap.String("search_id", searchID))
		return
	}

	// Guard against zero TTL (PT0S = immediate timeout, skip callback wait)
	if ttlDuration == 0 {
		h.logger.Warn("TTL is zero; skipping callback wait", zap.String("trace_id", traceID), zap.String("search_id", searchID))
		return
	}

	// Consume QUOTE_COMPUTED event (async, non-blocking)
	// Consumer group name: "uois-gateway-consumers" (shared across all UOIS Gateway handlers for consistency)
	quoteEvent, err := h.eventConsumer.ConsumeEvent(ctx, "quote:computed", "uois-gateway-consumers", searchID, ttlDuration)
	if err != nil {
		h.logger.Error("failed to consume QUOTE_COMPUTED event for callback", zap.Error(err), zap.String("trace_id", traceID), zap.String("search_id", searchID))
		return
	}

	// Build and send callback
	callbackURL := req.Context.BapURI + "/on_search"
	callbackPayload := h.buildOnSearchCallback(req, quoteEvent)

	if err := h.callbackService.SendCallback(ctx, callbackURL, callbackPayload); err != nil {
		h.logger.Error("failed to send /on_search callback", zap.Error(err), zap.String("trace_id", traceID), zap.String("callback_url", callbackURL), zap.String("search_id", searchID))
		h.logCallbackDelivery(ctx, req.Context.TransactionID, callbackURL, 1, "failed", err.Error())
	} else {
		h.logCallbackDelivery(ctx, req.Context.TransactionID, callbackURL, 1, "success", "")
	}
}

func (h *SearchHandler) buildOnSearchCallback(req *models.ONDCRequest, quoteEvent interface{}) models.ONDCResponse {
	quoteComputed, ok := quoteEvent.(*models.QuoteComputedEvent)
	if !ok {
		// Regenerate context for error callback
		callbackCtx := req.Context
		callbackCtx.MessageID = uuid.New().String()
		callbackCtx.Timestamp = time.Now().UTC()
		return models.ONDCResponse{
			Context: callbackCtx,
			Error: &models.ONDCError{
				Type:    "CONTEXT_ERROR",
				Code:    "65020",
				Message: map[string]string{"en": "internal error"},
			},
		}
	}

	// Regenerate callback context (ONDC protocol requirement)
	callbackCtx := req.Context
	callbackCtx.MessageID = uuid.New().String() // New message_id for callback
	callbackCtx.Timestamp = time.Now().UTC()    // New timestamp for callback
	callbackCtx.BppID = h.bppID                 // Set BPP ID
	callbackCtx.BppURI = h.bppURI               // Set BPP URI
	// transaction_id is preserved (required for correlation)

	// Extract category from intent (if provided)
	categoryID := h.extractCategoryID(req)
	if categoryID == "" {
		categoryID = "Immediate Delivery" // Default category for P2P
	}

	// If not serviceable, return empty catalog (ONDC-compliant)
	if !quoteComputed.Serviceable {
		return models.ONDCResponse{
			Context: callbackCtx,
			Message: map[string]interface{}{
				"catalog": map[string]interface{}{
					"bpp/descriptor": h.buildBPPDescriptor(),
					"bpp/providers":  []map[string]interface{}{},
				},
			},
		}
	}

	// Calculate absolute durations (not relative deltas)
	toPickupDuration := h.calculateAbsoluteDuration(quoteComputed.ETAOrigin, quoteComputed.Timestamp)
	toDropDuration := h.calculateAbsoluteDuration(quoteComputed.ETADestination, quoteComputed.Timestamp)

	// Build ONDC-compliant catalog structure
	message := map[string]interface{}{
		"catalog": map[string]interface{}{
			"bpp/descriptor": h.buildBPPDescriptor(),
			"bpp/providers": []map[string]interface{}{
				{
					"id": h.providerID, // Stable provider identifier (NOT search_id)
					"descriptor": map[string]interface{}{
						"name":       h.bppName,
						"short_desc": h.bppName,
						"long_desc":  h.bppName,
					},
					"categories": []map[string]interface{}{
						{
							"id": categoryID,
							"time": map[string]interface{}{
								"label":    "TAT",
								"duration": toDropDuration, // Total duration to drop
							},
						},
					},
					"fulfillments": []map[string]interface{}{
						{
							"id":   "1",
							"type": "Delivery",
							"start": map[string]interface{}{
								"time": map[string]interface{}{
									"duration": toPickupDuration, // Duration to pickup
								},
							},
							"tags": []map[string]interface{}{
								{
									"code": "distance",
									"list": []map[string]interface{}{
										{
											"code":  "motorable_distance_type",
											"value": "kilometer",
										},
										{
											"code":  "motorable_distance",
											"value": fmt.Sprintf("%.2f", quoteComputed.DistanceOriginToDestination),
										},
									},
								},
							},
						},
					},
					"items": []map[string]interface{}{
						{
							"id":             "I1",
							"parent_item_id": "",
							"category_id":    categoryID,
							"fulfillment_id": "1",
							"descriptor": map[string]interface{}{
								"code":       "P2P",
								"name":       h.buildItemName(toDropDuration),
								"short_desc": h.buildItemName(toDropDuration),
								"long_desc":  h.buildItemName(toDropDuration),
							},
							"price": map[string]interface{}{
								"currency": quoteComputed.Price.Currency,
								"value":    fmt.Sprintf("%.2f", quoteComputed.Price.Value),
							},
							"time": map[string]interface{}{
								"label":    "TAT",
								"duration": toDropDuration,
							},
						},
					},
				},
			},
		},
	}

	return models.ONDCResponse{
		Context: callbackCtx,
		Message: message,
	}
}

func (h *SearchHandler) buildBPPDescriptor() map[string]interface{} {
	descriptor := map[string]interface{}{
		"name": h.bppName,
	}
	if h.bppTermsURL != "" {
		descriptor["tags"] = []map[string]interface{}{
			{
				"code": "bpp_terms",
				"list": []map[string]interface{}{
					{
						"code":  "static_terms_new",
						"value": h.bppTermsURL,
					},
				},
			},
		}
	}
	return descriptor
}

func (h *SearchHandler) extractCategoryID(req *models.ONDCRequest) string {
	intent, ok := req.Message["intent"].(map[string]interface{})
	if !ok {
		return ""
	}
	category, ok := intent["category"].(map[string]interface{})
	if !ok {
		return ""
	}
	categoryID, ok := category["id"].(string)
	if !ok {
		return ""
	}
	return categoryID
}

func (h *SearchHandler) calculateAbsoluteDuration(eta *time.Time, baseTime time.Time) string {
	if eta == nil {
		return "PT0S"
	}
	// Calculate absolute duration from base time to ETA
	duration := eta.Sub(baseTime)
	if duration < 0 {
		return "PT0S"
	}
	// Convert to ISO8601 duration format (PT15M, PT30S, etc.)
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

func (h *SearchHandler) buildItemName(duration string) string {
	return fmt.Sprintf("P2P Delivery (%s)", duration)
}

func (h *SearchHandler) buildIdempotencyKey(transactionID, messageID string) string {
	return "search:" + transactionID + ":" + messageID
}

func (h *SearchHandler) respondACK(c *gin.Context, response interface{}) {
	c.JSON(http.StatusOK, response)
}

func (h *SearchHandler) respondNACK(c *gin.Context, err *errors.DomainError) {
	ctx := c.Request.Context()
	traceID := utils.ExtractTraceID(utils.EnsureTraceparent(c.GetHeader("traceparent")))

	// Extract context from request if available
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
		h.logRequestResponse(ctx, req, response, nil, "", clientID, traceID)
	}

	c.JSON(errors.GetHTTPStatus(err), response)
}

func (h *SearchHandler) logRequestResponse(ctx context.Context, req *models.ONDCRequest, ackResponse interface{}, callbackPayload interface{}, searchID, clientID, traceID string) {
	if h.auditService == nil {
		return
	}

	reqPayload := h.toMap(req)
	ackPayload := h.toMap(ackResponse)
	callbackPayloadMap := h.toMap(callbackPayload)

	params := &audit.RequestResponseLogParams{
		TransactionID:   req.Context.TransactionID,
		MessageID:       req.Context.MessageID,
		Action:          "search",
		RequestPayload:  reqPayload,
		ACKPayload:      ackPayload,
		CallbackPayload: callbackPayloadMap,
		TraceID:         traceID,
		ClientID:        clientID,
		SearchID:        searchID,
	}

	_ = h.auditService.LogRequestResponse(ctx, params)
}

func (h *SearchHandler) logCallbackDelivery(ctx context.Context, transactionID, callbackURL string, attemptNo int, status, errorMsg string) {
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

func (h *SearchHandler) toMap(v interface{}) map[string]interface{} {
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
