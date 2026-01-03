package ondc

// ExtractFulfillmentContactsFromRequest extracts fulfillment contacts from ONDC request
// Returns a map with "start" and "end" keys containing contact information
// Returns nil if contacts are not present or cannot be extracted
func ExtractFulfillmentContactsFromRequest(message map[string]interface{}) map[string]interface{} {
	order, ok := message["order"].(map[string]interface{})
	if !ok {
		return nil
	}

	fulfillments, ok := order["fulfillments"].([]interface{})
	if !ok || len(fulfillments) == 0 {
		return nil
	}

	// Extract contacts from first fulfillment (P2P typically has one fulfillment)
	fulfillment, ok := fulfillments[0].(map[string]interface{})
	if !ok {
		return nil
	}

	result := make(map[string]interface{})

	// Extract start contact
	if start, ok := fulfillment["start"].(map[string]interface{}); ok {
		if contact, ok := start["contact"].(map[string]interface{}); ok && contact != nil {
			result["start"] = contact
		}
	}

	// Extract end contact
	if end, ok := fulfillment["end"].(map[string]interface{}); ok {
		if contact, ok := end["contact"].(map[string]interface{}); ok && contact != nil {
			result["end"] = contact
		}
	}

	// Return nil if no contacts found
	if len(result) == 0 {
		return nil
	}

	return result
}

