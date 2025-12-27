package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestIssueStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		status   IssueStatus
		expected string
	}{
		{
			name:     "OPEN status",
			status:   IssueStatusOpen,
			expected: "OPEN",
		},
		{
			name:     "CLOSED status",
			status:   IssueStatusClosed,
			expected: "CLOSED",
		},
		{
			name:     "IN_PROGRESS status",
			status:   IssueStatusInProgress,
			expected: "IN_PROGRESS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestIssueStatus_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		status   IssueStatus
		expected bool
	}{
		{
			name:     "Valid OPEN status",
			status:   IssueStatusOpen,
			expected: true,
		},
		{
			name:     "Valid IN_PROGRESS status",
			status:   IssueStatusInProgress,
			expected: true,
		},
		{
			name:     "Valid CLOSED status",
			status:   IssueStatusClosed,
			expected: true,
		},
		{
			name:     "Invalid status",
			status:   IssueStatus("INVALID"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.IsValid())
		})
	}
}

func TestIssueType_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		issueType IssueType
		expected  bool
	}{
		{
			name:      "Valid ISSUE type",
			issueType: IssueTypeIssue,
			expected:  true,
		},
		{
			name:      "Valid GRIEVANCE type",
			issueType: IssueTypeGrievance,
			expected:  true,
		},
		{
			name:      "Valid DISPUTE type",
			issueType: IssueTypeDispute,
			expected:  true,
		},
		{
			name:      "Invalid type",
			issueType: IssueType("INVALID"),
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.issueType.IsValid())
		})
	}
}

func TestIssueType_String(t *testing.T) {
	tests := []struct {
		name      string
		issueType IssueType
		expected  string
	}{
		{
			name:      "ISSUE type",
			issueType: IssueTypeIssue,
			expected:  "ISSUE",
		},
		{
			name:      "GRIEVANCE type",
			issueType: IssueTypeGrievance,
			expected:  "GRIEVANCE",
		},
		{
			name:      "DISPUTE type",
			issueType: IssueTypeDispute,
			expected:  "DISPUTE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.issueType.String())
		})
	}
}

func TestIssue_Validate(t *testing.T) {
	tests := []struct {
		name    string
		issue   Issue
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid issue",
			issue: Issue{
				IssueID:       uuid.New().String(),
				TransactionID: uuid.New().String(),
				OrderID:       uuid.New().String(),
				IssueType:     IssueTypeIssue,
				Status:        IssueStatusOpen,
				Category:      "ORDER",
				SubCategory:   "DELAYED",
				Description:   "Order delayed",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			wantErr: false,
		},
		{
			name: "Missing issue_id",
			issue: Issue{
				TransactionID: uuid.New().String(),
				OrderID:       uuid.New().String(),
				IssueType:     IssueTypeIssue,
				Status:        IssueStatusOpen,
				Category:      "ORDER",
				Description:   "Order delayed",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			wantErr: true,
			errMsg:  "issue_id is required",
		},
		{
			name: "Missing transaction_id",
			issue: Issue{
				IssueID:     uuid.New().String(),
				OrderID:     uuid.New().String(),
				IssueType:   IssueTypeIssue,
				Status:      IssueStatusOpen,
				Category:    "ORDER",
				Description: "Order delayed",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			wantErr: true,
			errMsg:  "transaction_id is required",
		},
		{
			name: "Missing order_id",
			issue: Issue{
				IssueID:       uuid.New().String(),
				TransactionID: uuid.New().String(),
				IssueType:     IssueTypeIssue,
				Status:        IssueStatusOpen,
				Category:      "ORDER",
				Description:   "Order delayed",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			wantErr: true,
			errMsg:  "order_id is required",
		},
		{
			name: "Missing category",
			issue: Issue{
				IssueID:       uuid.New().String(),
				TransactionID: uuid.New().String(),
				OrderID:       uuid.New().String(),
				IssueType:     IssueTypeIssue,
				Status:        IssueStatusOpen,
				Description:   "Order delayed",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			wantErr: true,
			errMsg:  "category is required",
		},
		{
			name: "Missing description",
			issue: Issue{
				IssueID:       uuid.New().String(),
				TransactionID: uuid.New().String(),
				OrderID:       uuid.New().String(),
				IssueType:     IssueTypeIssue,
				Status:        IssueStatusOpen,
				Category:      "ORDER",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			wantErr: true,
			errMsg:  "description is required",
		},
		{
			name: "Zero created_at",
			issue: Issue{
				IssueID:       uuid.New().String(),
				TransactionID: uuid.New().String(),
				OrderID:       uuid.New().String(),
				IssueType:     IssueTypeIssue,
				Status:        IssueStatusOpen,
				Category:      "ORDER",
				Description:   "Order delayed",
				CreatedAt:     time.Time{},
				UpdatedAt:     time.Now(),
			},
			wantErr: true,
			errMsg:  "created_at is required",
		},
		{
			name: "Invalid issue_type",
			issue: Issue{
				IssueID:       uuid.New().String(),
				TransactionID: uuid.New().String(),
				OrderID:       uuid.New().String(),
				IssueType:     IssueType("INVALID_TYPE"), // Typo
				Status:        IssueStatusOpen,
				Category:      "ORDER",
				Description:   "Order delayed",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			wantErr: true,
			errMsg:  "invalid issue_type",
		},
		{
			name: "Invalid status",
			issue: Issue{
				IssueID:       uuid.New().String(),
				TransactionID: uuid.New().String(),
				OrderID:       uuid.New().String(),
				IssueType:     IssueTypeIssue,
				Status:        IssueStatus("INVALID_STATUS"), // Typo
				Category:      "ORDER",
				Description:   "Order delayed",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			wantErr: true,
			errMsg:  "invalid status",
		},
		{
			name: "Invalid category",
			issue: Issue{
				IssueID:       uuid.New().String(),
				TransactionID: uuid.New().String(),
				OrderID:       uuid.New().String(),
				IssueType:     IssueTypeIssue,
				Status:        IssueStatusOpen,
				Category:      "INVALID_CATEGORY", // Typo
				Description:   "Order delayed",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			wantErr: true,
			errMsg:  "invalid category",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.issue.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIssueAction_Validate(t *testing.T) {
	tests := []struct {
		name    string
		action  IssueAction
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid action",
			action: IssueAction{
				ActionType: "RESPOND",
				ShortDesc:  "Response provided",
				UpdatedAt:  time.Now(),
			},
			wantErr: false,
		},
		{
			name: "Missing action_type",
			action: IssueAction{
				ShortDesc: "Response provided",
				UpdatedAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "action_type is required",
		},
		{
			name: "Missing updated_at",
			action: IssueAction{
				ActionType: "RESPOND",
				ShortDesc:  "Response provided",
				UpdatedAt:  time.Time{},
			},
			wantErr: true,
			errMsg:  "updated_at is required",
		},
		{
			name: "Invalid action_type",
			action: IssueAction{
				ActionType: "INVALID_ACTION", // Typo
				ShortDesc:  "Response provided",
				UpdatedAt:  time.Now(),
			},
			wantErr: true,
			errMsg:  "invalid action_type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.action.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGRO_Validate(t *testing.T) {
	tests := []struct {
		name    string
		gro     GRO
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid GRO",
			gro: GRO{
				Level:       "L1",
				Name:        "John Doe",
				Email:       "john.doe@example.com",
				Phone:       "+1234567890",
				ContactType: "PRIMARY",
			},
			wantErr: false,
		},
		{
			name: "Missing level",
			gro: GRO{
				Name:        "John Doe",
				Email:       "john.doe@example.com",
				Phone:       "+1234567890",
				ContactType: "PRIMARY",
			},
			wantErr: true,
			errMsg:  "level is required",
		},
		{
			name: "Missing name",
			gro: GRO{
				Level:       "L1",
				Email:       "john.doe@example.com",
				Phone:       "+1234567890",
				ContactType: "PRIMARY",
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "Missing email",
			gro: GRO{
				Level:       "L1",
				Name:        "John Doe",
				Phone:       "+1234567890",
				ContactType: "PRIMARY",
			},
			wantErr: true,
			errMsg:  "email is required",
		},
		{
			name: "Invalid level",
			gro: GRO{
				Level:       "L4", // Invalid level
				Name:        "John Doe",
				Email:       "john.doe@example.com",
				Phone:       "+1234567890",
				ContactType: "PRIMARY",
			},
			wantErr: true,
			errMsg:  "invalid level",
		},
		{
			name: "Invalid contact_type",
			gro: GRO{
				Level:       "L1",
				Name:        "John Doe",
				Email:       "john.doe@example.com",
				Phone:       "+1234567890",
				ContactType: "INVALID_TYPE", // Invalid contact type
			},
			wantErr: true,
			errMsg:  "invalid contact_type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.gro.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFinancialResolution_Validate(t *testing.T) {
	tests := []struct {
		name    string
		res     FinancialResolution
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid financial resolution",
			res: FinancialResolution{
				RefundAmount:  100.50,
				PaymentMethod: "REFUND",
				Status:        "COMPLETED",
				ResolvedAt:    func() *time.Time { t := time.Now(); return &t }(),
			},
			wantErr: false,
		},
		{
			name: "Valid with empty status",
			res: FinancialResolution{
				RefundAmount: 100.50,
			},
			wantErr: false,
		},
		{
			name: "Invalid status",
			res: FinancialResolution{
				RefundAmount: 100.50,
				Status:       "INVALID_STATUS", // Typo
			},
			wantErr: true,
			errMsg:  "invalid status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.res.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGRO_GetLevelForIssueType(t *testing.T) {
	tests := []struct {
		name      string
		issueType IssueType
		expected  string
	}{
		{
			name:      "ISSUE maps to L1",
			issueType: IssueTypeIssue,
			expected:  "L1",
		},
		{
			name:      "GRIEVANCE maps to L2",
			issueType: IssueTypeGrievance,
			expected:  "L2",
		},
		{
			name:      "DISPUTE maps to L3",
			issueType: IssueTypeDispute,
			expected:  "L3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := GetGROLevelForIssueType(tt.issueType)
			assert.Equal(t, tt.expected, level)
		})
	}
}
