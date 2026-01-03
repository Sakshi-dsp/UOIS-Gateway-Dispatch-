# Contracts Implementation - Payment Type Validation

## UOIS Implementation

### `/on_search` Pricing Implementation

**ONDC Requirement**: Each item in `/on_search` must include a `price` object with `currency` and `value` (tax-inclusive).

**UOIS Gateway Implementation**:
- **Location**: `internal/handlers/ondc/search_handler.go` (lines 476-479)
- **Data Source**: `QUOTE_COMPUTED` event from Quote Service
- **Implementation**:
  ```go
  "price": map[string]interface{}{
      "currency": quoteComputed.Price.Currency,
      "value":    fmt.Sprintf("%.2f", quoteComputed.Price.Value),
  },
  ```

**Data Flow**:
1. UOIS Gateway publishes `SEARCH_REQUESTED` → Location Service
2. Location Service publishes `SERVICEABILITY_FOUND` → Quote Service
3. Quote Service computes pricing and publishes `QUOTE_COMPUTED` with `Price` field
4. UOIS Gateway consumes `QUOTE_COMPUTED` and extracts `Price`
5. UOIS Gateway builds `/on_search` callback with pricing from `quoteComputed.Price`

**Dependency**: Quote Service must provide valid `Price.Value` (tax-inclusive) and `Price.Currency` in `QUOTE_COMPUTED` event. If pricing is missing or invalid, `/on_search` response will be non-compliant with ONDC specification.

### `/on_init` Breakup Array Implementation

**ONDC Requirement**: `/on_init` callback must include a `breakup` array in `message.order.quote.breakup` for price transparency. The breakup array must itemize delivery charges and taxes separately.

**UOIS Gateway Implementation**:
- **Location**: `internal/handlers/ondc/init_handler.go` (lines 492-503, 544-557)
- **Data Source**: `QUOTE_CREATED` event from Order Service (includes `Breakup []BreakupItem` field)
- **Implementation**:
  ```go
  quoteMap := map[string]interface{}{
      "id": quoteCreated.QuoteID,
      "price": map[string]interface{}{
          "value":    quoteCreated.Price.Value,
          "currency": quoteCreated.Price.Currency,
      },
      "ttl": quoteCreated.TTL,
  }
  if len(quoteCreated.Breakup) > 0 {
      quoteMap["breakup"] = h.convertBreakupToMap(quoteCreated.Breakup)
  }
  ```

**BreakupItem Structure**:
```go
type BreakupItem struct {
    ItemID    string `json:"@ondc/org/item_id"`    // e.g., "delivery_charge", "tax"
    TitleType string `json:"@ondc/org/title_type"` // "delivery" or "tax"
    Price     Price  `json:"price"`                // Price object with value and currency
}
```

**convertBreakupToMap Helper**:
```go
func (h *InitHandler) convertBreakupToMap(breakup []models.BreakupItem) []map[string]interface{} {
    result := make([]map[string]interface{}, 0, len(breakup))
    for _, item := range breakup {
        result = append(result, map[string]interface{}{
            "@ondc/org/item_id":    item.ItemID,
            "@ondc/org/title_type": item.TitleType,
            "price": map[string]interface{}{
                "value":    item.Price.Value,
                "currency": item.Price.Currency,
            },
        })
    }
    return result
}
```

**Data Flow**:
1. Order Service calculates breakup (delivery + tax) using tax rate from Admin Service
2. Order Service publishes `QUOTE_CREATED` event with `Breakup []BreakupItem` field
3. UOIS Gateway consumes `QUOTE_CREATED` event from stream `stream.uois.quote_created`
4. UOIS Gateway extracts `Breakup` field from `QuoteCreatedEvent`
5. UOIS Gateway converts `[]BreakupItem` to ONDC-compliant JSON format using `convertBreakupToMap()`
6. UOIS Gateway includes breakup array in `/on_init` callback → `message.order.quote.breakup`

**ONDC Compliance**:
- ✅ Breakup array includes delivery charge (tax-exclusive base price)
- ✅ Breakup array includes tax amount (calculated from tax rate)
- ✅ Validation: `quote.price.value = Σ quote.breakup[].price.value` (Order Service ensures this)
- ✅ Breakup items use ONDC-compliant field names (`@ondc/org/item_id`, `@ondc/org/title_type`)

**Dependency**: Order Service must provide valid `Breakup []BreakupItem` in `QUOTE_CREATED` event. If breakup is missing, `/on_init` callback will not include breakup array (conditionally included). Order Service calculates breakup using tax rate from Admin Service (no hardcoded fallback).

## Business Logic

### Payment Type Support

**UOIS (Dispatch) supports:**
- ✅ `"ON-ORDER"` - Prepaid (payment before fulfillment)
- ✅ `"POST-FULFILLMENT"` - Post fulfillment billing

**UOIS (Dispatch) does NOT support:**
- ❌ `"ON-FULFILLMENT"` - Cash on Delivery (COD) - **REJECTED**

### Validation Logic

#### 1. Payment Type Validation

**Location**: All ONDC request handlers (`/search`, `/init`, `/confirm`)

**Pseudo Code**:
```
IF payment.type exists:
    IF payment.type == "ON-FULFILLMENT":
        RETURN NACK with error code 65004 (not serviceable)
    ELSE IF payment.type NOT IN ["ON-ORDER", "POST-FULFILLMENT"]:
        RETURN NACK with error code 65001 (invalid request)
    END IF
END IF
```

**Error Response**:
```json
{
  "error": {
    "type": "CONTEXT_ERROR",
    "code": "65004",
    "message": {
      "en": "Payment type ON-FULFILLMENT (COD) is not supported. Supported payment types: ON-ORDER, POST-FULFILLMENT"
    }
  }
}
```

#### 2. Collection Amount Validation

**For ON-FULFILLMENT** (if somehow passed validation):
- Collection amount is required (but ON-FULFILLMENT is rejected anyway)

**For POST-FULFILLMENT**:
- Collection amount is optional

**For ON-ORDER**:
- Collection amount should NOT be present (prepaid, no collection needed)

**Pseudo Code**:
```
IF payment.type == "ON-ORDER":
    IF payment.@ondc/org/collection_amount exists:
        LOG WARNING: "ON-ORDER payment should not have collection_amount"
        // Optionally: Remove collection_amount or return warning
    END IF
END IF
```

## Error Codes

### ONDC Standard Error Codes

| Error Code | Type | Description | Usage |
|------------|------|-------------|-------|
| **65001** | CONTEXT_ERROR | Invalid request | Invalid payment type enum value |
| **65004** | CONTEXT_ERROR | Not serviceable | COD (ON-FULFILLMENT) not supported |

### Error Code 65004 - Not Serviceable

**ONDC Definition**: Service not available for the given request parameters.

**Usage**: When payment type is ON-FULFILLMENT (COD), UOIS Gateway rejects the request as "not serviceable" because Dispatch does not support COD.

**HTTP Status**: 400 Bad Request

## Implementation Details

### Handler Integration Points

#### `/search` Handler
- **Location**: `internal/handlers/ondc/search_handler.go`
- **Validation Point**: After parsing request, before publishing SEARCH_REQUESTED event
- **Action**: Validate `intent.payment.type`

#### `/init` Handler
- **Location**: `internal/handlers/ondc/init_handler.go`
- **Validation Point**: After parsing request, before publishing INIT_REQUESTED event
- **Action**: Validate `order.payment.type`

#### `/confirm` Handler
- **Location**: `internal/handlers/ondc/confirm_handler.go`
- **Validation Point**: After parsing request, before publishing CONFIRM_REQUESTED event
- **Action**: Validate `order.payment.type`

### Validation Function Signature

```go
func validatePaymentType(paymentInfo map[string]interface{}) *errors.DomainError {
    // Extract payment type
    paymentType, ok := paymentInfo["type"].(string)
    if !ok {
        return nil // Payment is optional, no validation needed if not present
    }
    
    // Reject COD (ON-FULFILLMENT)
    if paymentType == "ON-FULFILLMENT" {
        return errors.NewDomainError(
            65004, 
            "not serviceable", 
            "Payment type ON-FULFILLMENT (COD) is not supported. Supported payment types: ON-ORDER, POST-FULFILLMENT",
        )
    }
    
    // Validate enum values
    validTypes := map[string]bool{
        "ON-ORDER":        true,
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
```

### Integration Example

```go
// In search_handler.go HandleSearch method
func (h *SearchHandler) HandleSearch(c *gin.Context) {
    // ... existing code ...
    
    // Extract payment info from intent
    intent, _ := req.Message["intent"].(map[string]interface{})
    paymentInfo, _ := intent["payment"].(map[string]interface{})
    
    // Validate payment type
    if err := validatePaymentType(paymentInfo); err != nil {
        h.respondNACK(c, err)
        return
    }
    
    // ... continue with existing flow ...
}
```

## Test Cases

### Valid Cases
1. ✅ `payment.type: "ON-ORDER"` - Should pass
2. ✅ `payment.type: "POST-FULFILLMENT"` - Should pass
3. ✅ No payment field - Should pass (optional)

### Invalid Cases
1. ❌ `payment.type: "ON-FULFILLMENT"` - Should reject with error 65004
2. ❌ `payment.type: "INVALID_TYPE"` - Should reject with error 65001
3. ❌ `payment.type: null` - Should reject with error 65001

## References

- [ONDC Error Codes](https://github.com/ONDC-Official/developer-docs/blob/main/protocol-network-extension/error-codes.md)
- [ONDC-DISPATCH Contracts](./ONDC-DISPATCH%20Contracts.md)
- [Payment Type Verification](./PAYMENT_TYPE_VERIFICATION.md)

