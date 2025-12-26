package auth

import (
	"context"

	"uois-gateway/internal/models"
	perrors "uois-gateway/pkg/errors"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const (
	ErrCodeAuthFailed            = 65002
	ErrCodeDependencyUnavailable = 65011
)

const (
	ErrMsgAuthFailed = "authentication failed"
)

type ClientRegistry interface {
	GetByClientID(ctx context.Context, clientID string) (*models.Client, error)
}

type ClientAuthService struct {
	registry ClientRegistry
	logger   *zap.Logger
}

func NewClientAuthService(registry ClientRegistry, logger *zap.Logger) *ClientAuthService {
	return &ClientAuthService{
		registry: registry,
		logger:   logger,
	}
}

func (s *ClientAuthService) AuthenticateClient(ctx context.Context, clientID, clientSecret, clientIP string) (*models.Client, error) {
	client, err := s.registry.GetByClientID(ctx, clientID)
	if err != nil {
		return s.handleRegistryError(err, clientID)
	}

	if !client.IsActive() {
		s.logger.Warn("authentication failed: client not active",
			zap.String("client_id", clientID),
			zap.String("status", client.Status),
			zap.String("client_ip", clientIP),
		)
		return nil, perrors.NewDomainError(ErrCodeAuthFailed, ErrMsgAuthFailed, "invalid credentials")
	}

	if !client.ValidateIP(clientIP) {
		s.logger.Warn("authentication failed: IP address not allowed",
			zap.String("client_id", clientID),
			zap.String("client_ip", clientIP),
			zap.Strings("allowed_ips", client.AllowedIPs),
		)
		return nil, perrors.NewDomainError(ErrCodeAuthFailed, ErrMsgAuthFailed, "invalid credentials")
	}

	if err := s.validateSecret(clientSecret, client.ClientSecretHash); err != nil {
		s.logger.Warn("authentication failed: invalid credentials",
			zap.String("client_id", clientID),
			zap.String("client_ip", clientIP),
		)
		return nil, perrors.NewDomainError(ErrCodeAuthFailed, ErrMsgAuthFailed, "invalid credentials")
	}

	return client, nil
}

func (s *ClientAuthService) handleRegistryError(err error, clientID string) (*models.Client, error) {
	domainErr, isDomainErr := err.(*perrors.DomainError)
	if isDomainErr && domainErr.Code == 65006 {
		s.logger.Info("authentication failed: client not found",
			zap.String("client_id", clientID),
		)
		return nil, perrors.NewDomainError(ErrCodeAuthFailed, ErrMsgAuthFailed, "invalid credentials")
	}

	s.logger.Error("authentication service unavailable: registry error",
		zap.String("client_id", clientID),
		zap.Error(err),
	)
	return nil, perrors.WrapDomainError(err, ErrCodeDependencyUnavailable, "authentication service unavailable", "dependency error")
}

func (s *ClientAuthService) validateSecret(secret, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret))
}
