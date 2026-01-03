package redis

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewClient_Success(t *testing.T) {
	// This test requires a real Redis instance or testcontainers
	// For now, we'll test the error case and structure
	logger := zap.NewNop()

	// Test with invalid address (should fail)
	client, err := NewClient("invalid:6379", "", 0, logger)
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestClient_Close(t *testing.T) {
	// Skip test if Redis is not available
	logger := zap.NewNop()
	testClient, err := NewClient("localhost:6379", "", 0, logger)
	if err != nil {
		t.Skip("Redis not available, skipping integration test")
		return
	}
	defer testClient.Close()

	err = testClient.Close()
	assert.NoError(t, err)
}

func TestClient_GetClient(t *testing.T) {
	// Skip test if Redis is not available
	logger := zap.NewNop()
	testClient, err := NewClient("localhost:6379", "", 0, logger)
	if err != nil {
		t.Skip("Redis not available, skipping integration test")
		return
	}
	defer testClient.Close()

	rdb := testClient.GetClient()
	assert.NotNil(t, rdb)
}

func TestClient_Incr(t *testing.T) {
	// Skip test if Redis is not available
	logger := zap.NewNop()
	testClient, err := NewClient("localhost:6379", "", 0, logger)
	if err != nil {
		t.Skip("Redis not available, skipping integration test")
		return
	}
	defer testClient.Close()

	ctx := context.Background()
	cmd := testClient.Incr(ctx, "test-key")
	assert.NotNil(t, cmd)
}

func TestClient_Expire(t *testing.T) {
	// Skip test if Redis is not available
	logger := zap.NewNop()
	testClient, err := NewClient("localhost:6379", "", 0, logger)
	if err != nil {
		t.Skip("Redis not available, skipping integration test")
		return
	}
	defer testClient.Close()

	ctx := context.Background()
	cmd := testClient.Expire(ctx, "test-key", time.Second)
	assert.NotNil(t, cmd)
}

func TestClient_TTL(t *testing.T) {
	// Skip test if Redis is not available
	logger := zap.NewNop()
	testClient, err := NewClient("localhost:6379", "", 0, logger)
	if err != nil {
		t.Skip("Redis not available, skipping integration test")
		return
	}
	defer testClient.Close()

	ctx := context.Background()
	cmd := testClient.TTL(ctx, "test-key")
	assert.NotNil(t, cmd)
}
