package storage

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

const (
	// Redis key prefix for ONDC fulfillment contacts
	fulfillmentContactsKeyPrefix = "ondc_fulfillment_contacts"
)

// FulfillmentContactsStorageService handles storage and retrieval of fulfillment contacts
type FulfillmentContactsStorageService interface {
	// StoreFulfillmentContacts stores fulfillment contacts for a transaction
	StoreFulfillmentContacts(ctx context.Context, transactionID string, contacts map[string]interface{}) error

	// GetFulfillmentContacts retrieves fulfillment contacts for a transaction
	GetFulfillmentContacts(ctx context.Context, transactionID string) (map[string]interface{}, error)

	// DeleteFulfillmentContacts removes fulfillment contacts for a transaction
	DeleteFulfillmentContacts(ctx context.Context, transactionID string) error
}

// FulfillmentContactsService implements FulfillmentContactsStorageService using Redis cache
type FulfillmentContactsService struct {
	cache  CacheService
	logger *zap.Logger
	ttl    time.Duration
}

// NewFulfillmentContactsStorageService creates a new fulfillment contacts storage service
func NewFulfillmentContactsStorageService(cache CacheService, ttl time.Duration, logger *zap.Logger) FulfillmentContactsStorageService {
	return &FulfillmentContactsService{
		cache:  cache,
		logger: logger,
		ttl:    ttl,
	}
}

// buildKey builds the Redis key for fulfillment contacts storage
func (s *FulfillmentContactsService) buildKey(transactionID string) string {
	return fmt.Sprintf("%s:%s", fulfillmentContactsKeyPrefix, transactionID)
}

// StoreFulfillmentContacts stores fulfillment contacts in Redis
func (s *FulfillmentContactsService) StoreFulfillmentContacts(ctx context.Context, transactionID string, contacts map[string]interface{}) error {
	if transactionID == "" {
		return fmt.Errorf("transaction_id is required")
	}

	if len(contacts) == 0 {
		// Contacts are optional per ONDC spec, but if provided, we store them
		s.logger.Debug("fulfillment contacts are empty, skipping storage", zap.String("transaction_id", transactionID))
		return nil
	}

	key := s.buildKey(transactionID)

	if err := s.cache.Set(ctx, key, contacts); err != nil {
		s.logger.Error("failed to store fulfillment contacts", zap.Error(err), zap.String("transaction_id", transactionID))
		return fmt.Errorf("failed to store fulfillment contacts: %w", err)
	}

	s.logger.Info("fulfillment contacts stored successfully", zap.String("transaction_id", transactionID), zap.String("key", key))
	return nil
}

// GetFulfillmentContacts retrieves fulfillment contacts from Redis
func (s *FulfillmentContactsService) GetFulfillmentContacts(ctx context.Context, transactionID string) (map[string]interface{}, error) {
	if transactionID == "" {
		return nil, fmt.Errorf("transaction_id is required")
	}

	key := s.buildKey(transactionID)
	var contacts map[string]interface{}

	exists, err := s.cache.Get(ctx, key, &contacts)
	if err != nil {
		return nil, fmt.Errorf("failed to get fulfillment contacts: %w", err)
	}

	if !exists {
		return nil, nil // Contacts not found (not an error, contacts are optional)
	}

	return contacts, nil
}

// DeleteFulfillmentContacts removes fulfillment contacts from Redis
func (s *FulfillmentContactsService) DeleteFulfillmentContacts(ctx context.Context, transactionID string) error {
	if transactionID == "" {
		return fmt.Errorf("transaction_id is required")
	}

	key := s.buildKey(transactionID)
	if err := s.cache.Delete(ctx, key); err != nil {
		s.logger.Error("failed to delete fulfillment contacts", zap.Error(err), zap.String("transaction_id", transactionID))
		return fmt.Errorf("failed to delete fulfillment contacts: %w", err)
	}

	s.logger.Info("fulfillment contacts deleted successfully", zap.String("transaction_id", transactionID), zap.String("key", key))
	return nil
}
