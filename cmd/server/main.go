package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"uois-gateway/internal/config"

	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Initialize dependencies (dependency injection)
	// TODO: Wire up all dependencies following Clean Architecture

	// Start HTTP server
	// TODO: Initialize HTTP router with handlers

	// Start event consumer
	// TODO: Initialize event consumer with context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = ctx // Placeholder until event consumer is implemented

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	logger.Info("UOIS Gateway starting", zap.Any("config", cfg.Server))
	<-sigChan
	logger.Info("Shutting down...")
	cancel()
}
