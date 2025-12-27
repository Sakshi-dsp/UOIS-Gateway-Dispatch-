package order_record

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"uois-gateway/internal/config"
	"uois-gateway/internal/handlers/ondc"
	"uois-gateway/pkg/errors"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisClient interface for Redis operations
type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
}

// Repository handles order record storage and retrieval
type Repository struct {
	redis  RedisClient
	config config.Config
	logger *zap.Logger
}

// NewRepository creates a new order record repository
func NewRepository(rdb RedisClient, cfg config.Config, logger *zap.Logger) *Repository {
	return &Repository{
		redis:  rdb,
		config: cfg,
		logger: logger,
	}
}

// StoreOrderRecord stores an order record
// CRITICAL: Must store all keys (search_id, quote_id, order_id, transaction_id)
// to ensure lookups work immediately after storage.
// Delegates to UpdateOrderRecord to avoid duplication.
func (r *Repository) StoreOrderRecord(ctx context.Context, record *ondc.OrderRecord) error {
	return r.UpdateOrderRecord(ctx, record)
}

// GetOrderRecordBySearchID retrieves order record by search_id
func (r *Repository) GetOrderRecordBySearchID(ctx context.Context, searchID string) (*ondc.OrderRecord, error) {
	key := r.buildKey("search_id", searchID)
	return r.getByKey(ctx, key)
}

// GetOrderRecordByQuoteID retrieves order record by quote_id
func (r *Repository) GetOrderRecordByQuoteID(ctx context.Context, quoteID string) (*ondc.OrderRecord, error) {
	key := r.buildKey("quote_id", quoteID)
	return r.getByKey(ctx, key)
}

// GetOrderRecordByOrderID retrieves order record by client_id + order.id
func (r *Repository) GetOrderRecordByOrderID(ctx context.Context, clientID, orderID string) (*ondc.OrderRecord, error) {
	key := r.buildKey("order_id", fmt.Sprintf("%s:%s", clientID, orderID))
	return r.getByKey(ctx, key)
}

// GetOrderRecordByTransactionID retrieves order record by transaction_id
func (r *Repository) GetOrderRecordByTransactionID(ctx context.Context, transactionID string) (*ondc.OrderRecord, error) {
	key := r.buildKey("transaction_id", transactionID)
	return r.getByKey(ctx, key)
}

// UpdateOrderRecord updates an existing order record
// NOTE: TTL extension on update is acceptable - status/track calls can keep extending lifetime
// NOTE: Atomicity - if Redis crashes mid-loop, partial state may occur
//
//	Optional future hardening: Redis pipeline or Lua script (not required for certification)
func (r *Repository) UpdateOrderRecord(ctx context.Context, record *ondc.OrderRecord) error {
	// Update all possible keys
	keys := []string{
		r.buildKey("search_id", record.SearchID),
		r.buildKey("quote_id", record.QuoteID),
		r.buildKey("order_id", fmt.Sprintf("%s:%s", record.ClientID, record.OrderID)),
		r.buildKey("transaction_id", record.TransactionID),
	}

	val, err := json.Marshal(record)
	if err != nil {
		return errors.WrapDomainError(err, 65020, "order record serialization failed", "failed to marshal record")
	}

	// Use longer TTL for order lifecycle (when order.id exists) vs search/quote TTL
	// Order lifecycle spans hours/days, search/quote TTL is minutes
	// Currently uses OrderMapping TTL (30 days) for all records
	// TODO: Consider adding OrderLifecycle TTL config if different TTL needed for post-confirmation records
	ttl := time.Duration(r.config.TTL.OrderMapping) * time.Second
	if record.OrderID != "" {
		// Future: Use OrderLifecycle TTL when config is added
		// ttl = time.Duration(r.config.TTL.OrderLifecycle) * time.Second
	}

	for _, key := range keys {
		if key != "" {
			if err := r.redis.Set(ctx, key, val, ttl).Err(); err != nil {
				return errors.WrapDomainError(err, 65011, "order record update failed", "redis error")
			}
		}
	}

	// Log successful update (debug level for observability)
	r.logger.Debug("order record updated",
		zap.String("search_id", record.SearchID),
		zap.String("quote_id", record.QuoteID),
		zap.String("order_id", record.OrderID),
		zap.String("transaction_id", record.TransactionID),
		zap.Duration("ttl", ttl),
	)

	return nil
}

func (r *Repository) buildKey(prefix, value string) string {
	if value == "" {
		return ""
	}
	return fmt.Sprintf("%s:order_record:%s:%s", r.config.Redis.KeyPrefix, prefix, value)
}

func (r *Repository) getByKey(ctx context.Context, key string) (*ondc.OrderRecord, error) {
	if key == "" {
		return nil, errors.NewDomainError(65006, "order not found", "invalid key")
	}

	val, err := r.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		r.logger.Debug("order record not found", zap.String("key", key))
		return nil, errors.NewDomainError(65006, "order not found", "order_id not found")
	}
	if err != nil {
		r.logger.Error("order record retrieval failed", zap.String("key", key), zap.Error(err))
		return nil, errors.WrapDomainError(err, 65011, "order record retrieval failed", "redis error")
	}

	var record ondc.OrderRecord
	if err := json.Unmarshal([]byte(val), &record); err != nil {
		r.logger.Error("order record deserialization failed", zap.String("key", key), zap.Error(err))
		return nil, errors.WrapDomainError(err, 65020, "order record deserialization failed", "invalid stored record")
	}

	r.logger.Debug("order record retrieved", zap.String("key", key), zap.String("order_id", record.OrderID))
	return &record, nil
}
