package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestONDCContext_Validate(t *testing.T) {
	tests := []struct {
		name    string
		context ONDCContext
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid context",
			context: ONDCContext{
				Domain:        "nic2004:60232",
				Action:        "search",
				BapID:         "buyer.example.com",
				BapURI:        "https://buyer.example.com",
				TransactionID: "txn_123",
				MessageID:     "msg_123",
				Timestamp:     time.Now(),
				TTL:           "PT30S",
			},
			wantErr: false,
		},
		{
			name: "Missing domain",
			context: ONDCContext{
				Action:        "search",
				BapID:         "buyer.example.com",
				BapURI:        "https://buyer.example.com",
				TransactionID: "txn_123",
				MessageID:     "msg_123",
				Timestamp:     time.Now(),
				TTL:           "PT30S",
			},
			wantErr: true,
			errMsg:  "domain is required",
		},
		{
			name: "Missing action",
			context: ONDCContext{
				Domain:        "nic2004:60232",
				BapID:         "buyer.example.com",
				BapURI:        "https://buyer.example.com",
				TransactionID: "txn_123",
				MessageID:     "msg_123",
				Timestamp:     time.Now(),
				TTL:           "PT30S",
			},
			wantErr: true,
			errMsg:  "action is required",
		},
		{
			name: "Missing transaction_id",
			context: ONDCContext{
				Domain:    "nic2004:60232",
				Action:    "search",
				BapID:     "buyer.example.com",
				BapURI:    "https://buyer.example.com",
				MessageID: "msg_123",
				Timestamp: time.Now(),
				TTL:       "PT30S",
			},
			wantErr: true,
			errMsg:  "transaction_id is required",
		},
		{
			name: "Missing message_id",
			context: ONDCContext{
				Domain:        "nic2004:60232",
				Action:        "search",
				BapID:         "buyer.example.com",
				BapURI:        "https://buyer.example.com",
				TransactionID: "txn_123",
				Timestamp:     time.Now(),
				TTL:           "PT30S",
			},
			wantErr: true,
			errMsg:  "message_id is required",
		},
		{
			name: "Missing bap_uri",
			context: ONDCContext{
				Domain:        "nic2004:60232",
				Action:        "search",
				BapID:         "buyer.example.com",
				TransactionID: "txn_123",
				MessageID:     "msg_123",
				Timestamp:     time.Now(),
				TTL:           "PT30S",
			},
			wantErr: true,
			errMsg:  "bap_uri is required",
		},
		{
			name: "Missing timestamp",
			context: ONDCContext{
				Domain:        "nic2004:60232",
				Action:        "search",
				BapID:         "buyer.example.com",
				BapURI:        "https://buyer.example.com",
				TransactionID: "txn_123",
				MessageID:     "msg_123",
				Timestamp:     time.Time{}, // Zero time
				TTL:           "PT30S",
			},
			wantErr: true,
			errMsg:  "timestamp is required",
		},
		{
			name: "Invalid action",
			context: ONDCContext{
				Domain:        "nic2004:60232",
				Action:        "on_sreach", // Typo
				BapID:         "buyer.example.com",
				BapURI:        "https://buyer.example.com",
				TransactionID: "txn_123",
				MessageID:     "msg_123",
				Timestamp:     time.Now(),
				TTL:           "PT30S",
			},
			wantErr: true,
			errMsg:  "invalid action: on_sreach",
		},
		{
			name: "Invalid TTL format",
			context: ONDCContext{
				Domain:        "nic2004:60232",
				Action:        "search",
				BapID:         "buyer.example.com",
				BapURI:        "https://buyer.example.com",
				TransactionID: "txn_123",
				MessageID:     "msg_123",
				Timestamp:     time.Now(),
				TTL:           "INVALID_TTL",
			},
			wantErr: true,
			errMsg:  "invalid ttl format",
		},
		{
			name: "Valid context with IGM action",
			context: ONDCContext{
				Domain:        "nic2004:60232",
				Action:        "issue",
				BapID:         "buyer.example.com",
				BapURI:        "https://buyer.example.com",
				TransactionID: "txn_123",
				MessageID:     "msg_123",
				Timestamp:     time.Now(),
				TTL:           "PT30S",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.context.Validate()
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

func TestONDCRequest_GetContext(t *testing.T) {
	req := &ONDCRequest{
		Context: ONDCContext{
			Domain:        "nic2004:60232",
			Action:        "search",
			TransactionID: "txn_123",
			MessageID:     "msg_123",
		},
		Message: map[string]interface{}{},
	}

	ctx := req.GetContext()
	assert.Equal(t, "nic2004:60232", ctx.Domain)
	assert.Equal(t, "search", ctx.Action)
	assert.Equal(t, "txn_123", ctx.TransactionID)
	assert.Equal(t, "msg_123", ctx.MessageID)
}

func TestONDCError_ToMap(t *testing.T) {
	err := ONDCError{
		Type: "CONTEXT_ERROR",
		Code: "65001",
		Path: "context.domain",
		Message: map[string]string{
			"en": "Invalid domain",
		},
	}

	errMap := err.ToMap()
	assert.Equal(t, "CONTEXT_ERROR", errMap["type"])
	assert.Equal(t, "65001", errMap["code"])
	assert.Equal(t, "context.domain", errMap["path"])
	assert.NotNil(t, errMap["message"])
}
