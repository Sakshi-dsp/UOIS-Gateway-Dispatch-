package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"Active", ClientStatusActive, true},
		{"Suspended", ClientStatusSuspended, false},
		{"Revoked", ClientStatusRevoked, false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				Status: tt.status,
			}
			assert.Equal(t, tt.expected, client.IsActive())
		})
	}
}

func TestClient_ValidateIP(t *testing.T) {
	client := &Client{
		AllowedIPs: []string{"192.168.1.0/24", "10.0.0.1/32"},
	}

	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"Within CIDR", "192.168.1.100", true},
		{"Exact match", "10.0.0.1", true},
		{"Outside CIDR", "192.168.2.100", false},
		{"Invalid IP", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.ValidateIP(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClient_ValidateIP_NoRestrictions_Nil(t *testing.T) {
	client := &Client{
		AllowedIPs: nil,
	}

	result := client.ValidateIP("192.168.1.100")

	// No IP allowlist configured → allow all (safe default)
	assert.True(t, result)
}

func TestClient_ValidateIP_NoRestrictions_Empty(t *testing.T) {
	client := &Client{
		AllowedIPs: []string{},
	}

	result := client.ValidateIP("192.168.1.100")

	// No IP allowlist configured → allow all (same behavior as nil for security)
	assert.True(t, result)
}

func TestClient_ValidateIP_InvalidCIDR(t *testing.T) {
	client := &Client{
		AllowedIPs: []string{"invalid-cidr", "192.168.1.0/24"},
	}

	// Invalid CIDR should be skipped, but valid CIDR should still work
	result := client.ValidateIP("192.168.1.100")
	assert.True(t, result)

	// IP not in valid CIDR should be denied
	result = client.ValidateIP("10.0.0.1")
	assert.False(t, result)
}
