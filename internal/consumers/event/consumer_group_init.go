package event

import (
	"context"
	"fmt"

	"uois-gateway/internal/config"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// ConsumerGroupClient interface for Redis consumer group operations
type ConsumerGroupClient interface {
	XGroupCreate(ctx context.Context, stream, group, start string, mkStream bool) *redis.StatusCmd
}

// InitializeConsumerGroups creates consumer groups for all configured streams
func InitializeConsumerGroups(ctx context.Context, rdb ConsumerGroupClient, cfg config.StreamsConfig, logger *zap.Logger) error {
	groupName := cfg.ConsumerGroupName
	if groupName == "" {
		return fmt.Errorf("consumer group name is required")
	}

	streams := []string{
		cfg.QuoteComputed,
		cfg.QuoteCreated,
		cfg.QuoteInvalidated,
		cfg.OrderConfirmed,
		cfg.OrderConfirmFailed,
		cfg.ClientEvents,
	}

	for _, stream := range streams {
		if stream == "" {
			continue
		}

		err := rdb.XGroupCreate(ctx, stream, groupName, "0", true).Err()
		if err != nil {
			if err.Error() == "BUSYGROUP Consumer Group name already exists" {
				logger.Debug("consumer group already exists", zap.String("stream", stream), zap.String("group", groupName))
				continue
			}
			logger.Error("failed to create consumer group", zap.Error(err), zap.String("stream", stream))
			return fmt.Errorf("failed to create consumer group for stream %s: %w", stream, err)
		}

		logger.Info("consumer group created", zap.String("stream", stream), zap.String("group", groupName))
	}

	return nil
}
