package errors

import (
	"fmt"
)

type DomainError struct {
	Code      int
	Message   string
	Details   string
	Retryable bool
	Cause     error
}

func (e *DomainError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error {
	return e.Cause
}

func (e *DomainError) WithRetryable(retryable bool) *DomainError {
	e.Retryable = retryable
	return e
}

func (e *DomainError) WithCause(cause error) *DomainError {
	e.Cause = cause
	return e
}

func NewDomainError(code int, message, details string) *DomainError {
	return &DomainError{
		Code:      code,
		Message:   message,
		Details:   details,
		Retryable: false,
	}
}

func WrapDomainError(err error, code int, message, details string) *DomainError {
	return &DomainError{
		Code:      code,
		Message:   message,
		Details:   details,
		Retryable: false,
		Cause:     err,
	}
}

func IsDomainError(err error) bool {
	_, ok := err.(*DomainError)
	return ok
}

func GetHTTPStatus(err error) int {
	domainErr, ok := err.(*DomainError)
	if !ok {
		return 500
	}

	switch domainErr.Code {
	case 65001, 65003, 65004, 65005, 65007:
		return 400
	case 65002:
		return 401
	case 65006:
		return 404
	case 65010, 65011:
		return 503
	case 65012:
		return 429
	case 65020, 65021:
		return 500
	default:
		return 500
	}
}
