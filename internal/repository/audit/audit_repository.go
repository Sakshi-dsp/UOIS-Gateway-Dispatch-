package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"uois-gateway/internal/config"
	"uois-gateway/pkg/errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RequestResponseLog represents an audit log entry for request/response
type RequestResponseLog struct {
	RequestID       string
	TransactionID   string
	MessageID       string
	Action          string
	RequestPayload  map[string]interface{}
	ACKPayload      map[string]interface{}
	CallbackPayload map[string]interface{}
	TraceID         string
	ClientID        string
	SearchID        string
	QuoteID         string
	OrderID         string
	DispatchOrderID string
	CreatedAt       time.Time
}

// CallbackDeliveryLog represents an audit log entry for callback delivery
type CallbackDeliveryLog struct {
	RequestID   string
	CallbackURL string
	AttemptNo   int
	Status      string
	Error       sql.NullString
	CreatedAt   time.Time
}

// DBClient interface for database operations
type DBClient interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// Repository handles audit log storage
type Repository struct {
	db     DBClient
	config config.Config
	logger *zap.Logger
}

// NewRepository creates a new audit repository
func NewRepository(db DBClient, cfg config.Config, logger *zap.Logger) *Repository {
	return &Repository{
		db:     db,
		config: cfg,
		logger: logger,
	}
}

// StoreRequestResponseLog stores a request/response log entry
// Note: created_at is set by the database (DEFAULT now())
func (r *Repository) StoreRequestResponseLog(ctx context.Context, log *RequestResponseLog) error {
	if log.RequestID == "" {
		log.RequestID = uuid.New().String()
	}

	requestPayloadJSON, err := json.Marshal(log.RequestPayload)
	if err != nil {
		return errors.WrapDomainError(err, 65020, "audit log serialization failed", "failed to marshal request payload")
	}

	var ackPayloadJSON []byte
	if log.ACKPayload != nil {
		ackPayloadJSON, err = json.Marshal(log.ACKPayload)
		if err != nil {
			return errors.WrapDomainError(err, 65020, "audit log serialization failed", "failed to marshal ack payload")
		}
	}

	var callbackPayloadJSON []byte
	if log.CallbackPayload != nil {
		callbackPayloadJSON, err = json.Marshal(log.CallbackPayload)
		if err != nil {
			return errors.WrapDomainError(err, 65020, "audit log serialization failed", "failed to marshal callback payload")
		}
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	query := `INSERT INTO audit.request_response_logs (
		request_id, transaction_id, message_id, action, request_payload,
		ack_payload, callback_payload, trace_id, client_id,
		search_id, quote_id, order_id, dispatch_order_id
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err = r.db.ExecContext(ctx, query,
		log.RequestID,
		log.TransactionID,
		log.MessageID,
		log.Action,
		requestPayloadJSON,
		sqlNullJSONB(ackPayloadJSON),
		sqlNullJSONB(callbackPayloadJSON),
		log.TraceID,
		log.ClientID,
		sqlNullUUID(log.SearchID),
		sqlNullUUID(log.QuoteID),
		sqlNullString(log.OrderID),
		sqlNullString(log.DispatchOrderID),
	)

	if err != nil {
		r.logger.Error("failed to store request response log", zap.Error(err))
		return errors.WrapDomainError(err, 65020, "audit log storage failed", "database error")
	}

	r.logger.Debug("audit request_response stored",
		zap.String("request_id", log.RequestID),
		zap.String("action", log.Action),
		zap.String("trace_id", log.TraceID),
	)

	return nil
}

// StoreCallbackDeliveryLog stores a callback delivery log entry
// Note: created_at is set by the database (DEFAULT now())
// Note: Caller must guarantee monotonic attempt numbers
func (r *Repository) StoreCallbackDeliveryLog(ctx context.Context, log *CallbackDeliveryLog) error {
	if log.RequestID == "" {
		log.RequestID = uuid.New().String()
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	query := `INSERT INTO audit.callback_delivery_logs (
		request_id, callback_url, attempt_no, status, error
	) VALUES ($1, $2, $3, $4, $5)`

	_, err := r.db.ExecContext(ctx, query,
		log.RequestID,
		log.CallbackURL,
		log.AttemptNo,
		log.Status,
		log.Error,
	)

	if err != nil {
		r.logger.Error("failed to store callback delivery log", zap.Error(err))
		return errors.WrapDomainError(err, 65020, "callback delivery log storage failed", "database error")
	}

	r.logger.Debug("audit callback_delivery stored",
		zap.String("request_id", log.RequestID),
		zap.String("callback_url", log.CallbackURL),
		zap.Int("attempt_no", log.AttemptNo),
		zap.String("status", log.Status),
	)

	return nil
}

func sqlNullJSONB(data []byte) sql.NullString {
	if len(data) == 0 {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: string(data), Valid: true}
}

func sqlNullUUID(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func sqlNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
