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
	"go.uber.org/zap"
)

// IssueStatusHandler handles /issue_status ONDC requests
type IssueStatusHandler struct {
	issueRepository   IssueRepository
	callbackService   CallbackService
	idempotencyService IdempotencyService
	groService        GROService
	bppID             string
	bppURI            string
	logger            *zap.Logger
}

// NewIssueStatusHandler creates a new issue status handler
func NewIssueStatusHandler(
	issueRepository IssueRepository,
	callbackService CallbackService,
	idempotencyService IdempotencyService,
	groService GROService,
	bppID string,
	bppURI string,
	logger *zap.Logger,
) *IssueStatusHandler {
	return &IssueStatusHandler{
		issueRepository:   issueRepository,
		callbackService:   callbackService,
		idempotencyService: idempotencyService,
		groService:        groService,
		bppID:             bppID,
		bppURI:            bppURI,
		logger:            logger,
	}
}

// HandleIssueStatus handles POST /issue_status requests
func (h *IssueStatusHandler) HandleIssueStatus(c *gin.Context) {
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

	issueID, err := h.extractIssueID(&req)
	if err != nil {
		h.respondNACK(c, errors.NewDomainError(65001, "invalid issue_id", err.Error()))
		return
	}

	issue, err := h.issueRepository.GetIssue(ctx, issueID)
	if err != nil {
		h.logger.Error("issue not found", zap.Error(err), zap.String("issue_id", issueID))
		h.respondNACK(c, errors.NewDomainError(65006, "issue not found", "issue_id not found"))
		return
	}

	gro, _ := h.groService.GetGRODetails(ctx, issue.IssueType)

	response := h.composeIssueStatusResponse()
	responseBytes, _ := json.Marshal(response)
	_ = h.idempotencyService.StoreIdempotency(ctx, idempotencyKey, responseBytes, 24*time.Hour)

	go h.sendIssueStatusCallback(ctx, &req, issue, gro, traceID)

	h.respondACK(c, response)
}

func (h *IssueStatusHandler) extractIssueID(req *models.ONDCRequest) (string, error) {
	message, ok := req.Message["issue_id"].(string)
	if ok && message != "" {
		return message, nil
	}

	issueMap, ok := req.Message["issue"].(map[string]interface{})
	if ok {
		if id, ok := issueMap["id"].(string); ok && id != "" {
			return id, nil
		}
	}

	return "", errors.NewDomainError(65001, "invalid request", "missing issue_id")
}

func (h *IssueStatusHandler) composeIssueStatusResponse() models.ONDCACKResponse {
	return models.ONDCACKResponse{
		Message: models.ONDCACKMessage{
			Ack: models.ONDCACKStatus{
				Status: "ACK",
			},
		},
	}
}

func (h *IssueStatusHandler) sendIssueStatusCallback(ctx context.Context, req *models.ONDCRequest, issue *models.Issue, gro *models.GRO, traceID string) {
	callbackPayload := h.buildOnIssueStatusCallback(req, issue, gro)
	callbackURL := h.buildCallbackURL(req.Context.BapURI, "on_issue_status")

	if err := h.callbackService.SendCallback(ctx, callbackURL, callbackPayload); err != nil {
		h.logger.Error("failed to send issue status callback", zap.Error(err), zap.String("trace_id", traceID))
	}
}

func (h *IssueStatusHandler) buildOnIssueStatusCallback(req *models.ONDCRequest, issue *models.Issue, gro *models.GRO) models.ONDCResponse {
	issueActions := []models.IssueAction{}
	if len(issue.IssueActions) > 0 {
		issueActions = issue.IssueActions
	} else {
		issueActions = []models.IssueAction{
			{
				ActionType: "RESPOND",
				UpdatedAt:  time.Now(),
			},
		}
	}

	resolutionProvider := &models.ResolutionProvider{
		GRO: gro,
	}

	if issue.ResolutionProvider != nil {
		resolutionProvider = issue.ResolutionProvider
	}

	response := map[string]interface{}{
		"id":                issue.IssueID,
		"status":            issue.Status.String(),
		"issue_actions":     issueActions,
		"resolution_provider": resolutionProvider,
	}

	if issue.FinancialResolution != nil {
		response["financial_resolution"] = issue.FinancialResolution
	}

	return models.ONDCResponse{
		Context: req.Context,
		Message: map[string]interface{}{
			"issue": response,
		},
	}
}

func (h *IssueStatusHandler) buildCallbackURL(bapURI, action string) string {
	if bapURI == "" {
		return ""
	}
	return bapURI + "/" + action
}

func (h *IssueStatusHandler) buildIdempotencyKey(transactionID, messageID string) string {
	return "issue_status:" + transactionID + ":" + messageID
}

func (h *IssueStatusHandler) respondACK(c *gin.Context, response interface{}) {
	c.JSON(200, response)
}

func (h *IssueStatusHandler) respondNACK(c *gin.Context, err *errors.DomainError) {
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

