package client

import (
	"context"
	"sync"
	"testing"

	"uois-gateway/internal/models"
	"uois-gateway/pkg/errors"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestInMemoryClientRegistry_GetByClientID_Success(t *testing.T) {
	logger := zap.NewNop()
	registry := NewInMemoryClientRegistry(logger)

	client := &models.Client{
		ID:     "test-client-1",
		Status: models.ClientStatusActive,
	}
	registry.AddClient(client)

	result, err := registry.GetByClientID(context.Background(), "test-client-1")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-client-1", result.ID)
	assert.Equal(t, models.ClientStatusActive, result.Status)
}

func TestInMemoryClientRegistry_GetByClientID_NotFound(t *testing.T) {
	logger := zap.NewNop()
	registry := NewInMemoryClientRegistry(logger)

	result, err := registry.GetByClientID(context.Background(), "non-existent")

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*errors.DomainError)
	assert.True(t, ok)
	assert.Equal(t, 65006, domainErr.Code)
	assert.Contains(t, domainErr.Message, "client not found")
}

func TestInMemoryClientRegistry_GetByClientID_EmptyID(t *testing.T) {
	logger := zap.NewNop()
	registry := NewInMemoryClientRegistry(logger)

	result, err := registry.GetByClientID(context.Background(), "")

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*errors.DomainError)
	assert.True(t, ok)
	assert.Equal(t, 65006, domainErr.Code)
}

func TestInMemoryClientRegistry_ConcurrentAccess(t *testing.T) {
	logger := zap.NewNop()
	registry := NewInMemoryClientRegistry(logger)

	// Add multiple clients
	client1 := &models.Client{ID: "client-1", Status: models.ClientStatusActive}
	client2 := &models.Client{ID: "client-2", Status: models.ClientStatusActive}
	registry.AddClient(client1)
	registry.AddClient(client2)

	// Concurrent reads
	var wg sync.WaitGroup
	concurrency := 10
	wg.Add(concurrency * 2) // 2 clients * 10 goroutines

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			client, err := registry.GetByClientID(context.Background(), "client-1")
			assert.NoError(t, err)
			assert.NotNil(t, client)
			assert.Equal(t, "client-1", client.ID)
		}()
		go func() {
			defer wg.Done()
			client, err := registry.GetByClientID(context.Background(), "client-2")
			assert.NoError(t, err)
			assert.NotNil(t, client)
			assert.Equal(t, "client-2", client.ID)
		}()
	}

	wg.Wait()
}

func TestInMemoryClientRegistry_ContextCancellation(t *testing.T) {
	logger := zap.NewNop()
	registry := NewInMemoryClientRegistry(logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := registry.GetByClientID(ctx, "test-client")

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*errors.DomainError)
	assert.True(t, ok)
	assert.Equal(t, 65011, domainErr.Code)
}
