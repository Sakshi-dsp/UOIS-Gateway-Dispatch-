package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSearchRequestedEvent_Validate(t *testing.T) {
	tests := []struct {
		name    string
		event   SearchRequestedEvent
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid event",
			event: SearchRequestedEvent{
				BaseEvent: BaseEvent{
					EventType:   "SEARCH_REQUESTED",
					EventID:     uuid.New().String(),
					Traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-8f2a1b2c3d4e5f6a-01",
					Timestamp:   time.Now(),
				},
				SearchID:       uuid.New().String(),
				OriginLat:      12.453544,
				OriginLng:      77.928379,
				DestinationLat: 12.9716,
				DestinationLng: 77.5946,
			},
			wantErr: false,
		},
		{
			name: "Missing search_id",
			event: SearchRequestedEvent{
				BaseEvent: BaseEvent{
					EventType:   "SEARCH_REQUESTED",
					EventID:     uuid.New().String(),
					Traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-8f2a1b2c3d4e5f6a-01",
					Timestamp:   time.Now(),
				},
				OriginLat:      12.453544,
				OriginLng:      77.928379,
				DestinationLat: 12.9716,
				DestinationLng: 77.5946,
			},
			wantErr: true,
			errMsg:  "search_id is required",
		},
		{
			name: "Missing event_id",
			event: SearchRequestedEvent{
				BaseEvent: BaseEvent{
					EventType:   "SEARCH_REQUESTED",
					Traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-8f2a1b2c3d4e5f6a-01",
					Timestamp:   time.Now(),
				},
				SearchID:       uuid.New().String(),
				OriginLat:      12.453544,
				OriginLng:      77.928379,
				DestinationLat: 12.9716,
				DestinationLng: 77.5946,
			},
			wantErr: true,
			errMsg:  "event_id is required",
		},
		{
			name: "Missing timestamp",
			event: SearchRequestedEvent{
				BaseEvent: BaseEvent{
					EventType:   "SEARCH_REQUESTED",
					EventID:     uuid.New().String(),
					Traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-8f2a1b2c3d4e5f6a-01",
					Timestamp:   time.Time{}, // Zero time
				},
				SearchID:       uuid.New().String(),
				OriginLat:      12.453544,
				OriginLng:      77.928379,
				DestinationLat: 12.9716,
				DestinationLng: 77.5946,
			},
			wantErr: true,
			errMsg:  "timestamp is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestQuoteComputedEvent_Validate(t *testing.T) {
	tests := []struct {
		name    string
		event   QuoteComputedEvent
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid event",
			event: QuoteComputedEvent{
				BaseEvent: BaseEvent{
					EventType:   "QUOTE_COMPUTED",
					EventID:     uuid.New().String(),
					Traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-8f2a1b2c3d4e5f6a-01",
					Timestamp:   time.Now(),
				},
				SearchID:    uuid.New().String(),
				Serviceable: true,
				Price: Price{
					Value:    58.00,
					Currency: "INR",
				},
				TTL:        "PT10M",
				TTLSeconds: 600,
			},
			wantErr: false,
		},
		{
			name: "Missing search_id",
			event: QuoteComputedEvent{
				BaseEvent: BaseEvent{
					EventType:   "QUOTE_COMPUTED",
					EventID:     uuid.New().String(),
					Traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-8f2a1b2c3d4e5f6a-01",
					Timestamp:   time.Now(),
				},
				Serviceable: true,
				Price: Price{
					Value:    58.00,
					Currency: "INR",
				},
				TTL:        "PT10M",
				TTLSeconds: 600,
			},
			wantErr: true,
			errMsg:  "search_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOrderConfirmedEvent_Validate(t *testing.T) {
	tests := []struct {
		name    string
		event   OrderConfirmedEvent
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid event",
			event: OrderConfirmedEvent{
				BaseEvent: BaseEvent{
					EventType:   "ORDER_CONFIRMED",
					EventID:     uuid.New().String(),
					Traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-8f2a1b2c3d4e5f6a-01",
					Timestamp:   time.Now(),
				},
				QuoteID:         uuid.New().String(),
				DispatchOrderID: "ABC0000001",
				RiderID:         "rider_123",
			},
			wantErr: false,
		},
		{
			name: "Missing quote_id",
			event: OrderConfirmedEvent{
				BaseEvent: BaseEvent{
					EventType:   "ORDER_CONFIRMED",
					EventID:     uuid.New().String(),
					Traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-8f2a1b2c3d4e5f6a-01",
					Timestamp:   time.Now(),
				},
				DispatchOrderID: "ABC0000001",
				RiderID:         "rider_123",
			},
			wantErr: true,
			errMsg:  "quote_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
