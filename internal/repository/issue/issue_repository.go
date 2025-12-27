package issue

import (
	"context"
	"encoding/json"
	"time"

	"uois-gateway/internal/config"
	"uois-gateway/internal/models"
	"uois-gateway/pkg/errors"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisClient interface for Redis operations
type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
}

// Repository handles issue storage and retrieval
type Repository struct {
	redis  RedisClient
	config config.Config
	logger *zap.Logger
}

// NewRepository creates a new issue repository
func NewRepository(rdb RedisClient, cfg config.Config, logger *zap.Logger) *Repository {
	return &Repository{
		redis:  rdb,
		config: cfg,
		logger: logger,
	}
}

// StoreIssue stores an issue in Redis
func (r *Repository) StoreIssue(ctx context.Context, issue *models.Issue) error {
	key := r.buildIssueKey(issue.IssueID)

	val, err := json.Marshal(issue)
	if err != nil {
		return errors.WrapDomainError(err, 65020, "issue serialization failed", "failed to marshal issue")
	}

	ttl := time.Duration(r.config.TTL.IssueStorage) * time.Second
	if err := r.redis.Set(ctx, key, val, ttl).Err(); err != nil {
		return errors.WrapDomainError(err, 65011, "issue storage failed", "redis error")
	}

	r.logger.Debug("issue stored", zap.String("issue_id", issue.IssueID))
	return nil
}

// GetIssue retrieves an issue by issue_id
func (r *Repository) GetIssue(ctx context.Context, issueID string) (*models.Issue, error) {
	key := r.buildIssueKey(issueID)

	val, err := r.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, errors.NewDomainError(65006, "issue not found", "issue_id not found")
	}
	if err != nil {
		return nil, errors.WrapDomainError(err, 65011, "issue retrieval failed", "redis error")
	}

	var issue models.Issue
	if err := json.Unmarshal([]byte(val), &issue); err != nil {
		return nil, errors.WrapDomainError(err, 65020, "issue deserialization failed", "invalid stored issue")
	}

	return &issue, nil
}

// StoreZendeskReference stores a Zendesk ticket reference
func (r *Repository) StoreZendeskReference(ctx context.Context, zendeskTicketID, issueID string) error {
	key := r.buildZendeskReferenceKey(zendeskTicketID)

	ttl := time.Duration(r.config.TTL.IssueStorage) * time.Second
	if err := r.redis.Set(ctx, key, issueID, ttl).Err(); err != nil {
		return errors.WrapDomainError(err, 65011, "zendesk reference storage failed", "redis error")
	}

	r.logger.Debug("zendesk reference stored", zap.String("ticket_id", zendeskTicketID), zap.String("issue_id", issueID))
	return nil
}

// GetIssueIDByZendeskTicket retrieves issue_id by Zendesk ticket ID
func (r *Repository) GetIssueIDByZendeskTicket(ctx context.Context, zendeskTicketID string) (string, error) {
	key := r.buildZendeskReferenceKey(zendeskTicketID)

	val, err := r.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", errors.NewDomainError(65006, "zendesk reference not found", "ticket_id not found")
	}
	if err != nil {
		return "", errors.WrapDomainError(err, 65011, "zendesk reference retrieval failed", "redis error")
	}

	return val, nil
}

func (r *Repository) buildIssueKey(issueID string) string {
	return r.config.Redis.KeyPrefix + ":issue:" + issueID
}

func (r *Repository) buildZendeskReferenceKey(ticketID string) string {
	return r.config.Redis.KeyPrefix + ":zendesk_ticket:" + ticketID
}
