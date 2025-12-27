package order

import (
	"context"
	"testing"

	"uois-gateway/internal/config"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestOrderServiceClient_ValidateSearchIDTTL_Success(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.OrderConfig{
		GRPCHost:    "localhost",
		GRPCPort:    50052,
		GRPCTimeout: 5,
		MaxRetries:  3,
	}

	client := NewClient(cfg, logger)

	valid, err := client.ValidateSearchIDTTL(context.Background(), "test-search-id")

	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestOrderServiceClient_ValidateQuoteIDTTL_Success(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.OrderConfig{
		GRPCHost:    "localhost",
		GRPCPort:    50052,
		GRPCTimeout: 5,
		MaxRetries:  3,
	}

	client := NewClient(cfg, logger)

	valid, err := client.ValidateQuoteIDTTL(context.Background(), "test-quote-id")

	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestOrderServiceClient_GetOrder_Success(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.OrderConfig{
		GRPCHost:    "localhost",
		GRPCPort:    50052,
		GRPCTimeout: 5,
		MaxRetries:  3,
	}

	client := NewClient(cfg, logger)

	order, err := client.GetOrder(context.Background(), "DISPATCH123")

	assert.NoError(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, "DISPATCH123", order.DispatchOrderID)
	assert.Equal(t, "IN_PROGRESS", order.State)
}

func TestOrderServiceClient_GetOrderTracking_Success(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.OrderConfig{
		GRPCHost:    "localhost",
		GRPCPort:    50052,
		GRPCTimeout: 5,
		MaxRetries:  3,
	}

	client := NewClient(cfg, logger)

	tracking, err := client.GetOrderTracking(context.Background(), "DISPATCH123")

	assert.NoError(t, err)
	assert.NotNil(t, tracking)
	assert.Equal(t, "DISPATCH123", tracking.DispatchOrderID)
	assert.Contains(t, tracking.TrackingURL, "DISPATCH123")
	assert.NotZero(t, tracking.ETA)
}

func TestOrderServiceClient_CancelOrder_Success(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.OrderConfig{
		GRPCHost:    "localhost",
		GRPCPort:    50052,
		GRPCTimeout: 5,
		MaxRetries:  3,
	}

	client := NewClient(cfg, logger)

	err := client.CancelOrder(context.Background(), "DISPATCH123", "001")

	assert.NoError(t, err)
}

func TestOrderServiceClient_UpdateOrder_Success(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.OrderConfig{
		GRPCHost:    "localhost",
		GRPCPort:    50052,
		GRPCTimeout: 5,
		MaxRetries:  3,
	}

	client := NewClient(cfg, logger)

	updates := map[string]interface{}{
		"weight": 5.0,
	}

	err := client.UpdateOrder(context.Background(), "DISPATCH123", updates)

	assert.NoError(t, err)
}

func TestOrderServiceClient_InitiateRTO_Success(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.OrderConfig{
		GRPCHost:    "localhost",
		GRPCPort:    50052,
		GRPCTimeout: 5,
		MaxRetries:  3,
	}

	client := NewClient(cfg, logger)

	err := client.InitiateRTO(context.Background(), "DISPATCH123")

	assert.NoError(t, err)
}
