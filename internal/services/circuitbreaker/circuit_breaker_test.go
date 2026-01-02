package circuitbreaker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCircuitBreaker_Execute_Success(t *testing.T) {
	cb := NewCircuitBreaker(DefaultConfig())
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.GetState())
}

func TestCircuitBreaker_Execute_Failure(t *testing.T) {
	cb := NewCircuitBreaker(DefaultConfig())
	testErr := errors.New("test error")
	err := cb.Execute(context.Background(), func() error {
		return testErr
	})
	assert.Error(t, err)
	assert.Equal(t, StateClosed, cb.GetState())
	assert.Equal(t, 1, cb.GetFailureCount())
}

func TestCircuitBreaker_OpenAfterThreshold(t *testing.T) {
	config := Config{
		FailureThreshold:    3,
		SuccessThreshold:    2,
		Timeout:             100 * time.Millisecond,
		MaxRequestsHalfOpen: 2,
	}
	cb := NewCircuitBreaker(config)

	// Fail 3 times to open the circuit
	for i := 0; i < 3; i++ {
		_ = cb.Execute(context.Background(), func() error {
			return errors.New("failure")
		})
	}

	assert.Equal(t, StateOpen, cb.GetState())

	// Next call should fail immediately
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	assert.Error(t, err)
	assert.Equal(t, ErrCircuitBreakerOpen, err)
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	config := Config{
		FailureThreshold:    2,
		SuccessThreshold:    1,
		Timeout:             50 * time.Millisecond,
		MaxRequestsHalfOpen: 2,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func() error {
			return errors.New("failure")
		})
	}
	assert.Equal(t, StateOpen, cb.GetState())

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Should transition to half-open
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateHalfOpen, cb.GetState())
}

func TestCircuitBreaker_CloseAfterSuccess(t *testing.T) {
	config := Config{
		FailureThreshold:    2,
		SuccessThreshold:    2,
		Timeout:             50 * time.Millisecond,
		MaxRequestsHalfOpen: 5,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func() error {
			return errors.New("failure")
		})
	}
	assert.Equal(t, StateOpen, cb.GetState())

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Succeed 2 times to close the circuit
	for i := 0; i < 2; i++ {
		err := cb.Execute(context.Background(), func() error {
			return nil
		})
		assert.NoError(t, err)
	}

	assert.Equal(t, StateClosed, cb.GetState())
}

func TestCircuitBreaker_ExecuteWithResult(t *testing.T) {
	cb := NewCircuitBreaker(DefaultConfig())
	result, err := cb.ExecuteWithResult(context.Background(), func() (interface{}, error) {
		return "success", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "success", result)
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := Config{
		FailureThreshold:    2,
		SuccessThreshold:    1,
		Timeout:             100 * time.Millisecond,
		MaxRequestsHalfOpen: 2,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func() error {
			return errors.New("failure")
		})
	}
	assert.Equal(t, StateOpen, cb.GetState())

	// Reset
	cb.Reset()
	assert.Equal(t, StateClosed, cb.GetState())
	assert.Equal(t, 0, cb.GetFailureCount())
}

func TestCircuitBreaker_MaxRequestsHalfOpen(t *testing.T) {
	config := Config{
		FailureThreshold:    2,
		SuccessThreshold:    1,
		Timeout:             50 * time.Millisecond,
		MaxRequestsHalfOpen: 2,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func() error {
			return errors.New("failure")
		})
	}
	assert.Equal(t, StateOpen, cb.GetState())

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// First request in half-open should succeed
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	assert.NoError(t, err)

	// Second request should succeed
	err = cb.Execute(context.Background(), func() error {
		return nil
	})
	assert.NoError(t, err)

	// Third request should fail (max requests exceeded)
	err = cb.Execute(context.Background(), func() error {
		return nil
	})
	assert.Error(t, err)
	assert.Equal(t, ErrCircuitBreakerHalfOpen, err)
}
