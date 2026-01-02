# Financial Notifications Integration - Implementation Plan

**Date:** 2025-01-XX  
**Purpose:** Implement financial notifications integration for Logistics Seller NP (P2P)  
**Context:** Settlement/reconciliation status notifications from Admin Backend to update IGM issues

---

## Current Status

### ✅ Implemented
- `FinancialResolution` model exists (`internal/models/issue.go`)
- Model supports: `refund_amount`, `payment_method`, `transaction_ref`, `status`, `resolved_at`
- Model validation implemented
- Can be included in `/on_issue_status` callback responses
- Issue repository supports storing financial resolution

### ❌ Missing
- Webhook endpoint to receive financial notifications from Admin Backend
- Service to process settlement/reconciliation notifications
- Redis storage for `ondc:financial:{issue_id}` (as per FR Section 9.4)
- Integration with Admin Backend notification system
- Update issue status when financial resolution is received
- Trigger `/on_issue_status` callback when financial resolution updates

---

## Requirements Analysis

### For Logistics Seller NP (P2P)

Per **ONDC IGM Contract v1.0.0** (Section 9.4 and IGM API Contract):
- **Financial Actions**: `RECONCILED`, `NOT-RECONCILED` (for settlement disputes)
- **Use Case**: Settlement/reconciliation status notifications for IGM issues
- **Storage**: Redis key `ondc:financial:{issue_id}` (TTL: 30 days)
- **Integration**: Receive notifications from Admin Backend and update related issues

### Functional Requirements (FR Section 9.4)

1. **Receive payment status notifications** from Admin Backend
2. **Receive settlement status notifications** (primary for Logistics NP)
3. **Receive RTO status notifications**
4. **Store financial resolution data** in Redis (`ondc:financial:{issue_id}`)
5. **Update related issues** with financial status information
6. **Support financial action tracking** (refunds, settlements)
7. **Link financial resolutions** to ONDC issues for status callbacks

---

## Implementation Plan

### Phase 1: Financial Notification Models & Storage

#### 1.1 Extend FinancialResolution Model

**File:** `internal/models/issue.go`

**Changes:**
- Add `ActionTriggered` field (enum: `RECONCILED`, `NOT-RECONCILED`, `REFUND`, `REPLACEMENT`, `CANCEL`, `NO-ACTION`)
- Add `SettlementAmount` field (for Logistics NP settlement disputes)
- Add `SettlementReference` field (settlement transaction reference)
- Update validation to support Logistics NP actions

**Code:**
```go
// FinancialResolution represents financial resolution details
type FinancialResolution struct {
	ActionTriggered    string     `json:"action_triggered,omitempty"`    // RECONCILED, NOT-RECONCILED, REFUND, etc.
	RefundAmount       float64    `json:"refund_amount,omitempty"`        // Refund amount (for REFUND action)
	SettlementAmount   float64    `json:"settlement_amount,omitempty"`   // Settlement amount (for RECONCILED/NOT-RECONCILED)
	PaymentMethod      string     `json:"payment_method,omitempty"`      // Payment method
	TransactionRef     string     `json:"transaction_ref,omitempty"`      // Transaction reference
	SettlementReference string    `json:"settlement_reference,omitempty"` // Settlement transaction reference
	Status             string     `json:"status,omitempty"`              // PENDING, COMPLETED, FAILED
	ResolvedAt         *time.Time `json:"resolved_at,omitempty"`          // Resolution timestamp
}
```

#### 1.2 Create Financial Notification Service

**File:** `internal/services/financial/financial_notification_service.go`

**Responsibilities:**
- Process financial notifications from Admin Backend
- Store financial resolution in Redis (`ondc:financial:{issue_id}`)
- Update issue with financial resolution
- Trigger `/on_issue_status` callback if issue status changes

**Interface:**
```go
type FinancialNotificationService interface {
	ProcessSettlementNotification(ctx context.Context, notification *SettlementNotification) error
	ProcessRefundNotification(ctx context.Context, notification *RefundNotification) error
	ProcessRTONotification(ctx context.Context, notification *RTONotification) error
	GetFinancialResolution(ctx context.Context, issueID string) (*models.FinancialResolution, error)
}
```

#### 1.3 Create Financial Notification Models

**File:** `internal/models/financial_notification.go`

**Models:**
```go
// SettlementNotification represents settlement status notification from Admin Backend
type SettlementNotification struct {
	IssueID              string    `json:"issue_id"`                // ONDC issue ID
	OrderID              string    `json:"order_id"`                // ONDC order ID
	ActionTriggered      string    `json:"action_triggered"`         // RECONCILED, NOT-RECONCILED
	SettlementAmount     float64   `json:"settlement_amount"`       // Settlement amount
	SettlementReference  string    `json:"settlement_reference"`    // Settlement transaction reference
	Status               string    `json:"status"`                  // PENDING, COMPLETED, FAILED
	ResolvedAt           time.Time `json:"resolved_at"`             // Resolution timestamp
	Message              string    `json:"message,omitempty"`       // Additional message
}

// RefundNotification represents refund status notification
type RefundNotification struct {
	IssueID         string    `json:"issue_id"`
	OrderID         string    `json:"order_id"`
	RefundAmount    float64   `json:"refund_amount"`
	PaymentMethod   string    `json:"payment_method"`
	TransactionRef  string    `json:"transaction_ref"`
	Status          string    `json:"status"`
	ResolvedAt      time.Time `json:"resolved_at"`
}

// RTONotification represents RTO status notification
type RTONotification struct {
	IssueID    string    `json:"issue_id"`
	OrderID    string    `json:"order_id"`
	RTOStatus  string    `json:"rto_status"`  // INITIATED, COMPLETED, FAILED
	UpdatedAt  time.Time `json:"updated_at"`
}
```

### Phase 2: Webhook Handler

#### 2.1 Create Financial Notification Webhook Handler

**File:** `internal/handlers/financial/financial_notification_handler.go`

**Endpoint:** `POST /webhooks/admin/financial-notification`

**Responsibilities:**
- Validate webhook signature (if Admin Backend signs webhooks)
- Parse notification payload
- Call `FinancialNotificationService` to process notification
- Return HTTP 200 OK on success

**Handler Structure:**
```go
type FinancialNotificationHandler struct {
	financialService FinancialNotificationService
	logger          *zap.Logger
}

func (h *FinancialNotificationHandler) HandleFinancialNotification(c *gin.Context) {
	// 1. Extract and validate webhook signature
	// 2. Parse notification payload
	// 3. Route to appropriate service method based on notification type
	// 4. Return success response
}
```

#### 2.2 Add Webhook Route

**File:** `cmd/server/main.go`

**Changes:**
- Add financial notification handler initialization
- Register webhook route: `router.POST("/webhooks/admin/financial-notification", financialHandler.HandleFinancialNotification)`
- Add authentication middleware for webhook (internal API key or signature validation)

### Phase 3: Redis Storage Integration

#### 3.1 Create Financial Resolution Repository

**File:** `internal/repository/financial/financial_repository.go`

**Responsibilities:**
- Store financial resolution in Redis: `ondc:financial:{issue_id}`
- Retrieve financial resolution by issue ID
- Set TTL: 30 days (as per FR Section 9.4)

**Interface:**
```go
type FinancialRepository interface {
	StoreFinancialResolution(ctx context.Context, issueID string, resolution *models.FinancialResolution) error
	GetFinancialResolution(ctx context.Context, issueID string) (*models.FinancialResolution, error)
	DeleteFinancialResolution(ctx context.Context, issueID string) error
}
```

### Phase 4: Issue Update Integration

#### 4.1 Update Issue Repository

**File:** `internal/repository/issue/issue_repository.go`

**Changes:**
- Add method to update issue with financial resolution
- Ensure financial resolution is persisted with issue

#### 4.2 Update Issue Status Handler

**File:** `internal/handlers/igm/issue_status_handler.go`

**Changes:**
- Already includes financial resolution in `/on_issue_status` response (✅ implemented)
- No changes needed

### Phase 5: Callback Triggering

#### 5.1 Trigger Callback on Financial Resolution

**File:** `internal/services/financial/financial_notification_service.go`

**Logic:**
- When financial resolution is received and issue status should change:
  1. Update issue with financial resolution
  2. Store in Redis
  3. Trigger `/on_issue_status` callback to Buyer NP
  4. Include financial resolution in callback payload

---

## Implementation Steps (TDD)

### Step 1: Financial Notification Models
1. ✅ Write tests for `SettlementNotification`, `RefundNotification`, `RTONotification`
2. ✅ Implement models with validation
3. ✅ Run tests and ensure green

### Step 2: Financial Repository
1. ✅ Write tests for `FinancialRepository` (Redis operations)
2. ✅ Implement repository
3. ✅ Run tests and ensure green

### Step 3: Financial Notification Service
1. ✅ Write tests for `FinancialNotificationService`
2. ✅ Mock dependencies (IssueRepository, FinancialRepository, CallbackService)
3. ✅ Implement service
4. ✅ Run tests and ensure green

### Step 4: Webhook Handler
1. ✅ Write tests for `FinancialNotificationHandler`
2. ✅ Mock `FinancialNotificationService`
3. ✅ Implement handler
4. ✅ Run tests and ensure green

### Step 5: Integration
1. ✅ Wire services in `main.go`
2. ✅ Add webhook route
3. ✅ Integration tests
4. ✅ Run full test suite

---

## File Structure

```
internal/
├── models/
│   ├── financial_notification.go (NEW)
│   └── issue.go (UPDATE - extend FinancialResolution)
├── services/
│   └── financial/ (NEW)
│       ├── financial_notification_service.go
│       └── financial_notification_service_test.go
├── repository/
│   └── financial/ (NEW)
│       ├── financial_repository.go
│       └── financial_repository_test.go
└── handlers/
    └── financial/ (NEW)
        ├── financial_notification_handler.go
        └── financial_notification_handler_test.go
```

---

## Configuration

### Environment Variables

**File:** `internal/config/config.go`

**Add:**
```go
type FinancialConfig struct {
	WebhookSecret     string `env:"FINANCIAL_WEBHOOK_SECRET" envDefault:""`     // Secret for webhook signature validation
	RedisKeyPrefix    string `env:"FINANCIAL_REDIS_KEY_PREFIX" envDefault:"ondc:financial:"`
	RedisTTL          int    `env:"FINANCIAL_REDIS_TTL_DAYS" envDefault:"30"`  // TTL in days
}
```

---

## Testing Strategy

### Unit Tests
- ✅ Financial notification models validation
- ✅ Financial repository (Redis operations)
- ✅ Financial notification service (business logic)
- ✅ Webhook handler (HTTP handling)

### Integration Tests
- ✅ End-to-end: Admin Backend → Webhook → Service → Redis → Issue Update → Callback
- ✅ Error scenarios: Invalid payload, missing issue, Redis failure

### Test Coverage Target
- **Minimum:** 80% code coverage
- **Critical paths:** 100% coverage

---

## Error Handling

### Error Scenarios
1. **Invalid notification payload** → Return HTTP 400 Bad Request
2. **Issue not found** → Log warning, return HTTP 200 OK (idempotent)
3. **Redis failure** → Log error, retry with exponential backoff
4. **Callback failure** → Use existing callback retry mechanism

### Error Codes
- Use existing error taxonomy (65001-65021)
- Add specific error codes if needed for financial notifications

---

## Security Considerations

### Webhook Authentication
- **Option 1:** HMAC signature validation (if Admin Backend signs webhooks)
- **Option 2:** Internal API key authentication
- **Option 3:** IP allowlist (if Admin Backend has static IPs)

### Data Validation
- Validate all notification fields
- Sanitize input to prevent injection attacks
- Validate issue_id exists before processing

---

## Monitoring & Observability

### Metrics
- `uois.financial.notifications.received.total` - Notifications received by type
- `uois.financial.notifications.processed.total` - Notifications processed successfully
- `uois.financial.notifications.failed.total` - Processing failures
- `uois.financial.callbacks.triggered.total` - Callbacks triggered due to financial updates

### Logging
- Log all financial notifications received
- Log issue updates with financial resolution
- Include `issue_id`, `order_id`, `action_triggered` in logs

---

## Dependencies

### External
- Admin Backend (webhook source)
- Redis (storage)
- Existing services: IssueRepository, CallbackService

### Internal
- `internal/models/issue.go` (extend FinancialResolution)
- `internal/repository/issue/issue_repository.go` (add update method)
- `internal/services/callback/callback_service.go` (existing)

---

## Timeline Estimate

- **Phase 1:** Models & Storage - 2-3 hours
- **Phase 2:** Webhook Handler - 2-3 hours
- **Phase 3:** Redis Integration - 1-2 hours
- **Phase 4:** Issue Update - 1-2 hours
- **Phase 5:** Callback Triggering - 2-3 hours
- **Testing & Integration:** 3-4 hours

**Total:** ~12-17 hours

---

## Success Criteria

1. ✅ Webhook endpoint receives financial notifications from Admin Backend
2. ✅ Financial resolution stored in Redis with 30-day TTL
3. ✅ Issue updated with financial resolution
4. ✅ `/on_issue_status` callback triggered when financial resolution updates
5. ✅ Financial resolution included in callback payload
6. ✅ All tests passing (unit + integration)
7. ✅ Error handling and logging implemented
8. ✅ Metrics exposed for monitoring

---

## Notes

- **Logistics Seller NP (P2P) Focus**: Primary use case is settlement/reconciliation (`RECONCILED`/`NOT-RECONCILED`)
- **Refund Support**: Model supports refunds but may not be primary use case for Logistics NP
- **RTO Notifications**: Can be integrated if Admin Backend sends RTO status updates
- **Idempotency**: Webhook handler should be idempotent (safe to replay notifications)

---

## Related Documents

- FR Section 9.4: Financial Notifications Integration
- ONDC IGM API Contract v1.0.0: Resolution actions (RECONCILED, NOT-RECONCILED)
- `internal/models/issue.go`: FinancialResolution model
- `internal/handlers/igm/issue_status_handler.go`: Callback response composition

