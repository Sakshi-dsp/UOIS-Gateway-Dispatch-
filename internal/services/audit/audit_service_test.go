package audit

import (
	"context"
	"errors"
	"testing"

	"uois-gateway/internal/repository/audit"
	domainerrors "uois-gateway/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) StoreRequestResponseLog(ctx context.Context, log *audit.RequestResponseLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockRepository) StoreCallbackDeliveryLog(ctx context.Context, log *audit.CallbackDeliveryLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func TestAuditService_LogRequestResponse_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zap.NewNop()
	service := NewService(mockRepo, logger)

	params := &RequestResponseLogParams{
		TransactionID:  "txn-123",
		MessageID:      "msg-456",
		Action:         "search",
		RequestPayload: map[string]interface{}{"test": "data"},
		TraceID:        "trace-789",
		ClientID:       "client-001",
	}

	mockRepo.On("StoreRequestResponseLog", mock.Anything, mock.MatchedBy(func(log *audit.RequestResponseLog) bool {
		return log.TransactionID == params.TransactionID &&
			log.MessageID == params.MessageID &&
			log.Action == params.Action &&
			log.TraceID == params.TraceID &&
			log.ClientID == params.ClientID
	})).Return(nil)

	err := service.LogRequestResponse(context.Background(), params)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditService_LogRequestResponse_RepositoryError(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zap.NewNop()
	service := NewService(mockRepo, logger)

	params := &RequestResponseLogParams{
		Action:         "search",
		RequestPayload: map[string]interface{}{"test": "data"},
	}

	mockRepo.On("StoreRequestResponseLog", mock.Anything, mock.Anything).
		Return(errors.New("database error"))

	err := service.LogRequestResponse(context.Background(), params)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsDomainError(err))
	mockRepo.AssertExpectations(t)
}

func TestAuditService_LogCallbackDelivery_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zap.NewNop()
	service := NewService(mockRepo, logger)

	params := &CallbackDeliveryLogParams{
		RequestID:   "req-123",
		CallbackURL: "https://example.com/callback",
		AttemptNo:   1,
		Status:      "success",
	}

	mockRepo.On("StoreCallbackDeliveryLog", mock.Anything, mock.MatchedBy(func(log *audit.CallbackDeliveryLog) bool {
		return log.RequestID == params.RequestID &&
			log.CallbackURL == params.CallbackURL &&
			log.AttemptNo == params.AttemptNo &&
			log.Status == params.Status
	})).Return(nil)

	err := service.LogCallbackDelivery(context.Background(), params)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditService_LogCallbackDelivery_WithError(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zap.NewNop()
	service := NewService(mockRepo, logger)

	params := &CallbackDeliveryLogParams{
		RequestID:   "req-123",
		CallbackURL: "https://example.com/callback",
		AttemptNo:   2,
		Status:      "failed",
		Error:       "timeout",
	}

	mockRepo.On("StoreCallbackDeliveryLog", mock.Anything, mock.MatchedBy(func(log *audit.CallbackDeliveryLog) bool {
		return log.RequestID == params.RequestID &&
			log.Status == params.Status &&
			log.Error.Valid &&
			log.Error.String == params.Error
	})).Return(nil)

	err := service.LogCallbackDelivery(context.Background(), params)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
