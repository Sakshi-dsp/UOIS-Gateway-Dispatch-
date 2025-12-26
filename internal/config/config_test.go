package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_Success(t *testing.T) {
	// Set required environment variables
	os.Setenv("POSTGRES_E_HOST", "localhost")
	os.Setenv("POSTGRES_E_PORT", "5432")
	os.Setenv("POSTGRES_E_USER", "test_user")
	os.Setenv("POSTGRES_E_DB", "test_db")
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("REDIS_PORT", "6379")
	os.Setenv("ORDER_SERVICE_GRPC_HOST", "localhost")
	os.Setenv("ORDER_SERVICE_GRPC_PORT", "50051")
	os.Setenv("ADMIN_SERVICE_GRPC_HOST", "localhost")
	os.Setenv("ADMIN_SERVICE_GRPC_PORT", "50052")
	os.Setenv("ONDC_PRIVATE_KEY_PATH", "/test/private.pem")
	os.Setenv("ONDC_PUBLIC_KEY_PATH", "/test/public.pem")
	defer func() {
		os.Unsetenv("POSTGRES_E_HOST")
		os.Unsetenv("POSTGRES_E_PORT")
		os.Unsetenv("POSTGRES_E_USER")
		os.Unsetenv("POSTGRES_E_DB")
		os.Unsetenv("REDIS_HOST")
		os.Unsetenv("REDIS_PORT")
		os.Unsetenv("ORDER_SERVICE_GRPC_HOST")
		os.Unsetenv("ORDER_SERVICE_GRPC_PORT")
		os.Unsetenv("ADMIN_SERVICE_GRPC_HOST")
		os.Unsetenv("ADMIN_SERVICE_GRPC_PORT")
		os.Unsetenv("ONDC_PRIVATE_KEY_PATH")
		os.Unsetenv("ONDC_PUBLIC_KEY_PATH")
	}()

	cfg, err := LoadConfig()

	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "localhost", cfg.PostgresE.Host)
	assert.Equal(t, 5432, cfg.PostgresE.Port)
}

func TestLoadConfig_MissingPostgresHost(t *testing.T) {
	os.Setenv("POSTGRES_E_PORT", "5432")
	os.Setenv("POSTGRES_E_USER", "test_user")
	os.Setenv("POSTGRES_E_DB", "test_db")
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("REDIS_PORT", "6379")
	os.Setenv("ORDER_SERVICE_GRPC_HOST", "localhost")
	os.Setenv("ORDER_SERVICE_GRPC_PORT", "50051")
	os.Setenv("ADMIN_SERVICE_GRPC_HOST", "localhost")
	os.Setenv("ADMIN_SERVICE_GRPC_PORT", "50052")
	os.Setenv("ONDC_PRIVATE_KEY_PATH", "/test/private.pem")
	os.Setenv("ONDC_PUBLIC_KEY_PATH", "/test/public.pem")
	defer func() {
		os.Unsetenv("POSTGRES_E_HOST")
		os.Unsetenv("POSTGRES_E_PORT")
		os.Unsetenv("POSTGRES_E_USER")
		os.Unsetenv("POSTGRES_E_DB")
		os.Unsetenv("REDIS_HOST")
		os.Unsetenv("REDIS_PORT")
		os.Unsetenv("ORDER_SERVICE_GRPC_HOST")
		os.Unsetenv("ORDER_SERVICE_GRPC_PORT")
		os.Unsetenv("ADMIN_SERVICE_GRPC_HOST")
		os.Unsetenv("ADMIN_SERVICE_GRPC_PORT")
		os.Unsetenv("ONDC_PRIVATE_KEY_PATH")
		os.Unsetenv("ONDC_PUBLIC_KEY_PATH")
	}()

	cfg, err := LoadConfig()

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "postgres-e config")
	assert.Contains(t, err.Error(), "host is required")
}

func TestLoadConfig_MissingRedisHost(t *testing.T) {
	os.Setenv("POSTGRES_E_HOST", "localhost")
	os.Setenv("POSTGRES_E_PORT", "5432")
	os.Setenv("POSTGRES_E_USER", "test_user")
	os.Setenv("POSTGRES_E_DB", "test_db")
	os.Setenv("REDIS_PORT", "6379")
	os.Setenv("ORDER_SERVICE_GRPC_HOST", "localhost")
	os.Setenv("ORDER_SERVICE_GRPC_PORT", "50051")
	os.Setenv("ADMIN_SERVICE_GRPC_HOST", "localhost")
	os.Setenv("ADMIN_SERVICE_GRPC_PORT", "50052")
	os.Setenv("ONDC_PRIVATE_KEY_PATH", "/test/private.pem")
	os.Setenv("ONDC_PUBLIC_KEY_PATH", "/test/public.pem")
	defer func() {
		os.Unsetenv("POSTGRES_E_HOST")
		os.Unsetenv("POSTGRES_E_PORT")
		os.Unsetenv("POSTGRES_E_USER")
		os.Unsetenv("POSTGRES_E_DB")
		os.Unsetenv("REDIS_HOST")
		os.Unsetenv("REDIS_PORT")
		os.Unsetenv("ORDER_SERVICE_GRPC_HOST")
		os.Unsetenv("ORDER_SERVICE_GRPC_PORT")
		os.Unsetenv("ADMIN_SERVICE_GRPC_HOST")
		os.Unsetenv("ADMIN_SERVICE_GRPC_PORT")
		os.Unsetenv("ONDC_PRIVATE_KEY_PATH")
		os.Unsetenv("ONDC_PUBLIC_KEY_PATH")
	}()

	cfg, err := LoadConfig()

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "redis config")
	assert.Contains(t, err.Error(), "host is required")
}

func TestLoadConfig_MissingONDCPrivateKey(t *testing.T) {
	os.Setenv("POSTGRES_E_HOST", "localhost")
	os.Setenv("POSTGRES_E_PORT", "5432")
	os.Setenv("POSTGRES_E_USER", "test_user")
	os.Setenv("POSTGRES_E_DB", "test_db")
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("REDIS_PORT", "6379")
	os.Setenv("ORDER_SERVICE_GRPC_HOST", "localhost")
	os.Setenv("ORDER_SERVICE_GRPC_PORT", "50051")
	os.Setenv("ADMIN_SERVICE_GRPC_HOST", "localhost")
	os.Setenv("ADMIN_SERVICE_GRPC_PORT", "50052")
	os.Setenv("ONDC_PUBLIC_KEY_PATH", "/test/public.pem")
	defer func() {
		os.Unsetenv("POSTGRES_E_HOST")
		os.Unsetenv("POSTGRES_E_PORT")
		os.Unsetenv("POSTGRES_E_USER")
		os.Unsetenv("POSTGRES_E_DB")
		os.Unsetenv("REDIS_HOST")
		os.Unsetenv("REDIS_PORT")
		os.Unsetenv("ORDER_SERVICE_GRPC_HOST")
		os.Unsetenv("ORDER_SERVICE_GRPC_PORT")
		os.Unsetenv("ADMIN_SERVICE_GRPC_HOST")
		os.Unsetenv("ADMIN_SERVICE_GRPC_PORT")
		os.Unsetenv("ONDC_PRIVATE_KEY_PATH")
		os.Unsetenv("ONDC_PUBLIC_KEY_PATH")
	}()

	cfg, err := LoadConfig()

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "ondc config")
	assert.Contains(t, err.Error(), "private key path is required")
}

func TestValidate_PostgresE_MissingHost(t *testing.T) {
	cfg := &Config{
		PostgresE: PostgresConfig{
			Port: 5432,
			User: "test_user",
			DB:   "test_db",
		},
	}

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "postgres-e config")
	assert.Contains(t, err.Error(), "host is required")
}

func TestValidate_PostgresE_MissingPort(t *testing.T) {
	cfg := &Config{
		PostgresE: PostgresConfig{
			Host: "localhost",
			User: "test_user",
			DB:   "test_db",
		},
	}

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "postgres-e config")
	assert.Contains(t, err.Error(), "port is required")
}

func TestValidate_PostgresE_MissingUser(t *testing.T) {
	cfg := &Config{
		PostgresE: PostgresConfig{
			Host: "localhost",
			Port: 5432,
			DB:   "test_db",
		},
	}

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "postgres-e config")
	assert.Contains(t, err.Error(), "user is required")
}

func TestValidate_PostgresE_MissingDB(t *testing.T) {
	cfg := &Config{
		PostgresE: PostgresConfig{
			Host: "localhost",
			Port: 5432,
			User: "test_user",
		},
	}

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "postgres-e config")
	assert.Contains(t, err.Error(), "database name is required")
}

func TestValidate_Redis_MissingHost(t *testing.T) {
	cfg := &Config{
		PostgresE: PostgresConfig{
			Host: "localhost",
			Port: 5432,
			User: "test_user",
			DB:   "test_db",
		},
		Redis: RedisConfig{
			Port: 6379,
		},
	}

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis config")
	assert.Contains(t, err.Error(), "host is required")
}

func TestValidate_Redis_MissingPort(t *testing.T) {
	cfg := &Config{
		PostgresE: PostgresConfig{
			Host: "localhost",
			Port: 5432,
			User: "test_user",
			DB:   "test_db",
		},
		Redis: RedisConfig{
			Host: "localhost",
		},
	}

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis config")
	assert.Contains(t, err.Error(), "port is required")
}

func TestValidate_OrderService_MissingHost(t *testing.T) {
	cfg := &Config{
		PostgresE: PostgresConfig{
			Host: "localhost",
			Port: 5432,
			User: "test_user",
			DB:   "test_db",
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
		Order: OrderConfig{
			GRPCPort: 50051,
		},
	}

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "order service config")
	assert.Contains(t, err.Error(), "grpc host is required")
}

func TestValidate_OrderService_MissingPort(t *testing.T) {
	cfg := &Config{
		PostgresE: PostgresConfig{
			Host: "localhost",
			Port: 5432,
			User: "test_user",
			DB:   "test_db",
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
		Order: OrderConfig{
			GRPCHost: "localhost",
		},
	}

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "order service config")
	assert.Contains(t, err.Error(), "grpc port is required")
}

func TestValidate_AdminService_MissingHost(t *testing.T) {
	cfg := &Config{
		PostgresE: PostgresConfig{
			Host: "localhost",
			Port: 5432,
			User: "test_user",
			DB:   "test_db",
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
		Order: OrderConfig{
			GRPCHost: "localhost",
			GRPCPort: 50051,
		},
		Admin: AdminConfig{
			GRPCPort: 50052,
		},
	}

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "admin service config")
	assert.Contains(t, err.Error(), "grpc host is required")
}

func TestValidate_AdminService_MissingPort(t *testing.T) {
	cfg := &Config{
		PostgresE: PostgresConfig{
			Host: "localhost",
			Port: 5432,
			User: "test_user",
			DB:   "test_db",
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
		Order: OrderConfig{
			GRPCHost: "localhost",
			GRPCPort: 50051,
		},
		Admin: AdminConfig{
			GRPCHost: "localhost",
		},
	}

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "admin service config")
	assert.Contains(t, err.Error(), "grpc port is required")
}

func TestValidate_ONDC_MissingPrivateKey(t *testing.T) {
	cfg := &Config{
		PostgresE: PostgresConfig{
			Host: "localhost",
			Port: 5432,
			User: "test_user",
			DB:   "test_db",
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
		Order: OrderConfig{
			GRPCHost: "localhost",
			GRPCPort: 50051,
		},
		Admin: AdminConfig{
			GRPCHost: "localhost",
			GRPCPort: 50052,
		},
		ONDC: ONDCConfig{
			PublicKeyPath: "/test/public.pem",
		},
	}

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ondc config")
	assert.Contains(t, err.Error(), "private key path is required")
}

func TestValidate_ONDC_MissingPublicKey(t *testing.T) {
	cfg := &Config{
		PostgresE: PostgresConfig{
			Host: "localhost",
			Port: 5432,
			User: "test_user",
			DB:   "test_db",
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
		Order: OrderConfig{
			GRPCHost: "localhost",
			GRPCPort: 50051,
		},
		Admin: AdminConfig{
			GRPCHost: "localhost",
			GRPCPort: 50052,
		},
		ONDC: ONDCConfig{
			PrivateKeyPath: "/test/private.pem",
		},
	}

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ondc config")
	assert.Contains(t, err.Error(), "public key path is required")
}

func TestValidate_Success(t *testing.T) {
	cfg := &Config{
		PostgresE: PostgresConfig{
			Host: "localhost",
			Port: 5432,
			User: "test_user",
			DB:   "test_db",
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
		Order: OrderConfig{
			GRPCHost: "localhost",
			GRPCPort: 50051,
		},
		Admin: AdminConfig{
			GRPCHost: "localhost",
			GRPCPort: 50052,
		},
		ONDC: ONDCConfig{
			PrivateKeyPath: "/test/private.pem",
			PublicKeyPath:  "/test/public.pem",
		},
		TTL: TTLConfig{
			ONDCRequestTTL: 30,
		},
		Retry: RetryConfig{
			CallbackMaxRetries: 5,
			CallbackBackoff:    []int{1, 2, 4, 8, 15},
		},
		Callback: CallbackConfig{
			HTTPTimeoutSeconds: 5,
			MaxConcurrent:      100,
		},
		Streams: StreamsConfig{
			ConsumerID: "test-consumer-123",
		},
	}

	err := cfg.Validate()

	assert.NoError(t, err)
}

func TestValidate_RetryBackoffExceedsTTL(t *testing.T) {
	cfg := &Config{
		PostgresE: PostgresConfig{
			Host: "localhost",
			Port: 5432,
			User: "test_user",
			DB:   "test_db",
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
		Order: OrderConfig{
			GRPCHost: "localhost",
			GRPCPort: 50051,
		},
		Admin: AdminConfig{
			GRPCHost: "localhost",
			GRPCPort: 50052,
		},
		ONDC: ONDCConfig{
			PrivateKeyPath: "/test/private.pem",
			PublicKeyPath:  "/test/public.pem",
		},
		TTL: TTLConfig{
			ONDCRequestTTL: 30,
		},
		Retry: RetryConfig{
			CallbackMaxRetries: 5,
			CallbackBackoff:    []int{1, 2, 4, 8, 16}, // Sum = 31 > 30
		},
		Callback: CallbackConfig{
			HTTPTimeoutSeconds: 5,
			MaxConcurrent:      100,
		},
	}

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "retry config")
	assert.Contains(t, err.Error(), "exceeds ONDC request TTL")
}

func TestValidate_CallbackConfig_InvalidTimeout(t *testing.T) {
	cfg := &Config{
		PostgresE: PostgresConfig{
			Host: "localhost",
			Port: 5432,
			User: "test_user",
			DB:   "test_db",
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
		Order: OrderConfig{
			GRPCHost: "localhost",
			GRPCPort: 50051,
		},
		Admin: AdminConfig{
			GRPCHost: "localhost",
			GRPCPort: 50052,
		},
		ONDC: ONDCConfig{
			PrivateKeyPath: "/test/private.pem",
			PublicKeyPath:  "/test/public.pem",
		},
		TTL: TTLConfig{
			ONDCRequestTTL: 30,
		},
		Retry: RetryConfig{
			CallbackMaxRetries: 5,
			CallbackBackoff:    []int{1, 2, 4, 8, 15},
		},
		Callback: CallbackConfig{
			HTTPTimeoutSeconds: 0, // Invalid
			MaxConcurrent:      100,
		},
	}

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "callback config")
	assert.Contains(t, err.Error(), "http timeout")
}

func TestValidate_CallbackConfig_DLQEnabledWithoutStream(t *testing.T) {
	cfg := &Config{
		PostgresE: PostgresConfig{
			Host: "localhost",
			Port: 5432,
			User: "test_user",
			DB:   "test_db",
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
		Order: OrderConfig{
			GRPCHost: "localhost",
			GRPCPort: 50051,
		},
		Admin: AdminConfig{
			GRPCHost: "localhost",
			GRPCPort: 50052,
		},
		ONDC: ONDCConfig{
			PrivateKeyPath: "/test/private.pem",
			PublicKeyPath:  "/test/public.pem",
		},
		TTL: TTLConfig{
			ONDCRequestTTL: 30,
		},
		Retry: RetryConfig{
			CallbackMaxRetries: 5,
			CallbackBackoff:    []int{1, 2, 4, 8, 15},
		},
		Callback: CallbackConfig{
			HTTPTimeoutSeconds: 5,
			MaxConcurrent:      100,
			DLQEnabled:         true,
			DLQStream:          "", // Missing
		},
	}

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "callback config")
	assert.Contains(t, err.Error(), "dlq stream is required")
}

func TestValidate_RateLimit_InvalidWhenEnabled(t *testing.T) {
	cfg := &Config{
		PostgresE: PostgresConfig{
			Host: "localhost",
			Port: 5432,
			User: "test_user",
			DB:   "test_db",
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
		Order: OrderConfig{
			GRPCHost: "localhost",
			GRPCPort: 50051,
		},
		Admin: AdminConfig{
			GRPCHost: "localhost",
			GRPCPort: 50052,
		},
		ONDC: ONDCConfig{
			PrivateKeyPath: "/test/private.pem",
			PublicKeyPath:  "/test/public.pem",
		},
		TTL: TTLConfig{
			ONDCRequestTTL: 30,
		},
		Retry: RetryConfig{
			CallbackMaxRetries: 5,
			CallbackBackoff:    []int{1, 2, 4, 8, 15},
		},
		Callback: CallbackConfig{
			HTTPTimeoutSeconds: 5,
			MaxConcurrent:      100,
		},
		RateLimit: RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 0, // Invalid
		},
		Streams: StreamsConfig{
			ConsumerID: "test-consumer-123",
		},
	}

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit config")
	assert.Contains(t, err.Error(), "requests per minute")
}

func TestValidate_Zendesk_PartialConfig(t *testing.T) {
	cfg := &Config{
		PostgresE: PostgresConfig{
			Host: "localhost",
			Port: 5432,
			User: "test_user",
			DB:   "test_db",
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
		Order: OrderConfig{
			GRPCHost: "localhost",
			GRPCPort: 50051,
		},
		Admin: AdminConfig{
			GRPCHost: "localhost",
			GRPCPort: 50052,
		},
		ONDC: ONDCConfig{
			PrivateKeyPath: "/test/private.pem",
			PublicKeyPath:  "/test/public.pem",
		},
		TTL: TTLConfig{
			ONDCRequestTTL: 30,
		},
		Retry: RetryConfig{
			CallbackMaxRetries: 5,
			CallbackBackoff:    []int{1, 2, 4, 8, 15},
		},
		Callback: CallbackConfig{
			HTTPTimeoutSeconds: 5,
			MaxConcurrent:      100,
		},
		Zendesk: ZendeskConfig{
			APIURL: "https://example.zendesk.com",
			// Missing APIEmail, APIToken, WebhookSecret
		},
		Streams: StreamsConfig{
			ConsumerID: "test-consumer-123",
		},
	}

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "zendesk config")
}

func TestValidate_Zendesk_MissingWebhookSecret(t *testing.T) {
	cfg := &Config{
		PostgresE: PostgresConfig{
			Host: "localhost",
			Port: 5432,
			User: "test_user",
			DB:   "test_db",
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
		Order: OrderConfig{
			GRPCHost: "localhost",
			GRPCPort: 50051,
		},
		Admin: AdminConfig{
			GRPCHost: "localhost",
			GRPCPort: 50052,
		},
		ONDC: ONDCConfig{
			PrivateKeyPath: "/test/private.pem",
			PublicKeyPath:  "/test/public.pem",
		},
		TTL: TTLConfig{
			ONDCRequestTTL: 30,
		},
		Retry: RetryConfig{
			CallbackMaxRetries: 5,
			CallbackBackoff:    []int{1, 2, 4, 8, 15},
		},
		Callback: CallbackConfig{
			HTTPTimeoutSeconds: 5,
			MaxConcurrent:      100,
		},
		Zendesk: ZendeskConfig{
			APIURL:   "https://example.zendesk.com",
			APIEmail: "test@example.com",
			APIToken: "token123",
			// Missing WebhookSecret
		},
		Streams: StreamsConfig{
			ConsumerID: "test-consumer-123",
		},
	}

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "zendesk config")
	assert.Contains(t, err.Error(), "webhook secret")
}
