package event

import (
	"context"
	"encoding/json"
	"time"

	"uois-gateway/internal/config"
	"uois-gateway/pkg/errors"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// EventEnvelope represents the structure of events stored in Redis streams
// Used for business ID filtering before ACK to ensure event isolation
// NOTE: UOIS Gateway uses business IDs (search_id, quote_id) for correlation, NOT correlation_id
// correlation_id is WebSocket Gateway responsibility and never enters UOIS Gateway
type EventEnvelope struct {
	EventType string          `json:"event_type"`
	SearchID  string          `json:"search_id,omitempty"` // Business ID for /search and /init correlation
	QuoteID   string          `json:"quote_id,omitempty"`  // Business ID for /confirm correlation
	Payload   json.RawMessage `json:"-"`                   // Full event data (not used currently)
}

// StreamConsumerClient interface for Redis stream consumer operations
type StreamConsumerClient interface {
	XReadGroup(ctx context.Context, args *redis.XReadGroupArgs) *redis.XStreamSliceCmd
	XAck(ctx context.Context, stream, group, id string) *redis.IntCmd
}

// EventIdempotencyService interface for event-level idempotency
type EventIdempotencyService interface {
	CheckAndStore(ctx context.Context, eventID string) (bool, error)
}

// Consumer consumes events from Redis streams with business ID filtering
//
// CRITICAL: This consumer implements event isolation to prevent consuming
// events meant for other requests. Events are only ACKed if they match the
// provided business ID (search_id or quote_id).
//
// ACK Strategy:
//   - Events are ACKed only after business ID match is confirmed
//   - If business ID doesn't match, event is skipped (not ACKed) and consumer
//     continues waiting for the next event
//   - This means consumption == ownership transfer - once ACKed, the event
//     is considered processed even if the caller crashes
//
// Business ID Matching:
// - Checks business ID fields: search_id (for /search and /init), quote_id (for /confirm)
// - UOIS Gateway uses business IDs for event correlation, NOT correlation_id
// - correlation_id is WebSocket Gateway responsibility and never enters UOIS Gateway
// - Empty business ID accepts all events (not recommended for production)
//
// For stronger isolation, consider using Option A: stream keys with business ID
// (e.g., "stream.uois.quote_created:<quote_id>") instead of filtering.
type Consumer struct {
	rdb                     StreamConsumerClient
	config                  config.StreamsConfig
	logger                  *zap.Logger
	eventIdempotencyService EventIdempotencyService
}

// NewConsumer creates a new event consumer
func NewConsumer(rdb StreamConsumerClient, cfg config.StreamsConfig, logger *zap.Logger) *Consumer {
	return &Consumer{
		rdb:    rdb,
		config: cfg,
		logger: logger,
	}
}

// NewConsumerWithIdempotency creates a new event consumer with event idempotency support
func NewConsumerWithIdempotency(rdb StreamConsumerClient, cfg config.StreamsConfig, eventIdempotencyService EventIdempotencyService, logger *zap.Logger) *Consumer {
	return &Consumer{
		rdb:                     rdb,
		config:                  cfg,
		logger:                  logger,
		eventIdempotencyService: eventIdempotencyService,
	}
}

// ConsumeEvent consumes an event from a Redis stream
// CRITICAL: Only ACKs events that match the provided businessID (search_id or quote_id)
// This prevents consuming events meant for other requests
// NOTE: businessID parameter receives business IDs (search_id, quote_id), NOT correlation_id
// correlation_id is WebSocket Gateway responsibility and never enters UOIS Gateway
func (c *Consumer) ConsumeEvent(ctx context.Context, stream, consumerGroup, businessID string, timeout time.Duration) (interface{}, error) {
	args := &redis.XReadGroupArgs{
		Group:    consumerGroup,
		Consumer: c.config.ConsumerID,
		Streams:  []string{stream, ">"},
		Count:    1,
		Block:    timeout,
	}

	streams, err := c.rdb.XReadGroup(ctx, args).Result()
	if err == redis.Nil {
		return nil, nil // No events available
	}
	if err != nil {
		return nil, errors.WrapDomainError(err, 65011, "event consumption failed", "redis error")
	}

	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, nil
	}

	msg := streams[0].Messages[0]
	dataStr, ok := msg.Values["data"].(string)
	if !ok {
		return nil, errors.NewDomainError(65020, "event deserialization failed", "invalid event data")
	}

	// Parse event to check business ID before ACK
	var envelope EventEnvelope
	if err := json.Unmarshal([]byte(dataStr), &envelope); err != nil {
		return nil, errors.WrapDomainError(err, 65020, "event deserialization failed", "failed to unmarshal event")
	}

	// CRITICAL: Filter by business ID before ACK
	// Check business ID fields (search_id, quote_id) for event correlation
	if !c.matchesBusinessID(&envelope, businessID) {
		// Event doesn't match - do NOT ACK, return nil to continue waiting
		c.logger.Debug("event business ID mismatch, skipping",
			zap.String("stream", stream),
			zap.String("expected_business_id", businessID),
			zap.String("event_id", msg.ID),
		)
		return nil, nil
	}

	// Event matches business ID - unmarshal full event and ACK
	var event interface{}
	if err := json.Unmarshal([]byte(dataStr), &event); err != nil {
		return nil, errors.WrapDomainError(err, 65020, "event deserialization failed", "failed to unmarshal event")
	}

	// Acknowledge the message only after business ID match confirmed
	if err := c.rdb.XAck(ctx, stream, consumerGroup, msg.ID).Err(); err != nil {
		c.logger.Warn("failed to ack event", zap.Error(err), zap.String("stream", stream), zap.String("id", msg.ID))
	}

	return event, nil
}

// matchesBusinessID checks if an event matches the expected business ID
//
// This function implements event isolation by checking business ID fields
// (search_id, quote_id) in the event. UOIS Gateway uses business IDs for
// event correlation, NOT correlation_id (which is WebSocket Gateway responsibility).
//
// IMPORTANT: For production systems with high concurrency, consider using
// Option A (stream keys with business ID) instead of filtering:
//   - Stream: "stream.uois.quote_created:<quote_id>"
//   - This provides stronger isolation and avoids head-of-line blocking
//
// Returns:
//   - true if business ID matches (event should be consumed and ACKed)
//   - false if business ID doesn't match (event should be skipped)
//   - true if expectedBusinessID is empty (accepts all events - not recommended)
func (c *Consumer) matchesBusinessID(envelope *EventEnvelope, expectedBusinessID string) bool {
	if expectedBusinessID == "" {
		// If no business ID expected, accept all events (not recommended but allowed)
		return true
	}

	// Check search_id field (used in /search and /init events)
	if envelope.SearchID != "" && envelope.SearchID == expectedBusinessID {
		return true
	}

	// Check quote_id field (used in /confirm events)
	if envelope.QuoteID != "" && envelope.QuoteID == expectedBusinessID {
		return true
	}

	return false
}
