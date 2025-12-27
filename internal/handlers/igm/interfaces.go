package igm

import (
	"context"
	"time"

	"uois-gateway/internal/models"
)

// IssueRepository handles issue storage and retrieval
type IssueRepository interface {
	StoreIssue(ctx context.Context, issue *models.Issue) error
	GetIssue(ctx context.Context, issueID string) (*models.Issue, error)
	StoreZendeskReference(ctx context.Context, zendeskTicketID, issueID string) error
	GetIssueIDByZendeskTicket(ctx context.Context, zendeskTicketID string) (string, error)
}

// CallbackService sends HTTP callbacks to client callback URLs
type CallbackService interface {
	SendCallback(ctx context.Context, callbackURL string, payload interface{}) error
}

// IdempotencyService handles request idempotency checks and storage
type IdempotencyService interface {
	CheckIdempotency(ctx context.Context, key string) ([]byte, bool, error)
	StoreIdempotency(ctx context.Context, key string, responseBytes []byte, ttl time.Duration) error
}

// GROService provides GRO (Grievance Redressal Officer) details
type GROService interface {
	GetGRODetails(ctx context.Context, issueType models.IssueType) (*models.GRO, error)
}

