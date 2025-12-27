package audit

import (
	"context"
	"database/sql"
	"time"

	"uois-gateway/internal/repository/audit"
	"uois-gateway/pkg/errors"

	"go.uber.org/zap"
)

// Repository interface for audit operations
type Repository interface {
	StoreRequestResponseLog(ctx context.Context, log *audit.RequestResponseLog) error
	StoreCallbackDeliveryLog(ctx context.Context, log *audit.CallbackDeliveryLog) error
}

// Service provides audit logging functionality
type Service struct {
	repo   Repository
	logger *zap.Logger
}

// NewService creates a new audit service
func NewService(repo Repository, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// LogRequestResponse logs a request/response pair
func (s *Service) LogRequestResponse(ctx context.Context, req *RequestResponseLogParams) error {
	log := &audit.RequestResponseLog{
		TransactionID:   req.TransactionID,
		MessageID:       req.MessageID,
		Action:          req.Action,
		RequestPayload:  req.RequestPayload,
		ACKPayload:      req.ACKPayload,
		CallbackPayload: req.CallbackPayload,
		TraceID:         req.TraceID,
		ClientID:        req.ClientID,
		SearchID:        req.SearchID,
		QuoteID:         req.QuoteID,
		OrderID:         req.OrderID,
		DispatchOrderID: req.DispatchOrderID,
		CreatedAt:       time.Now(),
	}

	if err := s.repo.StoreRequestResponseLog(ctx, log); err != nil {
		s.logger.Error("failed to log request response", zap.Error(err))
		return errors.WrapDomainError(err, 65020, "audit logging failed", "failed to store log")
	}

	return nil
}

// LogCallbackDelivery logs a callback delivery attempt
func (s *Service) LogCallbackDelivery(ctx context.Context, req *CallbackDeliveryLogParams) error {
	log := &audit.CallbackDeliveryLog{
		RequestID:   req.RequestID,
		CallbackURL: req.CallbackURL,
		AttemptNo:   req.AttemptNo,
		Status:      req.Status,
		Error:       sql.NullString{String: req.Error, Valid: req.Error != ""},
		CreatedAt:   time.Now(),
	}

	if err := s.repo.StoreCallbackDeliveryLog(ctx, log); err != nil {
		s.logger.Error("failed to log callback delivery", zap.Error(err))
		return errors.WrapDomainError(err, 65020, "callback delivery logging failed", "failed to store log")
	}

	return nil
}

// RequestResponseLogParams contains parameters for request/response logging
type RequestResponseLogParams struct {
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
}

// CallbackDeliveryLogParams contains parameters for callback delivery logging
type CallbackDeliveryLogParams struct {
	RequestID   string
	CallbackURL string
	AttemptNo   int
	Status      string
	Error       string
}
