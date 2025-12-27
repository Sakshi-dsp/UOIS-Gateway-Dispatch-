package models

import (
	"fmt"
	"time"
)

// IssueStatus represents the status of an issue
type IssueStatus string

const (
	IssueStatusOpen       IssueStatus = "OPEN"
	IssueStatusInProgress IssueStatus = "IN_PROGRESS"
	IssueStatusClosed     IssueStatus = "CLOSED"
)

// ValidIssueStatuses is the allowlist of valid issue statuses
var ValidIssueStatuses = map[string]bool{
	"OPEN":        true,
	"IN_PROGRESS": true,
	"CLOSED":      true,
}

// IsValid checks if the issue status is valid
func (s IssueStatus) IsValid() bool {
	return ValidIssueStatuses[string(s)]
}

// String returns the string representation of IssueStatus
func (s IssueStatus) String() string {
	return string(s)
}

// IssueType represents the type of issue
type IssueType string

const (
	IssueTypeIssue     IssueType = "ISSUE"
	IssueTypeGrievance IssueType = "GRIEVANCE"
	IssueTypeDispute   IssueType = "DISPUTE"
)

// ValidIssueTypes is the allowlist of valid issue types
var ValidIssueTypes = map[string]bool{
	"ISSUE":     true,
	"GRIEVANCE": true,
	"DISPUTE":   true,
}

// IsValid checks if the issue type is valid
func (t IssueType) IsValid() bool {
	return ValidIssueTypes[string(t)]
}

// String returns the string representation of IssueType
func (t IssueType) String() string {
	return string(t)
}

// Issue represents an ONDC issue/grievance/dispute
// ID Stack Compliance: Uses issue_id (business lifecycle ID) for correlation, NOT WebSocket correlation_id
type Issue struct {
	// Core identifiers
	IssueID         string `json:"issue_id"`                    // ONDC issue identifier (business lifecycle ID)
	ZendeskTicketID string `json:"zendesk_ticket_id,omitempty"` // Zendesk ticket identifier
	TransactionID   string `json:"transaction_id"`              // ONDC transaction ID for correlation
	OrderID         string `json:"order_id"`                    // ONDC order ID

	// Issue classification
	IssueType   IssueType   `json:"issue_type"`             // ISSUE, GRIEVANCE, DISPUTE
	Status      IssueStatus `json:"status"`                 // OPEN, IN_PROGRESS, CLOSED
	Category    string      `json:"category"`               // ORDER, FULFILLMENT, PAYMENT
	SubCategory string      `json:"sub_category,omitempty"` // Specific sub-category

	// Issue details
	Description string `json:"description"` // Issue description

	// Complainant information
	ComplainantInfo map[string]interface{} `json:"complainant_info,omitempty"` // Buyer NP information

	// Order details
	OrderDetails map[string]interface{} `json:"order_details,omitempty"` // Order ID, item IDs, fulfillment IDs

	// Resolution information
	ResolutionProvider  *ResolutionProvider  `json:"resolution_provider,omitempty"`  // Respondent info, GRO details
	FinancialResolution *FinancialResolution `json:"financial_resolution,omitempty"` // Refund amount, payment method, transaction ref

	// Issue actions (respondent actions, cascaded levels)
	IssueActions []IssueAction `json:"issue_actions,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Full ONDC payload for callback reconstruction
	FullONDCPayload map[string]interface{} `json:"full_ondc_payload,omitempty"`
}

// ValidCategories is the allowlist of valid issue categories
var ValidCategories = map[string]bool{
	"ORDER":       true,
	"FULFILLMENT": true,
	"PAYMENT":     true,
}

// Validate validates the Issue
func (i *Issue) Validate() error {
	if i.IssueID == "" {
		return fmt.Errorf("issue_id is required")
	}
	if i.TransactionID == "" {
		return fmt.Errorf("transaction_id is required")
	}
	if i.OrderID == "" {
		return fmt.Errorf("order_id is required")
	}
	// Validate issue type against allowlist (prevents typos)
	if !i.IssueType.IsValid() {
		return fmt.Errorf("invalid issue_type: %s (expected: ISSUE, GRIEVANCE, DISPUTE)", i.IssueType)
	}
	// Validate status against allowlist (prevents typos)
	if !i.Status.IsValid() {
		return fmt.Errorf("invalid status: %s (expected: OPEN, IN_PROGRESS, CLOSED)", i.Status)
	}
	if i.Category == "" {
		return fmt.Errorf("category is required")
	}
	// Validate category against allowlist (prevents typos)
	if !ValidCategories[i.Category] {
		return fmt.Errorf("invalid category: %s (expected: ORDER, FULFILLMENT, PAYMENT)", i.Category)
	}
	if i.Description == "" {
		return fmt.Errorf("description is required")
	}
	// Timestamp validation is mandatory for audit compliance
	if i.CreatedAt.IsZero() {
		return fmt.Errorf("created_at is required")
	}
	return nil
}

// ValidActionTypes is the allowlist of valid issue action types
var ValidActionTypes = map[string]bool{
	"RESPOND":  true,
	"ESCALATE": true,
	"RESOLVE":  true,
}

// IssueAction represents an action taken on an issue
type IssueAction struct {
	ActionType string    `json:"action_type"`          // RESPOND, ESCALATE, RESOLVE
	ShortDesc  string    `json:"short_desc,omitempty"` // Short description of action
	LongDesc   string    `json:"long_desc,omitempty"`  // Long description of action
	UpdatedAt  time.Time `json:"updated_at"`
	UpdatedBy  string    `json:"updated_by,omitempty"` // User/system that performed action
}

// Validate validates the IssueAction
func (a *IssueAction) Validate() error {
	if a.ActionType == "" {
		return fmt.Errorf("action_type is required")
	}
	// Validate action type against allowlist (prevents typos)
	if !ValidActionTypes[a.ActionType] {
		return fmt.Errorf("invalid action_type: %s (expected: RESPOND, ESCALATE, RESOLVE)", a.ActionType)
	}
	// Timestamp validation is mandatory for audit compliance
	if a.UpdatedAt.IsZero() {
		return fmt.Errorf("updated_at is required")
	}
	return nil
}

// ResolutionProvider represents the provider of issue resolution
type ResolutionProvider struct {
	RespondentInfo *RespondentInfo `json:"respondent_info,omitempty"` // Respondent information
	GRO            *GRO            `json:"gro,omitempty"`             // Grievance Redressal Officer details
}

// RespondentInfo represents respondent information
type RespondentInfo struct {
	Name     string                 `json:"name,omitempty"`
	Email    string                 `json:"email,omitempty"`
	Phone    string                 `json:"phone,omitempty"`
	Contact  map[string]interface{} `json:"contact,omitempty"` // Contact details
	ChatLink string                 `json:"chat_link,omitempty"`
	FAQs     []string               `json:"faqs,omitempty"`
}

// ValidGROLevels is the allowlist of valid GRO levels
var ValidGROLevels = map[string]bool{
	"L1": true,
	"L2": true,
	"L3": true,
}

// ValidContactTypes is the allowlist of valid contact types
var ValidContactTypes = map[string]bool{
	"PRIMARY":   true,
	"SECONDARY": true,
}

// GRO represents Grievance Redressal Officer details
type GRO struct {
	Level       string `json:"level"`                  // L1, L2, L3
	Name        string `json:"name"`                   // GRO name
	Email       string `json:"email"`                  // GRO email
	Phone       string `json:"phone,omitempty"`        // GRO phone
	ContactType string `json:"contact_type,omitempty"` // PRIMARY, SECONDARY
}

// Validate validates the GRO
func (g *GRO) Validate() error {
	if g.Level == "" {
		return fmt.Errorf("level is required")
	}
	// Validate level against allowlist (prevents typos)
	if !ValidGROLevels[g.Level] {
		return fmt.Errorf("invalid level: %s (expected: L1, L2, L3)", g.Level)
	}
	if g.Name == "" {
		return fmt.Errorf("name is required")
	}
	if g.Email == "" {
		return fmt.Errorf("email is required")
	}
	// Validate contact type if provided
	if g.ContactType != "" && !ValidContactTypes[g.ContactType] {
		return fmt.Errorf("invalid contact_type: %s (expected: PRIMARY, SECONDARY)", g.ContactType)
	}
	return nil
}

// ValidFinancialResolutionStatuses is the allowlist of valid financial resolution statuses
var ValidFinancialResolutionStatuses = map[string]bool{
	"PENDING":   true,
	"COMPLETED": true,
	"FAILED":    true,
}

// FinancialResolution represents financial resolution details
type FinancialResolution struct {
	RefundAmount   float64    `json:"refund_amount,omitempty"`   // Refund amount
	PaymentMethod  string     `json:"payment_method,omitempty"`  // Payment method
	TransactionRef string     `json:"transaction_ref,omitempty"` // Transaction reference
	Status         string     `json:"status,omitempty"`          // PENDING, COMPLETED, FAILED
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`     // Resolution timestamp
}

// Validate validates the FinancialResolution
func (f *FinancialResolution) Validate() error {
	// Validate status if provided
	if f.Status != "" && !ValidFinancialResolutionStatuses[f.Status] {
		return fmt.Errorf("invalid status: %s (expected: PENDING, COMPLETED, FAILED)", f.Status)
	}
	return nil
}

// GetGROLevelForIssueType returns the GRO level for a given issue type
// L1 for ISSUE, L2 for GRIEVANCE, L3 for DISPUTE
func GetGROLevelForIssueType(issueType IssueType) string {
	switch issueType {
	case IssueTypeIssue:
		return "L1"
	case IssueTypeGrievance:
		return "L2"
	case IssueTypeDispute:
		return "L3"
	default:
		return "L1" // Default to L1
	}
}
