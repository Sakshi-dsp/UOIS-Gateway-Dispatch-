package models

import (
	"fmt"
	"net"
)

const (
	ClientStatusActive    = "ACTIVE"
	ClientStatusSuspended = "SUSPENDED"
	ClientStatusRevoked   = "REVOKED"
)

// CIDRLogger is an optional interface for logging invalid CIDRs during client sync/load.
// If provided, invalid CIDRs will be logged during normalization instead of at runtime.
type CIDRLogger interface {
	Warn(msg string, fields ...interface{})
}

type Client struct {
	ID               string
	ClientCode       string
	ClientSecretHash string
	AllowedIPs       []string     // Original CIDR strings (for serialization/display)
	NormalizedIPs    []*net.IPNet // Pre-parsed CIDRs (for hot-path validation)
	Status           string
	Metadata         map[string]interface{}
}

func (c *Client) IsActive() bool {
	return c.Status == ClientStatusActive
}

// NormalizeIPs parses and validates AllowedIPs at load time, populating NormalizedIPs.
// Invalid CIDRs are skipped and optionally logged via the provided logger.
// This should be called when loading clients from DB/admin API to avoid repeated parsing on hot path.
func (c *Client) NormalizeIPs(logger CIDRLogger) {
	if len(c.AllowedIPs) == 0 {
		c.NormalizedIPs = nil
		return
	}

	normalized := make([]*net.IPNet, 0, len(c.AllowedIPs))
	for _, cidrStr := range c.AllowedIPs {
		_, ipNet, err := net.ParseCIDR(cidrStr)
		if err != nil {
			if logger != nil {
				logger.Warn("invalid CIDR in client allowed_ips, skipping",
					"client_id", c.ID,
					"client_code", c.ClientCode,
					"invalid_cidr", cidrStr,
					"error", err.Error(),
				)
			}
			continue
		}
		normalized = append(normalized, ipNet)
	}

	c.NormalizedIPs = normalized
}

// ValidateAllowedIPs validates AllowedIPs and returns any invalid CIDRs.
// This should be called during client sync/load to validate configuration.
// Returns list of invalid CIDR strings and their errors.
func (c *Client) ValidateAllowedIPs() (invalidCIDRs []string, errors []error) {
	if len(c.AllowedIPs) == 0 {
		return nil, nil
	}

	invalidCIDRs = make([]string, 0)
	errors = make([]error, 0)

	for _, cidrStr := range c.AllowedIPs {
		_, _, err := net.ParseCIDR(cidrStr)
		if err != nil {
			invalidCIDRs = append(invalidCIDRs, cidrStr)
			errors = append(errors, fmt.Errorf("invalid CIDR %q: %w", cidrStr, err))
		}
	}

	return invalidCIDRs, errors
}

// ValidateIP validates if the given IP address is allowed.
// Uses pre-parsed NormalizedIPs if available (hot path), otherwise falls back to parsing AllowedIPs.
func (c *Client) ValidateIP(ip string) bool {
	// No IP allowlist configured → allow all (safe default for gateway)
	// Treats both nil and empty slice the same way for security
	if len(c.AllowedIPs) == 0 && len(c.NormalizedIPs) == 0 {
		return true
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Use pre-parsed CIDRs if available (hot path optimization)
	if len(c.NormalizedIPs) > 0 {
		for _, ipNet := range c.NormalizedIPs {
			if ipNet.Contains(parsedIP) {
				return true
			}
		}
		return false
	}

	// Fallback: parse CIDRs on-the-fly (backward compatibility)
	// At least one CIDR configured → enforce strict matching
	for _, cidrStr := range c.AllowedIPs {
		_, ipNet, err := net.ParseCIDR(cidrStr)
		if err != nil {
			// Invalid CIDR: skip (should have been caught during normalization)
			continue
		}
		if ipNet.Contains(parsedIP) {
			return true
		}
	}

	return false
}
