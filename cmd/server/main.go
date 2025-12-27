package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"uois-gateway/internal/clients/order"
	"uois-gateway/internal/clients/redis"
	"uois-gateway/internal/config"
	"uois-gateway/internal/consumers/event"
	igmHandler "uois-gateway/internal/handlers/igm"
	"uois-gateway/internal/handlers/ondc"
	"uois-gateway/internal/middleware"
	auditRepo "uois-gateway/internal/repository/audit"
	"uois-gateway/internal/repository/issue"
	"uois-gateway/internal/repository/order_record"
	auditService "uois-gateway/internal/services/audit"
	"uois-gateway/internal/services/auth"
	"uois-gateway/internal/services/callback"
	"uois-gateway/internal/services/client"
	"uois-gateway/internal/services/idempotency"
	igmService "uois-gateway/internal/services/igm"
	ondcService "uois-gateway/internal/services/ondc"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
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

	// Initialize Redis client
	redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port)
	redisClient, err := redis.NewClient(redisAddr, cfg.Redis.Password, cfg.Redis.DB, logger)
	if err != nil {
		logger.Fatal("Failed to initialize Redis client", zap.Error(err))
	}
	defer redisClient.Close()

	// Initialize Postgres client (for audit)
	postgresDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.PostgresE.Host, cfg.PostgresE.Port, cfg.PostgresE.User,
		cfg.PostgresE.Password, cfg.PostgresE.DB, cfg.PostgresE.SSLMode)
	db, err := sql.Open("postgres", postgresDSN)
	if err != nil {
		logger.Fatal("Failed to initialize Postgres client", zap.Error(err))
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		logger.Fatal("Failed to ping Postgres", zap.Error(err))
	}

	// Initialize repositories
	orderRecordRepo := order_record.NewRepository(redisClient.GetClient(), *cfg, logger)
	issueRepo := issue.NewRepository(redisClient.GetClient(), *cfg, logger)
	auditRepoInstance := auditRepo.NewRepository(db, *cfg, logger)

	// Initialize services
	clientRegistry := client.NewInMemoryClientRegistry(logger)
	clientAuthService := auth.NewClientAuthService(clientRegistry, logger)
	rateLimitService := auth.NewRateLimitService(redisClient, cfg.RateLimit, logger)

	_, err = ondcService.NewONDCAuthService(&mockRegistryClient{}, cfg.ONDC, logger)
	if err != nil {
		logger.Fatal("Failed to initialize ONDC auth service", zap.Error(err))
	}

	callbackSigner, err := callback.NewONDCSignerFromConfig(cfg.ONDC, logger)
	if err != nil {
		logger.Warn("Failed to initialize callback signer, callbacks will be unsigned", zap.Error(err))
		callbackSigner = nil
	}

	callbackService := callback.NewService(cfg.Callback, callbackSigner, logger)
	idempotencyService := idempotency.NewService(redisClient.GetClient(), *cfg, logger)
	orderServiceClient := order.NewClient(cfg.Order, logger)
	eventPublisher := redis.NewEventPublisher(redisClient.GetClient(), logger)

	// Create adapters for event consumer and consumer group initialization
	streamConsumerAdapter := redis.NewStreamConsumerAdapter(redisClient.GetClient())
	consumerGroupAdapter := redis.NewConsumerGroupAdapter(redisClient.GetClient())

	eventConsumer := event.NewConsumer(streamConsumerAdapter, cfg.Streams, logger)
	groService := igmService.NewGROService(logger)
	_ = auditService.NewService(auditRepoInstance, logger) // Audit service ready for future integration

	// Initialize consumer groups
	initCtx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer initCancel()
	if err := event.InitializeConsumerGroups(initCtx, consumerGroupAdapter, cfg.Streams, logger); err != nil {
		logger.Warn("Failed to initialize consumer groups", zap.Error(err))
	}

	// Convert services to handler interfaces
	var (
		callbackServiceInterface    ondc.CallbackService        = callbackService
		idempotencyServiceInterface ondc.IdempotencyService     = idempotencyService
		orderServiceClientInterface ondc.OrderServiceClient     = orderServiceClient
		orderRecordServiceInterface ondc.OrderRecordService     = orderRecordRepo
		eventPublisherInterface     ondc.EventPublisher         = eventPublisher
		eventConsumerInterface      ondc.EventConsumer          = eventConsumer
		clientAuthServiceInterface  middleware.AuthService      = clientAuthService
		rateLimitServiceInterface   middleware.RateLimitService = rateLimitService
	)

	// Initialize ONDC handlers
	searchHandler := ondc.NewSearchHandler(
		eventPublisherInterface,
		eventConsumerInterface,
		callbackServiceInterface,
		idempotencyServiceInterface,
		orderRecordServiceInterface,
		cfg.ONDC.ProviderID,
		cfg.ONDC.BPPID,
		cfg.ONDC.BPPURI,
		cfg.ONDC.BPPName,
		cfg.ONDC.BPPTermsURL,
		logger,
	)

	initHandler := ondc.NewInitHandler(
		eventPublisherInterface,
		eventConsumerInterface,
		callbackServiceInterface,
		idempotencyServiceInterface,
		orderServiceClientInterface,
		orderRecordServiceInterface,
		cfg.ONDC.ProviderID,
		cfg.ONDC.BPPID,
		cfg.ONDC.BPPURI,
		logger,
	)

	confirmHandler := ondc.NewConfirmHandler(
		eventPublisherInterface,
		eventConsumerInterface,
		callbackServiceInterface,
		idempotencyServiceInterface,
		orderServiceClientInterface,
		orderRecordServiceInterface,
		cfg.ONDC.BPPID,
		cfg.ONDC.BPPURI,
		logger,
	)

	statusHandler := ondc.NewStatusHandler(
		callbackServiceInterface,
		idempotencyServiceInterface,
		orderServiceClientInterface,
		orderRecordServiceInterface,
		cfg.ONDC.BPPID,
		cfg.ONDC.BPPURI,
		logger,
	)

	trackHandler := ondc.NewTrackHandler(
		callbackServiceInterface,
		idempotencyServiceInterface,
		orderServiceClientInterface,
		orderRecordServiceInterface,
		cfg.ONDC.BPPID,
		cfg.ONDC.BPPURI,
		logger,
	)

	cancelHandler := ondc.NewCancelHandler(
		callbackServiceInterface,
		idempotencyServiceInterface,
		orderServiceClientInterface,
		orderRecordServiceInterface,
		cfg.ONDC.BPPID,
		cfg.ONDC.BPPURI,
		logger,
	)

	updateHandler := ondc.NewUpdateHandler(
		callbackServiceInterface,
		idempotencyServiceInterface,
		orderServiceClientInterface,
		orderRecordServiceInterface,
		cfg.ONDC.BPPID,
		cfg.ONDC.BPPURI,
		logger,
	)

	rtoHandler := ondc.NewRTOHandler(
		callbackServiceInterface,
		idempotencyServiceInterface,
		orderServiceClientInterface,
		orderRecordServiceInterface,
		cfg.ONDC.BPPID,
		cfg.ONDC.BPPURI,
		logger,
	)

	// Initialize IGM handlers
	issueHandler := igmHandler.NewIssueHandler(
		issueRepo,
		callbackServiceInterface,
		idempotencyServiceInterface,
		groService,
		cfg.ONDC.BPPID,
		cfg.ONDC.BPPURI,
		logger,
	)

	issueStatusHandler := igmHandler.NewIssueStatusHandler(
		issueRepo,
		callbackServiceInterface,
		idempotencyServiceInterface,
		groService,
		cfg.ONDC.BPPID,
		cfg.ONDC.BPPURI,
		logger,
	)

	// Initialize HTTP router
	router := setupRouter(
		searchHandler,
		initHandler,
		confirmHandler,
		statusHandler,
		trackHandler,
		cancelHandler,
		updateHandler,
		rtoHandler,
		issueHandler,
		issueStatusHandler,
		clientAuthServiceInterface,
		rateLimitServiceInterface,
		logger,
	)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start HTTP server in goroutine
	go func() {
		logger.Info("Starting HTTP server",
			zap.String("host", cfg.Server.Host),
			zap.Int("port", cfg.Server.Port),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Consumer groups already initialized above

	// Start event consumer (background goroutine)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// TODO: Start event consumer goroutines for each stream:
	// - QuoteComputed stream consumer (for /init handler)
	// - QuoteCreated stream consumer (for /init handler)
	// - OrderConfirmed stream consumer (for /confirm handler)
	// Example:
	// go func() {
	//     for {
	//         select {
	//         case <-ctx.Done():
	//             return
	//         default:
	//             event, err := eventConsumer.ConsumeEvent(ctx, cfg.Streams.QuoteComputed, cfg.Streams.ConsumerGroupName, "", 5*time.Second)
	//             if err != nil {
	//                 logger.Error("event consumption error", zap.Error(err))
	//                 continue
	//             }
	//             // Process event...
	//         }
	//     }
	// }()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	logger.Info("UOIS Gateway started", zap.Any("config", cfg.Server))
	<-sigChan

	logger.Info("Shutting down...")

	// Cancel event consumer context
	cancel()

	// Graceful shutdown: give server time to finish current requests
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}

	logger.Info("Shutdown complete")
}

// setupRouter configures the Gin router with all ONDC endpoints and middleware
func setupRouter(
	searchHandler *ondc.SearchHandler,
	initHandler *ondc.InitHandler,
	confirmHandler *ondc.ConfirmHandler,
	statusHandler *ondc.StatusHandler,
	trackHandler *ondc.TrackHandler,
	cancelHandler *ondc.CancelHandler,
	updateHandler *ondc.UpdateHandler,
	rtoHandler *ondc.RTOHandler,
	issueHandler *igmHandler.IssueHandler,
	issueStatusHandler *igmHandler.IssueStatusHandler,
	authService middleware.AuthService,
	rateLimitService middleware.RateLimitService,
	logger *zap.Logger,
) *gin.Engine {
	// Set Gin mode based on environment
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Health check endpoint (no auth required)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// ONDC API routes (require authentication and rate limiting)
	ondcGroup := router.Group("/ondc")
	ondcGroup.Use(middleware.AuthMiddleware(authService, rateLimitService, logger))

	// Register ONDC endpoints
	ondcGroup.POST("/search", searchHandler.HandleSearch)
	ondcGroup.POST("/init", initHandler.HandleInit)
	ondcGroup.POST("/confirm", confirmHandler.HandleConfirm)
	ondcGroup.POST("/status", statusHandler.HandleStatus)
	ondcGroup.POST("/track", trackHandler.HandleTrack)
	ondcGroup.POST("/cancel", cancelHandler.HandleCancel)
	ondcGroup.POST("/update", updateHandler.HandleUpdate)
	ondcGroup.POST("/rto", rtoHandler.HandleRTO)

	// Register IGM endpoints
	ondcGroup.POST("/issue", issueHandler.HandleIssue)
	ondcGroup.POST("/issue_status", issueStatusHandler.HandleIssueStatus)

	return router
}

// mockRegistryClient is a placeholder for ONDC registry client
// TODO: Implement actual registry client for production
type mockRegistryClient struct{}

func (m *mockRegistryClient) LookupPublicKey(ctx context.Context, subscriberID, ukID string) (string, error) {
	return "", fmt.Errorf("registry client not implemented")
}
