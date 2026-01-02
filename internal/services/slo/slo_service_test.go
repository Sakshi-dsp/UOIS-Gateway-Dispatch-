package slo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSLOService_CheckLatencySLO(t *testing.T) {
	tests := []struct {
		name           string
		endpoint       string
		p95Latency     time.Duration
		expectedPass   bool
		expectedReason string
	}{
		{
			name:           "search endpoint within SLO",
			endpoint:       "/ondc/search",
			p95Latency:     400 * time.Millisecond,
			expectedPass:   true,
			expectedReason: "",
		},
		{
			name:           "search endpoint exceeds SLO",
			endpoint:       "/ondc/search",
			p95Latency:     600 * time.Millisecond,
			expectedPass:   false,
			expectedReason: "p95 latency 600ms exceeds SLO 500ms",
		},
		{
			name:           "confirm endpoint within SLO",
			endpoint:       "/ondc/confirm",
			p95Latency:     800 * time.Millisecond,
			expectedPass:   true,
			expectedReason: "",
		},
		{
			name:           "confirm endpoint exceeds SLO",
			endpoint:       "/ondc/confirm",
			p95Latency:     1200 * time.Millisecond,
			expectedPass:   false,
			expectedReason: "p95 latency 1.2s exceeds SLO 1s",
		},
		{
			name:           "status endpoint within SLO",
			endpoint:       "/ondc/status",
			p95Latency:     150 * time.Millisecond,
			expectedPass:   true,
			expectedReason: "",
		},
		{
			name:           "status endpoint exceeds SLO",
			endpoint:       "/ondc/status",
			p95Latency:     250 * time.Millisecond,
			expectedPass:   false,
			expectedReason: "p95 latency 250ms exceeds SLO 200ms",
		},
		{
			name:           "callback within SLO",
			endpoint:       "callback",
			p95Latency:     1500 * time.Millisecond,
			expectedPass:   true,
			expectedReason: "",
		},
		{
			name:           "callback exceeds SLO",
			endpoint:       "callback",
			p95Latency:     2500 * time.Millisecond,
			expectedPass:   false,
			expectedReason: "p95 latency 2.5s exceeds SLO 2s",
		},
		{
			name:           "unknown endpoint",
			endpoint:       "/ondc/unknown",
			p95Latency:     100 * time.Millisecond,
			expectedPass:   true,
			expectedReason: "",
		},
	}

	service := NewService()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, reason := service.CheckLatencySLO(tt.endpoint, tt.p95Latency)
			assert.Equal(t, tt.expectedPass, pass)
			assert.Equal(t, tt.expectedReason, reason)
		})
	}
}

func TestSLOService_GetLatencySLO(t *testing.T) {
	service := NewService()

	tests := []struct {
		endpoint    string
		expectedSLO time.Duration
	}{
		{"/ondc/search", 500 * time.Millisecond},
		{"/ondc/confirm", 1000 * time.Millisecond},
		{"/ondc/status", 200 * time.Millisecond},
		{"/ondc/track", 200 * time.Millisecond},
		{"callback", 2000 * time.Millisecond},
		{"/ondc/unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			slo := service.GetLatencySLO(tt.endpoint)
			assert.Equal(t, tt.expectedSLO, slo)
		})
	}
}

func TestSLOService_CheckAvailabilitySLO(t *testing.T) {
	tests := []struct {
		name           string
		availability   float64
		expectedPass   bool
		expectedReason string
	}{
		{
			name:           "meets SLO",
			availability:   0.999,
			expectedPass:   true,
			expectedReason: "",
		},
		{
			name:           "exceeds SLO",
			availability:   0.9995,
			expectedPass:   true,
			expectedReason: "",
		},
		{
			name:           "below SLO",
			availability:   0.998,
			expectedPass:   false,
			expectedReason: "availability 99.8% below SLO 99.9%",
		},
		{
			name:           "exactly at SLO",
			availability:   0.999,
			expectedPass:   true,
			expectedReason: "",
		},
	}

	service := NewService()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, reason := service.CheckAvailabilitySLO(tt.availability)
			assert.Equal(t, tt.expectedPass, pass)
			assert.Equal(t, tt.expectedReason, reason)
		})
	}
}

func TestSLOService_GetAvailabilitySLO(t *testing.T) {
	service := NewService()
	slo := service.GetAvailabilitySLO()
	assert.Equal(t, 0.999, slo)
}
