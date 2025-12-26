# UOIS Gateway - Repository Setup Guide

**For AI:** Use this guide to set up a new repository for UOIS Gateway from scratch.

**Source of Truth:** `doc/UOISGateway_FR.md`  
**Development Rules:** `doc/DISPATCH_DEV_RULES.md` and  `.cursorrules`

---

## 🎯 CRITICAL: Follow These Rules Strictly

### From Development Rules:

1. **Function Size:** Keep every function under 20 lines
2. **TDD:** Write tests BEFORE implementation (strict TDD) using **Testify** (`testify/assert` for assertions, `testify/mock` for mocking)
3. **Dependency Injection:** All services via constructors, no global state
4. **Error Wrapping:** Use `fmt.Errorf("message: %w", err)` format
5. **Single Layer Logging:** Only one layer should log errors (ideally the handler or infra boundary). Avoid "logging AND returning" (double logging)
6. **Context:** Never store context inside structs, always pass as parameter
7. **No Hardcoded Values:** All business values from config (TTLs, retry counts, callback URLs, stream names, timeouts)
8. **Clean Architecture:** Handlers → Services → Repository → Clients
9. **Build Verification:** After each step, run `go build ./...` and ensure it succeeds
10. **Test Verification:** After each test pass, run full test suite
11. **Handler & Orchestration Rules:**
    - Each consumed event MUST have its own dedicated handler file
    - Event handlers MUST NOT publish events
    - Event handlers MUST NOT call gRPC directly
    - gRPC clients MUST NOT emit events
    - Only the service layer may orchestrate event publishing and external calls
12. **Coordinate Field Naming:** Use `origin_lat`, `origin_lng`, `destination_lat`, `destination_lng` (NOT `pickup_lat`, `pickup_lng`, `drop_lat`, `drop_lng`) in all event schemas, models, and transformations

---

## 📋 Prerequisites Setup

### Step 1: Initialize Go Module

```bash
# Create new directory
mkdir uois-gateway
cd uois-gateway

# Initialize Go module
go mod init uois-gateway

# Create initial go.mod (will be populated with dependencies)
```

### Step 2: Install Core Dependencies

```bash
# HTTP Server (Gin or Echo)
go get github.com/gin-gonic/gin@latest
# OR
go get github.com/labstack/echo/v4@latest

# gRPC and Protocol Buffers
go get google.golang.org/grpc@latest
go get google.golang.org/protobuf@latest
go get google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Redis Client (for caching and event streams)
go get github.com/redis/go-redis/v9@latest

# PostgreSQL Driver
go get github.com/lib/pq@latest
# OR
go get github.com/jackc/pgx/v5@latest

# Logging (Zap)
go get go.uber.org/zap@latest

# Configuration Management (Viper)
go get github.com/spf13/viper@latest

# UUID Generation
go get github.com/google/uuid@latest

# Distributed Tracing (OpenTelemetry)
go get go.opentelemetry.io/otel@latest
go get go.opentelemetry.io/otel/trace@latest
go get go.opentelemetry.io/otel/exporters/jaeger@latest
go get go.opentelemetry.io/otel/sdk@latest
go get go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin@latest
go get go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc@latest

# HTTP Client (for callbacks and Zendesk)
go get github.com/go-resty/resty/v2@latest

# Bcrypt/Argon2 (for client secret hashing)
go get golang.org/x/crypto@latest

# JSON Schema Validation (for ONDC payloads)
go get github.com/xeipuuv/gojsonschema@latest

# Clean up dependencies
go mod tidy
```

### Step 3: Install Testing Dependencies

```bash
# Testing Framework: Testify (required)
go get github.com/stretchr/testify@latest
go get github.com/stretchr/testify/mock@latest
go get github.com/stretchr/testify/assert@latest

# HTTP Testing
go get github.com/stretchr/testify/http@latest

# Clean up
go mod tidy
```

**Note:** This project uses **Testify** for all testing. Use `assert` for assertions and `mock` for mocking external dependencies.

### Step 4: Install Protocol Buffer Compiler

```bash
# For macOS
brew install protobuf

# For Linux
sudo apt-get install protobuf-compiler

# For Windows
# Download from: https://github.com/protocolbuffers/protobuf/releases
# Or use: choco install protoc

# Verify installation
protoc --version
```

### Step 5: Install Go Protocol Buffer Plugins

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Ensure $GOPATH/bin is in PATH
export PATH="$PATH:$(go env GOPATH)/bin"
```

---

## 📁 Project Structure

**Organization Principle:** Handlers and services are organized by domain (ondc, igm) in subfolders to:
- **Avoid God Files:** Each endpoint has its own handler and service file
- **Separation of Concerns:** Clear boundaries between different API domains
- **Future Multi-Client Support:** Easy to add new client integrations (e.g., `handlers/beckn/`, `handlers/custom_client/`) without modifying existing code
- **Maintainability:** Smaller, focused files are easier to understand, test, and maintain
- **Scalability:** New endpoints can be added without touching existing handlers/services

Create the following directory structure:

```
uois-gateway/
├── cmd/
│   └── server/
│       └── main.go                    # Application entry point
├── internal/
│   ├── config/
│   │   ├── config.go                  # Configuration struct and loader
│   │   └── config_test.go             # Config tests (TDD)
│   ├── handlers/
│   │   ├── ondc/                       # ONDC API handlers (Logistics Seller NP)
│   │   │   ├── search_handler.go       # /search endpoint handler
│   │   │   ├── search_handler_test.go  # Search handler tests (TDD)
│   │   │   ├── on_search_handler.go   # /on_search callback handler (receiving - optional, for callback acknowledgments)
│   │   │   ├── on_search_handler_test.go # OnSearch handler tests (TDD)
│   │   │   ├── init_handler.go        # /init endpoint handler
│   │   │   ├── init_handler_test.go    # Init handler tests (TDD)
│   │   │   ├── on_init_handler.go     # /on_init callback handler (receiving - optional, for callback acknowledgments)
│   │   │   ├── on_init_handler_test.go # OnInit handler tests (TDD)
│   │   │   ├── confirm_handler.go     # /confirm endpoint handler
│   │   │   ├── confirm_handler_test.go # Confirm handler tests (TDD)
│   │   │   ├── on_confirm_handler.go  # /on_confirm callback handler (receiving - optional, for callback acknowledgments)
│   │   │   ├── on_confirm_handler_test.go # OnConfirm handler tests (TDD)
│   │   │   ├── status_handler.go      # /status endpoint handler
│   │   │   ├── status_handler_test.go  # Status handler tests (TDD)
│   │   │   ├── on_status_handler.go   # /on_status callback handler (receiving - optional, for callback acknowledgments)
│   │   │   ├── on_status_handler_test.go # OnStatus handler tests (TDD)
│   │   │   ├── track_handler.go       # /track endpoint handler
│   │   │   ├── track_handler_test.go  # Track handler tests (TDD)
│   │   │   ├── on_track_handler.go    # /on_track callback handler (receiving - optional, for callback acknowledgments)
│   │   │   ├── on_track_handler_test.go # OnTrack handler tests (TDD)
│   │   │   ├── cancel_handler.go      # /cancel endpoint handler
│   │   │   ├── cancel_handler_test.go # Cancel handler tests (TDD)
│   │   │   ├── on_cancel_handler.go    # /on_cancel callback handler (receiving - optional, for callback acknowledgments)
│   │   │   ├── on_cancel_handler_test.go # OnCancel handler tests (TDD)
│   │   │   ├── update_handler.go      # /update endpoint handler
│   │   │   ├── update_handler_test.go  # Update handler tests (TDD)
│   │   │   ├── on_update_handler.go   # /on_update callback handler (receiving - optional, for callback acknowledgments)
│   │   │   ├── on_update_handler_test.go # OnUpdate handler tests (TDD)
│   │   │   └── rto_handler.go        # /rto endpoint handler (handled via /update)
│   │   │   └── rto_handler_test.go   # RTO handler tests (TDD)
│   │   ├── igm/                        # IGM API handlers (Issue & Grievance Management)
│   │   │   ├── issue_handler.go       # /issue endpoint handler
│   │   │   ├── issue_handler_test.go  # Issue handler tests (TDD)
│   │   │   ├── on_issue_handler.go    # /on_issue callback handler (receiving)
│   │   │   ├── on_issue_handler_test.go # OnIssue handler tests (TDD)
│   │   │   ├── issue_status_handler.go # /issue_status endpoint handler
│   │   │   ├── issue_status_handler_test.go # IssueStatus handler tests (TDD)
│   │   │   ├── on_issue_status_handler.go # /on_issue_status callback handler (receiving)
│   │   │   └── on_issue_status_handler_test.go # OnIssueStatus handler tests (TDD)
│   │   └── webhook/                    # Webhook handlers
│   │       ├── zendesk_webhook_handler.go # Zendesk Helpdesk webhook handler
│   │       └── zendesk_webhook_handler_test.go # Zendesk webhook handler tests (TDD)
│   ├── services/
│   │   ├── auth/                      # Authentication and authorization services
│   │   │   ├── auth_service.go        # Client authentication service
│   │   │   ├── auth_service_test.go   # Auth service tests (TDD)
│   │   │   └── rate_limit_service.go  # Rate limiting service
│   │   │   └── rate_limit_service_test.go # Rate limit service tests (TDD)
│   │   ├── callback/                  # Callback delivery services
│   │   │   ├── callback_service.go     # Callback delivery service
│   │   │   ├── callback_service_test.go # Callback service tests (TDD)
│   │   │   └── callback_retry_service.go # Callback retry logic
│   │   │   └── callback_retry_service_test.go # Callback retry tests (TDD)
│   │   ├── transformation/            # Protocol transformation services
│   │   │   ├── ondc_transformation_service.go # ONDC transformation service
│   │   │   ├── ondc_transformation_service_test.go # ONDC transformation tests  (TDD)
│   │   ├── ondc/                      # ONDC-specific business services
│   │   │   ├── search_service.go      # /search flow orchestration
│   │   │   ├── search_service_test.go # Search service tests (TDD)
│   │   │   ├── init_service.go        # /init flow orchestration
│   │   │   ├── init_service_test.go   # Init service tests (TDD)
│   │   │   ├── confirm_service.go     # /confirm flow orchestration
│   │   │   ├── confirm_service_test.go # Confirm service tests (TDD)
│   │   │   ├── status_service.go      # /status flow orchestration
│   │   │   ├── status_service_test.go # Status service tests (TDD)
│   │   │   ├── track_service.go       # /track flow orchestration
│   │   │   ├── track_service_test.go  # Track service tests (TDD)
│   │   │   ├── cancel_service.go      # /cancel flow orchestration
│   │   │   ├── cancel_service_test.go # Cancel service tests (TDD)
│   │   │   ├── update_service.go      # /update flow orchestration
│   │   │   ├── update_service_test.go # Update service tests (TDD)
│   │   │   └── rto_service.go         # /rto flow orchestration
│   │   │   └── rto_service_test.go    # RTO service tests (TDD)
│   │   ├── igm/                       # Issue & Grievance Management services
│   │   │   ├── issue_service.go       # Issue creation and management
│   │   │   ├── issue_service_test.go  # Issue service tests (TDD)
│   │   │   ├── issue_status_service.go # Issue status management
│   │   │   ├── issue_status_service_test.go # Issue status service tests (TDD)
│   │   │   ├── zendesk_service.go     # Zendesk Helpdesk integration service (wraps zendesk_client)
│   │   │   ├── zendesk_service_test.go # Zendesk service tests (TDD)
│   │   │   └── gro_service.go         # GRO (Grievance Redressal Officer) management service
│   │   │   └── gro_service_test.go    # GRO service tests (TDD)
│   │   ├── idempotency/               # Idempotency and deduplication services
│   │   │   ├── idempotency_service.go # Idempotency service
│   │   │   └── idempotency_service_test.go # Idempotency tests (TDD)
│   │   └── client/                    # Client configuration services
│   │       ├── client_config_service.go # Client configuration processing
│   │       └── client_config_service_test.go # Client config tests (TDD)
│   ├── consumers/
│   │   ├── event_consumer.go          # Redis Streams consumer for response events
│   │   └── event_consumer_test.go     # Consumer tests (TDD)
│   ├── repository/
│   │   ├── audit_repository.go        # Postgres-E audit log operations
│   │   ├── audit_repository_test.go   # Audit repository tests (TDD)
│   │   ├── order_mapping_repository.go # Order ID mapping operations
│   │   ├── order_mapping_repository_test.go # Order mapping tests (TDD)
│   │   ├── client_registry_repository.go # Client registry operations
│   │   ├── client_registry_repository_test.go # Client registry tests (TDD)
│   │   ├── idempotency_repository.go  # Idempotency key storage
│   │   ├── idempotency_repository_test.go # Idempotency repository tests (TDD)
│   │   ├── issue_repository.go        # Issue storage (Redis)
│   │   └── issue_repository_test.go   # Issue repository tests (TDD)
│   ├── clients/
│   │   ├── order_client.go            # Order Service gRPC client
│   │   ├── order_client_test.go       # Order client tests (TDD)
│   │   ├── admin_client.go            # Admin Service gRPC client
│   │   ├── admin_client_test.go       # Admin client tests (TDD)
│   │   ├── event_publisher.go        # Event publisher (Redis Streams)
│   │   ├── event_publisher_test.go    # Publisher tests (TDD)
│   │   ├── zendesk_client.go          # Zendesk Helpdesk HTTP client
│   │   └── zendesk_client_test.go     # Zendesk client tests (TDD)
│   ├── models/
│   │   ├── request.go                 # Request domain models
│   │   ├── response.go                # Response domain models
│   │   ├── events.go                   # Event DTOs
│   │   ├── errors.go                  # Domain errors
│   │   ├── client.go                  # Client domain models
│   │   └── issue.go                   # Issue domain models
│   └── utils/
│       ├── validation.go              # Request validation helpers
│       ├── validation_test.go         # Validation tests (TDD)
│       ├── signing.go                  # ONDC request/response signing (ed25519, Blake2b hash)
│       ├── signing_test.go            # Signing tests (TDD)
│       ├── registry_client.go         # ONDC network registry lookup client
│       ├── registry_client_test.go    # Registry client tests (TDD)
│       ├── tracing.go                 # Distributed tracing helpers
│       └── tracing_test.go            # Tracing tests (TDD)
├── contracts/
│   ├── README.md                      # Contracts directory documentation
│   ├── order/
│   │   ├── order.proto                # Order Service gRPC contract
│   │   └── order.pb.go                # Generated (do not edit)
│   ├── admin/
│   │   ├── admin.proto                # Admin Service gRPC contract
│   │   └── admin.pb.go                 # Generated (do not edit)
│   ├── events/
│   │   ├── produced/                  # Events published BY UOIS Gateway
│   │   │   ├── OWNERS                 # UOIS Gateway team ownership
│   │   │   ├── search_requested.json  # SEARCH_REQUESTED event schema
│   │   │   ├── init_requested.json   # INIT_REQUESTED event schema
│   │   │   ├── confirm_requested.json # CONFIRM_REQUESTED event schema
│   │   │   └── uois_events.proto      # Protobuf definitions for published events
│   │   └── consumed/                  # Events consumed BY UOIS Gateway
│   │       ├── OWNERS                 # Quote/Order Service team ownership
│   │       ├── quote_computed.json   # QUOTE_COMPUTED event schema
│   │       ├── quote_created.json    # QUOTE_CREATED event schema
│   │       ├── quote_invalidated.json # QUOTE_INVALIDATED event schema
│   │       ├── order_confirmed.json  # ORDER_CONFIRMED event schema
│   │       └── order_confirm_failed.json # ORDER_CONFIRM_FAILED event schema
│   └── apis/
│       └── ondc/                      # ONDC external contracts
│           ├── OWNERS                 # UOIS Gateway team ownership
│           └── README.md              # Canonical external-facing contract
├── pkg/
│   └── errors/
│       ├── errors.go                 # Error wrapping utilities
│       └── errors_test.go            # Error tests (TDD)
├── test/
│   ├── integration/                  # Integration tests (optional)
│   └── fixtures/                     # Test fixtures
├── .env.example                       # Environment variables template
├── .gitignore                        # Git ignore rules
├── Makefile                          # Build, test, proto generation
├── go.mod                            # Go module file
├── go.sum                            # Go checksums
└── README.md                         # Project documentation
```

**Create directories:**

```bash
mkdir -p cmd/server
mkdir -p internal/{config,handlers/{ondc,igm,webhook},services/{auth,callback,transformation,ondc,igm,idempotency,client},consumers,repository,clients,models,utils}
mkdir -p contracts/{order,admin,events/produced,events/consumed,apis/ondc}
mkdir -p pkg/errors
mkdir -p test/{integration,fixtures}
```

---

## 🔧 Initial Configuration Files

### 1. `.gitignore`

```gitignore
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
bin/
dist/

# Test binary
*.test

# Output of the go coverage tool
*.out
coverage.html

# Dependency directories
vendor/

# Go workspace file
go.work

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# Environment files
.env
.env.local

# Generated files
*.pb.go
*.pb.gw.go
contracts/**/*.pb.go

# Logs
*.log
logs/

# OS
.DS_Store
Thumbs.db
```

### 2. `.env.example`

```bash
# Server Configuration
SERVER_PORT=8080
SERVER_HOST=localhost
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s

# PostgreSQL-E (Audit Database) Configuration
POSTGRES_E_HOST=localhost
POSTGRES_E_PORT=5432
POSTGRES_E_USER=uois_gateway
POSTGRES_E_PASSWORD=
POSTGRES_E_DB=postgres_audit
POSTGRES_E_SSL_MODE=require
POSTGRES_E_MAX_CONNECTIONS=25
POSTGRES_E_MAX_IDLE_CONNECTIONS=5
POSTGRES_E_CONNECTION_MAX_LIFETIME=5m

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_TLS=false
REDIS_KEY_PREFIX=uois-gateway
REDIS_POOL_SIZE=10
REDIS_MIN_IDLE_CONNS=5

# Order Service gRPC
ORDER_SERVICE_GRPC_HOST=localhost
ORDER_SERVICE_GRPC_PORT=50051
ORDER_SERVICE_GRPC_TIMEOUT=30s

# Admin Service gRPC
ADMIN_SERVICE_GRPC_HOST=localhost
ADMIN_SERVICE_GRPC_PORT=50052
ADMIN_SERVICE_GRPC_TIMEOUT=5s

# Event Streams (Published)
STREAM_SEARCH_REQUESTED=stream.location.search
STREAM_INIT_REQUESTED=stream.uois.init_requested
STREAM_CONFIRM_REQUESTED=stream.uois.confirm_requested

# Event Streams (Consumed)
STREAM_QUOTE_COMPUTED=quote:computed
STREAM_QUOTE_CREATED=stream.uois.quote_created
STREAM_QUOTE_INVALIDATED=stream.uois.quote_invalidated
STREAM_ORDER_CONFIRMED=stream.uois.order_confirmed
STREAM_ORDER_CONFIRM_FAILED=stream.uois.order_confirm_failed
STREAM_CLIENT_EVENTS=stream:admin.client.events

# Consumer Group
CONSUMER_GROUP_NAME=uois-gateway-group
CONSUMER_ID=uois-gateway-1

# TTLs (in seconds) - NO HARDCODED VALUES
REQUEST_CONTEXT_TTL=3600
ORDER_MAPPING_TTL=2592000
IDEMPOTENCY_KEY_TTL=86400
ISSUE_STORAGE_TTL=2592000
CLIENT_CONFIG_CACHE_TTL=900
CLIENT_REGISTRY_CACHE_TTL=300

# Retry Configuration
CALLBACK_MAX_RETRIES=5
CALLBACK_RETRY_BACKOFF_1S=1s
CALLBACK_RETRY_BACKOFF_2S=2s
CALLBACK_RETRY_BACKOFF_4S=4s
CALLBACK_RETRY_BACKOFF_8S=8s
CALLBACK_RETRY_BACKOFF_16S=16s
ORDER_SERVICE_MAX_RETRIES=3
ADMIN_SERVICE_MAX_RETRIES=3
EVENT_PUBLISH_MAX_RETRIES=3

# TTL-Aware Defaults for ONDC Flows
# ONDC Request TTL: PT30S (30 seconds) - callback delivery deadline
# ONDC Quote TTL: PT15M (15 minutes) - quote validity period
# Formula: Total retry duration = sum(CALLBACK_RETRY_BACKOFF_*) <= 30 seconds
# Example: 1s + 2s + 4s + 8s + 16s = 31s (exceeds limit, adjust last retry to 15s)
# Recommended: 1s + 2s + 4s + 8s + 15s = 30s (within ONDC Request TTL)
ONDC_REQUEST_TTL_SECONDS=30
ONDC_QUOTE_TTL_SECONDS=900

# ONDC Configuration
ONDC_NETWORK_REGISTRY_URL=https://registry.ondc.org
ONDC_PRIVATE_KEY_PATH=/etc/uois/ondc_private_key.pem
ONDC_PUBLIC_KEY_PATH=/etc/uois/ondc_public_key.pem
ONDC_TIMESTAMP_WINDOW=300

# Zendesk Helpdesk Configuration
ZENDESK_API_URL=https://helpdesk.example.com/api
ZENDESK_API_EMAIL=
ZENDESK_API_TOKEN=
ZENDESK_WEBHOOK_SECRET=

# Logging
LOG_LEVEL=info
LOG_ENCODING=json

# Distributed Tracing
TRACING_ENABLED=true
TRACING_SAMPLE_RATE=0.1
JAEGER_ENDPOINT=http://localhost:14268/api/traces

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REDIS_KEY_PREFIX=rate_limit:uois
```

### 3. `Makefile`

```makefile
.PHONY: build test clean proto tidy run

# Build the application
build:
	go build ./...

# Run tests
test:
	go test ./... -v

# Run tests with coverage
test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Generate protocol buffers
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		contracts/order/order.proto
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		contracts/admin/admin.proto
	protoc --go_out=. --go_opt=paths=source_relative \
		contracts/events/produced/uois_events.proto

# Clean generated files
clean:
	rm -f contracts/order/*.pb.go
	rm -f contracts/admin/*.pb.go
	rm -f contracts/events/produced/*.pb.go
	rm -f coverage.out coverage.html

# Tidy dependencies
tidy:
	go mod tidy

# Run the application
run:
	go run cmd/server/main.go

# Format code
fmt:
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Verify build and tests
verify: build test
	@echo "✅ Build and tests passed"
```

---

## 📝 Initial File Templates

### 1. `cmd/server/main.go` (Minimal)

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"uois-gateway/internal/config"
	"uois-gateway/internal/consumers"
	"uois-gateway/internal/handlers"
	"uois-gateway/internal/services"
	"uois-gateway/internal/repository"
	"uois-gateway/internal/clients"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Info("Shutting down...")
	cancel()
}
```

### 2. `internal/config/config.go` (Template)

```go
package config

import (
	"fmt"
	"github.com/spf13/viper"
)

type Config struct {
	Server      ServerConfig
	PostgresE   PostgresConfig
	Redis       RedisConfig
	Order       OrderConfig
	Admin       AdminConfig
	Streams     StreamsConfig
	TTL         TTLConfig
	Retry       RetryConfig
	ONDC        ONDCConfig
	Zendesk     ZendeskConfig
	Logging     LoggingConfig
	Tracing     TracingConfig
	RateLimit   RateLimitConfig
}

type ServerConfig struct {
	Port         int
	Host         string
	ReadTimeout  string
	WriteTimeout string
}

type PostgresConfig struct {
	Host                string
	Port                int
	User                string
	Password            string
	DB                  string
	SSLMode             string
	MaxConnections      int
	MaxIdleConnections  int
	ConnectionMaxLifetime string
}

type RedisConfig struct {
	Host         string
	Port         int
	Password     string
	DB           int
	TLS          bool
	KeyPrefix    string
	PoolSize     int
	MinIdleConns int
}

type OrderConfig struct {
	GRPCHost    string
	GRPCPort    int
	GRPCTimeout string
	MaxRetries  int
}

type AdminConfig struct {
	GRPCHost    string
	GRPCPort    int
	GRPCTimeout string
	MaxRetries  int
}

type StreamsConfig struct {
	SearchRequested      string
	InitRequested         string
	ConfirmRequested     string
	QuoteComputed         string
	QuoteCreated          string
	QuoteInvalidated      string
	OrderConfirmed        string
	OrderConfirmFailed    string
	ClientEvents          string
	ConsumerGroupName     string
	ConsumerID            string
}

type TTLConfig struct {
	RequestContext      int
	OrderMapping        int
	IdempotencyKey      int
	IssueStorage        int
	ClientConfigCache   int
	ClientRegistryCache int
	ONDCRequestTTL      int // ONDC Request TTL in seconds (PT30S = 30)
	ONDCQuoteTTL        int // ONDC Quote TTL in seconds (PT15M = 900)
}

type RetryConfig struct {
	CallbackMaxRetries int
	CallbackBackoff    []string
	OrderServiceMaxRetries int
	AdminServiceMaxRetries int
	EventPublishMaxRetries int
}

type ONDCConfig struct {
	NetworkRegistryURL string
	PrivateKeyPath     string
	PublicKeyPath      string
	TimestampWindow    int
}

type ZendeskConfig struct {
	APIURL        string
	APIEmail      string
	APIToken      string
	WebhookSecret string
}

type LoggingConfig struct {
	Level    string
	Encoding string
}

type TracingConfig struct {
	Enabled     bool
	SampleRate   float64
	JaegerEndpoint string
}

type RateLimitConfig struct {
	Enabled      bool
	RedisKeyPrefix string
}

func LoadConfig() (*Config, error) {
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	// Set defaults (these are infrastructure defaults, not business values)
	viper.SetDefault("SERVER_PORT", 8080)
	viper.SetDefault("LOG_LEVEL", "info")

	cfg := &Config{
		Server: ServerConfig{
			Port: viper.GetInt("SERVER_PORT"),
			Host: viper.GetString("SERVER_HOST"),
		},
		// ... load all config sections
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if err := c.validatePostgresE(); err != nil {
		return fmt.Errorf("postgres-e config: %w", err)
	}
	if err := c.validateRedis(); err != nil {
		return fmt.Errorf("redis config: %w", err)
	}
	if err := c.validateOrderService(); err != nil {
		return fmt.Errorf("order service config: %w", err)
	}
	if err := c.validateAdminService(); err != nil {
		return fmt.Errorf("admin service config: %w", err)
	}
	if err := c.validateONDC(); err != nil {
		return fmt.Errorf("ondc config: %w", err)
	}
	return nil
}

func (c *Config) validatePostgresE() error {
	if c.PostgresE.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.PostgresE.Port == 0 {
		return fmt.Errorf("port is required")
	}
	if c.PostgresE.User == "" {
		return fmt.Errorf("user is required")
	}
	if c.PostgresE.DB == "" {
		return fmt.Errorf("database name is required")
	}
	return nil
}

func (c *Config) validateRedis() error {
	if c.Redis.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.Redis.Port == 0 {
		return fmt.Errorf("port is required")
	}
	return nil
}

func (c *Config) validateOrderService() error {
	if c.Order.GRPCHost == "" {
		return fmt.Errorf("grpc host is required")
	}
	if c.Order.GRPCPort == 0 {
		return fmt.Errorf("grpc port is required")
	}
	return nil
}

func (c *Config) validateAdminService() error {
	if c.Admin.GRPCHost == "" {
		return fmt.Errorf("grpc host is required")
	}
	if c.Admin.GRPCPort == 0 {
		return fmt.Errorf("grpc port is required")
	}
	return nil
}

func (c *Config) validateONDC() error {
	if c.ONDC.PrivateKeyPath == "" {
		return fmt.Errorf("private key path is required")
	}
	if c.ONDC.PublicKeyPath == "" {
		return fmt.Errorf("public key path is required")
	}
	return nil
}
```

**CRITICAL:** Keep `LoadConfig()` and `Validate()` functions under 20 lines each. Split if needed.

---

## 📄 Contracts Directory Structure

The `contracts/` directory contains all service contracts, event schemas, and API definitions. **See `contracts/README.md` for full documentation.**

### Ownership Model

Each contract directory has an `OWNERS` file specifying:
- **Owning team** responsible for the contract
- **Review requirements** for changes
- **Contact information** for questions

### `contracts/order/`
- **Purpose:** Order Service gRPC contracts
- **Files:**
  - `order.proto` - Protocol buffer definition for Order Service APIs (GetOrder, GetOrderTracking, CancelOrder, UpdateOrder, InitiateRTO)
  - `order.pb.go` - Generated Go code (do not edit manually)

### `contracts/admin/`
- **Purpose:** Admin Service gRPC contracts
- **Files:**
  - `admin.proto` - Protocol buffer definition for Admin Service APIs (GetClientConfig, AuthenticateClient)
  - `admin.pb.go` - Generated Go code (do not edit manually)

### `contracts/events/produced/` (UOIS Gateway Owned)
- **Purpose:** Events published BY UOIS Gateway
- **Ownership:** UOIS Gateway Team (see `OWNERS` file)
- **Files:**
  - `search_requested.json` - Schema for SEARCH_REQUESTED event
  - `init_requested.json` - Schema for INIT_REQUESTED event
  - `confirm_requested.json` - Schema for CONFIRM_REQUESTED event
  - `uois_events.proto` - Protobuf definitions for published events
- **Critical Requirements:**
  - All events MUST include `search_id` (for `/search` and `/init`) or `quote_id` (for `/confirm`) as correlation key
  - All events MUST include `traceparent` (W3C format) for distributed tracing
  - All events MUST include `event_id` (UUID v4) for event-level deduplication
  - Events published to streams: `stream.location.search`, `stream.uois.init_requested`, `stream.uois.confirm_requested`

### `contracts/events/consumed/` (Not Owned by UOIS Gateway)
- **Purpose:** Events consumed BY UOIS Gateway
- **Ownership:** Quote Service Team / Order Service Team (see `OWNERS` file)
- **Files:**
  - `quote_computed.json` - Schema for QUOTE_COMPUTED event (from Quote Service)
  - `quote_created.json` - Schema for QUOTE_CREATED event (from Order Service)
  - `quote_invalidated.json` - Schema for QUOTE_INVALIDATED event (from Order Service)
  - `order_confirmed.json` - Schema for ORDER_CONFIRMED event (from Order Service)
  - `order_confirm_failed.json` - Schema for ORDER_CONFIRM_FAILED event (from Order Service)
- **Note:** UOIS Gateway consumes these but does not own them. Changes require review from owning team.

### `contracts/apis/ondc/` (UOIS Gateway Owned)
- **Purpose:** ONDC external-facing contracts
- **Ownership:** UOIS Gateway Team (see `OWNERS` file)
- **Files:**
  - `README.md` - Canonical external-facing contract documentation
- **Note:** This document is the canonical mapping between internal events and ONDC `/on_search`, `/on_init`, `/on_confirm`, `/on_status`, `/on_track`, `/on_cancel`, `/on_update` responses.

**Note:** JSON schema files are used for validation and documentation. Protocol buffer files are compiled to Go code using `make proto`.

---

## ✅ Setup Verification Checklist

After initial setup, verify:

1. **Go Module:**
   ```bash
   go mod tidy
   go build ./...
   ```
   ✅ Should succeed (even if main.go is minimal)

2. **Project Structure:**
   ```bash
   tree -L 3 -I 'vendor|.git'
   ```
   ✅ All directories should exist

3. **Dependencies:**
   ```bash
   go list -m all
   ```
   ✅ Should show all required dependencies

4. **Protocol Buffer Setup:**
   ```bash
   protoc --version
   ```
   ✅ Should show protoc version

5. **Test Framework (Testify):**
   ```bash
   go test ./... -v
   ```
   ✅ Should run (may have no tests yet, but should not error)
   ✅ All tests must use **Testify** (`github.com/stretchr/testify/assert` for assertions, `github.com/stretchr/testify/mock` for mocking)

---

## 🚀 Next Steps (After Setup)

1. **Follow TDD Strictly:**
   - Start with `internal/config/config_test.go`
   - Write tests first using **Testify** (`assert` for assertions, `mock` for mocking)
   - Then implement `internal/config/config.go`
   - Verify: `go test ./internal/config -v` passes
   - Verify: `go build ./...` succeeds

2. **Organize by Domain:**
   - **Handlers**: One handler per endpoint in domain subfolders (`handlers/ondc/`, `handlers/igm/`)
   - **Services**: One service per flow in domain subfolders (`services/ondc/`, `services/igm/`)
   - **Separation of Concerns**: Each handler/service file should be focused on a single responsibility
   - **No God Files**: Avoid large files with multiple responsibilities

3. **Follow Implementation Plan:**
   - Use `UOISGateway_FR.md` as guide
   - Implement phase by phase
   - Never skip tests
   - Never leave repo in broken state

4. **Follow Development Rules:**
   - Functions < 20 lines
   - Dependency injection
   - Error wrapping with `%w`
   - No hardcoded business values
   - Only handler layer logs errors
   - Never store context in structs

---

## 📚 Key References

- **Functional Requirements:** `doc/UOISGateway_FR.md`
- **Development Rules:** `doc/DISPATCH_DEV_RULES.md` (or `.cursorrules`)
- **ONDC API Contract:** `doc/ONDC - API Contract for Logistics (v1.2.0).md`

---

## ⚠️ Critical Reminders

1. **Separation of Concerns:** Handlers and services are organized by domain (ondc, igm) in subfolders. Each endpoint has its own handler and service file to avoid god files and enable future multi-client integrations.

2. **Protocol Translation Architecture:** UOIS Gateway translates ONDC/Beckn requests to internal events and gRPC calls, then transforms responses back to ONDC/Beckn format.

3. **Event-Driven Request Processing:** All ONDC APIs (`/search`, `/init`, `/confirm`) follow async ACK + callback pattern. Return HTTP 200 OK immediately, process asynchronously, send callback within TTL.

4. **No Hardcoded Values:** All TTLs, retry counts, callback URLs, stream names, timeouts must come from config.

5. **Client Authentication:** UOIS Gateway maintains local client registry (Postgres-E) synced via events from Admin Service. No synchronous Admin Service calls in hot-path.

6. **Order ID Resolution:** Maintain mappings: `client_order_id` ↔ `dispatch_order_id` ↔ `quote_id` ↔ `search_id` in Redis (30 days) and Postgres-E (7 years).

7. **TDD:** Write tests first using **Testify**, then implementation. Never skip this step. Use `testify/assert` for assertions and `testify/mock` for mocking external dependencies.

8. **Build Verification:** After each change, run `go build ./...` and `go test ./...`.

9. **Distributed Tracing:** Generate W3C `traceparent` at edge, propagate through all events and gRPC calls, extract from consumed events, maintain trace continuity across sync + async hops.

10. **Audit Logging:** Persist all request/response pairs to Postgres-E (`audit.request_response_logs`) with 7-year retention. Include `trace_id`, `search_id`, `quote_id`, `dispatch_order_id`, `transaction_id`, `message_id` for correlation.

11. **Callback Delivery:** All callbacks (`/on_search`, `/on_init`, `/on_confirm`, `/on_status`, `/on_track`, `/on_cancel`, `/on_update`, `/on_issue`, `/on_issue_status`) must be idempotent. Use exponential backoff retry (1s → 2s → 4s → 8s → 16s, max 5 attempts). Total retry duration MUST NOT exceed ONDC Request TTL (`PT30S`).

12. **TTL Handling:** Request TTL (`PT30S`) is callback delivery deadline. Quote TTL (`PT15M`) is quote validity period. These are independent.

13. **Issue & Grievance Management (IGM):** UOIS Gateway acts as bridge between ONDC Network and Zendesk Helpdesk. Maintain bidirectional sync: ONDC issues → Zendesk tickets, Zendesk ticket updates → ONDC status callbacks.

14. **Storage Architecture:** Use Postgres-E (separate database instance) for audit logs (7-year retention). Use Redis for temporary storage (order mappings, idempotency keys, request context, issue storage).

15. **Error Handling:** Map internal errors to ONDC error codes (`65001`-`65021`). Return appropriate HTTP status codes. Include actionable error messages.

16. **ONDC Signing:** Validate incoming ONDC request signatures per ONDC API Contract v1.2.0:
    - Use ed25519 for signing and X25519 for encryption
    - Generate Blake2b hash from UTF-8 byte array of raw payload
    - Verify signatures using public keys fetched from ONDC network registry via `/lookup` API
    - Parse `Authorization` header with format: `keyId="{subscriber_id}|{unique_key_id}|{algorithm}"`
    - Sign all ONDC responses with gateway private key (ed25519)
    - Return HTTP 401 with error code `65002` on verification failure
    - Maintain key rotation capability and local registry cache (TTL: 1 hour)

17. **Client Configuration Processing:** Fetch client config from Admin Service (gRPC), cache in Redis (15 minutes), persist snapshot to Postgres-E client registry for audit.

18. **Event Subscription:** Subscribe to events with correlation IDs (`search_id`, `quote_id`, `dispatch_order_id`). Handle timeout scenarios when events not received within request TTL.

19. **Idempotency:** Track request hashes (ONDC `transaction_id` + `message_id`), support idempotent request replay. Store idempotency keys in Redis (24 hours) and Postgres-E (7 years).

20. **Rate Limiting:** Apply per-client rate limiting. Return HTTP 429 when rate limit exceeded. Support configurable rate limits per client.

21. **Post-confirmation Flows:** All post-confirmation flows (`/status`, `/track`, `/cancel`, `/update`, `/rto`) resolve `client_order_id` to `dispatch_order_id` before calling Order Service gRPC methods.

22. **Logistics Seller NP Role:** UOIS Gateway acts as Logistics Seller NP (BPP) in ONDC network. All ONDC endpoints follow the Logistics API Contract (v1.2.0) for Seller NP responsibilities.

---

## 🔧 Technology Stack

### Go Version
- **Minimum:** Go 1.21+
- **Recommended:** Go 1.22+

### Required Go Libraries

| Library | Purpose | Version |
|---------|---------|---------|
| `github.com/gin-gonic/gin` or `github.com/labstack/echo/v4` | HTTP server framework | Latest |
| `google.golang.org/grpc` | gRPC client for Order/Admin services | Latest |
| `google.golang.org/protobuf` | Protocol buffers | Latest |
| `github.com/redis/go-redis/v9` | Redis client (streams + cache) | Latest |
| `github.com/lib/pq` or `github.com/jackc/pgx/v5` | PostgreSQL driver | Latest |
| `go.uber.org/zap` | Structured logging | Latest |
| `github.com/spf13/viper` | Configuration management | Latest |
| `go.opentelemetry.io/otel` | Distributed tracing | Latest |
| `github.com/go-resty/resty/v2` | HTTP client (callbacks, Zendesk) | Latest |
| `golang.org/x/crypto` | Password hashing (bcrypt/argon2) | Latest |
| `github.com/xeipuuv/gojsonschema` | JSON schema validation | Latest |
| `github.com/stretchr/testify` | Testing framework | Latest |

### Protocols
- **HTTP/HTTPS:** ONDC API endpoints, callbacks, Zendesk Helpdesk
- **gRPC:** Order Service, Admin Service
- **Redis Streams:** Event publishing and consumption
- **PostgreSQL:** Audit logs (Postgres-E)

### Observability Stack
- **Logging:** Zap (structured JSON logging)
- **Tracing:** OpenTelemetry with Jaeger
- **Metrics:** 
  - **Prometheus** (suggested for on-premise/self-hosted deployments)
  - **CloudWatch** (suggested for AWS deployments)
  - Metrics to include: request rate, latency, error rate, callback delivery success/failure rate, event publishing/consumption rate, database connection pool metrics

### Config Management
- **Viper:** Environment variable and config file loading
- **No hardcoded values:** All business values from config

---

## 🌍 Environment Setup

### `.env` Structure

See `.env.example` for complete structure. Key sections:

- **Server Configuration:** Port, host, timeouts
- **PostgreSQL-E:** Audit database connection
- **Redis:** Cache and event streams
- **Service Endpoints:** Order Service, Admin Service gRPC
- **Event Streams:** Published and consumed stream names
- **TTLs:** All time-to-live values (no hardcoded)
- **Retry Configuration:** Exponential backoff intervals
- **ONDC Configuration:** Network registry, signing keys
- **Zendesk Configuration:** Helpdesk API credentials
- **Tracing:** OpenTelemetry settings

### Required Environment Variables

**Critical (must be set):**
- `POSTGRES_E_HOST`, `POSTGRES_E_PORT`, `POSTGRES_E_USER`, `POSTGRES_E_PASSWORD`, `POSTGRES_E_DB`
- `REDIS_HOST`, `REDIS_PORT`
- `ORDER_SERVICE_GRPC_HOST`, `ORDER_SERVICE_GRPC_PORT`
- `ADMIN_SERVICE_GRPC_HOST`, `ADMIN_SERVICE_GRPC_PORT`
- `ONDC_PRIVATE_KEY_PATH` (ed25519 private key for signing responses)
- `ONDC_PUBLIC_KEY_PATH` (ed25519 public key, also registered in ONDC network registry)
- `ONDC_NETWORK_REGISTRY_URL` (default: https://registry.ondc.org)
- `ONDC_TIMESTAMP_WINDOW` (default: 300 seconds for replay protection)
- `ZENDESK_API_URL`, `ZENDESK_API_EMAIL`, `ZENDESK_API_TOKEN`

**Optional (have defaults):**
- `SERVER_PORT` (default: 8080)
- `LOG_LEVEL` (default: info)
- `TRACING_ENABLED` (default: true)
- `RATE_LIMIT_ENABLED` (default: true)

### Config Loading Rules

1. Load from `.env` file (if present)
2. Override with environment variables
3. Validate all required fields
4. Set infrastructure defaults (not business values)

### Secrets Handling

- **Never commit secrets** to repository
- Use environment variables or secret management service (AWS Secrets Manager, HashiCorp Vault)
- Store signing keys in secure location (`/etc/uois/` or equivalent)
- Rotate keys periodically

---

## 🔌 Application Bootstrapping

### Entry Point

`cmd/server/main.go` is the application entry point. It should:

1. Load configuration
2. Initialize logger (Zap)
3. Initialize database connections (Postgres-E, Redis)
4. Initialize gRPC clients (Order Service, Admin Service)
5. Initialize HTTP server with handlers
6. Start event consumers
7. Start HTTP server
8. Handle graceful shutdown

### Dependency Initialization

Follow dependency injection pattern:

```go
// Initialize repositories
auditRepo := repository.NewAuditRepository(postgresDB)
orderMappingRepo := repository.NewOrderMappingRepository(redisClient, postgresDB)
clientRegistryRepo := repository.NewClientRegistryRepository(redisClient, postgresDB)

// Initialize clients
orderClient := clients.NewOrderClient(grpcConn, logger)
adminClient := clients.NewAdminClient(grpcConn, logger)
eventPublisher := clients.NewEventPublisher(redisClient, logger)
zendeskClient := clients.NewZendeskClient(httpClient, logger)

// Initialize services
authService := auth.NewAuthService(clientRegistryRepo, adminClient, logger)
rateLimitService := auth.NewRateLimitService(redisClient, logger)
callbackService := callback.NewCallbackService(httpClient, logger)
callbackRetryService := callback.NewCallbackRetryService(callbackService, logger)
ondcTransformationService := transformation.NewONDCTransformationService(logger)
idempotencyService := idempotency.NewIdempotencyService(orderMappingRepo, logger)

// Initialize ONDC services
searchService := ondc.NewSearchService(eventPublisher, eventConsumer, callbackService, ondcTransformationService, logger)
initService := ondc.NewInitService(eventPublisher, eventConsumer, callbackService, ondcTransformationService, orderClient, logger)
confirmService := ondc.NewConfirmService(eventPublisher, eventConsumer, callbackService, ondcTransformationService, orderClient, logger)
statusService := ondc.NewStatusService(orderClient, callbackService, ondcTransformationService, logger)
trackService := ondc.NewTrackService(orderClient, callbackService, ondcTransformationService, logger)
cancelService := ondc.NewCancelService(orderClient, callbackService, ondcTransformationService, logger)
updateService := ondc.NewUpdateService(orderClient, callbackService, ondcTransformationService, logger)
rtoService := ondc.NewRTOService(orderClient, callbackService, ondcTransformationService, logger)

// Initialize IGM services
zendeskService := igm.NewZendeskService(zendeskClient, logger)
groService := igm.NewGROService(redisClient, logger)
issueService := igm.NewIssueService(issueRepo, zendeskService, groService, callbackService, logger)
issueStatusService := igm.NewIssueStatusService(issueRepo, zendeskService, groService, callbackService, logger)

// Initialize ONDC handlers
searchHandler := ondc.NewSearchHandler(searchService, authService, rateLimitService, idempotencyService, logger)
initHandler := ondc.NewInitHandler(initService, authService, rateLimitService, idempotencyService, logger)
confirmHandler := ondc.NewConfirmHandler(confirmService, authService, rateLimitService, idempotencyService, logger)
statusHandler := ondc.NewStatusHandler(statusService, authService, rateLimitService, idempotencyService, logger)
trackHandler := ondc.NewTrackHandler(trackService, authService, rateLimitService, idempotencyService, logger)
cancelHandler := ondc.NewCancelHandler(cancelService, authService, rateLimitService, idempotencyService, logger)
updateHandler := ondc.NewUpdateHandler(updateService, authService, rateLimitService, idempotencyService, logger)
rtoHandler := ondc.NewRTOHandler(rtoService, authService, rateLimitService, idempotencyService, logger)

// Initialize IGM handlers
issueHandler := igm.NewIssueHandler(issueService, authService, rateLimitService, logger)
issueStatusHandler := igm.NewIssueStatusHandler(issueStatusService, authService, rateLimitService, logger)
```

### Graceful Shutdown

Handle SIGTERM and SIGINT:

1. Stop accepting new HTTP requests
2. Wait for in-flight requests to complete (with timeout)
3. Close event consumers
4. Close database connections
5. Close gRPC connections
6. Flush logs

### Health Checks

Implement health check endpoints:

- `GET /health` - Basic health check (returns 200 if service is up)
- `GET /health/ready` - Readiness check (checks dependencies: Postgres-E, Redis, gRPC clients)
- `GET /health/live` - Liveness check (returns 200 if service is alive)

---

## 📡 Event Handling Model

### How UOIS Consumes Events

UOIS Gateway subscribes to Redis Streams using consumer groups:

1. **Event Streams:**
   - `quote:computed` - QUOTE_COMPUTED events (for `/search` response)
   - `stream.uois.quote_created` - QUOTE_CREATED events (for `/init` response)
   - `stream.uois.quote_invalidated` - QUOTE_INVALIDATED events (for `/init` error response)
   - `stream.uois.order_confirmed` - ORDER_CONFIRMED events (for `/confirm` response)
   - `stream.uois.order_confirm_failed` - ORDER_CONFIRM_FAILED events (for `/confirm` error response)
   - `stream:admin.client.events` - Client registry sync events

2. **Consumer Pattern:**
   - Use Redis Streams consumer groups for reliable consumption
   - Track processed event IDs for idempotency
   - ACK events after successful processing
   - NACK events on failure (retry or DLQ)

3. **Event Correlation:**
   - Match events to pending requests using correlation IDs:
     - `search_id` for `/search` and `/init`
     - `quote_id` for `/init` and `/confirm`
     - `dispatch_order_id` for post-confirmation flows

### How UOIS Publishes Events

UOIS Gateway publishes events to Redis Streams:

1. **Published Events:**
   - `SEARCH_REQUESTED` → `stream.location.search` (for `/search` flow)
   - `INIT_REQUESTED` → `stream.uois.init_requested` (for `/init` flow)
   - `CONFIRM_REQUESTED` → `stream.uois.confirm_requested` (for `/confirm` flow)

2. **Event Payload:**
   - Include `search_id` or `quote_id` as correlation key
   - Include `traceparent` (W3C format) for distributed tracing
   - Include `event_id` (UUID v4) for event-level deduplication
   - Include all required fields per event schema

### Retry, Idempotency, and Ordering Guarantees

1. **Retry Policy:**
   - Event publishing: 3 retries with exponential backoff
   - Callback delivery: 5 retries with exponential backoff (1s → 2s → 4s → 8s → 16s)
   - Total retry duration MUST NOT exceed ONDC Request TTL (`PT30S`)

2. **Idempotency:**
   - Track processed request hashes (ONDC `transaction_id` + `message_id`)
   - Track processed event IDs (Redis Stream message IDs)
   - Support idempotent request replay

3. **Ordering:**
   - Redis Streams provide ordering within a stream
   - Use correlation IDs (`search_id`, `quote_id`) to match events to requests
   - Handle out-of-order events gracefully (store pending requests, match on correlation ID)

---

## 🧪 Testing Strategy

### Unit Tests

- **Location:** `*_test.go` files alongside source files
- **Framework:** Testify (`testify/assert`, `testify/mock`)
- **Coverage:** All handlers, services, repositories, clients, utils
- **Mocking:** Mock all external dependencies (gRPC clients, HTTP clients, Redis, Postgres)

### Integration Tests

- **Location:** `test/integration/`
- **Purpose:** Test full request flows with real dependencies (optional, can use testcontainers)
- **Scope:** End-to-end flows (e.g., `/search` → event → callback)

### Contract Tests

- **Location:** `test/contracts/`
- **Purpose:** Validate event schemas and API contracts
- **Tools:** JSON Schema validation, Protobuf validation

### Recommended Tools

- **Testify:** Assertions and mocking
- **Testcontainers (optional):** Integration tests with real Redis/Postgres
- **httptest:** HTTP handler testing
- **gomock (optional):** Alternative mocking tool

### Test Coverage Requirements

- **Minimum:** 80% code coverage
- **Critical paths:** 100% coverage (authentication, callback delivery, event publishing)

---

## 🏗️ Build & Run Instructions

### Local Setup

1. **Prerequisites:**
   - Go 1.21+
   - PostgreSQL (for Postgres-E)
   - Redis (for streams and cache)
   - Protocol Buffer compiler

2. **Environment Setup:**
   ```bash
   cp .env.example .env
   # Edit .env with your local configuration
   ```

3. **Database Setup:**
   ```bash
   # Create Postgres-E database
   createdb postgres_audit
   # Run migrations (if applicable)
   ```

4. **Dependencies:**
   ```bash
   go mod download
   go mod tidy
   ```

### Build Commands

```bash
# Build all packages
make build
# OR
go build ./...

# Build binary
go build -o bin/uois-gateway cmd/server/main.go
```

### Test Commands

```bash
# Run all tests
make test
# OR
go test ./... -v

# Run tests with coverage
make test-coverage
# OR
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Linting and Formatting

```bash
# Format code
make fmt
# OR
go fmt ./...

# Lint code (requires golangci-lint)
make lint
# OR
golangci-lint run
```

### Running the Service

```bash
# Run directly
make run
# OR
go run cmd/server/main.go

# Run binary
./bin/uois-gateway
```

---

## 📊 Non-Functional Requirements

### Performance Expectations

- **Latency Requirements:**
  - Quote requests (`/search`): < 500ms (p95) - includes event subscription wait
  - Order creation (`/confirm`): < 1s (p95) - includes event subscription wait
  - Status queries (`/status`): < 200ms (p95) - direct gRPC call
  - Callback delivery: < 2s (p95) - HTTP POST to client

- **Throughput:**
  - Support minimum 1000 requests/second
  - Scale horizontally for peak loads
  - Use connection pooling for Postgres-E and Redis

### Latency Bounds

- **Synchronous Operations:**
  - HTTP request validation: < 10ms
  - Client authentication: < 50ms (Redis cache hit)
  - Order ID resolution: < 10ms (Redis cache hit)
  - gRPC calls (Order Service, Admin Service): < 100ms (p95)

- **Asynchronous Operations:**
  - Event publishing: < 50ms
  - Event subscription wait: < 30s (request TTL)
  - Callback delivery: < 2s (p95)

### Reliability Goals

- **Availability:** 99.9% uptime
- **Error Rate:** < 0.1% (5xx errors)
- **Graceful Degradation:** Continue operating when non-critical dependencies fail

### Retry Policies

- **Event Publishing:**
  - Max retries: 3
  - Exponential backoff: 1s → 2s → 4s

- **Callback Delivery:**
  - Max retries: 5
  - Exponential backoff: 1s → 2s → 4s → 8s → 15s (adjusted to fit within PT30S)
  - Total retry duration MUST NOT exceed ONDC Request TTL (`PT30S`)
  - Formula: `sum(CALLBACK_RETRY_BACKOFF_*) <= ONDC_REQUEST_TTL_SECONDS`
  - Example calculation: 1s + 2s + 4s + 8s + 15s = 30s (within limit)

- **gRPC Calls:**
  - Max retries: 3
  - Exponential backoff: 1s → 2s → 4s

### Observability Requirements

- **Logging:**
  - Structured JSON logging (Zap)
  - Log level: INFO for user actions, WARN for retries, ERROR for failures
  - Include `trace_id`, `search_id`, `quote_id`, `dispatch_order_id` in all logs

- **Tracing:**
  - OpenTelemetry with Jaeger
  - Generate `traceparent` at edge
  - Propagate through all events and gRPC calls
  - Sample rate: 10% (configurable)

- **Metrics:**
  - Request rate, latency, error rate
  - Callback delivery success/failure rate
  - Event publishing/consumption rate
  - Database connection pool metrics

### Security Expectations

- **Authentication:**
  - Validate client credentials (Basic/Bearer auth)
  - Validate ONDC request signatures
  - Enforce IP allowlisting (CIDR matching)

- **Authorization:**
  - Enforce client-specific API scopes
  - Rate limiting per client

- **Data Protection:**
  - Sanitize sensitive data in logs (PII, payment details)
  - Encrypt data at rest (Postgres-E, Redis)
  - Encrypt data in transit (TLS for HTTP, gRPC)

---

## 📋 Operational Guidelines

### Logging Standards

- **Format:** Structured JSON (Zap)
- **Levels:**
  - `INFO`: User actions (request received, callback sent, event published)
  - `WARN`: Retries, transient failures
  - `ERROR`: Failures, exceptions
- **Fields:**
  - `trace_id`: Distributed tracing ID
  - `search_id`, `quote_id`, `dispatch_order_id`: Business correlation IDs
  - `client_id`: Client identifier
  - `request_id`: Unique request ID
  - `error`: Error message (if applicable)

- **Only handler layer logs errors** (avoid double logging)

### Alerting Expectations

- **Critical Alerts:**
  - Service unavailable (health check fails)
  - High error rate (> 1% 5xx errors)
  - Database connection failures
  - Redis connection failures
  - gRPC client failures

- **Warning Alerts:**
  - High latency (p95 > SLA)
  - Callback delivery failures (> 5% failure rate)
  - Event publishing failures
  - Rate limit violations

### Failure Handling

- **Circuit Breaker:** Implement circuit breaker for external service calls (Order Service, Admin Service, Zendesk)
- **Dead Letter Queue:** Move failed callbacks to DLQ after max retries
- **Graceful Degradation:** Continue operating when non-critical dependencies fail (e.g., Zendesk Helpdesk)

### Safe Deployment Practices

- **Blue-Green Deployment:** Deploy new version alongside old version, switch traffic gradually
- **Health Checks:** Verify health before routing traffic
- **Rollback Plan:** Ability to rollback to previous version quickly
- **Database Migrations:** Run migrations separately, verify before deployment
- **Feature Flags:** Use feature flags for gradual rollout

---

## 📖 References

### Functional Requirements
- **UOIS Gateway Functional Requirements:** `doc/UOISGateway_FR.md`

### ONDC Specifications
- **ONDC API Contract:** `doc/ONDC - API Contract for Logistics (v1.2.0).md`
- **ONDC Network Registry:** https://registry.ondc.org

### Shared Engineering Guidelines
- **Development Rules:** `doc/DISPATCH_DEV_RULES.md` (or `.cursorrules`)
- **Quote Service Repository Setup Guide:** (Reference for structure and style)

### Related Service Contracts
- **Location Service Contracts:** `/docs/04_DispatchContracts/06_location_service/`
- **Order Service Contracts:** `Order-Service-Dispatch/contracts/`
- **Quote Service Contracts:** `Quote-Service-Dispatch/contracts/`

---

## 🎓 Additional Notes

### ID Stack & Ownership

**UOIS Gateway Responsibilities:**
- **Generates**: `trace_id` (via W3C `traceparent` header at edge)
- **Extracts from auth**: `client_id` (from JWT/API key)
- **Generates**: `search_id` (Serviceability ID for `/search` requests)
- **Passes downstream**: `traceparent`, `client_id`, business IDs (`search_id`, `quote_id`)
- **Never generates or uses**: `correlation_id` (WebSocket Gateway responsibility only)

**ID Stack Summary:**
```
┌────────────────────────────────────┐
│ trace_id                           │  ← observability (generated by UOIS)
├────────────────────────────────────┤
│ correlation_id                     │  ← UI / session (WebSocket Gateway only)
├────────────────────────────────────┤
│ client_id                          │  ← tenant boundary (extracted from auth)
├────────────────────────────────────┤
│ search_id / quote_id /             │
│ dispatch_order_id                  │  ← business lifecycle
├────────────────────────────────────┤
│ event_id                           │  ← event-level idempotency
├────────────────────────────────────┤
│ message_id (Redis Stream ID)       │  ← transport sequencing
└────────────────────────────────────┘
```

**One-line meaning:**
- **trace_id** → *What happened across services* (UOIS generates)
- **correlation_id** → *What belongs to one UI/session* (WebSocket Gateway only)
- **client_id** → *Who owns this business* (extracted from auth)
- **business IDs** → *What the business object is* (search_id/quote_id/dispatch_order_id)
- **event_id** → *Did we already process this event* (for deduplication)
- **message_id** → *Where is this message in the stream* (Redis Streams, ACK only)

**Strict Rules:**
- `trace_id`: Generated by UOIS Gateway, propagated everywhere, logs + spans only, ❌ never business logic
- `correlation_id`: Generated by WebSocket Gateway, ❌ never stored in DB, ❌ never enters core services
- `client_id`: Extracted from auth, passed to all core services, used for pricing/billing/multi-tenancy
- `search_id/quote_id/dispatch_order_id`: Pure business lifecycle IDs, one replaces the other as lifecycle advances
- `event_id`: Generated by event publisher, used only for deduplication, TTL-based storage
- `message_id`: Generated by Redis Streams, used only for ACK/replay/lag monitoring, ❌ never stored in business tables

**One-Line Law:**
> **UOIS Gateway generates `trace_id`, extracts `client_id`, generates `search_id`, passes business IDs downstream, uses `event_id` for deduplication, and NEVER generates or uses `correlation_id` (WebSocket Gateway responsibility exclusively).**

### Service Boundaries

**UOIS Gateway owns:**
- Protocol translation and validation
- Client authentication and rate limiting
- Request/response signing (ONDC)
- Event publishing and subscription
- Callback relay via event consumption
- Idempotency and deduplication
- Issue & Grievance Management (IGM) API endpoints
- Client configuration processing and caching
- Audit logging to Postgres-E

**UOIS Gateway does NOT own:**
- Business logic (pricing, capacity, routing) → Quote Service, Location Service, DroneAI
- Order lifecycle management → Order Service
- Issue resolution and ticket content → External Helpdesk (Zendesk)
- Client configuration source of truth → Admin Service

### Common Request Processing Contract

All ONDC APIs follow the same processing pattern (unless explicitly overridden):

1. **Edge Processing:** Generate `traceparent`, validate auth, validate request
2. **Immediate Response:** Return HTTP 200 OK ACK/NACK immediately (< 1 second)
3. **Asynchronous Processing:** Publish events, subscribe to response events, call Order Service
4. **Callback Delivery:** Send callback within TTL period with retry
5. **Audit & Observability:** Persist to Postgres-E, log with correlation IDs

### Event-Driven Architecture

UOIS Gateway is event-driven:
- Publishes events for async processing (`SEARCH_REQUESTED`, `INIT_REQUESTED`, `CONFIRM_REQUESTED`)
- Subscribes to events for response composition (`QUOTE_COMPUTED`, `QUOTE_CREATED`, `ORDER_CONFIRMED`)
- Uses Redis Streams for reliable event delivery
- Maintains correlation across sync + async hops using `search_id`, `quote_id`, `dispatch_order_id`

### Handler and Service Organization Pattern

**Domain-Based Structure:**
- **Handlers** are organized by domain: `handlers/ondc/`, `handlers/igm/`, `handlers/webhook/`
- **Services** are organized by domain: `services/ondc/`, `services/igm/`, `services/auth/`, `services/callback/`
- Each endpoint has its own handler file (e.g., `search_handler.go`, `init_handler.go`)
- Each flow has its own service file (e.g., `search_service.go`, `init_service.go`)

**Benefits:**
- **No God Files:** Each file has a single, focused responsibility
- **Easy to Navigate:** Developers can quickly find the handler/service for a specific endpoint
- **Independent Testing:** Each handler/service can be tested in isolation
- **Future Multi-Client Support:** New client integrations (e.g., Beckn, custom clients) can be added as new subfolders:
  - `handlers/beckn/` - Beckn protocol handlers
  - `handlers/custom_client/` - Custom client-specific handlers
  - `services/beckn/` - Beckn-specific services
- **Clear Boundaries:** Domain separation makes it clear which code belongs to which API contract

**Example Structure for Future Multi-Client:**
```
handlers/
├── ondc/          # ONDC Logistics Seller NP handlers
├── beckn/         # Beckn protocol handlers (future)
├── igm/           # IGM handlers (ONDC-specific)
└── webhook/       # Webhook handlers (domain-agnostic)

services/
├── ondc/          # ONDC-specific business logic
├── beckn/         # Beckn-specific business logic (future)
├── auth/          # Shared authentication (all clients)
├── callback/      # Shared callback delivery (all clients)
└── transformation/ # Protocol transformation (ondc, beckn, etc.)
```

---

**End of Repository Setup Guide**

