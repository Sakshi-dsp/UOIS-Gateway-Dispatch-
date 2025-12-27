package models

import (
	"fmt"
	"strings"
	"time"
)

// ONDCContext represents the ONDC request/response context
// ID Stack Compliance:
// - transaction_id + message_id: Used for ONDC callback correlation and idempotency (NOT business correlation)
// - message_id: ONDC message identifier from context (NOT Redis Stream message_id used for ACK)
// - UOIS Gateway NEVER generates or uses correlation_id (WebSocket Gateway responsibility only)
type ONDCContext struct {
	Domain        string    `json:"domain"`
	Action        string    `json:"action"`
	BapID         string    `json:"bap_id,omitempty"`
	BapURI        string    `json:"bap_uri,omitempty"`
	BppID         string    `json:"bpp_id,omitempty"`
	BppURI        string    `json:"bpp_uri,omitempty"`
	TransactionID string    `json:"transaction_id"` // ONDC transaction identifier (for callback correlation)
	MessageID     string    `json:"message_id"`     // ONDC message identifier (NOT Redis Stream message_id)
	Timestamp     time.Time `json:"timestamp"`
	TTL           string    `json:"ttl,omitempty"` // ISO 8601 duration (e.g., "PT30S", "PT15M")
}

// validActions is the allowlist of valid ONDC actions
// Prevents hard-to-debug silent failures from typos like "on_sreach"
var validActions = map[string]bool{
	"search":          true,
	"init":            true,
	"confirm":         true,
	"status":          true,
	"track":           true,
	"cancel":          true,
	"update":          true,
	"rto":             true,
	"on_search":       true,
	"on_init":         true,
	"on_confirm":      true,
	"on_status":       true,
	"on_track":        true,
	"on_cancel":       true,
	"on_update":       true,
	"on_rto":          true,
	"issue":           true,
	"issue_status":    true,
	"on_issue":        true,
	"on_issue_status": true,
}

// Validate validates the ONDC context
func (c *ONDCContext) Validate() error {
	if c.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	if c.Action == "" {
		return fmt.Errorf("action is required")
	}
	// Validate action against allowlist (prevents typos like "on_sreach")
	if !validActions[c.Action] {
		return fmt.Errorf("invalid action: %s", c.Action)
	}
	if c.TransactionID == "" {
		return fmt.Errorf("transaction_id is required")
	}
	if c.MessageID == "" {
		return fmt.Errorf("message_id is required")
	}
	// Timestamp validation is mandatory for ONDC compliance
	// Prevents replay attacks, invalid signed payloads, and audit corruption
	if c.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}
	// Validate bap_uri requirement based on action
	if c.Action != "on_search" && c.Action != "on_init" && c.Action != "on_confirm" &&
		c.Action != "on_status" && c.Action != "on_track" && c.Action != "on_cancel" &&
		c.Action != "on_update" && c.Action != "on_rto" &&
		c.Action != "issue" && c.Action != "issue_status" &&
		c.BapURI == "" {
		return fmt.Errorf("bap_uri is required")
	}
	// Optional: Validate TTL format if provided
	if c.TTL != "" {
		// Convert ISO 8601 duration (PT30S) to Go duration format (30s)
		ttlStr := strings.ReplaceAll(c.TTL, "PT", "")
		ttlStr = strings.ReplaceAll(ttlStr, "H", "h")
		ttlStr = strings.ReplaceAll(ttlStr, "M", "m")
		ttlStr = strings.ReplaceAll(ttlStr, "S", "s")
		if _, err := time.ParseDuration(ttlStr); err != nil {
			return fmt.Errorf("invalid ttl format: %s (expected ISO 8601 duration, e.g., PT30S, PT15M)", c.TTL)
		}
	}
	return nil
}

// ONDCRequest represents an incoming ONDC request
type ONDCRequest struct {
	Context ONDCContext            `json:"context"`
	Message map[string]interface{} `json:"message"`
}

// GetContext returns the request context
func (r *ONDCRequest) GetContext() *ONDCContext {
	return &r.Context
}

// ONDCResponse represents an outgoing ONDC response
type ONDCResponse struct {
	Context ONDCContext            `json:"context"`
	Message map[string]interface{} `json:"message,omitempty"`
	Error   *ONDCError             `json:"error,omitempty"`
}

// ONDCError represents an ONDC error response
type ONDCError struct {
	Type    string            `json:"type"`
	Code    string            `json:"code"`
	Path    string            `json:"path,omitempty"`
	Message map[string]string `json:"message"`
}

// ToMap converts ONDCError to map for JSON serialization
func (e *ONDCError) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"type":    e.Type,
		"code":    e.Code,
		"message": e.Message,
	}
	if e.Path != "" {
		result["path"] = e.Path
	}
	return result
}

// ONDCACKResponse represents a simple ACK response
type ONDCACKResponse struct {
	Message ONDCACKMessage `json:"message"`
}

// ONDCACKMessage represents the ACK message
type ONDCACKMessage struct {
	Ack ONDCACKStatus `json:"ack"`
}

// ONDCACKStatus represents ACK status
type ONDCACKStatus struct {
	Status string `json:"status"` // "ACK" or "NACK"
}
