package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// BillingStorageService handles storage and retrieval of billing information
type BillingStorageService interface {
	// StoreBilling stores billing information for a transaction
	StoreBilling(ctx context.Context, transactionID string, billing map[string]interface{}) error

	// GetBilling retrieves billing information for a transaction
	GetBilling(ctx context.Context, transactionID string) (map[string]interface{}, error)

	// DeleteBilling removes billing information for a transaction
	DeleteBilling(ctx context.Context, transactionID string) error
}

// CacheService interface for cache operations (matches cache.Service interface)
type CacheService interface {
	Get(ctx context.Context, key string, dest interface{}) (bool, error)
	Set(ctx context.Context, key string, value interface{}) error
	Delete(ctx context.Context, key string) error
}

// Service implements BillingStorageService using Redis cache
type Service struct {
	cache  CacheService
	logger *zap.Logger
	ttl    time.Duration
}

// NewBillingStorageService creates a new billing storage service
func NewBillingStorageService(cache CacheService, ttl time.Duration, logger *zap.Logger) BillingStorageService {
	return &Service{
		cache:  cache,
		logger: logger,
		ttl:    ttl,
	}
}

// buildKey builds the Redis key for billing storage
func (s *Service) buildKey(transactionID string) string {
	return fmt.Sprintf("ondc_billing:%s", transactionID)
}

// StoreBilling stores billing information in Redis
func (s *Service) StoreBilling(ctx context.Context, transactionID string, billing map[string]interface{}) error {
	if transactionID == "" {
		return fmt.Errorf("transaction_id is required")
	}

	if billing == nil || len(billing) == 0 {
		// Billing is optional per ONDC spec, but if provided, we store it
		s.logger.Debug("billing is empty, skipping storage", zap.String("transaction_id", transactionID))
		return nil
	}

	key := s.buildKey(transactionID)

	// Marshal billing to JSON for storage
	billingJSON, err := json.Marshal(billing)
	if err != nil {
		return fmt.Errorf("failed to marshal billing: %w", err)
	}

	// Store in cache with TTL
	var billingMap map[string]interface{}
	if err := json.Unmarshal(billingJSON, &billingMap); err != nil {
		return fmt.Errorf("failed to unmarshal billing: %w", err)
	}

	if err := s.cache.Set(ctx, key, billingMap); err != nil {
		s.logger.Error("failed to store billing", zap.Error(err), zap.String("transaction_id", transactionID))
		return fmt.Errorf("failed to store billing: %w", err)
	}

	s.logger.Info("billing stored successfully", zap.String("transaction_id", transactionID), zap.String("key", key))
	return nil
}

// GetBilling retrieves billing information from Redis
func (s *Service) GetBilling(ctx context.Context, transactionID string) (map[string]interface{}, error) {
	if transactionID == "" {
		return nil, fmt.Errorf("transaction_id is required")
	}

	key := s.buildKey(transactionID)
	var billing map[string]interface{}

	exists, err := s.cache.Get(ctx, key, &billing)
	if err != nil {
		return nil, fmt.Errorf("failed to get billing: %w", err)
	}

	if !exists {
		return nil, nil // Billing not found (not an error, billing is optional)
	}

	return billing, nil
}

// DeleteBilling removes billing information from Redis
func (s *Service) DeleteBilling(ctx context.Context, transactionID string) error {
	if transactionID == "" {
		return fmt.Errorf("transaction_id is required")
	}

	key := s.buildKey(transactionID)
	if err := s.cache.Delete(ctx, key); err != nil {
		s.logger.Error("failed to delete billing", zap.Error(err), zap.String("transaction_id", transactionID))
		return fmt.Errorf("failed to delete billing: %w", err)
	}

	s.logger.Info("billing deleted successfully", zap.String("transaction_id", transactionID), zap.String("key", key))
	return nil
}
