package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Service provides Prometheus metrics for UOIS Gateway
type Service struct {
	// Business Metrics
	requestsTotal           *prometheus.CounterVec
	ordersCreatedTotal      prometheus.Counter
	quotesComputedTotal     prometheus.Counter
	quotesCreatedTotal      prometheus.Counter
	callbacksDeliveredTotal *prometheus.CounterVec
	callbacksFailedTotal    *prometheus.CounterVec
	issuesCreatedTotal      prometheus.Counter
	issuesResolvedTotal     prometheus.Counter

	// Latency Metrics
	requestDuration          *prometheus.HistogramVec
	callbackDeliveryDuration *prometheus.HistogramVec
	eventProcessingDuration  *prometheus.HistogramVec
	dbQueryDuration          *prometheus.HistogramVec
	grpcCallDuration         *prometheus.HistogramVec
	authDuration             *prometheus.HistogramVec

	// Error Metrics
	errorsTotal            *prometheus.CounterVec
	errorsByCategory       *prometheus.CounterVec
	timeoutsTotal          *prometheus.CounterVec
	rateLimitExceededTotal *prometheus.CounterVec
	callbackRetriesTotal   prometheus.Counter

	// Service Health Metrics
	serviceAvailability       prometheus.Gauge
	dependenciesHealth        *prometheus.GaugeVec
	dependenciesLatency       *prometheus.HistogramVec
	circuitBreakerState       *prometheus.GaugeVec
	dbConnectionPoolActive    prometheus.Gauge
	dbConnectionPoolIdle      prometheus.Gauge
	redisConnectionPoolActive prometheus.Gauge

	// Cache Metrics
	cacheHitsTotal      *prometheus.CounterVec
	cacheMissesTotal    *prometheus.CounterVec
	cacheHitRate        *prometheus.GaugeVec
	cacheSize           *prometheus.GaugeVec
	cacheEvictionsTotal *prometheus.CounterVec

	// Idempotency Metrics
	idempotencyDuplicateRequestsTotal prometheus.Counter
	idempotencyReplaysTotal           prometheus.Counter

	// ONDC-Specific Metrics
	ondcSignatureVerificationsTotal prometheus.Counter
	ondcSignatureGenerationsTotal   prometheus.Counter
	ondcRegistryLookupsTotal        prometheus.Counter
	ondcTimestampValidationsTotal   prometheus.Counter

	// IGM Metrics
	igmIssuesByStatus       *prometheus.GaugeVec
	igmIssuesResolutionTime *prometheus.HistogramVec

	// Database Metrics
	dbAuditLogsWrittenTotal *prometheus.CounterVec
	dbAuditLogsSize         prometheus.Gauge
	dbQueryErrorsTotal      prometheus.Counter
	dbTransactionDuration   *prometheus.HistogramVec

	// SLO/SLI Metrics
	latencySLIViolations      *prometheus.CounterVec
	availabilitySLI           prometheus.Gauge
	availabilitySLIViolations prometheus.Counter
}

// NewService creates a new metrics service
func NewService(serviceName, environment string) *Service {
	return &Service{
		// Business Metrics
		requestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "uois_requests_total",
				Help: "Total number of requests by endpoint, client_id, protocol, status",
			},
			[]string{"endpoint", "client_id", "protocol", "status"},
		),
		ordersCreatedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "uois_orders_created_total",
				Help: "Total number of orders created via /confirm",
			},
		),
		quotesComputedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "uois_quotes_computed_total",
				Help: "Total number of quotes computed for /search",
			},
		),
		quotesCreatedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "uois_quotes_created_total",
				Help: "Total number of quotes created for /init",
			},
		),
		callbacksDeliveredTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "uois_callbacks_delivered_total",
				Help: "Total number of callbacks delivered successfully",
			},
			[]string{"callback_type", "client_id"},
		),
		callbacksFailedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "uois_callbacks_failed_total",
				Help: "Total number of callback delivery failures",
			},
			[]string{"callback_type", "client_id", "error_code"},
		),
		issuesCreatedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "uois_issues_created_total",
				Help: "Total number of IGM issues created",
			},
		),
		issuesResolvedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "uois_issues_resolved_total",
				Help: "Total number of IGM issues resolved",
			},
		),

		// Latency Metrics
		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "uois_request_duration_seconds",
				Help:    "Request processing time in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"endpoint", "status"},
		),
		callbackDeliveryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "uois_callback_delivery_duration_seconds",
				Help:    "Callback delivery time in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"callback_type"},
		),
		eventProcessingDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "uois_event_processing_duration_seconds",
				Help:    "Event processing time in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"event_type"},
		),
		dbQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "uois_db_query_duration_seconds",
				Help:    "Database query time in seconds",
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"query_type"},
		),
		grpcCallDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "uois_grpc_call_duration_seconds",
				Help:    "gRPC call duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"service", "method"},
		),
		authDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "uois_auth_duration_seconds",
				Help:    "Authentication/authorization time in seconds",
				Buckets: []float64{.001, .005, .01, .025, .05, .1},
			},
			[]string{"auth_type"},
		),

		// Error Metrics
		errorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "uois_errors_total",
				Help: "Total errors by error_code, endpoint, client_id",
			},
			[]string{"error_code", "endpoint", "client_id"},
		),
		errorsByCategory: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "uois_errors_by_category_total",
				Help: "Errors by category (validation, auth, dependency, internal)",
			},
			[]string{"category"},
		),
		timeoutsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "uois_timeouts_total",
				Help: "Request timeouts by endpoint, dependency",
			},
			[]string{"endpoint", "dependency"},
		),
		rateLimitExceededTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "uois_rate_limit_exceeded_total",
				Help: "Rate limit violations",
			},
			[]string{"client_id"},
		),
		callbackRetriesTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "uois_callback_retries_total",
				Help: "Callback retry attempts",
			},
		),

		// Service Health Metrics
		serviceAvailability: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "uois_service_availability",
				Help: "Service availability (1 = healthy, 0 = unhealthy)",
			},
		),
		dependenciesHealth: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "uois_dependencies_health",
				Help: "Dependency health status (1 = healthy, 0 = unhealthy)",
			},
			[]string{"dependency"},
		),
		dependenciesLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "uois_dependencies_latency_seconds",
				Help:    "Dependency latency in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"dependency"},
		),
		circuitBreakerState: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "uois_circuit_breaker_state",
				Help: "Circuit breaker state (0 = closed, 1 = open, 2 = half-open)",
			},
			[]string{"dependency"},
		),
		dbConnectionPoolActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "uois_db_connection_pool_active",
				Help: "Active database connections",
			},
		),
		dbConnectionPoolIdle: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "uois_db_connection_pool_idle",
				Help: "Idle database connections",
			},
		),
		redisConnectionPoolActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "uois_redis_connection_pool_active",
				Help: "Active Redis connections",
			},
		),

		// Cache Metrics
		cacheHitsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "uois_cache_hits_total",
				Help: "Cache hits by cache type",
			},
			[]string{"cache_type"},
		),
		cacheMissesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "uois_cache_misses_total",
				Help: "Cache misses by cache type",
			},
			[]string{"cache_type"},
		),
		cacheHitRate: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "uois_cache_hit_rate",
				Help: "Cache hit rate by cache type",
			},
			[]string{"cache_type"},
		),
		cacheSize: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "uois_cache_size",
				Help: "Cache size by cache type",
			},
			[]string{"cache_type"},
		),
		cacheEvictionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "uois_cache_evictions_total",
				Help: "Cache evictions by cache type",
			},
			[]string{"cache_type"},
		),

		// Idempotency Metrics
		idempotencyDuplicateRequestsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "uois_idempotency_duplicate_requests_total",
				Help: "Duplicate requests detected",
			},
		),
		idempotencyReplaysTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "uois_idempotency_replays_total",
				Help: "Idempotent request replays",
			},
		),

		// ONDC-Specific Metrics
		ondcSignatureVerificationsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "uois_ondc_signature_verifications_total",
				Help: "ONDC signature verifications",
			},
		),
		ondcSignatureGenerationsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "uois_ondc_signature_generations_total",
				Help: "ONDC signature generations",
			},
		),
		ondcRegistryLookupsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "uois_ondc_registry_lookups_total",
				Help: "ONDC network registry lookups",
			},
		),
		ondcTimestampValidationsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "uois_ondc_timestamp_validations_total",
				Help: "Timestamp validations",
			},
		),

		// IGM Metrics
		igmIssuesByStatus: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "uois_igm_issues_by_status",
				Help: "Current issue count by status",
			},
			[]string{"status"},
		),
		igmIssuesResolutionTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "uois_igm_issues_resolution_time_seconds",
				Help:    "Issue resolution time in seconds",
				Buckets: []float64{60, 300, 600, 1800, 3600, 7200, 86400},
			},
			[]string{"issue_type"},
		),

		// Database Metrics
		dbAuditLogsWrittenTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "uois_db_audit_logs_written_total",
				Help: "Audit log writes by status",
			},
			[]string{"status"},
		),
		dbAuditLogsSize: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "uois_db_audit_logs_size",
				Help: "Total audit log entries",
			},
		),
		dbQueryErrorsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "uois_db_query_errors_total",
				Help: "Database query errors",
			},
		),
		dbTransactionDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "uois_db_transaction_duration_seconds",
				Help:    "Database transaction duration in seconds",
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
			},
			[]string{"transaction_type"},
		),

		// SLO/SLI Metrics
		latencySLIViolations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "uois_latency_sli_violations_total",
				Help: "Total latency SLO violations by endpoint",
			},
			[]string{"endpoint"},
		),
		availabilitySLI: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "uois_availability_sli",
				Help: "Current availability SLI (0-1)",
			},
		),
		availabilitySLIViolations: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "uois_availability_sli_violations_total",
				Help: "Total availability SLO violations",
			},
		),
	}
}

// RecordRequest records a request metric
func (s *Service) RecordRequest(endpoint, clientID, protocol, status string) {
	s.requestsTotal.WithLabelValues(endpoint, clientID, protocol, status).Inc()
}

// RecordRequestDuration records request processing duration
func (s *Service) RecordRequestDuration(endpoint, status string, duration time.Duration) {
	s.requestDuration.WithLabelValues(endpoint, status).Observe(duration.Seconds())
}

// RecordOrderCreated records an order creation
func (s *Service) RecordOrderCreated() {
	s.ordersCreatedTotal.Inc()
}

// RecordQuoteComputed records a quote computation
func (s *Service) RecordQuoteComputed() {
	s.quotesComputedTotal.Inc()
}

// RecordQuoteCreated records a quote creation
func (s *Service) RecordQuoteCreated() {
	s.quotesCreatedTotal.Inc()
}

// RecordCallbackDelivered records a successful callback delivery
func (s *Service) RecordCallbackDelivered(callbackType, clientID string) {
	s.callbacksDeliveredTotal.WithLabelValues(callbackType, clientID).Inc()
}

// RecordCallbackFailed records a failed callback delivery
func (s *Service) RecordCallbackFailed(callbackType, clientID, errorCode string) {
	s.callbacksFailedTotal.WithLabelValues(callbackType, clientID, errorCode).Inc()
}

// RecordCallbackDeliveryDuration records callback delivery duration
func (s *Service) RecordCallbackDeliveryDuration(callbackType string, duration time.Duration) {
	s.callbackDeliveryDuration.WithLabelValues(callbackType).Observe(duration.Seconds())
}

// RecordCallbackRetry records a callback retry attempt
func (s *Service) RecordCallbackRetry() {
	s.callbackRetriesTotal.Inc()
}

// RecordError records an error
func (s *Service) RecordError(errorCode, endpoint, clientID string) {
	s.errorsTotal.WithLabelValues(errorCode, endpoint, clientID).Inc()
}

// RecordErrorByCategory records an error by category
func (s *Service) RecordErrorByCategory(category string) {
	s.errorsByCategory.WithLabelValues(category).Inc()
}

// RecordTimeout records a timeout
func (s *Service) RecordTimeout(endpoint, dependency string) {
	s.timeoutsTotal.WithLabelValues(endpoint, dependency).Inc()
}

// RecordRateLimitExceeded records a rate limit violation
func (s *Service) RecordRateLimitExceeded(clientID string) {
	s.rateLimitExceededTotal.WithLabelValues(clientID).Inc()
}

// RecordEventProcessingDuration records event processing duration
func (s *Service) RecordEventProcessingDuration(eventType string, duration time.Duration) {
	s.eventProcessingDuration.WithLabelValues(eventType).Observe(duration.Seconds())
}

// RecordDBQueryDuration records database query duration
func (s *Service) RecordDBQueryDuration(queryType string, duration time.Duration) {
	s.dbQueryDuration.WithLabelValues(queryType).Observe(duration.Seconds())
}

// RecordDBQueryError records a database query error
func (s *Service) RecordDBQueryError() {
	s.dbQueryErrorsTotal.Inc()
}

// RecordGRPCCallDuration records gRPC call duration
func (s *Service) RecordGRPCCallDuration(service, method string, duration time.Duration) {
	s.grpcCallDuration.WithLabelValues(service, method).Observe(duration.Seconds())
}

// RecordAuthDuration records authentication duration
func (s *Service) RecordAuthDuration(authType string, duration time.Duration) {
	s.authDuration.WithLabelValues(authType).Observe(duration.Seconds())
}

// SetServiceAvailability sets service availability status
func (s *Service) SetServiceAvailability(available bool) {
	if available {
		s.serviceAvailability.Set(1)
	} else {
		s.serviceAvailability.Set(0)
	}
}

// SetDependencyHealth sets dependency health status
func (s *Service) SetDependencyHealth(dependency string, healthy bool) {
	if healthy {
		s.dependenciesHealth.WithLabelValues(dependency).Set(1)
	} else {
		s.dependenciesHealth.WithLabelValues(dependency).Set(0)
	}
}

// RecordDependencyLatency records dependency latency
func (s *Service) RecordDependencyLatency(dependency string, duration time.Duration) {
	s.dependenciesLatency.WithLabelValues(dependency).Observe(duration.Seconds())
}

// SetCircuitBreakerState sets circuit breaker state
func (s *Service) SetCircuitBreakerState(dependency string, state int) {
	s.circuitBreakerState.WithLabelValues(dependency).Set(float64(state))
}

// SetDBConnectionPoolActive sets active database connections
func (s *Service) SetDBConnectionPoolActive(count float64) {
	s.dbConnectionPoolActive.Set(count)
}

// SetDBConnectionPoolIdle sets idle database connections
func (s *Service) SetDBConnectionPoolIdle(count float64) {
	s.dbConnectionPoolIdle.Set(count)
}

// SetRedisConnectionPoolActive sets active Redis connections
func (s *Service) SetRedisConnectionPoolActive(count float64) {
	s.redisConnectionPoolActive.Set(count)
}

// RecordCacheHit records a cache hit
func (s *Service) RecordCacheHit(cacheType string) {
	s.cacheHitsTotal.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss records a cache miss
func (s *Service) RecordCacheMiss(cacheType string) {
	s.cacheMissesTotal.WithLabelValues(cacheType).Inc()
}

// SetCacheHitRate sets cache hit rate
func (s *Service) SetCacheHitRate(cacheType string, rate float64) {
	s.cacheHitRate.WithLabelValues(cacheType).Set(rate)
}

// SetCacheSize sets cache size
func (s *Service) SetCacheSize(cacheType string, size float64) {
	s.cacheSize.WithLabelValues(cacheType).Set(size)
}

// RecordCacheEviction records a cache eviction
func (s *Service) RecordCacheEviction(cacheType string) {
	s.cacheEvictionsTotal.WithLabelValues(cacheType).Inc()
}

// RecordIdempotencyDuplicate records a duplicate request
func (s *Service) RecordIdempotencyDuplicate() {
	s.idempotencyDuplicateRequestsTotal.Inc()
}

// RecordIdempotencyReplay records an idempotent replay
func (s *Service) RecordIdempotencyReplay() {
	s.idempotencyReplaysTotal.Inc()
}

// RecordONDCSignatureVerification records an ONDC signature verification
func (s *Service) RecordONDCSignatureVerification() {
	s.ondcSignatureVerificationsTotal.Inc()
}

// RecordONDCSignatureGeneration records an ONDC signature generation
func (s *Service) RecordONDCSignatureGeneration() {
	s.ondcSignatureGenerationsTotal.Inc()
}

// RecordONDCRegistryLookup records an ONDC registry lookup
func (s *Service) RecordONDCRegistryLookup() {
	s.ondcRegistryLookupsTotal.Inc()
}

// RecordONDCTimestampValidation records a timestamp validation
func (s *Service) RecordONDCTimestampValidation() {
	s.ondcTimestampValidationsTotal.Inc()
}

// RecordIssueCreated records an issue creation
func (s *Service) RecordIssueCreated() {
	s.issuesCreatedTotal.Inc()
}

// RecordIssueResolved records an issue resolution
func (s *Service) RecordIssueResolved() {
	s.issuesResolvedTotal.Inc()
}

// SetIssuesByStatus sets issue count by status
func (s *Service) SetIssuesByStatus(status string, count float64) {
	s.igmIssuesByStatus.WithLabelValues(status).Set(count)
}

// RecordIssueResolutionTime records issue resolution time
func (s *Service) RecordIssueResolutionTime(issueType string, duration time.Duration) {
	s.igmIssuesResolutionTime.WithLabelValues(issueType).Observe(duration.Seconds())
}

// RecordDBAuditLogWritten records an audit log write
func (s *Service) RecordDBAuditLogWritten(status string) {
	s.dbAuditLogsWrittenTotal.WithLabelValues(status).Inc()
}

// SetDBAuditLogsSize sets total audit log entries
func (s *Service) SetDBAuditLogsSize(size float64) {
	s.dbAuditLogsSize.Set(size)
}

// RecordDBTransactionDuration records database transaction duration
func (s *Service) RecordDBTransactionDuration(transactionType string, duration time.Duration) {
	s.dbTransactionDuration.WithLabelValues(transactionType).Observe(duration.Seconds())
}

// RecordLatencySLIViolation records a latency SLO violation
func (s *Service) RecordLatencySLIViolation(endpoint string) {
	s.latencySLIViolations.WithLabelValues(endpoint).Inc()
}

// SetAvailabilitySLI sets the current availability SLI
func (s *Service) SetAvailabilitySLI(availability float64) {
	s.availabilitySLI.Set(availability)
}

// RecordAvailabilitySLIViolation records an availability SLO violation
func (s *Service) RecordAvailabilitySLIViolation() {
	s.availabilitySLIViolations.Inc()
}
