package client_registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"uois-gateway/internal/config"
	"uois-gateway/internal/models"
	"uois-gateway/pkg/errors"

	"github.com/lib/pq"
	"go.uber.org/zap"
)

// DBClient interface for database operations
type DBClient interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// Repository handles client registry storage
type Repository struct {
	db     DBClient
	config config.Config
	logger *zap.Logger
}

// NewRepository creates a new client registry repository
func NewRepository(db DBClient, cfg config.Config, logger *zap.Logger) *Repository {
	return &Repository{
		db:     db,
		config: cfg,
		logger: logger,
	}
}

// UpsertClient upserts a client into the registry
func (r *Repository) UpsertClient(ctx context.Context, client *models.Client) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	metadataJSON, err := json.Marshal(client.Metadata)
	if err != nil {
		return errors.WrapDomainError(err, 65020, "client metadata serialization failed", "failed to marshal metadata")
	}

	allowedIPs := pq.Array(client.AllowedIPs)
	if len(client.AllowedIPs) == 0 {
		allowedIPs = pq.Array([]string{})
	}

	query := `INSERT INTO client_registry.clients (
		client_id, client_code, client_secret_hash, api_key_hash,
		bap_id, bap_uri, allowed_ips, rate_limit, status, metadata
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	ON CONFLICT (client_id) DO UPDATE SET
		client_code = EXCLUDED.client_code,
		client_secret_hash = EXCLUDED.client_secret_hash,
		api_key_hash = EXCLUDED.api_key_hash,
		bap_id = EXCLUDED.bap_id,
		bap_uri = EXCLUDED.bap_uri,
		allowed_ips = EXCLUDED.allowed_ips,
		rate_limit = EXCLUDED.rate_limit,
		status = EXCLUDED.status,
		metadata = EXCLUDED.metadata,
		updated_at = CURRENT_TIMESTAMP,
		last_synced_at = CURRENT_TIMESTAMP`

	apiKeyHash := client.ClientSecretHash
	bapID := sql.NullString{Valid: false}
	bapURI := sql.NullString{Valid: false}
	rateLimit := int64(100)

	if client.Metadata != nil {
		if val, ok := client.Metadata["bap_id"].(string); ok && val != "" {
			bapID = sql.NullString{String: val, Valid: true}
		}
		if val, ok := client.Metadata["bap_uri"].(string); ok && val != "" {
			bapURI = sql.NullString{String: val, Valid: true}
		}
		if val, ok := client.Metadata["rate_limit"].(int64); ok && val > 0 {
			rateLimit = val
		}
	}

	_, err = r.db.ExecContext(ctx, query,
		client.ID,
		client.ClientCode,
		client.ClientSecretHash,
		apiKeyHash,
		bapID,
		bapURI,
		allowedIPs,
		rateLimit,
		client.Status,
		metadataJSON,
	)

	if err != nil {
		r.logger.Error("failed to upsert client", zap.Error(err), zap.String("client_id", client.ID))
		return errors.WrapDomainError(err, 65020, "client registry storage failed", "database error")
	}

	r.logger.Debug("client upserted",
		zap.String("client_id", client.ID),
		zap.String("client_code", client.ClientCode),
		zap.String("status", client.Status),
	)

	return nil
}

// GetByClientID retrieves a client by ID
func (r *Repository) GetByClientID(ctx context.Context, clientID string) (*models.Client, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	query := `SELECT 
		client_id, client_code, client_secret_hash,
		api_key_hash, bap_id, bap_uri, allowed_ips,
		rate_limit, status, metadata
	FROM client_registry.clients
	WHERE client_id = $1`

	var (
		dbClientID       string
		clientCode       string
		clientSecretHash string
		apiKeyHash       sql.NullString
		bapID            sql.NullString
		bapURI           sql.NullString
		allowedIPs       pq.StringArray
		rateLimit        sql.NullInt64
		status           string
		metadataJSON     sql.NullString
	)

	err := r.db.QueryRowContext(ctx, query, clientID).Scan(
		&dbClientID,
		&clientCode,
		&clientSecretHash,
		&apiKeyHash,
		&bapID,
		&bapURI,
		&allowedIPs,
		&rateLimit,
		&status,
		&metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, errors.NewDomainError(65006, "client not found", fmt.Sprintf("client_id %s not found", clientID))
	}

	if err != nil {
		r.logger.Error("failed to get client", zap.Error(err), zap.String("client_id", clientID))
		return nil, errors.WrapDomainError(err, 65011, "client registry unavailable", "database error")
	}

	client := &models.Client{
		ID:               dbClientID,
		ClientCode:       clientCode,
		ClientSecretHash: clientSecretHash,
		Status:           status,
	}

	if len(allowedIPs) > 0 {
		client.AllowedIPs = []string(allowedIPs)
	}

	if metadataJSON.Valid && metadataJSON.String != "" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(metadataJSON.String), &metadata); err == nil {
			client.Metadata = metadata
		}
	}

	if bapID.Valid {
		if client.Metadata == nil {
			client.Metadata = make(map[string]interface{})
		}
		client.Metadata["bap_id"] = bapID.String
	}

	if bapURI.Valid {
		if client.Metadata == nil {
			client.Metadata = make(map[string]interface{})
		}
		client.Metadata["bap_uri"] = bapURI.String
	}

	if rateLimit.Valid {
		if client.Metadata == nil {
			client.Metadata = make(map[string]interface{})
		}
		client.Metadata["rate_limit"] = rateLimit.Int64
	}

	return client, nil
}

// UpdateStatus updates client status
func (r *Repository) UpdateStatus(ctx context.Context, clientID, status string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	query := `UPDATE client_registry.clients
	SET status = $1, updated_at = CURRENT_TIMESTAMP
	WHERE client_id = $2`

	result, err := r.db.ExecContext(ctx, query, status, clientID)
	if err != nil {
		r.logger.Error("failed to update client status", zap.Error(err), zap.String("client_id", clientID))
		return errors.WrapDomainError(err, 65020, "client status update failed", "database error")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.WrapDomainError(err, 65020, "client status update failed", "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return errors.NewDomainError(65006, "client not found", fmt.Sprintf("client_id %s not found", clientID))
	}

	r.logger.Debug("client status updated",
		zap.String("client_id", clientID),
		zap.String("status", status),
	)

	return nil
}

