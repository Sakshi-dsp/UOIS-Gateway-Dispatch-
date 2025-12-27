package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractTraceID_ValidTraceparent(t *testing.T) {
	traceparent := "00-4bf92f3577b34da6a811ce9a12345678-1234567890abcdef-01"
	traceID := ExtractTraceID(traceparent)

	assert.Equal(t, "4bf92f3577b34da6a811ce9a12345678", traceID)
}

func TestExtractTraceID_EmptyString(t *testing.T) {
	traceID := ExtractTraceID("")
	assert.Equal(t, "", traceID)
}

func TestExtractTraceID_InvalidFormat(t *testing.T) {
	traceID := ExtractTraceID("invalid-format")
	assert.Equal(t, "", traceID)
}

func TestExtractTraceID_MissingPrefix(t *testing.T) {
	traceID := ExtractTraceID("01-4bf92f3577b34da6a811ce9a-1234567890abcdef-01")
	assert.Equal(t, "", traceID)
}

func TestExtractTraceID_ShortFormat(t *testing.T) {
	traceID := ExtractTraceID("00-abc")
	assert.Equal(t, "", traceID)
}

func TestGenerateTraceparent_ValidFormat(t *testing.T) {
	traceparent := GenerateTraceparent()

	// Validate format: 00-<trace-id>-<span-id>-<flags>
	parts := strings.Split(traceparent, "-")
	assert.Equal(t, 4, len(parts))
	assert.Equal(t, "00", parts[0])
	assert.Equal(t, 32, len(parts[1])) // trace-id: 32 hex chars
	assert.Equal(t, 16, len(parts[2])) // span-id: 16 hex chars
	assert.Equal(t, "01", parts[3])    // flags: 01 (sampled)
}

func TestGenerateTraceparent_Unique(t *testing.T) {
	traceparent1 := GenerateTraceparent()
	traceparent2 := GenerateTraceparent()

	assert.NotEqual(t, traceparent1, traceparent2)
}

func TestGenerateTraceparent_Format(t *testing.T) {
	traceparent := GenerateTraceparent()

	assert.True(t, strings.HasPrefix(traceparent, "00-"))
	assert.True(t, strings.HasSuffix(traceparent, "-01"))

	parts := strings.Split(traceparent, "-")
	assert.Equal(t, 4, len(parts))

	// Validate hex characters only
	for _, char := range parts[1] {
		assert.Contains(t, "0123456789abcdef", string(char))
	}
	for _, char := range parts[2] {
		assert.Contains(t, "0123456789abcdef", string(char))
	}
}
