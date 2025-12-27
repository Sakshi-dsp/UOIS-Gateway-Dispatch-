package models

import (
	"net"
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

func TestClient_NormalizeIPs(t *testing.T) {
	t.Run("ValidCIDRs", func(t *testing.T) {
		client := &Client{
			ID:         "test-client",
			ClientCode: "TEST",
			AllowedIPs: []string{"192.168.1.0/24", "10.0.0.1/32"},
		}

		client.NormalizeIPs(nil)

		assert.Len(t, client.NormalizedIPs, 2)
		assert.True(t, client.NormalizedIPs[0].Contains(net.ParseIP("192.168.1.100")))
		assert.True(t, client.NormalizedIPs[1].Contains(net.ParseIP("10.0.0.1")))
	})

	t.Run("EmptyAllowedIPs", func(t *testing.T) {
		client := &Client{
			AllowedIPs: []string{},
		}

		client.NormalizeIPs(nil)

		assert.Nil(t, client.NormalizedIPs)
	})

	t.Run("NilAllowedIPs", func(t *testing.T) {
		client := &Client{
			AllowedIPs: nil,
		}

		client.NormalizeIPs(nil)

		assert.Nil(t, client.NormalizedIPs)
	})

	t.Run("InvalidCIDRsSkipped", func(t *testing.T) {
		client := &Client{
			AllowedIPs: []string{"invalid-cidr", "192.168.1.0/24", "also-invalid"},
		}

		client.NormalizeIPs(nil)

		// Only valid CIDR should be normalized
		assert.Len(t, client.NormalizedIPs, 1)
		assert.True(t, client.NormalizedIPs[0].Contains(net.ParseIP("192.168.1.100")))
	})
}

func TestClient_ValidateAllowedIPs(t *testing.T) {
	t.Run("ValidCIDRs", func(t *testing.T) {
		client := &Client{
			AllowedIPs: []string{"192.168.1.0/24", "10.0.0.1/32"},
		}

		invalidCIDRs, errors := client.ValidateAllowedIPs()

		assert.Empty(t, invalidCIDRs)
		assert.Empty(t, errors)
	})

	t.Run("InvalidCIDRs", func(t *testing.T) {
		client := &Client{
			AllowedIPs: []string{"invalid-cidr", "192.168.1.0/24", "also-invalid"},
		}

		invalidCIDRs, errors := client.ValidateAllowedIPs()

		assert.Len(t, invalidCIDRs, 2)
		assert.Len(t, errors, 2)
		assert.Contains(t, invalidCIDRs, "invalid-cidr")
		assert.Contains(t, invalidCIDRs, "also-invalid")
	})

	t.Run("EmptyAllowedIPs", func(t *testing.T) {
		client := &Client{
			AllowedIPs: []string{},
		}

		invalidCIDRs, errors := client.ValidateAllowedIPs()

		assert.Empty(t, invalidCIDRs)
		assert.Empty(t, errors)
	})
}

func TestClient_ValidateIP_WithNormalizedIPs(t *testing.T) {
	client := &Client{
		AllowedIPs: []string{"192.168.1.0/24", "10.0.0.1/32"},
	}

	// Normalize IPs at load time
	client.NormalizeIPs(nil)

	// ValidateIP should use pre-parsed NormalizedIPs (hot path)
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

func TestClient_ValidateIP_BackwardCompatibility(t *testing.T) {
	// Test that ValidateIP still works without normalization (backward compatibility)
	client := &Client{
		AllowedIPs:    []string{"192.168.1.0/24"},
		NormalizedIPs: nil, // Not normalized
	}

	// Should fall back to parsing AllowedIPs
	result := client.ValidateIP("192.168.1.100")
	assert.True(t, result)

	result = client.ValidateIP("10.0.0.1")
	assert.False(t, result)
}
