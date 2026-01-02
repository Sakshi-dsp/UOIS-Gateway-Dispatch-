package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"uois-gateway/internal/config"
	"uois-gateway/internal/models"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisClient interface for Redis operations
// Can accept either *redis.Client directly or any interface with Get/Set/Del methods
type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

// ClientRegistryRepository interface for database operations
type ClientRegistryRepository interface {
	GetByClientID(ctx context.Context, clientID string) (*models.Client, error)
	UpsertClient(ctx context.Context, client *models.Client) error
	UpdateStatus(ctx context.Context, clientID, status string) error
}

// DBClientRegistry is a DB-backed client registry with Redis caching
// Implements ClientRegistry interface for use in authentication
type DBClientRegistry struct {
	repo   ClientRegistryRepository
	redis  RedisClient
	config config.Config
	logger *zap.Logger
}

// Ensure DBClientRegistry implements ClientRegistry interface
var _ ClientRegistry = (*DBClientRegistry)(nil)

// NewDBClientRegistry creates a new DB-backed client registry with Redis cache
func NewDBClientRegistry(
	repo ClientRegistryRepository,
	redis RedisClient,
	cfg config.Config,
	logger *zap.Logger,
) *DBClientRegistry {
	return &DBClientRegistry{
		repo:   repo,
		redis:  redis,
		config: cfg,
		logger: logger,
	}
}

// GetByClientID retrieves a client by ID (cache-first, then DB)
func (r *DBClientRegistry) GetByClientID(ctx context.Context, clientID string) (*models.Client, error) {
	cacheKey := fmt.Sprintf("client:%s", clientID)
	cacheTTL := time.Duration(r.config.TTL.ClientRegistryCache) * time.Second

	// Try cache first
	cached, err := r.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var client models.Client
		if err := json.Unmarshal([]byte(cached), &client); err == nil {
			r.logger.Debug("client found in cache", zap.String("client_id", clientID))
			client.NormalizeIPs(&zapCIDRLogger{logger: r.logger})
			return &client, nil
		}
		r.logger.Warn("failed to unmarshal cached client", zap.Error(err))
	}

	// Cache miss or error - query DB
	client, err := r.repo.GetByClientID(ctx, clientID)
	if err != nil {
		return nil, err
	}

	// Normalize IPs for hot-path optimization
	client.NormalizeIPs(&zapCIDRLogger{logger: r.logger})

	// Cache the result
	clientJSON, err := json.Marshal(client)
	if err == nil {
		if err := r.redis.Set(ctx, cacheKey, clientJSON, cacheTTL).Err(); err != nil {
			r.logger.Warn("failed to cache client", zap.Error(err), zap.String("client_id", clientID))
		}
	}

	return client, nil
}

// UpsertClient upserts a client (updates DB and invalidates cache)
func (r *DBClientRegistry) UpsertClient(ctx context.Context, client *models.Client) error {
	if err := r.repo.UpsertClient(ctx, client); err != nil {
		return err
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("client:%s", client.ID)
	if err := r.redis.Del(ctx, cacheKey).Err(); err != nil {
		r.logger.Warn("failed to invalidate client cache", zap.Error(err), zap.String("client_id", client.ID))
	}

	return nil
}

// UpdateStatus updates client status (updates DB and invalidates cache)
func (r *DBClientRegistry) UpdateStatus(ctx context.Context, clientID, status string) error {
	if err := r.repo.UpdateStatus(ctx, clientID, status); err != nil {
		return err
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("client:%s", clientID)
	if err := r.redis.Del(ctx, cacheKey).Err(); err != nil {
		r.logger.Warn("failed to invalidate client cache", zap.Error(err), zap.String("client_id", clientID))
	}

	return nil
}

// zapCIDRLogger adapts zap.Logger to models.CIDRLogger interface
type zapCIDRLogger struct {
	logger *zap.Logger
}

func (z *zapCIDRLogger) Warn(msg string, fields ...interface{}) {
	z.logger.Warn(msg, zap.Any("fields", fields))
}
