package eventidempotency

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Service handles event-level idempotency
type Service struct {
	redis  *redis.Client
	logger *zap.Logger
	ttl    time.Duration
}

// NewService creates a new event idempotency service
func NewService(redis *redis.Client, ttl time.Duration, logger *zap.Logger) *Service {
	return &Service{
		redis:  redis,
		logger: logger,
		ttl:    ttl,
	}
}

// CheckAndStore checks if an event ID has been processed and stores it if not
func (s *Service) CheckAndStore(ctx context.Context, eventID string) (bool, error) {
	key := s.buildKey(eventID)

	// Try to set the key with NX (only if not exists)
	// This is atomic and prevents race conditions
	result, err := s.redis.SetNX(ctx, key, "1", s.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("event idempotency check error: %w", err)
	}

	// If result is false, the event was already processed
	if !result {
		return true, nil // Already processed
	}

	return false, nil // New event
}

// IsProcessed checks if an event ID has been processed without storing it
func (s *Service) IsProcessed(ctx context.Context, eventID string) (bool, error) {
	key := s.buildKey(eventID)
	exists, err := s.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("event idempotency check error: %w", err)
	}
	return exists > 0, nil
}

// buildKey builds a Redis key for event idempotency
func (s *Service) buildKey(eventID string) string {
	// Hash the event ID to ensure consistent key format
	hash := sha256.Sum256([]byte(eventID))
	hashStr := hex.EncodeToString(hash[:])
	return fmt.Sprintf("event:idempotency:%s", hashStr)
}

// GenerateEventID generates a unique event ID from event data
func GenerateEventID(stream string, eventData interface{}) string {
	// In a real implementation, extract event_id from eventData
	// For now, create a hash of stream + event data
	data := fmt.Sprintf("%s:%v", stream, eventData)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16]) // Use first 16 bytes as event ID
}

