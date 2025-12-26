package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig
	PostgresE PostgresConfig
	Redis     RedisConfig
	Order     OrderConfig
	Admin     AdminConfig
	Streams   StreamsConfig
	TTL       TTLConfig
	Retry     RetryConfig
	Callback  CallbackConfig
	ONDC      ONDCConfig
	Zendesk   ZendeskConfig
	Logging   LoggingConfig
	Tracing   TracingConfig
	RateLimit RateLimitConfig
}

type ServerConfig struct {
	Port         int
	Host         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type PostgresConfig struct {
	Host                  string
	Port                  int
	User                  string
	Password              string
	DB                    string
	SSLMode               string
	MaxConnections        int
	MaxIdleConnections    int
	ConnectionMaxLifetime time.Duration
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
	GRPCTimeout time.Duration
	MaxRetries  int
}

type AdminConfig struct {
	GRPCHost    string
	GRPCPort    int
	GRPCTimeout time.Duration
	MaxRetries  int
}

type StreamsConfig struct {
	SearchRequested    string
	InitRequested      string
	ConfirmRequested   string
	QuoteComputed      string
	QuoteCreated       string
	QuoteInvalidated   string
	OrderConfirmed     string
	OrderConfirmFailed string
	ClientEvents       string
	ConsumerGroupName  string
	ConsumerID         string
}

type TTLConfig struct {
	RequestContext      int
	OrderMapping        int
	IdempotencyKey      int
	IssueStorage        int
	ClientConfigCache   int
	ClientRegistryCache int
	ONDCRequestTTL      int
	ONDCQuoteTTL        int
}

type RetryConfig struct {
	CallbackMaxRetries     int
	CallbackBackoff        []int // Backoff durations in seconds
	OrderServiceMaxRetries int
	AdminServiceMaxRetries int
	EventPublishMaxRetries int
}

type CallbackConfig struct {
	HTTPTimeoutSeconds int
	MaxConcurrent      int
	DLQStream          string
	DLQEnabled         bool
}

type ONDCConfig struct {
	NetworkRegistryURL string
	PrivateKeyPath     string
	PublicKeyPath      string
	TimestampWindow    int
	SubscriberID       string
	UkID               string
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
	Enabled        bool
	SampleRate     float64
	JaegerEndpoint string
}

type RateLimitConfig struct {
	Enabled           bool
	RedisKeyPrefix    string
	RequestsPerMinute int
	Burst             int
	WindowSeconds     int
}

func LoadConfig() (*Config, error) {
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	viper.SetDefault("SERVER_PORT", 8080)
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("SERVER_READ_TIMEOUT", "10s")
	viper.SetDefault("SERVER_WRITE_TIMEOUT", "10s")
	viper.SetDefault("ORDER_SERVICE_GRPC_TIMEOUT", "5s")
	viper.SetDefault("ADMIN_SERVICE_GRPC_TIMEOUT", "5s")
	viper.SetDefault("POSTGRES_E_CONNECTION_MAX_LIFETIME", "1h")
	viper.SetDefault("CLIENT_CONFIG_CACHE_TTL", 900)   // 15 minutes
	viper.SetDefault("CLIENT_REGISTRY_CACHE_TTL", 300) // 5 minutes
	viper.SetDefault("ONDC_REQUEST_TTL_SECONDS", 30)   // PT30S
	viper.SetDefault("ONDC_QUOTE_TTL_SECONDS", 900)    // PT15M = 15 minutes
	viper.SetDefault("CALLBACK_MAX_RETRIES", 5)
	viper.SetDefault("CALLBACK_BACKOFF", "1s,2s,4s,8s,15s") // Must sum to ≤ 30s
	viper.SetDefault("CALLBACK_HTTP_TIMEOUT_SECONDS", 5)
	viper.SetDefault("CALLBACK_MAX_CONCURRENT", 100)
	viper.SetDefault("RATE_LIMIT_REQUESTS_PER_MINUTE", 60)
	viper.SetDefault("RATE_LIMIT_BURST", 10)
	viper.SetDefault("RATE_LIMIT_WINDOW_SECONDS", 60)

	readTimeout, err := parseDurationWithDefault(viper.GetString("SERVER_READ_TIMEOUT"), 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("invalid SERVER_READ_TIMEOUT: %w", err)
	}
	writeTimeout, err := parseDurationWithDefault(viper.GetString("SERVER_WRITE_TIMEOUT"), 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("invalid SERVER_WRITE_TIMEOUT: %w", err)
	}

	cfg := &Config{
		Server: ServerConfig{
			Port:         viper.GetInt("SERVER_PORT"),
			Host:         viper.GetString("SERVER_HOST"),
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
		},
		PostgresE: func() PostgresConfig {
			connMaxLifetime, _ := parseDurationWithDefault(viper.GetString("POSTGRES_E_CONNECTION_MAX_LIFETIME"), time.Hour)
			return PostgresConfig{
				Host:                  viper.GetString("POSTGRES_E_HOST"),
				Port:                  viper.GetInt("POSTGRES_E_PORT"),
				User:                  viper.GetString("POSTGRES_E_USER"),
				Password:              viper.GetString("POSTGRES_E_PASSWORD"),
				DB:                    viper.GetString("POSTGRES_E_DB"),
				SSLMode:               viper.GetString("POSTGRES_E_SSL_MODE"),
				MaxConnections:        viper.GetInt("POSTGRES_E_MAX_CONNECTIONS"),
				MaxIdleConnections:    viper.GetInt("POSTGRES_E_MAX_IDLE_CONNECTIONS"),
				ConnectionMaxLifetime: connMaxLifetime,
			}
		}(),
		Redis: RedisConfig{
			Host:         viper.GetString("REDIS_HOST"),
			Port:         viper.GetInt("REDIS_PORT"),
			Password:     viper.GetString("REDIS_PASSWORD"),
			DB:           viper.GetInt("REDIS_DB"),
			TLS:          viper.GetBool("REDIS_TLS"),
			KeyPrefix:    viper.GetString("REDIS_KEY_PREFIX"),
			PoolSize:     viper.GetInt("REDIS_POOL_SIZE"),
			MinIdleConns: viper.GetInt("REDIS_MIN_IDLE_CONNS"),
		},
		Order: func() OrderConfig {
			orderTimeout, err := parseDurationWithDefault(viper.GetString("ORDER_SERVICE_GRPC_TIMEOUT"), 5*time.Second)
			if err != nil {
				orderTimeout = 5 * time.Second
			}
			return OrderConfig{
				GRPCHost:    viper.GetString("ORDER_SERVICE_GRPC_HOST"),
				GRPCPort:    viper.GetInt("ORDER_SERVICE_GRPC_PORT"),
				GRPCTimeout: orderTimeout,
				MaxRetries:  viper.GetInt("ORDER_SERVICE_MAX_RETRIES"),
			}
		}(),
		Admin: func() AdminConfig {
			adminTimeout, err := parseDurationWithDefault(viper.GetString("ADMIN_SERVICE_GRPC_TIMEOUT"), 5*time.Second)
			if err != nil {
				adminTimeout = 5 * time.Second
			}
			return AdminConfig{
				GRPCHost:    viper.GetString("ADMIN_SERVICE_GRPC_HOST"),
				GRPCPort:    viper.GetInt("ADMIN_SERVICE_GRPC_PORT"),
				GRPCTimeout: adminTimeout,
				MaxRetries:  viper.GetInt("ADMIN_SERVICE_MAX_RETRIES"),
			}
		}(),
		Streams: func() StreamsConfig {
			consumerID := viper.GetString("CONSUMER_ID")
			if consumerID == "" {
				consumerID = generateConsumerID()
			}
			return StreamsConfig{
				SearchRequested:    viper.GetString("STREAM_SEARCH_REQUESTED"),
				InitRequested:      viper.GetString("STREAM_INIT_REQUESTED"),
				ConfirmRequested:   viper.GetString("STREAM_CONFIRM_REQUESTED"),
				QuoteComputed:      viper.GetString("STREAM_QUOTE_COMPUTED"),
				QuoteCreated:       viper.GetString("STREAM_QUOTE_CREATED"),
				QuoteInvalidated:   viper.GetString("STREAM_QUOTE_INVALIDATED"),
				OrderConfirmed:     viper.GetString("STREAM_ORDER_CONFIRMED"),
				OrderConfirmFailed: viper.GetString("STREAM_ORDER_CONFIRM_FAILED"),
				ClientEvents:       viper.GetString("STREAM_CLIENT_EVENTS"),
				ConsumerGroupName:  viper.GetString("CONSUMER_GROUP_NAME"),
				ConsumerID:         consumerID,
			}
		}(),
		TTL: TTLConfig{
			RequestContext:      viper.GetInt("REQUEST_CONTEXT_TTL"),
			OrderMapping:        viper.GetInt("ORDER_MAPPING_TTL"),
			IdempotencyKey:      viper.GetInt("IDEMPOTENCY_KEY_TTL"),
			IssueStorage:        viper.GetInt("ISSUE_STORAGE_TTL"),
			ClientConfigCache:   viper.GetInt("CLIENT_CONFIG_CACHE_TTL"),
			ClientRegistryCache: viper.GetInt("CLIENT_REGISTRY_CACHE_TTL"),
			ONDCRequestTTL:      viper.GetInt("ONDC_REQUEST_TTL_SECONDS"),
			ONDCQuoteTTL:        viper.GetInt("ONDC_QUOTE_TTL_SECONDS"),
		},
		Retry: RetryConfig{
			CallbackMaxRetries:     viper.GetInt("CALLBACK_MAX_RETRIES"),
			CallbackBackoff:        parseBackoffDurations(viper.GetString("CALLBACK_BACKOFF")),
			OrderServiceMaxRetries: viper.GetInt("ORDER_SERVICE_MAX_RETRIES"),
			AdminServiceMaxRetries: viper.GetInt("ADMIN_SERVICE_MAX_RETRIES"),
			EventPublishMaxRetries: viper.GetInt("EVENT_PUBLISH_MAX_RETRIES"),
		},
		Callback: CallbackConfig{
			HTTPTimeoutSeconds: viper.GetInt("CALLBACK_HTTP_TIMEOUT_SECONDS"),
			MaxConcurrent:      viper.GetInt("CALLBACK_MAX_CONCURRENT"),
			DLQStream:          viper.GetString("CALLBACK_DLQ_STREAM"),
			DLQEnabled:         viper.GetBool("CALLBACK_DLQ_ENABLED"),
		},
		ONDC: ONDCConfig{
			NetworkRegistryURL: viper.GetString("ONDC_NETWORK_REGISTRY_URL"),
			PrivateKeyPath:     viper.GetString("ONDC_PRIVATE_KEY_PATH"),
			PublicKeyPath:      viper.GetString("ONDC_PUBLIC_KEY_PATH"),
			TimestampWindow:    viper.GetInt("ONDC_TIMESTAMP_WINDOW"),
			SubscriberID:       viper.GetString("ONDC_SUBSCRIBER_ID"),
			UkID:               viper.GetString("ONDC_UK_ID"),
		},
		Zendesk: ZendeskConfig{
			APIURL:        viper.GetString("ZENDESK_API_URL"),
			APIEmail:      viper.GetString("ZENDESK_API_EMAIL"),
			APIToken:      viper.GetString("ZENDESK_API_TOKEN"),
			WebhookSecret: viper.GetString("ZENDESK_WEBHOOK_SECRET"),
		},
		Logging: LoggingConfig{
			Level:    viper.GetString("LOG_LEVEL"),
			Encoding: viper.GetString("LOG_ENCODING"),
		},
		Tracing: TracingConfig{
			Enabled:        viper.GetBool("TRACING_ENABLED"),
			SampleRate:     viper.GetFloat64("TRACING_SAMPLE_RATE"),
			JaegerEndpoint: viper.GetString("JAEGER_ENDPOINT"),
		},
		RateLimit: RateLimitConfig{
			Enabled:           viper.GetBool("RATE_LIMIT_ENABLED"),
			RedisKeyPrefix:    viper.GetString("RATE_LIMIT_REDIS_KEY_PREFIX"),
			RequestsPerMinute: viper.GetInt("RATE_LIMIT_REQUESTS_PER_MINUTE"),
			Burst:             viper.GetInt("RATE_LIMIT_BURST"),
			WindowSeconds:     viper.GetInt("RATE_LIMIT_WINDOW_SECONDS"),
		},
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
	if err := c.validateRetryVsTTL(); err != nil {
		return fmt.Errorf("retry config: %w", err)
	}
	if err := c.validateCallback(); err != nil {
		return fmt.Errorf("callback config: %w", err)
	}
	if err := c.validateRateLimit(); err != nil {
		return fmt.Errorf("rate limit config: %w", err)
	}
	if err := c.validateZendesk(); err != nil {
		return fmt.Errorf("zendesk config: %w", err)
	}
	if err := c.validateStreams(); err != nil {
		return fmt.Errorf("streams config: %w", err)
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
	if c.ONDC.SubscriberID == "" {
		return fmt.Errorf("subscriber id is required")
	}
	if c.ONDC.UkID == "" {
		return fmt.Errorf("uk id is required")
	}
	return nil
}

func (c *Config) validateRetryVsTTL() error {
	if len(c.Retry.CallbackBackoff) == 0 {
		return fmt.Errorf("callback backoff must not be empty")
	}
	if c.Retry.CallbackMaxRetries <= 0 {
		return fmt.Errorf("callback max retries must be greater than 0")
	}
	if len(c.Retry.CallbackBackoff) < c.Retry.CallbackMaxRetries {
		return fmt.Errorf("callback backoff array length (%d) must be >= max retries (%d)", len(c.Retry.CallbackBackoff), c.Retry.CallbackMaxRetries)
	}
	totalBackoff := sumBackoff(c.Retry.CallbackBackoff[:c.Retry.CallbackMaxRetries])
	if c.TTL.ONDCRequestTTL <= 0 {
		return fmt.Errorf("ondc request ttl must be greater than 0")
	}
	if totalBackoff > c.TTL.ONDCRequestTTL {
		return fmt.Errorf("callback retry backoff total (%ds) exceeds ONDC request TTL (%ds): sum(CALLBACK_BACKOFF) must be <= ONDC_REQUEST_TTL_SECONDS", totalBackoff, c.TTL.ONDCRequestTTL)
	}
	return nil
}

func (c *Config) validateCallback() error {
	if c.Callback.HTTPTimeoutSeconds <= 0 {
		return fmt.Errorf("callback http timeout must be greater than 0")
	}
	if c.Callback.MaxConcurrent <= 0 {
		return fmt.Errorf("callback max concurrent must be greater than 0")
	}
	if c.Callback.DLQEnabled && c.Callback.DLQStream == "" {
		return fmt.Errorf("callback dlq stream is required when dlq is enabled")
	}
	return nil
}

func (c *Config) validateRateLimit() error {
	if !c.RateLimit.Enabled {
		return nil
	}
	if c.RateLimit.RequestsPerMinute <= 0 {
		return fmt.Errorf("rate limit requests per minute must be greater than 0 when enabled")
	}
	if c.RateLimit.Burst <= 0 {
		return fmt.Errorf("rate limit burst must be greater than 0 when enabled")
	}
	if c.RateLimit.WindowSeconds <= 0 {
		return fmt.Errorf("rate limit window seconds must be greater than 0 when enabled")
	}
	return nil
}

func (c *Config) validateZendesk() error {
	if c.Zendesk.APIURL == "" && c.Zendesk.APIEmail == "" && c.Zendesk.APIToken == "" {
		return nil
	}
	if c.Zendesk.APIURL == "" {
		return fmt.Errorf("zendesk api url is required when zendesk is configured")
	}
	if c.Zendesk.APIEmail == "" {
		return fmt.Errorf("zendesk api email is required when zendesk is configured")
	}
	if c.Zendesk.APIToken == "" {
		return fmt.Errorf("zendesk api token is required when zendesk is configured")
	}
	if c.Zendesk.WebhookSecret == "" {
		return fmt.Errorf("zendesk webhook secret is required when zendesk is configured (for IGM webhook validation)")
	}
	return nil
}

func parseBackoffDurations(backoffStr string) []int {
	if backoffStr == "" {
		return []int{1, 2, 4, 8, 15}
	}
	parts := strings.Split(backoffStr, ",")
	durations := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		duration, err := parseDuration(part)
		if err != nil {
			continue
		}
		durations = append(durations, int(duration.Seconds()))
	}
	if len(durations) == 0 {
		return []int{1, 2, 4, 8, 15}
	}
	return durations
}

func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration string")
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	return d, nil
}

func sumBackoff(backoff []int) int {
	sum := 0
	for _, b := range backoff {
		sum += b
	}
	return sum
}

func parseDurationWithDefault(s string, defaultVal time.Duration) (time.Duration, error) {
	if s == "" {
		return defaultVal, nil
	}
	return parseDuration(s)
}

func generateConsumerID() string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	pid := os.Getpid()
	return fmt.Sprintf("%s-%d", hostname, pid)
}

func (c *Config) validateStreams() error {
	if c.Streams.ConsumerID == "" {
		return fmt.Errorf("consumer id must not be empty (auto-generation failed)")
	}
	if c.Streams.ConsumerGroupName == "" && (c.Streams.QuoteComputed != "" || c.Streams.QuoteCreated != "" || c.Streams.OrderConfirmed != "") {
		return fmt.Errorf("consumer group name is required when event consumption is enabled")
	}
	return nil
}
