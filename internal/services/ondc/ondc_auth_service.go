package ondc

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"uois-gateway/internal/config"
	"uois-gateway/pkg/errors"

	"go.uber.org/zap"
	"golang.org/x/crypto/blake2b"
)

// RegistryClient interface for ONDC network registry lookup
// NOTE: For production optimization, consider implementing an LRU cache with TTL
// to avoid hot-path registry lookups. This is optional and does not affect correctness.
type RegistryClient interface {
	LookupPublicKey(ctx context.Context, subscriberID, ukID string) (string, error)
}

// ONDCAuthService handles ONDC request/response signing and verification
type ONDCAuthService struct {
	registry     RegistryClient
	config       config.ONDCConfig
	logger       *zap.Logger
	privateKey   ed25519.PrivateKey
	publicKey    ed25519.PublicKey
	subscriberID string
	ukID         string
}

// NewONDCAuthService creates a new ONDC authentication service
func NewONDCAuthService(registry RegistryClient, cfg config.ONDCConfig, logger *zap.Logger) (*ONDCAuthService, error) {
	service := &ONDCAuthService{
		registry:     registry,
		config:       cfg,
		logger:       logger,
		subscriberID: cfg.SubscriberID,
		ukID:         cfg.UkID,
	}

	if err := service.loadKeys(); err != nil {
		return nil, fmt.Errorf("failed to load keys: %w", err)
	}

	return service, nil
}

// loadKeys loads ed25519 keys from file paths
func (s *ONDCAuthService) loadKeys() error {
	privateKeyBytes, err := os.ReadFile(s.config.PrivateKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key file: %w", err)
	}

	publicKeyBytes, err := os.ReadFile(s.config.PublicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key file: %w", err)
	}

	privateKeyDecoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(privateKeyBytes)))
	if err != nil {
		return fmt.Errorf("failed to decode private key: %w", err)
	}

	publicKeyDecoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(publicKeyBytes)))
	if err != nil {
		return fmt.Errorf("failed to decode public key: %w", err)
	}

	if len(privateKeyDecoded) != ed25519.PrivateKeySize {
		return fmt.Errorf("invalid private key size: expected %d, got %d", ed25519.PrivateKeySize, len(privateKeyDecoded))
	}

	if len(publicKeyDecoded) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key size: expected %d, got %d", ed25519.PublicKeySize, len(publicKeyDecoded))
	}

	s.privateKey = ed25519.PrivateKey(privateKeyDecoded)
	s.publicKey = ed25519.PublicKey(publicKeyDecoded)

	return nil
}

// VerifyRequestSignature verifies the signature of an incoming ONDC request
// NOTE: payload must be the exact raw JSON bytes as received. Do not re-marshal or normalize
// whitespace, as ONDC requires exact byte-for-byte matching for signature verification.
func (s *ONDCAuthService) VerifyRequestSignature(ctx context.Context, authHeader string, payload []byte) error {
	if strings.TrimSpace(authHeader) == "" {
		return errors.NewDomainError(65002, "authentication failed", "empty authorization header")
	}

	authParams, err := s.parseAuthHeader(authHeader)
	if err != nil {
		return errors.NewDomainError(65002, "authentication failed", "invalid auth header")
	}

	keyID := authParams["keyId"]
	signature := authParams["signature"]

	subscriberID, ukID, algorithm, err := s.parseKeyID(keyID)
	if err != nil {
		return errors.NewDomainError(65002, "authentication failed", "invalid keyId format")
	}

	if algorithm != "ed25519" {
		return errors.NewDomainError(65002, "authentication failed", "unsupported algorithm")
	}

	publicKeyBase64, err := s.registry.LookupPublicKey(ctx, subscriberID, ukID)
	if err != nil {
		return errors.WrapDomainError(err, 65011, "registry unavailable", "dependency")
	}

	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return errors.NewDomainError(65002, "authentication failed", "invalid public key format")
	}

	if len(publicKeyBytes) != ed25519.PublicKeySize {
		return errors.NewDomainError(65002, "authentication failed", "invalid registry public key size")
	}

	publicKey := ed25519.PublicKey(publicKeyBytes)

	// Generate Blake2b hash of payload
	hash := blake2b.Sum256(payload)

	// Decode signature
	signatureBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return errors.NewDomainError(65002, "authentication failed", "invalid signature format")
	}

	// Verify signature (signature is over the Blake2b hash directly)
	if !ed25519.Verify(publicKey, hash[:], signatureBytes) {
		return errors.NewDomainError(65002, "authentication failed", "signature verification failed")
	}

	return nil
}

// VerifyTimestamp validates the request timestamp to prevent replay attacks
func (s *ONDCAuthService) VerifyTimestamp(timestampStr string) error {
	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return errors.NewDomainError(65003, "stale request", "invalid timestamp format")
	}

	now := time.Now().UTC()
	window := time.Duration(s.config.TimestampWindow) * time.Second
	diff := now.Sub(timestamp)

	if diff > window || diff < -window {
		return errors.NewDomainError(65003, "stale request", "timestamp outside acceptable window")
	}

	return nil
}

// SignResponse signs an outgoing ONDC response
// NOTE: payload must be the exact raw JSON bytes to be sent. Do not re-marshal or normalize
// whitespace, as ONDC requires exact byte-for-byte matching for signature verification.
func (s *ONDCAuthService) SignResponse(payload []byte) (string, error) {
	if len(s.privateKey) == 0 {
		return "", errors.NewDomainError(65020, "internal error", "private key not loaded")
	}

	if s.subscriberID == "" || s.ukID == "" {
		return "", errors.NewDomainError(65020, "internal error", "subscriber identity not configured")
	}

	hash := blake2b.Sum256(payload)
	signature := ed25519.Sign(s.privateKey, hash[:])
	signatureBase64 := base64.StdEncoding.EncodeToString(signature)

	keyID := fmt.Sprintf("%s|%s|ed25519", s.subscriberID, s.ukID)
	authHeader := fmt.Sprintf(`keyId="%s", signature="%s"`, keyID, signatureBase64)

	return authHeader, nil
}

// parseAuthHeader parses Authorization header as key-value pairs
// Returns error if required fields (keyId, signature) are missing
func (s *ONDCAuthService) parseAuthHeader(header string) (map[string]string, error) {
	params := make(map[string]string)
	parts := strings.Split(header, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		idx := strings.Index(part, "=")
		if idx == -1 {
			continue
		}

		key := strings.TrimSpace(part[:idx])
		value := strings.TrimSpace(part[idx+1:])
		value = strings.Trim(value, `"`)

		if key != "" && value != "" {
			params[key] = value
		}
	}

	if _, ok := params["keyId"]; !ok || params["keyId"] == "" {
		return nil, fmt.Errorf("missing required keyId")
	}

	if _, ok := params["signature"]; !ok || params["signature"] == "" {
		return nil, fmt.Errorf("missing required signature")
	}

	return params, nil
}

// parseKeyID extracts subscriber_id, ukID, and algorithm from keyID
func (s *ONDCAuthService) parseKeyID(keyID string) (subscriberID, ukID, algorithm string, err error) {
	parts := strings.Split(keyID, "|")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid keyID format")
	}

	return parts[0], parts[1], parts[2], nil
}
