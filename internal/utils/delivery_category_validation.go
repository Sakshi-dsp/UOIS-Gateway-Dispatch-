package utils

import (
	"fmt"
	"strconv"
	"strings"

	"uois-gateway/pkg/errors"
)

// ValidateDeliveryCategory validates delivery category according to Dispatch business rules
// Dispatch only supports:
//   - "Immediate Delivery"
//   - "Standard Delivery" with immediate subcategory (time.duration <= PT60M)
//
// Returns DomainError if category is invalid or not supported
func ValidateDeliveryCategory(categoryID string, timeDuration string) *errors.DomainError {
	if categoryID == "" {
		return nil // Category is optional, no validation needed if not present
	}

	// Supported categories
	switch categoryID {
	case "Immediate Delivery":
		// For Immediate Delivery, validate duration <= PT60M
		if timeDuration != "" {
			if err := validateDurationWithinLimit(timeDuration, 60); err != nil {
				return errors.NewDomainError(
					66002,
					"order validation failure",
					fmt.Sprintf("Order Validation Failed: 'Immediate Delivery' requires time.duration <= PT60M, got %s", timeDuration),
				)
			}
		}
		return nil // Valid

	case "Standard Delivery":
		// For Standard Delivery, only accept if duration <= PT60M (immediate subcategory)
		if timeDuration == "" {
			return errors.NewDomainError(
				66002,
				"order validation failure",
				"Order Validation Failed: unsupported delivery category 'Standard Delivery'. Standard Delivery is only accepted with immediate subcategory (time.duration <= PT60M)",
			)
		}
		if err := validateDurationWithinLimit(timeDuration, 60); err != nil {
			return errors.NewDomainError(
				66002,
				"order validation failure",
				fmt.Sprintf("Order Validation Failed: unsupported delivery category 'Standard Delivery'. Standard Delivery is only accepted with immediate subcategory (time.duration <= PT60M), got %s", timeDuration),
			)
		}
		return nil // Valid

	case "Same Day Delivery", "Next Day Delivery", "Express Delivery":
		// Reject unsupported categories
		return errors.NewDomainError(
			66002,
			"order validation failure",
			fmt.Sprintf("Order Validation Failed: unsupported delivery category '%s'. Only 'Immediate Delivery' or 'Standard Delivery' with immediate subcategory (time.duration <= PT60M) is supported", categoryID),
		)

	default:
		// Unknown category - reject
		return errors.NewDomainError(
			66002,
			"order validation failure",
			fmt.Sprintf("Order Validation Failed: unsupported delivery category '%s'. Only 'Immediate Delivery' or 'Standard Delivery' with immediate subcategory (time.duration <= PT60M) is supported", categoryID),
		)
	}
}

// validateDurationWithinLimit validates ISO8601 duration format and checks if it's within limit (in minutes)
// Returns error if duration exceeds limit or is invalid format
func validateDurationWithinLimit(duration string, maxMinutes int) error {
	if duration == "" {
		return fmt.Errorf("duration is empty")
	}

	// Parse ISO8601 duration format (PT30M, PT60M, PT1H30M, etc.)
	if !strings.HasPrefix(duration, "PT") {
		return fmt.Errorf("invalid duration format: must start with PT")
	}

	durationStr := strings.TrimPrefix(duration, "PT")
	if durationStr == "" {
		return fmt.Errorf("invalid duration format: no time components")
	}

	// Extract hours, minutes, seconds
	var hours, minutes, seconds int

	// Parse hours (H suffix)
	if idx := strings.Index(durationStr, "H"); idx != -1 {
		hoursStr := durationStr[:idx]
		if h, err := strconv.Atoi(hoursStr); err == nil {
			hours = h
		}
		durationStr = durationStr[idx+1:]
	}

	// Parse minutes (M suffix) - but need to distinguish between minutes and months
	// In ISO8601 duration, M after H or at start means minutes
	// We'll use regex to find M that's not preceded by T (which would be months in date format)
	// For duration format PT...M, M always means minutes
	if idx := strings.Index(durationStr, "M"); idx != -1 {
		minutesStr := durationStr[:idx]
		if m, err := strconv.Atoi(minutesStr); err == nil {
			minutes = m
		}
		durationStr = durationStr[idx+1:]
	}

	// Parse seconds (S suffix)
	if idx := strings.Index(durationStr, "S"); idx != -1 {
		secondsStr := durationStr[:idx]
		if s, err := strconv.Atoi(secondsStr); err == nil {
			seconds = s
		}
	}

	// Convert to total minutes
	totalMinutes := hours*60 + minutes
	if seconds > 0 {
		// Round up if there are seconds
		totalMinutes++
	}

	// Check if exceeds limit
	if totalMinutes > maxMinutes {
		return fmt.Errorf("duration %s exceeds limit of %d minutes", duration, maxMinutes)
	}

	return nil
}

// ExtractCategoryID extracts category ID from intent map
func ExtractCategoryID(intent map[string]interface{}) string {
	if intent == nil {
		return ""
	}
	category, ok := intent["category"].(map[string]interface{})
	if !ok {
		return ""
	}
	categoryID, ok := category["id"].(string)
	if !ok {
		return ""
	}
	return categoryID
}

// ExtractTimeDuration extracts time.duration from intent map
func ExtractTimeDuration(intent map[string]interface{}) string {
	if intent == nil {
		return ""
	}
	provider, ok := intent["provider"].(map[string]interface{})
	if !ok {
		return ""
	}
	timeInfo, ok := provider["time"].(map[string]interface{})
	if !ok {
		return ""
	}
	duration, ok := timeInfo["duration"].(string)
	if !ok {
		return ""
	}
	return duration
}

// ExtractCategoryIDFromOrder extracts category ID from order items
func ExtractCategoryIDFromOrder(order map[string]interface{}) string {
	if order == nil {
		return ""
	}
	items, ok := order["items"].([]interface{})
	if !ok || len(items) == 0 {
		return ""
	}
	item, ok := items[0].(map[string]interface{})
	if !ok {
		return ""
	}
	categoryID, ok := item["category_id"].(string)
	if !ok {
		return ""
	}
	return categoryID
}

// ExtractTimeDurationFromOrder extracts time.duration from order items
func ExtractTimeDurationFromOrder(order map[string]interface{}) string {
	if order == nil {
		return ""
	}
	items, ok := order["items"].([]interface{})
	if !ok || len(items) == 0 {
		return ""
	}
	item, ok := items[0].(map[string]interface{})
	if !ok {
		return ""
	}
	timeInfo, ok := item["time"].(map[string]interface{})
	if !ok {
		return ""
	}
	duration, ok := timeInfo["duration"].(string)
	if !ok {
		return ""
	}
	return duration
}
