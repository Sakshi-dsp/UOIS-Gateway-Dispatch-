package ondc

import (
	"context"
	"time"

	"uois-gateway/internal/services/audit"
)

// EventPublisher publishes events to Redis streams
type EventPublisher interface {
	PublishEvent(ctx context.Context, stream string, event interface{}) error
}

// EventConsumer consumes events from Redis streams
// NOTE: The correlationID parameter receives business IDs (search_id, quote_id), NOT correlation_id
// correlation_id is WebSocket Gateway responsibility and never enters UOIS Gateway
type EventConsumer interface {
	ConsumeEvent(ctx context.Context, stream, consumerGroup, correlationID string, timeout time.Duration) (interface{}, error)
}

// CallbackService sends HTTP callbacks to client callback URLs
type CallbackService interface {
	SendCallback(ctx context.Context, callbackURL string, payload interface{}) error
}

// CacheService provides caching functionality
type CacheService interface {
	Get(ctx context.Context, key string, dest interface{}) (bool, error)
	Set(ctx context.Context, key string, value interface{}) error
	Delete(ctx context.Context, key string) error
}

// IdempotencyService handles request idempotency checks and storage
type IdempotencyService interface {
	CheckIdempotency(ctx context.Context, key string) ([]byte, bool, error)
	StoreIdempotency(ctx context.Context, key string, responseBytes []byte, ttl time.Duration) error
}

// OrderServiceClient handles Order Service gRPC calls
type OrderServiceClient interface {
	ValidateSearchIDTTL(ctx context.Context, searchID string) (bool, error)
	ValidateQuoteIDTTL(ctx context.Context, quoteID string) (bool, error)
	GetOrder(ctx context.Context, dispatchOrderID string) (*OrderStatus, error)
	GetOrderTracking(ctx context.Context, dispatchOrderID string) (*OrderTracking, error)
	CancelOrder(ctx context.Context, dispatchOrderID string, reason string) error
	UpdateOrder(ctx context.Context, dispatchOrderID string, updates map[string]interface{}) error
	InitiateRTO(ctx context.Context, dispatchOrderID string) error
}

// OrderRecord represents an order record with all identifiers stored together
// Identifiers are stored together on the same order record for correlation only.
// No identifier represents, replaces, or derives another.
type OrderRecord struct {
	SearchID        string // UOIS Gateway-generated (internal-only, for /search and /init correlation)
	QuoteID         string // Order Service-generated (ONDC-visible, for /init and /confirm correlation)
	DispatchOrderID string // Order Service-generated (internal-only, execution identifier)
	OrderID         string // Seller-generated ONDC order.id (network-facing, sent in /on_confirm)
	ClientID        string // Client identifier (for multi-tenant lookup)
	TransactionID   string // ONDC transaction_id (for /init correlation lookup)
	MessageID       string // ONDC message_id (for /init correlation lookup)
	FulfillmentID   string // UOIS Gateway-generated (ONDC-visible, stable per order, used in /init and /confirm)
}

// OrderRecordService handles order record storage and retrieval
// Order records store multiple identifiers together; no identifier maps to or derives from another
type OrderRecordService interface {
	// StoreOrderRecord stores an order record with all identifiers together
	// Used when order is created/confirmed to store all IDs on same record
	StoreOrderRecord(ctx context.Context, record *OrderRecord) error

	// GetOrderRecordBySearchID retrieves order record by search_id
	// Used for /init flow to validate search_id was previously generated
	GetOrderRecordBySearchID(ctx context.Context, searchID string) (*OrderRecord, error)

	// GetOrderRecordByQuoteID retrieves order record by quote_id
	// Used for /confirm flow to validate quote_id was previously received
	GetOrderRecordByQuoteID(ctx context.Context, quoteID string) (*OrderRecord, error)

	// GetOrderRecordByOrderID retrieves order record by client_id + order.id (ONDC)
	// Used for post-confirmation flows (/status, /track, /cancel, /update, /rto)
	// order.id is seller-generated ONDC identifier (network-facing)
	GetOrderRecordByOrderID(ctx context.Context, clientID, orderID string) (*OrderRecord, error)

	// GetOrderRecordByTransactionID retrieves order record by transaction_id
	// Used for /init flow to lookup search_id from original /search request
	// transaction_id is the primary flow key; message_id is for idempotency only
	GetOrderRecordByTransactionID(ctx context.Context, transactionID string) (*OrderRecord, error)

	// UpdateOrderRecord updates an existing order record with additional identifiers
	// Used to append newly generated identifiers to an existing order record.
	// Existing identifiers must never be modified or replaced.
	UpdateOrderRecord(ctx context.Context, record *OrderRecord) error
}

// AuditService provides audit logging functionality
type AuditService interface {
	LogRequestResponse(ctx context.Context, req *audit.RequestResponseLogParams) error
	LogCallbackDelivery(ctx context.Context, req *audit.CallbackDeliveryLogParams) error
}

// BillingStorageService handles storage and retrieval of billing information
type BillingStorageService interface {
	StoreBilling(ctx context.Context, transactionID string, billing map[string]interface{}) error
	GetBilling(ctx context.Context, transactionID string) (map[string]interface{}, error)
	DeleteBilling(ctx context.Context, transactionID string) error
}

// OrderStatus represents order status from Order Service
type OrderStatus struct {
	DispatchOrderID string
	State           string
	RiderID         string
	Timeline        []OrderTimelineEvent
	Fulfillment     FulfillmentStatus
}

// OrderTimelineEvent represents a timeline event
type OrderTimelineEvent struct {
	Timestamp time.Time
	Event     string
	State     string
}

// FulfillmentStatus represents fulfillment status
type FulfillmentStatus struct {
	State           string
	ProofOfPickup   string
	ProofOfDelivery string
}

// OrderTracking represents order tracking information
type OrderTracking struct {
	DispatchOrderID string
	CurrentLocation Location
	ETA             time.Time
	Timeline        []OrderTimelineEvent
	TrackingURL     string
}

// Location represents GPS coordinates
type Location struct {
	Lat float64
	Lng float64
}
