package utils

import (
	"strings"

	"github.com/google/uuid"
)

// ExtractTraceID extracts trace_id from W3C traceparent header
// Format: 00-<trace-id>-<span-id>-<flags>
// trace-id must be 32 hex characters
// Returns empty string if traceparent is invalid or missing
func ExtractTraceID(traceparent string) string {
	if traceparent == "" || !strings.HasPrefix(traceparent, "00-") {
		return ""
	}

	parts := strings.Split(traceparent, "-")
	if len(parts) < 2 {
		return ""
	}

	traceID := parts[1]
	// Validate trace-id is 32 hex characters
	if len(traceID) != 32 {
		return ""
	}

	return traceID
}

// GenerateTraceparent generates a W3C traceparent header
// Format: 00-<trace-id>-<span-id>-<flags>
// trace-id: 32 hex chars (no hyphens)
// span-id: 16 hex chars (no hyphens)
// flags: 01 (sampled)
func GenerateTraceparent() string {
	traceID := strings.ReplaceAll(uuid.New().String(), "-", "")
	spanID := strings.ReplaceAll(uuid.New().String(), "-", "")[:16]
	return "00-" + traceID + "-" + spanID + "-01"
}

// EnsureTraceparent ensures a valid traceparent exists, generating one if needed
// Returns the provided traceparent if valid, otherwise generates a new one
func EnsureTraceparent(traceparent string) string {
	if traceparent == "" || !strings.HasPrefix(traceparent, "00-") {
		return GenerateTraceparent()
	}
	return traceparent
}
