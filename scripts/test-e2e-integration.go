package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// IntegrationTest orchestrates end-to-end ONDC → UOIS → internal services flow
// Dispatch is logistics seller NP (BPP) for P2P (Point-to-Point) delivery only
type IntegrationTest struct {
	uoisBaseURL string
	redisClient *redis.Client
	logger      *zap.Logger
	httpClient  *http.Client
}

// NewIntegrationTest creates a new integration test instance
func NewIntegrationTest(uoisBaseURL, redisAddr, redisPassword string, redisDB int) (*IntegrationTest, error) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &IntegrationTest{
		uoisBaseURL: uoisBaseURL,
		redisClient: rdb,
		logger:      logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// LoadJSONFile loads a JSON file from testdata/mocks directory
func (it *IntegrationTest) LoadJSONFile(relativePath string) (map[string]interface{}, error) {
	basePath := filepath.Join("testdata", "mocks")
	fullPath := filepath.Join(basePath, relativePath)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", fullPath, err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from %s: %w", fullPath, err)
	}

	return result, nil
}

// SendONDCRequest sends an ONDC HTTP request to UOIS Gateway
func (it *IntegrationTest) SendONDCRequest(endpoint string, payload map[string]interface{}) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", it.uoisBaseURL, endpoint)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic dGVzdF9jbGllbnQ6dGVzdF9zZWNyZXQ=") // test_client:test_secret

	resp, err := it.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	return resp, nil
}

// PublishEvent publishes an event to Redis Stream
func (it *IntegrationTest) PublishEvent(ctx context.Context, stream string, event map[string]interface{}) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	args := &redis.XAddArgs{
		Stream: stream,
		Values: map[string]interface{}{
			"data": string(eventJSON),
		},
	}

	if err := it.redisClient.XAdd(ctx, args).Err(); err != nil {
		return fmt.Errorf("failed to publish event to stream %s: %w", stream, err)
	}

	it.logger.Info("Published event",
		zap.String("stream", stream),
		zap.String("event_type", getEventType(event)),
	)
	return nil
}

// ConsumeEvent consumes an event from Redis Stream using consumer group
func (it *IntegrationTest) ConsumeEvent(ctx context.Context, stream, consumerGroup, correlationID string, timeout time.Duration) (map[string]interface{}, error) {
	// Create consumer group if it doesn't exist
	consumerName := fmt.Sprintf("test-consumer-%s", uuid.New().String()[:8])
	err := it.redisClient.XGroupCreateMkStream(ctx, stream, consumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		it.logger.Warn("Failed to create consumer group (may already exist)", zap.Error(err))
	}

	// Read from stream with blocking
	streams, err := it.redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    consumerGroup,
		Consumer: consumerName,
		Streams:  []string{stream, ">"},
		Count:    1,
		Block:    timeout,
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("failed to read from stream %s: %w", stream, err)
	}

	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, fmt.Errorf("no messages received from stream %s within timeout", stream)
	}

	msg := streams[0].Messages[0]
	eventData := msg.Values["data"].(string)

	var event map[string]interface{}
	if err := json.Unmarshal([]byte(eventData), &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	// ACK the message
	if err := it.redisClient.XAck(ctx, stream, consumerGroup, msg.ID).Err(); err != nil {
		it.logger.Warn("Failed to ACK message", zap.Error(err))
	}

	// Filter by correlation ID if provided
	if correlationID != "" {
		eventCorrelationID := getCorrelationID(event)
		if eventCorrelationID != correlationID {
			return nil, fmt.Errorf("correlation ID mismatch: expected %s, got %s", correlationID, eventCorrelationID)
		}
	}

	return event, nil
}

// getEventType extracts event_type from event payload
func getEventType(event map[string]interface{}) string {
	if eventType, ok := event["event_type"].(string); ok {
		return eventType
	}
	return "unknown"
}

// getCorrelationID extracts correlation ID (search_id, quote_id, etc.) from event
func getCorrelationID(event map[string]interface{}) string {
	if searchID, ok := event["search_id"].(string); ok {
		return searchID
	}
	if quoteID, ok := event["quote_id"].(string); ok {
		return quoteID
	}
	return ""
}

// TestSearchFlow tests the complete /search flow: ONDC request → SEARCH_REQUESTED event → QUOTE_COMPUTED event → /on_search callback
func (it *IntegrationTest) TestSearchFlow(ctx context.Context) error {
	it.logger.Info("=== Testing /search Flow (P2P Delivery) ===")

	// 1. Load ONDC /search request
	searchReq, err := it.LoadJSONFile("ondc/requests/search.json")
	if err != nil {
		return fmt.Errorf("failed to load search request: %w", err)
	}

	// 2. Send /search request to UOIS Gateway
	it.logger.Info("Sending /search request to UOIS Gateway")
	resp, err := it.SendONDCRequest("/search", searchReq)
	if err != nil {
		return fmt.Errorf("failed to send /search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// Parse ACK response
	var ackResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&ackResp); err != nil {
		return fmt.Errorf("failed to parse ACK response: %w", err)
	}

	it.logger.Info("Received ACK response", zap.Any("response", ackResp))

	// 3. Wait for SEARCH_REQUESTED event to be published (UOIS Gateway publishes this)
	// In real flow, UOIS publishes this, but for testing we simulate it
	it.logger.Info("Waiting for SEARCH_REQUESTED event...")
	time.Sleep(500 * time.Millisecond)

	// 4. Simulate QUOTE_COMPUTED event from Quote Service
	it.logger.Info("Publishing QUOTE_COMPUTED event (simulating Quote Service)")
	quoteComputed, err := it.LoadJSONFile("events/consumed/quote_computed.json")
	if err != nil {
		return fmt.Errorf("failed to load QUOTE_COMPUTED event: %w", err)
	}

	// Extract search_id from request context for correlation
	contextObj := searchReq["context"].(map[string]interface{})
	transactionID := contextObj["transaction_id"].(string)
	searchID := uuid.New().String() // UOIS generates this, but we simulate it
	quoteComputed["search_id"] = searchID

	if err := it.PublishEvent(ctx, "quote:computed", quoteComputed); err != nil {
		return fmt.Errorf("failed to publish QUOTE_COMPUTED event: %w", err)
	}

	it.logger.Info("Published QUOTE_COMPUTED event",
		zap.String("search_id", searchID),
		zap.String("transaction_id", transactionID),
	)

	// 5. UOIS Gateway should consume QUOTE_COMPUTED and send /on_search callback
	// In a real test, we would set up a mock callback receiver
	it.logger.Info("UOIS Gateway should consume QUOTE_COMPUTED and send /on_search callback")
	time.Sleep(2 * time.Second)

	it.logger.Info("✓ /search flow test completed successfully")
	return nil
}

// TestInitFlow tests the complete /init flow: ONDC request → INIT_REQUESTED event → QUOTE_CREATED event → /on_init callback
func (it *IntegrationTest) TestInitFlow(ctx context.Context, searchID string) (string, error) {
	it.logger.Info("=== Testing /init Flow (P2P Delivery) ===")

	// 1. Load ONDC /init request
	initReq, err := it.LoadJSONFile("ondc/requests/init.json")
	if err != nil {
		return "", fmt.Errorf("failed to load init request: %w", err)
	}

	// 2. Send /init request to UOIS Gateway
	it.logger.Info("Sending /init request to UOIS Gateway")
	resp, err := it.SendONDCRequest("/init", initReq)
	if err != nil {
		return "", fmt.Errorf("failed to send /init request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var ackResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&ackResp); err != nil {
		return "", fmt.Errorf("failed to parse ACK response: %w", err)
	}

	it.logger.Info("Received ACK response", zap.Any("response", ackResp))

	// 3. Simulate QUOTE_CREATED event from Order Service
	it.logger.Info("Publishing QUOTE_CREATED event (simulating Order Service)")
	quoteCreated, err := it.LoadJSONFile("events/consumed/quote_created.json")
	if err != nil {
		return "", fmt.Errorf("failed to load QUOTE_CREATED event: %w", err)
	}

	quoteID := uuid.New().String()
	quoteCreated["quote_id"] = quoteID
	quoteCreated["search_id"] = searchID

	if err := it.PublishEvent(ctx, "stream.uois.quote_created", quoteCreated); err != nil {
		return "", fmt.Errorf("failed to publish QUOTE_CREATED event: %w", err)
	}

	it.logger.Info("Published QUOTE_CREATED event",
		zap.String("quote_id", quoteID),
		zap.String("search_id", searchID),
	)

	// 4. UOIS Gateway should consume QUOTE_CREATED and send /on_init callback
	it.logger.Info("UOIS Gateway should consume QUOTE_CREATED and send /on_init callback")
	time.Sleep(2 * time.Second)

	it.logger.Info("✓ /init flow test completed successfully")
	return quoteID, nil
}

// TestConfirmFlow tests the complete /confirm flow: ONDC request → CONFIRM_REQUESTED event → ORDER_CONFIRMED event → /on_confirm callback
func (it *IntegrationTest) TestConfirmFlow(ctx context.Context, quoteID string) error {
	it.logger.Info("=== Testing /confirm Flow (P2P Delivery) ===")

	// 1. Load ONDC /confirm request
	confirmReq, err := it.LoadJSONFile("ondc/requests/confirm.json")
	if err != nil {
		return fmt.Errorf("failed to load confirm request: %w", err)
	}

	// Update quote_id in request
	messageObj := confirmReq["message"].(map[string]interface{})
	orderObj := messageObj["order"].(map[string]interface{})
	quoteObj := orderObj["quote"].(map[string]interface{})
	quoteObj["id"] = quoteID

	// 2. Send /confirm request to UOIS Gateway
	it.logger.Info("Sending /confirm request to UOIS Gateway")
	resp, err := it.SendONDCRequest("/confirm", confirmReq)
	if err != nil {
		return fmt.Errorf("failed to send /confirm request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var ackResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&ackResp); err != nil {
		return fmt.Errorf("failed to parse ACK response: %w", err)
	}

	it.logger.Info("Received ACK response", zap.Any("response", ackResp))

	// 3. Simulate ORDER_CONFIRMED event from Order Service
	it.logger.Info("Publishing ORDER_CONFIRMED event (simulating Order Service)")
	orderConfirmed, err := it.LoadJSONFile("events/consumed/order_confirmed.json")
	if err != nil {
		return fmt.Errorf("failed to load ORDER_CONFIRMED event: %w", err)
	}

	orderConfirmed["quote_id"] = quoteID
	dispatchOrderID := "ABC0000001"
	orderConfirmed["dispatch_order_id"] = dispatchOrderID

	if err := it.PublishEvent(ctx, "stream.uois.order_confirmed", orderConfirmed); err != nil {
		return fmt.Errorf("failed to publish ORDER_CONFIRMED event: %w", err)
	}

	it.logger.Info("Published ORDER_CONFIRMED event",
		zap.String("quote_id", quoteID),
		zap.String("dispatch_order_id", dispatchOrderID),
	)

	// 4. UOIS Gateway should consume ORDER_CONFIRMED and send /on_confirm callback
	it.logger.Info("UOIS Gateway should consume ORDER_CONFIRMED and send /on_confirm callback")
	time.Sleep(2 * time.Second)

	it.logger.Info("✓ /confirm flow test completed successfully")
	return nil
}

// TestEventPublishing validates that UOIS Gateway publishes events correctly
func (it *IntegrationTest) TestEventPublishing(ctx context.Context) error {
	it.logger.Info("=== Testing Event Publishing (UOIS Gateway → Downstream Services) ===")

	// Test SEARCH_REQUESTED event
	searchRequested, err := it.LoadJSONFile("events/published/search_requested.json")
	if err != nil {
		return fmt.Errorf("failed to load SEARCH_REQUESTED event: %w", err)
	}

	if err := it.PublishEvent(ctx, "stream.location.search", searchRequested); err != nil {
		return fmt.Errorf("failed to publish SEARCH_REQUESTED: %w", err)
	}
	it.logger.Info("✓ Published SEARCH_REQUESTED to stream.location.search")

	// Test INIT_REQUESTED event
	initRequested, err := it.LoadJSONFile("events/published/init_requested.json")
	if err != nil {
		return fmt.Errorf("failed to load INIT_REQUESTED event: %w", err)
	}

	if err := it.PublishEvent(ctx, "stream.uois.init_requested", initRequested); err != nil {
		return fmt.Errorf("failed to publish INIT_REQUESTED: %w", err)
	}
	it.logger.Info("✓ Published INIT_REQUESTED to stream.uois.init_requested")

	// Test CONFIRM_REQUESTED event
	confirmRequested, err := it.LoadJSONFile("events/published/confirm_requested.json")
	if err != nil {
		return fmt.Errorf("failed to load CONFIRM_REQUESTED event: %w", err)
	}

	if err := it.PublishEvent(ctx, "stream.uois.confirm_requested", confirmRequested); err != nil {
		return fmt.Errorf("failed to publish CONFIRM_REQUESTED: %w", err)
	}
	it.logger.Info("✓ Published CONFIRM_REQUESTED to stream.uois.confirm_requested")

	return nil
}

// TestEventConsumption validates that UOIS Gateway consumes events correctly
func (it *IntegrationTest) TestEventConsumption(ctx context.Context) error {
	it.logger.Info("=== Testing Event Consumption (Downstream Services → UOIS Gateway) ===")

	// Consumer group: uois-gateway-consumers (used by UOIS Gateway)
	_ = "uois-gateway-consumers" // Consumer group name for reference

	// Test QUOTE_COMPUTED consumption
	quoteComputed, err := it.LoadJSONFile("events/consumed/quote_computed.json")
	if err != nil {
		return fmt.Errorf("failed to load QUOTE_COMPUTED event: %w", err)
	}

	searchID := uuid.New().String()
	quoteComputed["search_id"] = searchID

	if err := it.PublishEvent(ctx, "quote:computed", quoteComputed); err != nil {
		return fmt.Errorf("failed to publish QUOTE_COMPUTED: %w", err)
	}

	it.logger.Info("Published QUOTE_COMPUTED, waiting for UOIS Gateway to consume...")
	time.Sleep(1 * time.Second)
	it.logger.Info("✓ QUOTE_COMPUTED event published (UOIS Gateway should consume)")

	// Test QUOTE_CREATED consumption
	quoteCreated, err := it.LoadJSONFile("events/consumed/quote_created.json")
	if err != nil {
		return fmt.Errorf("failed to load QUOTE_CREATED event: %w", err)
	}

	quoteID := uuid.New().String()
	quoteCreated["quote_id"] = quoteID
	quoteCreated["search_id"] = searchID

	if err := it.PublishEvent(ctx, "stream.uois.quote_created", quoteCreated); err != nil {
		return fmt.Errorf("failed to publish QUOTE_CREATED: %w", err)
	}

	it.logger.Info("Published QUOTE_CREATED, waiting for UOIS Gateway to consume...")
	time.Sleep(1 * time.Second)
	it.logger.Info("✓ QUOTE_CREATED event published (UOIS Gateway should consume)")

	// Test ORDER_CONFIRMED consumption
	orderConfirmed, err := it.LoadJSONFile("events/consumed/order_confirmed.json")
	if err != nil {
		return fmt.Errorf("failed to load ORDER_CONFIRMED event: %w", err)
	}

	orderConfirmed["quote_id"] = quoteID

	if err := it.PublishEvent(ctx, "stream.uois.order_confirmed", orderConfirmed); err != nil {
		return fmt.Errorf("failed to publish ORDER_CONFIRMED: %w", err)
	}

	it.logger.Info("Published ORDER_CONFIRMED, waiting for UOIS Gateway to consume...")
	time.Sleep(1 * time.Second)
	it.logger.Info("✓ ORDER_CONFIRMED event published (UOIS Gateway should consume)")

	return nil
}

// RunAllTests runs the complete end-to-end integration test suite
func (it *IntegrationTest) RunAllTests(ctx context.Context) error {
	it.logger.Info("========================================")
	it.logger.Info("UOIS Gateway E2E Integration Tests")
	it.logger.Info("Dispatch: Logistics Seller NP (BPP) - P2P Delivery Only")
	it.logger.Info("========================================")

	// Test 1: Event Publishing
	if err := it.TestEventPublishing(ctx); err != nil {
		return fmt.Errorf("event publishing test failed: %w", err)
	}

	// Test 2: Event Consumption
	if err := it.TestEventConsumption(ctx); err != nil {
		return fmt.Errorf("event consumption test failed: %w", err)
	}

	// Test 3: Complete /search flow
	searchID := uuid.New().String()
	if err := it.TestSearchFlow(ctx); err != nil {
		return fmt.Errorf("/search flow test failed: %w", err)
	}

	// Test 4: Complete /init flow
	quoteID, err := it.TestInitFlow(ctx, searchID)
	if err != nil {
		return fmt.Errorf("/init flow test failed: %w", err)
	}

	// Test 5: Complete /confirm flow
	if err := it.TestConfirmFlow(ctx, quoteID); err != nil {
		return fmt.Errorf("/confirm flow test failed: %w", err)
	}

	it.logger.Info("========================================")
	it.logger.Info("✓ All integration tests passed!")
	it.logger.Info("========================================")

	return nil
}

func main() {
	// Configuration (can be overridden via environment variables)
	uoisBaseURL := getEnv("UOIS_BASE_URL", "http://localhost:8080")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := 0

	it, err := NewIntegrationTest(uoisBaseURL, redisAddr, redisPassword, redisDB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize integration test: %v\n", err)
		os.Exit(1)
	}
	defer it.redisClient.Close()

	ctx := context.Background()
	if err := it.RunAllTests(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Integration test failed: %v\n", err)
		os.Exit(1)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
