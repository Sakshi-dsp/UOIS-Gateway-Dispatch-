package models

import (
	"fmt"
	"time"
)

// BaseEvent contains common fields for all events
// ID Stack Compliance:
// - event_id: Used only for event-level deduplication (NOT business logic)
// - traceparent: W3C traceparent format for distributed tracing (logs+spans only, NOT business logic)
// - trace_id: Extracted from traceparent for logging convenience (NOT business logic)
// - UOIS Gateway NEVER generates or uses correlation_id (WebSocket Gateway responsibility only)
type BaseEvent struct {
	EventType   string    `json:"event_type"`
	EventID     string    `json:"event_id"`           // UUID v4 for event-level deduplication
	Traceparent string    `json:"traceparent"`        // W3C traceparent format (00-<trace-id>-<span-id>-<flags>)
	TraceID     string    `json:"trace_id,omitempty"` // Extracted from traceparent for logging convenience
	Timestamp   time.Time `json:"timestamp"`
}

// ValidateBaseEvent validates common event fields
func (e *BaseEvent) ValidateBaseEvent() error {
	if e.EventType == "" {
		return fmt.Errorf("event_type is required")
	}
	if e.EventID == "" {
		return fmt.Errorf("event_id is required")
	}
	if e.Traceparent == "" {
		return fmt.Errorf("traceparent is required")
	}
	if e.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}
	return nil
}

// Price represents a monetary value with currency
type Price struct {
	Value    float64 `json:"value"`
	Currency string  `json:"currency"`
}

// Events Published by UOIS Gateway

// SearchRequestedEvent is published to stream.location.search
// ID Stack Compliance: Uses search_id (business correlation ID) for event correlation, NOT WebSocket correlation_id
type SearchRequestedEvent struct {
	BaseEvent
	SearchID       string  `json:"search_id"`       // Business correlation ID (NOT WebSocket correlation_id)
	OriginLat      float64 `json:"origin_lat"`      // Internal format (NOT pickup_lat)
	OriginLng      float64 `json:"origin_lng"`      // Internal format (NOT pickup_lng)
	DestinationLat float64 `json:"destination_lat"` // Internal format (NOT drop_lat)
	DestinationLng float64 `json:"destination_lng"` // Internal format (NOT drop_lng)
}

// Validate validates SearchRequestedEvent
// NOTE: EventType consistency check is optional - only add if you fully control all publishers
// and want hard schema enforcement. Current design allows flexibility for external publishers.
func (e *SearchRequestedEvent) Validate() error {
	if err := e.ValidateBaseEvent(); err != nil {
		return err
	}
	// Optional: Uncomment for strict EventType enforcement
	// if e.EventType != "SEARCH_REQUESTED" {
	// 	return fmt.Errorf("invalid event_type for SearchRequestedEvent: expected SEARCH_REQUESTED, got %s", e.EventType)
	// }
	if e.SearchID == "" {
		return fmt.Errorf("search_id is required")
	}
	return nil
}

// InitRequestedEvent is published to stream.uois.init_requested
// ID Stack Compliance: Uses search_id (business correlation ID) for event correlation, NOT WebSocket correlation_id
type InitRequestedEvent struct {
	BaseEvent
	SearchID           string                 `json:"search_id"`  // Business correlation ID (NOT WebSocket correlation_id)
	OriginLat          float64                `json:"origin_lat"` // Internal format
	OriginLng          float64                `json:"origin_lng"` // Internal format
	OriginAddress      map[string]interface{} `json:"origin_address,omitempty"`
	DestinationLat     float64                `json:"destination_lat"` // Internal format
	DestinationLng     float64                `json:"destination_lng"` // Internal format
	DestinationAddress map[string]interface{} `json:"destination_address,omitempty"`
	PackageInfo        map[string]interface{} `json:"package_info,omitempty"`
}

// Validate validates InitRequestedEvent
// NOTE: EventType consistency check is optional - only add if you fully control all publishers
func (e *InitRequestedEvent) Validate() error {
	if err := e.ValidateBaseEvent(); err != nil {
		return err
	}
	// Optional: Uncomment for strict EventType enforcement
	// if e.EventType != "INIT_REQUESTED" {
	// 	return fmt.Errorf("invalid event_type for InitRequestedEvent: expected INIT_REQUESTED, got %s", e.EventType)
	// }
	if e.SearchID == "" {
		return fmt.Errorf("search_id is required")
	}
	return nil
}

// ConfirmRequestedEvent is published to stream.uois.confirm_requested
// ID Stack Compliance: Uses quote_id (business correlation ID) for event correlation, NOT WebSocket correlation_id
// client_id is extracted from auth context (tenant boundary, NOT business correlation)
type ConfirmRequestedEvent struct {
	BaseEvent
	QuoteID       string                 `json:"quote_id"`        // Business correlation ID (NOT WebSocket correlation_id)
	ClientID      string                 `json:"client_id"`       // Extracted from auth (tenant boundary)
	ClientOrderID string                 `json:"client_order_id"` // ONDC order ID from message.order.id
	PaymentInfo   map[string]interface{} `json:"payment_info,omitempty"`
}

// Validate validates ConfirmRequestedEvent
// NOTE: EventType consistency check is optional - only add if you fully control all publishers
func (e *ConfirmRequestedEvent) Validate() error {
	if err := e.ValidateBaseEvent(); err != nil {
		return err
	}
	// Optional: Uncomment for strict EventType enforcement
	// if e.EventType != "CONFIRM_REQUESTED" {
	// 	return fmt.Errorf("invalid event_type for ConfirmRequestedEvent: expected CONFIRM_REQUESTED, got %s", e.EventType)
	// }
	if e.QuoteID == "" {
		return fmt.Errorf("quote_id is required")
	}
	return nil
}

// Events Consumed by UOIS Gateway

// QuoteComputedEvent is consumed from quote:computed
// ID Stack Compliance: Uses search_id (business correlation ID) for event correlation, NOT WebSocket correlation_id
type QuoteComputedEvent struct {
	BaseEvent
	SearchID                    string                 `json:"search_id"` // Business correlation ID (NOT WebSocket correlation_id)
	Serviceable                 bool                   `json:"serviceable"`
	Price                       Price                  `json:"price"`
	TTL                         string                 `json:"ttl"` // ISO8601 duration (e.g., "PT10M")
	TTLSeconds                  int                    `json:"ttl_seconds,omitempty"`
	ETAOrigin                   *time.Time             `json:"eta_origin,omitempty"`                     // Pass-through from SERVICEABILITY_FOUND
	ETADestination              *time.Time             `json:"eta_destination,omitempty"`                // Pass-through from SERVICEABILITY_FOUND
	DistanceOriginToDestination float64                `json:"distance_origin_to_destination,omitempty"` // Pass-through from SERVICEABILITY_FOUND
	Metadata                    map[string]interface{} `json:"metadata,omitempty"`
}

// Validate validates QuoteComputedEvent
// NOTE: EventType consistency check is optional - only add if you fully control all publishers
func (e *QuoteComputedEvent) Validate() error {
	if err := e.ValidateBaseEvent(); err != nil {
		return err
	}
	// Optional: Uncomment for strict EventType enforcement
	// if e.EventType != "QUOTE_COMPUTED" {
	// 	return fmt.Errorf("invalid event_type for QuoteComputedEvent: expected QUOTE_COMPUTED, got %s", e.EventType)
	// }
	if e.SearchID == "" {
		return fmt.Errorf("search_id is required")
	}
	return nil
}

// QuoteCreatedEvent is consumed from stream.uois.quote_created
// ID Stack Compliance: Uses search_id and quote_id (business correlation IDs) for event correlation
type QuoteCreatedEvent struct {
	BaseEvent
	SearchID                    string     `json:"search_id"` // Business correlation ID (NOT WebSocket correlation_id)
	QuoteID                     string     `json:"quote_id"`  // Business correlation ID
	Price                       Price      `json:"price"`
	TTL                         string     `json:"ttl"` // ISO8601 duration (e.g., "PT15M")
	TTLSeconds                  int        `json:"ttl_seconds,omitempty"`
	DistanceOriginToDestination float64    `json:"distance_origin_to_destination,omitempty"`
	ETAOrigin                   *time.Time `json:"eta_origin,omitempty"`
	ETADestination              *time.Time `json:"eta_destination,omitempty"`
}

// Validate validates QuoteCreatedEvent
// NOTE: EventType consistency check is optional - only add if you fully control all publishers
func (e *QuoteCreatedEvent) Validate() error {
	if err := e.ValidateBaseEvent(); err != nil {
		return err
	}
	// Optional: Uncomment for strict EventType enforcement
	// if e.EventType != "QUOTE_CREATED" {
	// 	return fmt.Errorf("invalid event_type for QuoteCreatedEvent: expected QUOTE_CREATED, got %s", e.EventType)
	// }
	if e.SearchID == "" {
		return fmt.Errorf("search_id is required")
	}
	if e.QuoteID == "" {
		return fmt.Errorf("quote_id is required")
	}
	return nil
}

// QuoteInvalidatedEvent is consumed from stream.uois.quote_invalidated
// ID Stack Compliance: Uses search_id and quote_id (business correlation IDs) for event correlation
type QuoteInvalidatedEvent struct {
	BaseEvent
	SearchID         string `json:"search_id"` // Business correlation ID (NOT WebSocket correlation_id)
	QuoteID          string `json:"quote_id"`  // Business correlation ID
	Error            string `json:"error"`
	Message          string `json:"message"`
	RequiresResearch bool   `json:"requires_research,omitempty"`
}

// Validate validates QuoteInvalidatedEvent
// NOTE: EventType consistency check is optional - only add if you fully control all publishers
func (e *QuoteInvalidatedEvent) Validate() error {
	if err := e.ValidateBaseEvent(); err != nil {
		return err
	}
	// Optional: Uncomment for strict EventType enforcement
	// if e.EventType != "QUOTE_INVALIDATED" {
	// 	return fmt.Errorf("invalid event_type for QuoteInvalidatedEvent: expected QUOTE_INVALIDATED, got %s", e.EventType)
	// }
	if e.SearchID == "" {
		return fmt.Errorf("search_id is required")
	}
	return nil
}

// OrderConfirmedEvent is consumed from stream.uois.order_confirmed
// ID Stack Compliance: Uses quote_id (business correlation ID) for event correlation, NOT WebSocket correlation_id
type OrderConfirmedEvent struct {
	BaseEvent
	QuoteID         string `json:"quote_id"`          // Business correlation ID (NOT WebSocket correlation_id)
	DispatchOrderID string `json:"dispatch_order_id"` // Business lifecycle ID
	RiderID         string `json:"rider_id,omitempty"`
}

// Validate validates OrderConfirmedEvent
// NOTE: EventType consistency check is optional - only add if you fully control all publishers
func (e *OrderConfirmedEvent) Validate() error {
	if err := e.ValidateBaseEvent(); err != nil {
		return err
	}
	// Optional: Uncomment for strict EventType enforcement
	// if e.EventType != "ORDER_CONFIRMED" {
	// 	return fmt.Errorf("invalid event_type for OrderConfirmedEvent: expected ORDER_CONFIRMED, got %s", e.EventType)
	// }
	if e.QuoteID == "" {
		return fmt.Errorf("quote_id is required")
	}
	if e.DispatchOrderID == "" {
		return fmt.Errorf("dispatch_order_id is required")
	}
	return nil
}

// OrderConfirmFailedEvent is consumed from stream.uois.order_confirm_failed
// ID Stack Compliance: Uses quote_id (business correlation ID) for event correlation, NOT WebSocket correlation_id
type OrderConfirmFailedEvent struct {
	BaseEvent
	QuoteID         string `json:"quote_id"`                    // Business correlation ID (NOT WebSocket correlation_id)
	DispatchOrderID string `json:"dispatch_order_id,omitempty"` // Business lifecycle ID
	Reason          string `json:"reason"`
}

// Validate validates OrderConfirmFailedEvent
// NOTE: EventType consistency check is optional - only add if you fully control all publishers
func (e *OrderConfirmFailedEvent) Validate() error {
	if err := e.ValidateBaseEvent(); err != nil {
		return err
	}
	// Optional: Uncomment for strict EventType enforcement
	// if e.EventType != "ORDER_CONFIRM_FAILED" {
	// 	return fmt.Errorf("invalid event_type for OrderConfirmFailedEvent: expected ORDER_CONFIRM_FAILED, got %s", e.EventType)
	// }
	if e.QuoteID == "" {
		return fmt.Errorf("quote_id is required")
	}
	return nil
}
