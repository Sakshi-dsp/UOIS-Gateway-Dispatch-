package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDomainError(t *testing.T) {
	err := NewDomainError(65001, "validation failed", "missing required field")

	assert.NotNil(t, err)
	assert.Equal(t, 65001, err.Code)
	assert.Equal(t, "validation failed", err.Message)
	assert.Equal(t, "missing required field", err.Details)
	assert.False(t, err.Retryable)
}

func TestNewDomainError_WithRetryable(t *testing.T) {
	err := NewDomainError(65010, "timeout", "service unavailable").WithRetryable(true)

	assert.NotNil(t, err)
	assert.Equal(t, 65010, err.Code)
	assert.True(t, err.Retryable)
}

func TestDomainError_Error(t *testing.T) {
	err := NewDomainError(65001, "validation failed", "missing field")

	errorMsg := err.Error()

	assert.Contains(t, errorMsg, "65001")
	assert.Contains(t, errorMsg, "validation failed")
}

func TestDomainError_WithCause(t *testing.T) {
	originalErr := errors.New("original error")
	err := NewDomainError(65020, "internal error", "").WithCause(originalErr)

	assert.NotNil(t, err)
	assert.Equal(t, originalErr, err.Cause)
}

func TestWrapDomainError(t *testing.T) {
	originalErr := errors.New("database connection failed")
	domainErr := WrapDomainError(originalErr, 65020, "internal error", "database unavailable")

	assert.NotNil(t, domainErr)
	assert.Equal(t, 65020, domainErr.Code)
	assert.Equal(t, originalErr, domainErr.Cause)
}

func TestIsDomainError(t *testing.T) {
	domainErr := NewDomainError(65001, "validation", "")
	regularErr := errors.New("regular error")

	assert.True(t, IsDomainError(domainErr))
	assert.False(t, IsDomainError(regularErr))
}

func TestGetHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected int
	}{
		{"Validation", 65001, 400},
		{"Authentication", 65002, 401},
		{"Stale Request", 65003, 400},
		{"Quote Expired", 65004, 400},
		{"Quote Invalid", 65005, 400},
		{"Order Not Found", 65006, 404},
		{"Invalid State", 65007, 400},
		{"Dependency Timeout", 65010, 503},
		{"Dependency Unavailable", 65011, 503},
		{"Rate Limit", 65012, 429},
		{"Internal Error", 65020, 500},
		{"Unknown Code", 99999, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewDomainError(tt.code, "test", "")
			status := GetHTTPStatus(err)
			assert.Equal(t, tt.expected, status)
		})
	}
}

func TestGetHTTPStatus_NonDomainError(t *testing.T) {
	regularErr := errors.New("regular error")
	status := GetHTTPStatus(regularErr)

	assert.Equal(t, 500, status)
}
