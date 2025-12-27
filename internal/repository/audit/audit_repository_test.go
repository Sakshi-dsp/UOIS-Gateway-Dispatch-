package audit

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"uois-gateway/internal/config"
	"uois-gateway/pkg/errors"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestAuditRepository_StoreRequestResponseLog_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	logger := zap.NewNop()
	cfg := config.Config{
		PostgresE: config.PostgresConfig{
			DB: "test_db",
		},
	}

	repo := NewRepository(db, cfg, logger)

	log := &RequestResponseLog{
		TransactionID:  "txn-123",
		MessageID:      "msg-456",
		Action:         "search",
		RequestPayload: map[string]interface{}{"test": "data"},
		TraceID:        "trace-789",
		ClientID:       "client-001",
		CreatedAt:      time.Now(),
	}

	mock.ExpectExec(`INSERT INTO audit\.request_response_logs`).
		WithArgs(
			sqlmock.AnyArg(), // request_id (UUID)
			log.TransactionID,
			log.MessageID,
			log.Action,
			sqlmock.AnyArg(), // request_payload (JSONB)
			sqlmock.AnyArg(), // ack_payload (JSONB, nullable)
			sqlmock.AnyArg(), // callback_payload (JSONB, nullable)
			log.TraceID,
			log.ClientID,
			sqlmock.AnyArg(), // search_id (UUID, nullable)
			sqlmock.AnyArg(), // quote_id (UUID, nullable)
			sqlmock.AnyArg(), // order_id (nullable)
			sqlmock.AnyArg(), // dispatch_order_id (nullable)
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.StoreRequestResponseLog(context.Background(), log)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuditRepository_StoreRequestResponseLog_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	logger := zap.NewNop()
	cfg := config.Config{
		PostgresE: config.PostgresConfig{
			DB: "test_db",
		},
	}

	repo := NewRepository(db, cfg, logger)

	log := &RequestResponseLog{
		Action:         "search",
		RequestPayload: map[string]interface{}{"test": "data"},
		CreatedAt:      time.Now(),
	}

	mock.ExpectExec(`INSERT INTO audit\.request_response_logs`).
		WillReturnError(sql.ErrConnDone)

	err = repo.StoreRequestResponseLog(context.Background(), log)
	assert.Error(t, err)
	assert.True(t, errors.IsDomainError(err))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuditRepository_StoreCallbackDeliveryLog_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	logger := zap.NewNop()
	cfg := config.Config{
		PostgresE: config.PostgresConfig{
			DB: "test_db",
		},
	}

	repo := NewRepository(db, cfg, logger)

	log := &CallbackDeliveryLog{
		RequestID:   "req-123",
		CallbackURL: "https://example.com/callback",
		AttemptNo:   1,
		Status:      "success",
		CreatedAt:   time.Now(),
	}

	mock.ExpectExec(`INSERT INTO audit\.callback_delivery_logs`).
		WithArgs(
			sqlmock.AnyArg(), // request_id (UUID)
			log.CallbackURL,
			log.AttemptNo,
			log.Status,
			sqlmock.AnyArg(), // error (nullable)
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.StoreCallbackDeliveryLog(context.Background(), log)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuditRepository_StoreCallbackDeliveryLog_WithError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	logger := zap.NewNop()
	cfg := config.Config{
		PostgresE: config.PostgresConfig{
			DB: "test_db",
		},
	}

	repo := NewRepository(db, cfg, logger)

	log := &CallbackDeliveryLog{
		RequestID:   "req-123",
		CallbackURL: "https://example.com/callback",
		AttemptNo:   2,
		Status:      "failed",
		Error:       sql.NullString{String: "timeout", Valid: true},
		CreatedAt:   time.Now(),
	}

	mock.ExpectExec(`INSERT INTO audit\.callback_delivery_logs`).
		WithArgs(
			sqlmock.AnyArg(),
			log.CallbackURL,
			log.AttemptNo,
			log.Status,
			log.Error,
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.StoreCallbackDeliveryLog(context.Background(), log)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
