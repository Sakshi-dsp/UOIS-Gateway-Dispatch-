package callback

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"uois-gateway/internal/config"
	"uois-gateway/internal/services/ondc"

	"go.uber.org/zap"
)

// ONDCSigner adapts ONDCAuthService to implement Signer interface for HTTP callbacks
type ONDCSigner struct {
	ondcAuth *ondc.ONDCAuthService
}

// NewONDCSigner creates a new ONDC signer
func NewONDCSigner(ondcAuth *ondc.ONDCAuthService) *ONDCSigner {
	return &ONDCSigner{
		ondcAuth: ondcAuth,
	}
}

// SignRequest generates an HTTP signature for the given request
// ONDC requires signing the digest and headers for callback requests
func (s *ONDCSigner) SignRequest(ctx context.Context, method, url string, body []byte, headers map[string]string) (string, error) {
	hash := sha256.Sum256(body)
	digest := base64.StdEncoding.EncodeToString(hash[:])
	
	signaturePayload := fmt.Sprintf("%s %s\nContent-Type: %s\nDigest: SHA-256=%s",
		method,
		url,
		headers["Content-Type"],
		digest,
	)
	
	signatureBytes := []byte(signaturePayload)
	authHeader, err := s.ondcAuth.SignResponse(signatureBytes)
	if err != nil {
		return "", err
	}
	
	return authHeader, nil
}

// NewONDCSignerFromConfig creates a new ONDC signer from config
// This is a helper function for main.go initialization
func NewONDCSignerFromConfig(cfg config.ONDCConfig, logger *zap.Logger) (*ONDCSigner, error) {
	registry := &mockRegistryClient{}
	ondcAuth, err := ondc.NewONDCAuthService(registry, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create ONDC auth service: %w", err)
	}
	return NewONDCSigner(ondcAuth), nil
}

// mockRegistryClient is a placeholder for registry client
// TODO: Implement actual registry client
type mockRegistryClient struct{}

func (m *mockRegistryClient) LookupPublicKey(ctx context.Context, subscriberID, ukID string) (string, error) {
	return "", fmt.Errorf("registry client not implemented")
}

