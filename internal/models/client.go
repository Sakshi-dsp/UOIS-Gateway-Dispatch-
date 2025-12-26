package models

import (
	"net"
)

const (
	ClientStatusActive    = "ACTIVE"
	ClientStatusSuspended = "SUSPENDED"
	ClientStatusRevoked   = "REVOKED"
)

type Client struct {
	ID               string
	ClientCode       string
	ClientSecretHash string
	AllowedIPs       []string
	Status           string
	Metadata         map[string]interface{}
}

func (c *Client) IsActive() bool {
	return c.Status == ClientStatusActive
}

func (c *Client) ValidateIP(ip string) bool {
	// No IP allowlist configured → allow all (safe default for gateway)
	// Treats both nil and empty slice the same way for security
	if len(c.AllowedIPs) == 0 {
		return true
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// At least one CIDR configured → enforce strict matching
	for _, cidrStr := range c.AllowedIPs {
		_, ipNet, err := net.ParseCIDR(cidrStr)
		if err != nil {
			// Invalid CIDR: log at sync/load time, skip at runtime
			continue
		}
		if ipNet.Contains(parsedIP) {
			return true
		}
	}

	return false
}
