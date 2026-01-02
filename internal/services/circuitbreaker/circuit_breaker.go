package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrCircuitBreakerOpen     = errors.New("circuit breaker is open")
	ErrCircuitBreakerHalfOpen = errors.New("circuit breaker is half-open")
)

// State represents the circuit breaker state
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Config holds circuit breaker configuration
type Config struct {
	FailureThreshold    int           // Number of failures before opening
	SuccessThreshold    int           // Number of successes in half-open before closing
	Timeout             time.Duration // Time to wait before attempting half-open
	MaxRequestsHalfOpen int           // Max requests allowed in half-open state
}

// DefaultConfig returns a default circuit breaker configuration
func DefaultConfig() Config {
	return Config{
		FailureThreshold:    5,
		SuccessThreshold:    2,
		Timeout:             60 * time.Second,
		MaxRequestsHalfOpen: 3,
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config        Config
	state         State
	failureCount  int
	successCount  int
	halfOpenCount int
	lastFailure   time.Time
	mu            sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config Config) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if err := cb.beforeCall(); err != nil {
		return err
	}

	err := fn()
	cb.afterCall(err)
	return err
}

// ExecuteWithResult executes a function that returns a result with circuit breaker protection
func (cb *CircuitBreaker) ExecuteWithResult(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	if err := cb.beforeCall(); err != nil {
		return nil, err
	}

	result, err := fn()
	cb.afterCall(err)
	return result, err
}

// beforeCall checks if the circuit breaker allows the call
func (cb *CircuitBreaker) beforeCall() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return nil
	case StateOpen:
		if time.Since(cb.lastFailure) >= cb.config.Timeout {
			cb.state = StateHalfOpen
			cb.halfOpenCount = 0
			// Fall through to half-open logic to increment count
		} else {
			return ErrCircuitBreakerOpen
		}
		fallthrough
	case StateHalfOpen:
		// Check if we've reached the max requests limit
		// We check >= because if count equals max, we've already used all allowed requests
		if cb.halfOpenCount >= cb.config.MaxRequestsHalfOpen {
			return ErrCircuitBreakerHalfOpen
		}
		// Increment count before allowing the request
		cb.halfOpenCount++
		return nil
	default:
		return nil
	}
}

// afterCall updates the circuit breaker state based on the call result
func (cb *CircuitBreaker) afterCall(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.onFailure()
	} else {
		cb.onSuccess()
	}
}

// onFailure handles a failed call
func (cb *CircuitBreaker) onFailure() {
	cb.failureCount++
	cb.lastFailure = time.Now()

	switch cb.state {
	case StateClosed:
		if cb.failureCount >= cb.config.FailureThreshold {
			cb.state = StateOpen
		}
	case StateHalfOpen:
		cb.state = StateOpen
		cb.failureCount = cb.config.FailureThreshold
	}
}

// onSuccess handles a successful call
func (cb *CircuitBreaker) onSuccess() {
	cb.failureCount = 0

	switch cb.state {
	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.config.SuccessThreshold {
			cb.state = StateClosed
			cb.successCount = 0
			cb.halfOpenCount = 0
		}
	case StateClosed:
		cb.successCount = 0
	}
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetFailureCount returns the current failure count
func (cb *CircuitBreaker) GetFailureCount() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failureCount
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.halfOpenCount = 0
}
