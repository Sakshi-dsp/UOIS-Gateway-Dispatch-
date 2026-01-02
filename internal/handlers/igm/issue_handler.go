package igm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"uois-gateway/internal/models"
	"uois-gateway/internal/utils"
	"uois-gateway/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// IssueHandler handles /issue ONDC requests
type IssueHandler struct {
	issueRepository    IssueRepository
	callbackService    CallbackService
	idempotencyService IdempotencyService
	groService         GROService
	bppID              string
	bppURI             string
	logger             *zap.Logger
}

// NewIssueHandler creates a new issue handler
func NewIssueHandler(
	issueRepository IssueRepository,
	callbackService CallbackService,
	idempotencyService IdempotencyService,
	groService GROService,
	bppID string,
	bppURI string,
	logger *zap.Logger,
) *IssueHandler {
	return &IssueHandler{
		issueRepository:    issueRepository,
		callbackService:    callbackService,
		idempotencyService: idempotencyService,
		groService:         groService,
		bppID:              bppID,
		bppURI:             bppURI,
		logger:             logger,
	}
}

// HandleIssue handles POST /issue requests
func (h *IssueHandler) HandleIssue(c *gin.Context) {
	ctx := c.Request.Context()

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

	issue, err := h.extractIssueData(&req)
	if err != nil {
		h.respondNACK(c, errors.NewDomainError(65001, "invalid issue data", err.Error()))
		return
	}

	// Client ID available from middleware if needed
	_, _ = c.Get("client")

	// Set timestamps before validation (validation requires CreatedAt)
	issue.CreatedAt = time.Now()
	issue.UpdatedAt = time.Now()

	if err := issue.Validate(); err != nil {
		h.respondNACK(c, errors.NewDomainError(65001, "invalid issue", err.Error()))
		return
	}

	if err := h.issueRepository.StoreIssue(ctx, issue); err != nil {
		h.logger.Error("failed to store issue", zap.Error(err), zap.String("trace_id", traceID))
		h.respondNACK(c, errors.NewDomainError(65020, "internal error", "failed to store issue"))
		return
	}

	response := h.composeIssueResponse()
	responseBytes, _ := json.Marshal(response)
	_ = h.idempotencyService.StoreIdempotency(ctx, idempotencyKey, responseBytes, 24*time.Hour)

	go h.sendIssueCallback(ctx, &req, issue, traceID)

	h.respondACK(c, response)
}

func (h *IssueHandler) extractIssueData(req *models.ONDCRequest) (*models.Issue, error) {
	message, ok := req.Message["issue"].(map[string]interface{})
	if !ok {
		return nil, errors.NewDomainError(65001, "invalid request", "missing issue")
	}

	issueID, _ := message["id"].(string)
	if issueID == "" {
		issueID = uuid.New().String()
	}

	issueTypeStr, _ := message["issue_type"].(string)
	issueType := models.IssueType(issueTypeStr)
	if !issueType.IsValid() {
		issueType = models.IssueTypeIssue
	}

	category, _ := message["category"].(string)
	subCategory, _ := message["sub_category"].(string)
	description, _ := message["description"].(string)

	orderDetails, _ := message["order_details"].(map[string]interface{})
	complainantInfo, _ := message["complainant_info"].(map[string]interface{})

	orderID := ""
	if orderDetails != nil {
		if od, ok := orderDetails["order_id"].(string); ok {
			orderID = od
		}
	}

	return &models.Issue{
		IssueID:         issueID,
		TransactionID:   req.Context.TransactionID,
		OrderID:         orderID,
		IssueType:       issueType,
		Status:          models.IssueStatusOpen,
		Category:        category,
		SubCategory:     subCategory,
		Description:     description,
		OrderDetails:    orderDetails,
		ComplainantInfo: complainantInfo,
		FullONDCPayload: message,
	}, nil
}

func (h *IssueHandler) composeIssueResponse() models.ONDCACKResponse {
	return models.ONDCACKResponse{
		Message: models.ONDCACKMessage{
			Ack: models.ONDCACKStatus{
				Status: "ACK",
			},
		},
	}
}

func (h *IssueHandler) sendIssueCallback(ctx context.Context, req *models.ONDCRequest, issue *models.Issue, traceID string) {
	gro, _ := h.groService.GetGRODetails(ctx, issue.IssueType)

	callbackPayload := h.buildOnIssueCallback(req, issue, gro)
	callbackURL := h.buildCallbackURL(req.Context.BapURI, "on_issue")

	if err := h.callbackService.SendCallback(ctx, callbackURL, callbackPayload); err != nil {
		h.logger.Error("failed to send issue callback", zap.Error(err), zap.String("trace_id", traceID))
	}
}

func (h *IssueHandler) buildOnIssueCallback(req *models.ONDCRequest, issue *models.Issue, gro *models.GRO) models.ONDCResponse {
	issueActions := []models.IssueAction{
		{
			ActionType: "RESPOND",
			UpdatedAt:  time.Now(),
		},
	}

	resolutionProvider := &models.ResolutionProvider{
		GRO: gro,
	}

	return models.ONDCResponse{
		Context: req.Context,
		Message: map[string]interface{}{
			"issue": map[string]interface{}{
				"id":                  issue.IssueID,
				"status":              issue.Status.String(),
				"issue_actions":       issueActions,
				"resolution_provider": resolutionProvider,
			},
		},
	}
}

func (h *IssueHandler) buildCallbackURL(bapURI, action string) string {
	if bapURI == "" {
		return ""
	}
	return bapURI + "/" + action
}

func (h *IssueHandler) buildIdempotencyKey(transactionID, messageID string) string {
	return "issue:" + transactionID + ":" + messageID
}

func (h *IssueHandler) respondACK(c *gin.Context, response interface{}) {
	c.JSON(200, response)
}

// HandleOnIssue handles POST /on_issue callbacks from Buyer NPs
func (h *IssueHandler) HandleOnIssue(c *gin.Context) {
	ctx := c.Request.Context()

	traceparent := utils.EnsureTraceparent(c.GetHeader("traceparent"))
	traceID := utils.ExtractTraceID(traceparent)

	var req models.ONDCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid callback request", zap.Error(err), zap.String("trace_id", traceID))
		h.respondNACK(c, errors.NewDomainError(65001, "invalid request", err.Error()))
		return
	}

	c.Set("ondc_request", &req)

	if err := req.Context.Validate(); err != nil {
		h.logger.Error("invalid context", zap.Error(err), zap.String("trace_id", traceID))
		h.respondNACK(c, errors.NewDomainError(65001, "invalid context", err.Error()))
		return
	}

	issueID, issueActions, err := h.extractOnIssueCallback(&req)
	if err != nil {
		h.respondNACK(c, errors.NewDomainError(65001, "invalid callback data", err.Error()))
		return
	}

	issue, err := h.issueRepository.GetIssue(ctx, issueID)
	if err != nil {
		h.logger.Error("issue not found", zap.Error(err), zap.String("issue_id", issueID))
		h.respondNACK(c, errors.NewDomainError(65006, "issue not found", "issue_id not found"))
		return
	}

	issue.IssueActions = issueActions
	issue.UpdatedAt = time.Now()

	if err := h.issueRepository.StoreIssue(ctx, issue); err != nil {
		h.logger.Error("failed to update issue", zap.Error(err), zap.String("trace_id", traceID))
		h.respondNACK(c, errors.NewDomainError(65020, "internal error", "failed to update issue"))
		return
	}

	response := h.composeIssueResponse()
	h.respondACK(c, response)
}

func (h *IssueHandler) extractOnIssueCallback(req *models.ONDCRequest) (string, []models.IssueAction, error) {
	message, ok := req.Message["issue"].(map[string]interface{})
	if !ok {
		return "", nil, errors.NewDomainError(65001, "invalid request", "missing issue")
	}

	issueID, _ := message["id"].(string)
	if issueID == "" {
		return "", nil, errors.NewDomainError(65001, "invalid request", "missing issue_id")
	}

	issueActionsMap, _ := message["issue_actions"].(map[string]interface{})
	if issueActionsMap == nil {
		return issueID, []models.IssueAction{}, nil
	}

	respondentActions, _ := issueActionsMap["respondent_actions"].([]interface{})

	actions := []models.IssueAction{}
	for _, ra := range respondentActions {
		raMap, ok := ra.(map[string]interface{})
		if !ok {
			continue
		}

		respondentAction, _ := raMap["respondent_action"].(string)
		shortDesc, _ := raMap["short_desc"].(string)
		updatedAtStr, _ := raMap["updated_at"].(string)

		var updatedAt time.Time
		if updatedAtStr != "" {
			var err error
			updatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
			if err != nil {
				updatedAt = time.Now()
			}
		} else {
			updatedAt = time.Now()
		}

		actionType := h.mapRespondentActionToActionType(respondentAction)
		actions = append(actions, models.IssueAction{
			ActionType: actionType,
			ShortDesc:  shortDesc,
			UpdatedAt:  updatedAt,
		})
	}

	return issueID, actions, nil
}

func (h *IssueHandler) mapRespondentActionToActionType(respondentAction string) string {
	switch respondentAction {
	case "PROCESSING", "CASCADED", "RESOLVED", "NEED-MORE-INFO":
		return "RESPOND"
	default:
		return "RESPOND"
	}
}

func (h *IssueHandler) respondNACK(c *gin.Context, err *errors.DomainError) {
	var ctx models.ONDCContext
	if req, ok := c.Get("ondc_request"); ok {
		if ondcReq, ok := req.(*models.ONDCRequest); ok {
			ctx = ondcReq.Context
		}
	}

	c.JSON(errors.GetHTTPStatus(err), models.ONDCResponse{
		Context: ctx,
		Error: &models.ONDCError{
			Type:    "DOMAIN_ERROR",
			Code:    fmt.Sprintf("%d", err.Code),
			Message: map[string]string{"en": err.Message},
		},
	})
}
