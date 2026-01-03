package utils

import (
	"fmt"

	"uois-gateway/pkg/errors"
)

// ValidatePaymentType validates payment type according to Dispatch business rules
// Dispatch does NOT support COD (ON-FULFILLMENT) payment type
// Returns DomainError if payment type is invalid or not supported
func ValidatePaymentType(paymentInfo map[string]interface{}) *errors.DomainError {
	if paymentInfo == nil || len(paymentInfo) == 0 {
		return nil // Payment is optional, no validation needed if not present
	}

	paymentType, ok := paymentInfo["type"].(string)
	if !ok {
		return nil // Payment type not provided, optional field
	}

	// Reject COD (ON-FULFILLMENT) - Dispatch does not support COD
	if paymentType == "ON-FULFILLMENT" {
		return errors.NewDomainError(
			65004,
			"not serviceable",
			"Payment type ON-FULFILLMENT (COD) is not supported. Supported payment types: ON-ORDER, POST-FULFILLMENT",
		)
	}

	// Validate enum values - only ON-ORDER and POST-FULFILLMENT are supported
	validTypes := map[string]bool{
		"ON-ORDER":         true,
		"POST-FULFILLMENT": true,
	}

	if !validTypes[paymentType] {
		return errors.NewDomainError(
			65001,
			"invalid request",
			fmt.Sprintf("Invalid payment type: %s. Supported types: ON-ORDER, POST-FULFILLMENT", paymentType),
		)
	}

	return nil // Valid payment type
}
