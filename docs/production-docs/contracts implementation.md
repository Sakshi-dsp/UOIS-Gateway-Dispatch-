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

### Delivery Category Validation

**ONDC Requirement**: LSPs may validate delivery categories and reject unsupported ones. ONDC contract defines validations for different categories (lines 549-552).

**UOIS Gateway Implementation**:
- **Location**: `internal/utils/delivery_category_validation.go`
- **Validation Function**: `ValidateDeliveryCategory(categoryID string, timeDuration string) *errors.DomainError`
- **Supported Categories**:
  - ✅ `"Immediate Delivery"` - Requires `time.duration <= PT60M` if duration is provided
  - ✅ `"Standard Delivery"` - Only accepted with immediate subcategory (`time.duration <= PT60M`)
- **Rejected Categories**:
  - ❌ `"Same Day Delivery"` - Rejected with error code 66002
  - ❌ `"Next Day Delivery"` - Rejected with error code 66002
  - ❌ `"Express Delivery"` - Rejected with error code 66002
  - ❌ `"Standard Delivery"` without duration or with duration > PT60M - Rejected with error code 66002

**Implementation Details**:

#### `/search` Handler
- **Location**: `internal/handlers/ondc/search_handler.go` (lines 103-110)
- **Validation Point**: After payment type validation, before publishing SEARCH_REQUESTED event
- **Data Source**: `intent.category.id` and `intent.provider.time.duration`
- **Helper Functions**: `utils.ExtractCategoryID(intent)` and `utils.ExtractTimeDuration(intent)`

```go
// Validate delivery category (Dispatch only supports Immediate Delivery or Standard Delivery with immediate subcategory)
categoryID := utils.ExtractCategoryID(intent)
timeDuration := utils.ExtractTimeDuration(intent)
if err := utils.ValidateDeliveryCategory(categoryID, timeDuration); err != nil {
    h.logger.Warn("delivery category validation failed", zap.Error(err), zap.String("trace_id", traceID), zap.String("category_id", categoryID), zap.String("time_duration", timeDuration))
    h.respondNACK(c, err)
    return
}
```

#### `/init` Handler
- **Location**: `internal/handlers/ondc/init_handler.go` (lines 100-107)
- **Validation Point**: After payment type validation, before publishing INIT_REQUESTED event
- **Data Source**: `order.items[0].category_id` and `order.items[0].time.duration`
- **Helper Functions**: `utils.ExtractCategoryIDFromOrder(order)` and `utils.ExtractTimeDurationFromOrder(order)`

```go
// Validate delivery category (Dispatch only supports Immediate Delivery or Standard Delivery with immediate subcategory)
categoryID := utils.ExtractCategoryIDFromOrder(order)
timeDuration := utils.ExtractTimeDurationFromOrder(order)
if err := utils.ValidateDeliveryCategory(categoryID, timeDuration); err != nil {
    h.logger.Warn("delivery category validation failed", zap.Error(err), zap.String("trace_id", traceID), zap.String("category_id", categoryID), zap.String("time_duration", timeDuration))
    h.respondNACK(c, err)
    return
}
```

**Duration Validation**:
- Parses ISO8601 duration format (PT30M, PT60M, PT1H30M, etc.)
- Converts to total minutes (hours*60 + minutes, rounding up seconds)
- Validates that duration <= 60 minutes for supported categories
- Returns error if duration exceeds limit or format is invalid

**Error Responses**:
- **Error Code**: `66002` (order validation failure)
- **Error Type**: `CONTEXT_ERROR`
- **Error Messages**:
  - For unsupported categories: `"Order Validation Failed: unsupported delivery category '<category>'. Only 'Immediate Delivery' or 'Standard Delivery' with immediate subcategory (time.duration <= PT60M) is supported"`
  - For Standard Delivery without immediate subcategory: `"Order Validation Failed: unsupported delivery category 'Standard Delivery'. Standard Delivery is only accepted with immediate subcategory (time.duration <= PT60M)"`
  - For Immediate Delivery with invalid duration: `"Order Validation Failed: 'Immediate Delivery' requires time.duration <= PT60M, got <duration>"`

**ONDC Compliance**:
- ✅ Validates delivery categories per ONDC contract (lines 549-552)
- ✅ Returns appropriate error codes (66002) for validation failures
- ✅ Provides clear error messages indicating supported categories
- ✅ Validates duration constraints for supported categories

**Dependency**: None - validation is self-contained in utils package. No external service calls required.

### Billing Information Storage

**ONDC Requirement**: Billing information is provided by Buyer NP in `/init` request (`message.order.billing`) and must be preserved for use in `/on_confirm` and other post-order APIs. ONDC contract specifies that billing should be the same as in `/init` (footnote [^118]).

**UOIS Gateway Implementation**:
- **Location**: `internal/services/ondc/storage/billing_storage_service.go`
- **Service Interface**: `BillingStorageService`
- **Storage Backend**: Redis (via `CacheService`)
- **Redis Key Pattern**: `ondc_billing:{transaction_id}`
- **TTL**: 24 hours

**Implementation Details**:

#### Billing Storage Service
- **Location**: `internal/services/ondc/storage/billing_storage_service.go`
- **Interface Methods**:
  - `StoreBilling(ctx context.Context, transactionID string, billing map[string]interface{}) error`
  - `GetBilling(ctx context.Context, transactionID string) (map[string]interface{}, error)`
  - `DeleteBilling(ctx context.Context, transactionID string) error`

**Service Implementation**:
```go
type Service struct {
    cache  CacheService
    logger *zap.Logger
    ttl    time.Duration // 24 hours
}

func (s *Service) StoreBilling(ctx context.Context, transactionID string, billing map[string]interface{}) error {
    key := fmt.Sprintf("ondc_billing:%s", transactionID)
    return s.cache.Set(ctx, key, billing) // TTL set at service creation
}
```

#### `/init` Handler Integration
- **Location**: `internal/handlers/ondc/init_handler.go` (lines 169-177)
- **Integration Point**: After payment type validation, before extracting coordinates
- **Data Source**: `message.order.billing` from `/init` request
- **Storage Key**: Uses `transaction_id` from request context

```go
// Extract billing information and store in Redis
if h.billingStorageService != nil {
    billing := h.extractBilling(&req)
    if billing != nil {
        if err := h.billingStorageService.StoreBilling(ctx, req.Context.TransactionID, billing); err != nil {
            h.logger.Warn("failed to store billing", zap.Error(err), zap.String("trace_id", traceID), zap.String("transaction_id", req.Context.TransactionID))
            // Non-fatal error - continue processing even if billing storage fails
        }
    }
}
```

**Billing Extraction**:
- **Location**: `internal/handlers/ondc/init_handler.go` (method `extractBilling`)
- **Extraction Logic**: Extracts `order.billing` from request message
- **Optional Field**: Billing is optional per ONDC spec - returns `nil` if not present

```go
func (h *InitHandler) extractBilling(req *models.ONDCRequest) map[string]interface{} {
    order, ok := req.Message["order"].(map[string]interface{})
    if !ok {
        return nil
    }
    billing, ok := order["billing"].(map[string]interface{})
    if !ok {
        return nil // Billing is optional per ONDC spec
    }
    return billing
}
```

**Billing Structure** (per ONDC spec):
```json
{
  "name": "ONDC Logistics Buyer NP",
  "address": {
    "name": "My house or building no",
    "building": "My house or building name",
    "locality": "Jayanagar",
    "city": "Bengaluru",
    "state": "Karnataka",
    "country": "India",
    "area_code": "560076"
  },
  "tax_number": "XXXXXXXXXXXXXXX",  // Required - GST no for logistics buyer NP
  "phone": "9886098860",
  "email": "abcd.efgh@gmail.com",   // Required
  "created_at": "2023-02-06T21:30:00.000Z",
  "updated_at": "2023-02-06T21:30:00.000Z"
}
```

**ONDC Compliance**:
- ✅ Billing extracted from `/init` request (`message.order.billing`)
- ✅ Stored with transaction_id as key for correlation
- ✅ TTL of 24 hours matches typical order lifecycle
- ✅ Non-fatal storage - order processing continues even if billing storage fails
- ✅ Billing can be retrieved for use in `/on_confirm` and other post-order APIs

**Usage in Post-Order APIs**:
- Billing information stored during `/init` can be retrieved using `GetBilling(transactionID)` for use in:
  - `/on_confirm` callback (billing should be same as in /init per ONDC spec)
  - `/on_status` callback
  - `/on_cancel` callback
  - `/on_update` callback

**Dependency**: 
- Requires `CacheService` (Redis) to be configured and available
- Billing storage is non-blocking - order processing continues even if storage fails
- Storage failures are logged but do not cause request rejection

**Error Handling**:
- Storage failures are logged as warnings but do not block order processing
- Missing billing (optional field) is handled gracefully
- Invalid transaction_id returns error

### Fulfillment Contacts Storage

**ONDC Requirement**: Fulfillment contacts (`start.contact` and `end.contact`) are provided by Buyer NP in `/init` and `/confirm` requests. These contacts represent the pickup location contact (merchant/warehouse staff) and delivery location contact (customer). ONDC contract requires these contacts to be included in `/on_init`, `/on_confirm`, `/on_status`, and `/on_cancel` responses.

**UOIS Gateway Implementation**:
- **Location**: `internal/services/ondc/storage/fulfillment_contacts_storage_service.go`
- **Service Interface**: `FulfillmentContactsStorageService`
- **Storage Backend**: Redis (via `CacheService`)
- **Redis Key Pattern**: `ondc_fulfillment_contacts:{transaction_id}`
- **TTL**: 30 days (same as order mapping)

**Implementation Details**:

#### Fulfillment Contacts Storage Service
- **Location**: `internal/services/ondc/storage/fulfillment_contacts_storage_service.go`
- **Interface Methods**:
  - `StoreFulfillmentContacts(ctx context.Context, transactionID string, contacts map[string]interface{}) error`
  - `GetFulfillmentContacts(ctx context.Context, transactionID string) (map[string]interface{}, error)`
  - `DeleteFulfillmentContacts(ctx context.Context, transactionID string) error`

**Service Implementation**:
```go
type FulfillmentContactsService struct {
    cache  CacheService
    logger *zap.Logger
    ttl    time.Duration // 30 days
}

func (s *FulfillmentContactsService) StoreFulfillmentContacts(ctx context.Context, transactionID string, contacts map[string]interface{}) error {
    key := fmt.Sprintf("ondc_fulfillment_contacts:%s", transactionID)
    return s.cache.Set(ctx, key, contacts) // TTL set at service creation
}
```

**Contact Extraction Utility**:
- **Location**: `internal/utils/ondc/request_parser.go`
- **Function**: `ExtractFulfillmentContactsFromRequest(message map[string]interface{}) map[string]interface{}`
- **Extraction Logic**: Extracts `fulfillments[0].start.contact` and `fulfillments[0].end.contact` from request message
- **Return Format**: Map with `"start"` and `"end"` keys, each containing contact object

```go
func ExtractFulfillmentContactsFromRequest(message map[string]interface{}) map[string]interface{} {
    order, ok := message["order"].(map[string]interface{})
    if !ok {
        return nil
    }
    fulfillments, ok := order["fulfillments"].([]interface{})
    if !ok || len(fulfillments) == 0 {
        return nil
    }
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
    if len(result) == 0 {
        return nil
    }
    return result
}
```

#### `/init` Handler Integration
- **Location**: `internal/handlers/ondc/init_handler.go` (lines 187-195)
- **Integration Point**: After billing storage, before publishing INIT_REQUESTED event
- **Data Source**: `message.order.fulfillments[0].start.contact` and `message.order.fulfillments[0].end.contact` from `/init` request
- **Storage Key**: Uses `transaction_id` from request context

```go
// Extract fulfillment contacts and store in Redis
if h.fulfillmentContactsStorageService != nil {
    contacts := h.extractFulfillmentContacts(&req)
    if contacts != nil {
        if err := h.fulfillmentContactsStorageService.StoreFulfillmentContacts(ctx, req.Context.TransactionID, contacts); err != nil {
            h.logger.Warn("failed to store fulfillment contacts", zap.Error(err), zap.String("trace_id", traceID), zap.String("transaction_id", req.Context.TransactionID))
            // Non-fatal error - continue processing even if storage fails
        }
    }
}
```

#### `/confirm` Handler Integration
- **Location**: `internal/handlers/ondc/confirm_handler.go` (lines 208-218)
- **Integration Point**: After order record update, before composing response
- **Data Source**: `message.order.fulfillments[0].start.contact` and `message.order.fulfillments[0].end.contact` from `/confirm` request
- **Storage Key**: Uses `transaction_id` from request context
- **Note**: Contacts from `/confirm` may update or supplement contacts stored during `/init`

```go
// Extract fulfillment contacts and store in Redis (if not already stored by /init)
if h.fulfillmentContactsStorageService != nil {
    contacts := ondcUtils.ExtractFulfillmentContactsFromRequest(req.Message)
    if contacts != nil {
        if err := h.fulfillmentContactsStorageService.StoreFulfillmentContacts(ctx, req.Context.TransactionID, contacts); err != nil {
            h.logger.Warn("failed to store fulfillment contacts during /confirm", zap.Error(err), zap.String("trace_id", traceID), zap.String("transaction_id", req.Context.TransactionID))
            // Non-fatal error - continue processing even if storage fails
        }
    }
}
```

#### Contact Retrieval in Callback Builders
- **Location**: All callback builders (`buildOnInitCallback`, `buildOnConfirmCallback`, `buildOnStatusCallback`, `buildOnCancelCallback`)
- **Retrieval Priority**:
  1. First: Check if contacts exist in the current request (for `/confirm`, `/status`, `/cancel`)
  2. Second: Retrieve from Redis using `transaction_id` (if stored during `/init` or `/confirm`)
- **Integration**: Contacts are retrieved and included in fulfillment structure using `buildFulfillmentWithContacts` helper methods

**Contact Structure** (per ONDC spec):
```json
{
  "name": "Sherlock Holmes",  // Optional
  "phone": "9886098860",      // Required
  "email": "sherlock@detective.com"  // Required
}
```

**ONDC Compliance**:
- ✅ Contacts extracted from `/init` and `/confirm` requests
- ✅ Stored with transaction_id as key for correlation
- ✅ TTL of 30 days matches order mapping lifecycle
- ✅ Non-fatal storage - order processing continues even if contact storage fails
- ✅ Contacts retrieved and included in all callback responses (`/on_init`, `/on_confirm`, `/on_status`, `/on_cancel`)
- ✅ Ensures consistency - Buyer NP receives the same contact information they originally provided

**Usage in Callback Responses**:
- Contacts stored during `/init` or `/confirm` are retrieved using `GetFulfillmentContacts(transactionID)` for use in:
  - `/on_init` callback - `fulfillment.start.contact` and `fulfillment.end.contact`
  - `/on_confirm` callback - `fulfillment.start.contact` and `fulfillment.end.contact`
  - `/on_status` callback - `fulfillment.start.contact` and `fulfillment.end.contact`
  - `/on_cancel` callback - `fulfillment.start.contact` and `fulfillment.end.contact`

**Why Store Contacts**:
- The orchestrator's location data may not always include the exact contact details (email, phone, name) that the Buyer NP provided
- By storing the original contacts from the Buyer NP's request, we ensure consistency across all ONDC responses
- This prevents mismatches between what the Buyer NP sent and what they receive in callbacks

**Dependency**: 
- Requires `CacheService` (Redis) to be configured and available
- Contact storage is non-blocking - order processing continues even if storage fails
- Storage failures are logged but do not cause request rejection

**Error Handling**:
- Storage failures are logged as warnings but do not block order processing
- Missing contacts (optional field) are handled gracefully
- Invalid transaction_id returns error
- Contacts are optional per ONDC spec - absence does not cause errors

## References

- [ONDC Error Codes](https://github.com/ONDC-Official/developer-docs/blob/main/protocol-network-extension/error-codes.md)
- [ONDC-DISPATCH Contracts](./ONDC-DISPATCH%20Contracts.md)
- [Payment Type Verification](./PAYMENT_TYPE_VERIFICATION.md)

