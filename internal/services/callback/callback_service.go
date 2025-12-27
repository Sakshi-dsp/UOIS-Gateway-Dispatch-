package callback

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"uois-gateway/internal/config"
	"uois-gateway/pkg/errors"

	"go.uber.org/zap"
)

// Service sends HTTP callbacks to client callback URLs
// ONDC requires all Seller NP callbacks to be HTTP-signed with digest and Authorization headers
type Service struct {
	httpClient *http.Client
	config     config.CallbackConfig
	signer     Signer // ONDC-required: HTTP signature generator
	logger     *zap.Logger
}

// NewService creates a new callback service
// signer can be nil for testing, but production must provide a valid signer
func NewService(cfg config.CallbackConfig, signer Signer, logger *zap.Logger) *Service {
	return &Service{
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.HTTPTimeoutSeconds) * time.Second,
		},
		config: cfg,
		signer: signer,
		logger: logger,
	}
}

// SendCallback sends a callback to the specified URL
// ONDC requires: Digest header (SHA-256) and Authorization header (HTTP signature)
func (s *Service) SendCallback(ctx context.Context, callbackURL string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return errors.WrapDomainError(err, 65020, "callback serialization failed", "failed to marshal payload")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callbackURL, bytes.NewReader(body))
	if err != nil {
		return errors.WrapDomainError(err, 65020, "callback request failed", "failed to create request")
	}

	// Set Content-Type header (required for ONDC signature)
	req.Header.Set("Content-Type", "application/json")

	// Calculate and set Digest header (ONDC requirement: SHA-256 hash of body)
	digest := s.calculateDigest(body)
	req.Header.Set("Digest", digest)

	// Generate HTTP signature if signer is provided (ONDC requirement)
	if s.signer != nil {
		headers := map[string]string{
			"Content-Type": "application/json",
			"Digest":       digest,
		}

		authHeader, err := s.signer.SignRequest(ctx, http.MethodPost, callbackURL, body, headers)
		if err != nil {
			return errors.WrapDomainError(err, 65020, "callback signing failed", "failed to generate HTTP signature")
		}

		req.Header.Set("Authorization", authHeader)
	} else {
		// Log warning if signer is not provided (non-production scenario)
		s.logger.Warn("callback sent without HTTP signature (signer not provided)", zap.String("url", callbackURL))
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return errors.WrapDomainError(err, 65020, "callback delivery failed", "http request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.NewDomainError(65020, "callback delivery failed", fmt.Sprintf("unexpected status: %d", resp.StatusCode))
	}

	return nil
}

// calculateDigest computes SHA-256 digest of the body in ONDC format
// Format: "SHA-256=<base64(sha256(body))>"
func (s *Service) calculateDigest(body []byte) string {
	hash := sha256.Sum256(body)
	digest := base64.StdEncoding.EncodeToString(hash[:])
	return fmt.Sprintf("SHA-256=%s", digest)
}
