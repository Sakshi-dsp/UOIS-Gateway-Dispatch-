package client_events

import (
	"context"
	"encoding/json"

	"uois-gateway/internal/models"
	"uois-gateway/pkg/errors"

	"go.uber.org/zap"
)

// ClientEvent represents a client event from Admin Service
type ClientEvent struct {
	EventType        string                 `json:"event_type"`
	ClientID         string                 `json:"client_id"`
	ClientCode       string                 `json:"client_code,omitempty"`
	ClientSecretHash string                 `json:"client_secret_hash,omitempty"`
	BapID            string                 `json:"bap_id,omitempty"`
	BapURI           string                 `json:"bap_uri,omitempty"`
	AllowedIPs       []string               `json:"allowed_ips,omitempty"`
	Status           string                 `json:"status,omitempty"`
	RateLimit        int64                  `json:"rate_limit,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	Timestamp        string                 `json:"timestamp"`
	EventID          string                 `json:"event_id"`
}

// ClientRegistryService interface for client registry operations
// Matches DBClientRegistry methods for event-driven sync
type ClientRegistryService interface {
	UpsertClient(ctx context.Context, client *models.Client) error
	UpdateStatus(ctx context.Context, clientID, status string) error
}

// Consumer handles client events from Admin Service
type Consumer struct {
	registry ClientRegistryService
	logger   *zap.Logger
}

// NewConsumer creates a new client event consumer
func NewConsumer(registry ClientRegistryService, logger *zap.Logger) *Consumer {
	return &Consumer{
		registry: registry,
		logger:   logger,
	}
}

// HandleClientEvent processes a client event
func (c *Consumer) HandleClientEvent(ctx context.Context, eventData []byte) error {
	var event ClientEvent
	if err := json.Unmarshal(eventData, &event); err != nil {
		c.logger.Error("failed to unmarshal client event", zap.Error(err))
		return errors.WrapDomainError(err, 65020, "client event parsing failed", "invalid JSON")
	}

	client := &models.Client{
		ID:               event.ClientID,
		ClientCode:       event.ClientCode,
		ClientSecretHash: event.ClientSecretHash,
		AllowedIPs:       event.AllowedIPs,
		Status:           event.Status,
		Metadata:         event.Metadata,
	}

	if event.Metadata == nil {
		client.Metadata = make(map[string]interface{})
	}

	if event.BapID != "" {
		client.Metadata["bap_id"] = event.BapID
	}

	if event.BapURI != "" {
		client.Metadata["bap_uri"] = event.BapURI
	}

	if event.RateLimit > 0 {
		client.Metadata["rate_limit"] = event.RateLimit
	}

	switch event.EventType {
	case "client.created", "client.updated", "client.api_key_rotated":
		return c.registry.UpsertClient(ctx, client)
	case "client.suspended":
		return c.registry.UpdateStatus(ctx, event.ClientID, models.ClientStatusSuspended)
	case "client.revoked":
		return c.registry.UpdateStatus(ctx, event.ClientID, models.ClientStatusRevoked)
	default:
		c.logger.Warn("unknown client event type", zap.String("event_type", event.EventType))
		return nil
	}
}
