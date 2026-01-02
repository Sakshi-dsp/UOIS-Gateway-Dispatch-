package client_registry

import (
	"context"
	"database/sql"
	"testing"

	"uois-gateway/internal/config"
	"uois-gateway/internal/models"
	domainErrors "uois-gateway/pkg/errors"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestClientRegistryRepository_UpsertClient_Success(t *testing.T) {
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

	client := &models.Client{
		ID:               "550e8400-e29b-41d4-a716-446655440000",
		ClientCode:       "ABC",
		ClientSecretHash: "hashed_secret",
		AllowedIPs:       []string{"192.168.1.0/24"},
		Status:           models.ClientStatusActive,
		Metadata:         map[string]interface{}{"key": "value"},
	}

	mock.ExpectExec(`INSERT INTO client_registry\.clients`).
		WithArgs(
			client.ID,
			client.ClientCode,
			client.ClientSecretHash,
			sqlmock.AnyArg(), // api_key_hash
			sqlmock.AnyArg(), // bap_id
			sqlmock.AnyArg(), // bap_uri
			sqlmock.AnyArg(), // allowed_ips
			sqlmock.AnyArg(), // rate_limit
			client.Status,
			sqlmock.AnyArg(), // metadata
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.UpsertClient(context.Background(), client)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClientRegistryRepository_GetByClientID_Success(t *testing.T) {
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

	clientID := "550e8400-e29b-41d4-a716-446655440000"

	rows := sqlmock.NewRows([]string{
		"client_id", "client_code", "client_secret_hash",
		"api_key_hash", "bap_id", "bap_uri", "allowed_ips",
		"rate_limit", "status", "metadata",
	}).
		AddRow(
			clientID,
			"ABC",
			"hashed_secret",
			"hashed_secret",             // api_key_hash
			"buyer.example.com",         // bap_id
			"https://buyer.example.com", // bap_uri
			"{192.168.1.0/24}",          // allowed_ips (PostgreSQL array format)
			100,                         // rate_limit
			models.ClientStatusActive,
			`{"key":"value"}`, // metadata
		)

	mock.ExpectQuery(`SELECT client_id, client_code, client_secret_hash`).
		WithArgs(clientID).
		WillReturnRows(rows)

	client, err := repo.GetByClientID(context.Background(), clientID)
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, clientID, client.ID)
	assert.Equal(t, "ABC", client.ClientCode)
	assert.Equal(t, models.ClientStatusActive, client.Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClientRegistryRepository_GetByClientID_NotFound(t *testing.T) {
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

	clientID := "550e8400-e29b-41d4-a716-446655440000"

	mock.ExpectQuery(`SELECT client_id, client_code, client_secret_hash`).
		WithArgs(clientID).
		WillReturnError(sql.ErrNoRows)

	client, err := repo.GetByClientID(context.Background(), clientID)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.True(t, domainErrors.IsDomainError(err))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClientRegistryRepository_GetByClientID_DBError(t *testing.T) {
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

	clientID := "550e8400-e29b-41d4-a716-446655440000"

	mock.ExpectQuery(`SELECT client_id, client_code, client_secret_hash`).
		WithArgs(clientID).
		WillReturnError(sql.ErrConnDone)

	client, err := repo.GetByClientID(context.Background(), clientID)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.True(t, domainErrors.IsDomainError(err))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClientRegistryRepository_UpdateStatus_Success(t *testing.T) {
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

	clientID := "550e8400-e29b-41d4-a716-446655440000"
	newStatus := models.ClientStatusSuspended

	mock.ExpectExec(`UPDATE client_registry\.clients`).
		WithArgs(newStatus, clientID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateStatus(context.Background(), clientID, newStatus)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
