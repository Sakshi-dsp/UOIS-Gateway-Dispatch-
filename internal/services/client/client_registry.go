package client

import (
	"context"
	"sync"

	"uois-gateway/internal/models"
	"uois-gateway/pkg/errors"

	"go.uber.org/zap"
)

// ClientRegistry provides client lookup functionality
type ClientRegistry interface {
	GetByClientID(ctx context.Context, clientID string) (*models.Client, error)
}

// InMemoryClientRegistry is a simple in-memory implementation for development/testing
// TODO: Replace with Redis-backed or DB-backed implementation for production
// Thread-safe implementation using RWMutex for concurrent access
type InMemoryClientRegistry struct {
	mu      sync.RWMutex // Protects clients map from concurrent access
	clients map[string]*models.Client
	logger  *zap.Logger
}

// NewInMemoryClientRegistry creates a new in-memory client registry
func NewInMemoryClientRegistry(logger *zap.Logger) *InMemoryClientRegistry {
	return &InMemoryClientRegistry{
		clients: make(map[string]*models.Client),
		logger:  logger,
	}
}

// AddClient adds a client to the registry (for testing/development)
func (r *InMemoryClientRegistry) AddClient(client *models.Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[client.ID] = client
}

// GetByClientID retrieves a client by ID
// TODO: For Redis/DB implementations, honor ctx.Done() for cancellation
func (r *InMemoryClientRegistry) GetByClientID(ctx context.Context, clientID string) (*models.Client, error) {
	// Check context cancellation (for future Redis/DB implementations)
	select {
	case <-ctx.Done():
		return nil, errors.WrapDomainError(ctx.Err(), 65011, "client lookup cancelled", "context cancelled")
	default:
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	client, exists := r.clients[clientID]
	if !exists {
		return nil, errors.NewDomainError(65006, "client not found", "client_id not found")
	}
	return client, nil
}
