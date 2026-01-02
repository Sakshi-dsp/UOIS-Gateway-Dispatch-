package callback

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"uois-gateway/internal/config"
	"uois-gateway/internal/services/audit"
	"uois-gateway/pkg/errors"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// CallbackSender interface for sending callbacks (to avoid circular dependency)
type CallbackSender interface {
	SendCallbackDirect(ctx context.Context, callbackURL string, payload interface{}) error
}

// RetryService handles callback retries with exponential backoff and DLQ
type RetryService struct {
	callbackSender CallbackSender
	config         config.RetryConfig
	callbackConfig config.CallbackConfig
	redis          *redis.Client
	auditService   AuditService
	logger         *zap.Logger
}

// AuditService interface for logging callback delivery attempts
type AuditService interface {
	LogCallbackDelivery(ctx context.Context, req *audit.CallbackDeliveryLogParams) error
}

// NewRetryService creates a new retry service wrapper around callback sender
func NewRetryService(
	callbackSender CallbackSender,
	retryConfig config.RetryConfig,
	callbackConfig config.CallbackConfig,
	redis *redis.Client,
	auditService AuditService,
	logger *zap.Logger,
) *RetryService {
	return &RetryService{
		callbackSender: callbackSender,
		config:         retryConfig,
		callbackConfig: callbackConfig,
		redis:          redis,
		auditService:   auditService,
		logger:         logger,
	}
}

// SendCallbackWithRetry sends a callback with exponential backoff retry and DLQ support
func (r *RetryService) SendCallbackWithRetry(
	ctx context.Context,
	callbackURL string,
	payload interface{},
	requestID string,
	ttlSeconds int,
) error {
	var lastErr error
	attempt := 0

	for attempt < r.config.CallbackMaxRetries {
		attempt++
		err := r.callbackSender.SendCallbackDirect(ctx, callbackURL, payload)

		// Log attempt
		status := "success"
		errorMsg := ""
		if err != nil {
			status = "failed"
			errorMsg = err.Error()
			lastErr = err
		}

		// Log to audit service
		if r.auditService != nil {
			logCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = r.auditService.LogCallbackDelivery(logCtx, &audit.CallbackDeliveryLogParams{
				RequestID:   requestID,
				CallbackURL: callbackURL,
				AttemptNo:   attempt,
				Status:      status,
				Error:       errorMsg,
			})
			cancel()
		}

		if err == nil {
			return nil
		}

		// Check if we should retry
		if attempt >= r.config.CallbackMaxRetries {
			break
		}

		// Calculate backoff duration
		backoffDuration := r.calculateBackoff(attempt, ttlSeconds)
		if backoffDuration <= 0 {
			r.logger.Warn("backoff duration is zero or negative, stopping retries",
				zap.Int("attempt", attempt),
				zap.String("request_id", requestID),
			)
			break
		}

		r.logger.Info("callback failed, retrying",
			zap.Int("attempt", attempt),
			zap.Duration("backoff", backoffDuration),
			zap.String("request_id", requestID),
			zap.Error(err),
		)

		// Wait before retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoffDuration):
			// Continue to next retry
		}
	}

	// All retries exhausted, send to DLQ if enabled
	if r.callbackConfig.DLQEnabled && r.callbackConfig.DLQStream != "" {
		if err := r.sendToDLQ(ctx, callbackURL, payload, requestID, lastErr); err != nil {
			r.logger.Error("failed to send callback to DLQ",
				zap.String("request_id", requestID),
				zap.Error(err),
			)
		}
	}

	return errors.WrapDomainError(
		lastErr,
		65020,
		"callback delivery failed after retries",
		fmt.Sprintf("failed after %d attempts", attempt),
	)
}

// calculateBackoff calculates exponential backoff duration bounded by TTL
func (r *RetryService) calculateBackoff(attempt int, ttlSeconds int) time.Duration {
	if attempt <= 0 || len(r.config.CallbackBackoff) == 0 {
		return 0
	}

	// Use configured backoff if available
	backoffIndex := attempt - 1
	if backoffIndex < len(r.config.CallbackBackoff) {
		backoffSeconds := r.config.CallbackBackoff[backoffIndex]
		backoffDuration := time.Duration(backoffSeconds) * time.Second

		// Ensure backoff doesn't exceed remaining TTL
		if ttlSeconds > 0 {
			remainingTTL := time.Duration(ttlSeconds) * time.Second
			if backoffDuration > remainingTTL {
				backoffDuration = remainingTTL
			}
		}

		return backoffDuration
	}

	// Fallback: exponential backoff (2^attempt seconds, max 30s)
	backoffSeconds := 1 << attempt
	if backoffSeconds > 30 {
		backoffSeconds = 30
	}

	backoffDuration := time.Duration(backoffSeconds) * time.Second
	if ttlSeconds > 0 {
		remainingTTL := time.Duration(ttlSeconds) * time.Second
		if backoffDuration > remainingTTL {
			backoffDuration = remainingTTL
		}
	}

	return backoffDuration
}

// sendToDLQ sends failed callback to Dead Letter Queue
func (r *RetryService) sendToDLQ(
	ctx context.Context,
	callbackURL string,
	payload interface{},
	requestID string,
	lastErr error,
) error {
	dlqPayload := map[string]interface{}{
		"request_id":   requestID,
		"callback_url": callbackURL,
		"payload":       payload,
		"error":         lastErr.Error(),
		"timestamp":     time.Now().Unix(),
		"retries":       r.config.CallbackMaxRetries,
	}

	payloadJSON, err := json.Marshal(dlqPayload)
	if err != nil {
		return errors.WrapDomainError(err, 65020, "dlq serialization failed", "failed to marshal dlq payload")
	}

	args := &redis.XAddArgs{
		Stream: r.callbackConfig.DLQStream,
		Values: map[string]interface{}{
			"data":      string(payloadJSON),
			"request_id": requestID,
			"error":     lastErr.Error(),
		},
	}

	if err := r.redis.XAdd(ctx, args).Err(); err != nil {
		return errors.WrapDomainError(err, 65020, "dlq publish failed", "redis error")
	}

	r.logger.Info("callback sent to DLQ",
		zap.String("request_id", requestID),
		zap.String("dlq_stream", r.callbackConfig.DLQStream),
	)

	return nil
}

