package auth

import (
	"context"
	"errors"
	"testing"

	"uois-gateway/internal/models"
	perrors "uois-gateway/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type MockClientRegistry struct {
	mock.Mock
}

func (m *MockClientRegistry) GetByClientID(ctx context.Context, clientID string) (*models.Client, error) {
	args := m.Called(ctx, clientID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Client), args.Error(1)
}

func TestAuthenticateClient_Success(t *testing.T) {
	mockRepo := new(MockClientRegistry)
	logger := zap.NewNop()
	service := NewClientAuthService(mockRepo, logger)

	clientID := "test-client-123"
	clientSecret := "secret123"
	clientIP := "192.168.1.100"

	hashedSecret, _ := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)

	client := &models.Client{
		ID:               "uuid-123",
		ClientCode:       "TEST_CLIENT",
		ClientSecretHash: string(hashedSecret),
		Status:           "ACTIVE",
		AllowedIPs:       []string{"192.168.1.0/24"},
	}

	mockRepo.On("GetByClientID", mock.Anything, clientID).Return(client, nil)

	ctx := context.Background()
	authenticatedClient, err := service.AuthenticateClient(ctx, clientID, clientSecret, clientIP)

	assert.NoError(t, err)
	assert.NotNil(t, authenticatedClient)
	assert.Equal(t, client.ClientCode, authenticatedClient.ClientCode)
	mockRepo.AssertExpectations(t)
}

func TestAuthenticateClient_ClientNotFound(t *testing.T) {
	mockRepo := new(MockClientRegistry)
	logger := zap.NewNop()
	service := NewClientAuthService(mockRepo, logger)

	clientID := "non-existent"
	clientSecret := "secret123"
	clientIP := "192.168.1.100"

	mockRepo.On("GetByClientID", mock.Anything, clientID).Return(nil, perrors.NewDomainError(65006, "client not found", ""))

	ctx := context.Background()
	authenticatedClient, err := service.AuthenticateClient(ctx, clientID, clientSecret, clientIP)

	assert.Error(t, err)
	assert.Nil(t, authenticatedClient)
	assert.True(t, perrors.IsDomainError(err))
	domainErr, _ := err.(*perrors.DomainError)
	assert.Equal(t, ErrCodeAuthFailed, domainErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestAuthenticateClient_InactiveStatus(t *testing.T) {
	mockRepo := new(MockClientRegistry)
	logger := zap.NewNop()
	service := NewClientAuthService(mockRepo, logger)

	clientID := "test-client-123"
	clientSecret := "secret123"
	clientIP := "192.168.1.100"

	hashedSecret, _ := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)

	client := &models.Client{
		ID:               "uuid-123",
		ClientCode:       "TEST_CLIENT",
		ClientSecretHash: string(hashedSecret),
		Status:           "SUSPENDED",
		AllowedIPs:       []string{"192.168.1.0/24"},
	}

	mockRepo.On("GetByClientID", mock.Anything, clientID).Return(client, nil)

	ctx := context.Background()
	authenticatedClient, err := service.AuthenticateClient(ctx, clientID, clientSecret, clientIP)

	assert.Error(t, err)
	assert.Nil(t, authenticatedClient)
	assert.True(t, perrors.IsDomainError(err))
	domainErr, _ := err.(*perrors.DomainError)
	assert.Equal(t, ErrCodeAuthFailed, domainErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestAuthenticateClient_IPNotAllowed(t *testing.T) {
	mockRepo := new(MockClientRegistry)
	logger := zap.NewNop()
	service := NewClientAuthService(mockRepo, logger)

	clientID := "test-client-123"
	clientSecret := "secret123"
	clientIP := "10.0.0.1"

	hashedSecret, _ := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)

	client := &models.Client{
		ID:               "uuid-123",
		ClientCode:       "TEST_CLIENT",
		ClientSecretHash: string(hashedSecret),
		Status:           "ACTIVE",
		AllowedIPs:       []string{"192.168.1.0/24"},
	}

	mockRepo.On("GetByClientID", mock.Anything, clientID).Return(client, nil)

	ctx := context.Background()
	authenticatedClient, err := service.AuthenticateClient(ctx, clientID, clientSecret, clientIP)

	assert.Error(t, err)
	assert.Nil(t, authenticatedClient)
	assert.True(t, perrors.IsDomainError(err))
	domainErr, _ := err.(*perrors.DomainError)
	assert.Equal(t, ErrCodeAuthFailed, domainErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestAuthenticateClient_NoIPRestrictions(t *testing.T) {
	mockRepo := new(MockClientRegistry)
	logger := zap.NewNop()
	service := NewClientAuthService(mockRepo, logger)

	clientID := "test-client-123"
	clientSecret := "secret123"
	clientIP := "10.0.0.1"

	hashedSecret, _ := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)

	client := &models.Client{
		ID:               "uuid-123",
		ClientCode:       "TEST_CLIENT",
		ClientSecretHash: string(hashedSecret),
		Status:           "ACTIVE",
		AllowedIPs:       nil,
	}

	mockRepo.On("GetByClientID", mock.Anything, clientID).Return(client, nil)

	ctx := context.Background()
	authenticatedClient, err := service.AuthenticateClient(ctx, clientID, clientSecret, clientIP)

	assert.NoError(t, err)
	assert.NotNil(t, authenticatedClient)
	mockRepo.AssertExpectations(t)
}

func TestAuthenticateClient_InvalidSecret(t *testing.T) {
	mockRepo := new(MockClientRegistry)
	logger := zap.NewNop()
	service := NewClientAuthService(mockRepo, logger)

	clientID := "test-client-123"
	clientSecret := "wrong-secret"
	clientIP := "192.168.1.100"

	hashedSecret, _ := bcrypt.GenerateFromPassword([]byte("correct-secret"), bcrypt.DefaultCost)

	client := &models.Client{
		ID:               "uuid-123",
		ClientCode:       "TEST_CLIENT",
		ClientSecretHash: string(hashedSecret),
		Status:           "ACTIVE",
		AllowedIPs:       []string{"192.168.1.0/24"},
	}

	mockRepo.On("GetByClientID", mock.Anything, clientID).Return(client, nil)

	ctx := context.Background()
	authenticatedClient, err := service.AuthenticateClient(ctx, clientID, clientSecret, clientIP)

	assert.Error(t, err)
	assert.Nil(t, authenticatedClient)
	assert.True(t, perrors.IsDomainError(err))
	domainErr, _ := err.(*perrors.DomainError)
	assert.Equal(t, ErrCodeAuthFailed, domainErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestAuthenticateClient_RegistryInfrastructureError(t *testing.T) {
	mockRepo := new(MockClientRegistry)
	logger := zap.NewNop()
	service := NewClientAuthService(mockRepo, logger)

	clientID := "test-client-123"
	clientSecret := "secret123"
	clientIP := "192.168.1.100"

	infraError := errors.New("redis connection timeout")
	mockRepo.On("GetByClientID", mock.Anything, clientID).Return(nil, infraError)

	ctx := context.Background()
	authenticatedClient, err := service.AuthenticateClient(ctx, clientID, clientSecret, clientIP)

	assert.Error(t, err)
	assert.Nil(t, authenticatedClient)
	assert.True(t, perrors.IsDomainError(err))
	domainErr, _ := err.(*perrors.DomainError)
	assert.Equal(t, ErrCodeDependencyUnavailable, domainErr.Code)
	mockRepo.AssertExpectations(t)
}
