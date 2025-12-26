package ondc

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
	"testing"
	"time"

	"uois-gateway/internal/config"
	"uois-gateway/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"golang.org/x/crypto/blake2b"
)

// MockRegistryClient is a mock for ONDC registry client
type MockRegistryClient struct {
	mock.Mock
}

func (m *MockRegistryClient) LookupPublicKey(ctx context.Context, subscriberID, ukID string) (string, error) {
	args := m.Called(ctx, subscriberID, ukID)
	return args.String(0), args.Error(1)
}

func createTestService(t *testing.T, cfg config.ONDCConfig) (*ONDCAuthService, *MockRegistryClient) {
	mockRegistry := new(MockRegistryClient)
	logger := zap.NewNop()

	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)

	privateKeyBase64 := base64.StdEncoding.EncodeToString(privateKey)
	publicKeyBase64 := base64.StdEncoding.EncodeToString(publicKey)

	tmpDir := t.TempDir()
	privateKeyPath := tmpDir + "/private.key"
	publicKeyPath := tmpDir + "/public.key"

	err = os.WriteFile(privateKeyPath, []byte(privateKeyBase64), 0600)
	assert.NoError(t, err)
	err = os.WriteFile(publicKeyPath, []byte(publicKeyBase64), 0600)
	assert.NoError(t, err)

	cfg.PrivateKeyPath = privateKeyPath
	cfg.PublicKeyPath = publicKeyPath

	if cfg.SubscriberID == "" {
		cfg.SubscriberID = "seller.com"
	}
	if cfg.UkID == "" {
		cfg.UkID = "SELLER_UKID"
	}

	service, err := NewONDCAuthService(mockRegistry, cfg, logger)
	assert.NoError(t, err)
	return service, mockRegistry
}

func TestONDCAuthService_NewONDCAuthService_KeyLoadFailure(t *testing.T) {
	mockRegistry := new(MockRegistryClient)
	logger := zap.NewNop()
	cfg := config.ONDCConfig{
		PrivateKeyPath: "/nonexistent/private.key",
		PublicKeyPath:  "/nonexistent/public.key",
		SubscriberID:   "seller.com",
		UkID:           "SELLER_UKID",
	}

	service, err := NewONDCAuthService(mockRegistry, cfg, logger)

	assert.Error(t, err)
	assert.Nil(t, service)
}

func TestONDCAuthService_VerifyRequestSignature_Success(t *testing.T) {
	cfg := config.ONDCConfig{
		NetworkRegistryURL: "https://registry.ondc.org",
		TimestampWindow:    300,
		SubscriberID:       "seller.com",
		UkID:               "SELLER_UKID",
	}
	service, mockRegistry := createTestService(t, cfg)

	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)
	publicKeyBase64 := base64.StdEncoding.EncodeToString(publicKey)

	mockRegistry.On("LookupPublicKey", mock.Anything, "buyer.com", "UKID1").
		Return(publicKeyBase64, nil)

	payload := []byte(`{"context":{"timestamp":"2024-01-15T10:00:00Z"}}`)
	hash := blake2b.Sum256(payload)
	signature := ed25519.Sign(privateKey, hash[:])
	signatureBase64 := base64.StdEncoding.EncodeToString(signature)

	authHeader := fmt.Sprintf(`keyId="buyer.com|UKID1|ed25519", signature="%s"`, signatureBase64)

	err = service.VerifyRequestSignature(context.Background(), authHeader, payload)

	assert.NoError(t, err)
	mockRegistry.AssertExpectations(t)
}

func TestONDCAuthService_VerifyRequestSignature_EmptyHeader(t *testing.T) {
	service, _ := createTestService(t, config.ONDCConfig{})

	authHeader := ""
	payload := []byte(`{"context":{"timestamp":"2024-01-15T10:00:00Z"}}`)

	err := service.VerifyRequestSignature(context.Background(), authHeader, payload)

	assert.Error(t, err)
	assert.True(t, errors.IsDomainError(err))
	domainErr, _ := err.(*errors.DomainError)
	assert.Equal(t, 65002, domainErr.Code)
	assert.Contains(t, domainErr.Details, "empty authorization header")
}

func TestONDCAuthService_VerifyRequestSignature_WhitespaceOnlyHeader(t *testing.T) {
	service, _ := createTestService(t, config.ONDCConfig{})

	authHeader := "   \t\n  "
	payload := []byte(`{"context":{"timestamp":"2024-01-15T10:00:00Z"}}`)

	err := service.VerifyRequestSignature(context.Background(), authHeader, payload)

	assert.Error(t, err)
	assert.True(t, errors.IsDomainError(err))
	domainErr, _ := err.(*errors.DomainError)
	assert.Equal(t, 65002, domainErr.Code)
	assert.Contains(t, domainErr.Details, "empty authorization header")
}

func TestONDCAuthService_VerifyRequestSignature_InvalidHeader(t *testing.T) {
	service, _ := createTestService(t, config.ONDCConfig{})

	authHeader := "invalid_header_format"
	payload := []byte(`{"context":{"timestamp":"2024-01-15T10:00:00Z"}}`)

	err := service.VerifyRequestSignature(context.Background(), authHeader, payload)

	assert.Error(t, err)
	assert.True(t, errors.IsDomainError(err))
	domainErr, _ := err.(*errors.DomainError)
	assert.Equal(t, 65002, domainErr.Code)
}

func TestONDCAuthService_VerifyRequestSignature_MissingComma(t *testing.T) {
	service, _ := createTestService(t, config.ONDCConfig{})

	authHeader := `keyId="buyer.com|UKID1|ed25519" signature="base64_signature"`
	payload := []byte(`{"context":{"timestamp":"2024-01-15T10:00:00Z"}}`)

	err := service.VerifyRequestSignature(context.Background(), authHeader, payload)

	assert.Error(t, err)
	assert.True(t, errors.IsDomainError(err))
	domainErr, _ := err.(*errors.DomainError)
	assert.Equal(t, 65002, domainErr.Code)
}

func TestONDCAuthService_VerifyRequestSignature_ExtraFields(t *testing.T) {
	cfg := config.ONDCConfig{
		NetworkRegistryURL: "https://registry.ondc.org",
		TimestampWindow:    300,
		SubscriberID:       "seller.com",
		UkID:               "SELLER_UKID",
	}
	service, mockRegistry := createTestService(t, cfg)

	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)
	publicKeyBase64 := base64.StdEncoding.EncodeToString(publicKey)

	mockRegistry.On("LookupPublicKey", mock.Anything, "buyer.com", "UKID1").
		Return(publicKeyBase64, nil)

	payload := []byte(`{"context":{"timestamp":"2024-01-15T10:00:00Z"}}`)
	hash := blake2b.Sum256(payload)
	signature := ed25519.Sign(privateKey, hash[:])
	signatureBase64 := base64.StdEncoding.EncodeToString(signature)

	authHeader := fmt.Sprintf(`keyId="buyer.com|UKID1|ed25519", signature="%s", extraField="value", anotherField="test"`, signatureBase64)

	err = service.VerifyRequestSignature(context.Background(), authHeader, payload)

	assert.NoError(t, err)
	mockRegistry.AssertExpectations(t)
}

func TestONDCAuthService_VerifyRequestSignature_InvalidBase64Signature(t *testing.T) {
	service, mockRegistry := createTestService(t, config.ONDCConfig{})

	publicKey, _, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)
	publicKeyBase64 := base64.StdEncoding.EncodeToString(publicKey)

	mockRegistry.On("LookupPublicKey", mock.Anything, "buyer.com", "UKID1").
		Return(publicKeyBase64, nil)

	authHeader := `keyId="buyer.com|UKID1|ed25519", signature="invalid-base64!!!"`
	payload := []byte(`{"context":{"timestamp":"2024-01-15T10:00:00Z"}}`)

	err = service.VerifyRequestSignature(context.Background(), authHeader, payload)

	assert.Error(t, err)
	assert.True(t, errors.IsDomainError(err))
	domainErr, _ := err.(*errors.DomainError)
	assert.Equal(t, 65002, domainErr.Code)
	assert.Contains(t, domainErr.Details, "invalid signature format")
	mockRegistry.AssertExpectations(t)
}

func TestONDCAuthService_VerifyRequestSignature_MissingKeyId(t *testing.T) {
	service, _ := createTestService(t, config.ONDCConfig{})

	authHeader := `signature="some_signature"`
	payload := []byte(`{"context":{"timestamp":"2024-01-15T10:00:00Z"}}`)

	err := service.VerifyRequestSignature(context.Background(), authHeader, payload)

	assert.Error(t, err)
	assert.True(t, errors.IsDomainError(err))
	domainErr, _ := err.(*errors.DomainError)
	assert.Equal(t, 65002, domainErr.Code)
}

func TestONDCAuthService_VerifyRequestSignature_MissingSignature(t *testing.T) {
	service, _ := createTestService(t, config.ONDCConfig{})

	authHeader := `keyId="buyer.com|UKID1|ed25519"`
	payload := []byte(`{"context":{"timestamp":"2024-01-15T10:00:00Z"}}`)

	err := service.VerifyRequestSignature(context.Background(), authHeader, payload)

	assert.Error(t, err)
	assert.True(t, errors.IsDomainError(err))
	domainErr, _ := err.(*errors.DomainError)
	assert.Equal(t, 65002, domainErr.Code)
}

func TestONDCAuthService_VerifyRequestSignature_InvalidAlgorithm(t *testing.T) {
	service, _ := createTestService(t, config.ONDCConfig{})

	authHeader := `keyId="buyer.com|UKID1|rsa256", signature="signature"`
	payload := []byte(`{"context":{"timestamp":"2024-01-15T10:00:00Z"}}`)

	err := service.VerifyRequestSignature(context.Background(), authHeader, payload)

	assert.Error(t, err)
	assert.True(t, errors.IsDomainError(err))
	domainErr, _ := err.(*errors.DomainError)
	assert.Equal(t, 65002, domainErr.Code)
	assert.Contains(t, domainErr.Details, "unsupported algorithm")
}

func TestONDCAuthService_VerifyRequestSignature_RegistryError(t *testing.T) {
	service, mockRegistry := createTestService(t, config.ONDCConfig{})

	mockRegistry.On("LookupPublicKey", mock.Anything, "buyer.com", "UKID1").
		Return("", errors.NewDomainError(65011, "registry unavailable", "dependency"))

	authHeader := `keyId="buyer.com|UKID1|ed25519", signature="base64_signature"`
	payload := []byte(`{"context":{"timestamp":"2024-01-15T10:00:00Z"}}`)

	err := service.VerifyRequestSignature(context.Background(), authHeader, payload)

	assert.Error(t, err)
	assert.True(t, errors.IsDomainError(err))
	mockRegistry.AssertExpectations(t)
}

func TestONDCAuthService_VerifyRequestSignature_InvalidSignature(t *testing.T) {
	service, mockRegistry := createTestService(t, config.ONDCConfig{})

	publicKey, _, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)
	publicKeyBase64 := base64.StdEncoding.EncodeToString(publicKey)

	mockRegistry.On("LookupPublicKey", mock.Anything, "buyer.com", "UKID1").
		Return(publicKeyBase64, nil)

	authHeader := `keyId="buyer.com|UKID1|ed25519", signature="invalid_signature"`
	payload := []byte(`{"context":{"timestamp":"2024-01-15T10:00:00Z"}}`)

	err = service.VerifyRequestSignature(context.Background(), authHeader, payload)

	assert.Error(t, err)
	assert.True(t, errors.IsDomainError(err))
	domainErr, _ := err.(*errors.DomainError)
	assert.Equal(t, 65002, domainErr.Code)
	mockRegistry.AssertExpectations(t)
}

func TestONDCAuthService_VerifyRequestSignature_InvalidRegistryKeySize(t *testing.T) {
	service, mockRegistry := createTestService(t, config.ONDCConfig{})

	invalidKeyBase64 := base64.StdEncoding.EncodeToString([]byte("too-short-key"))

	mockRegistry.On("LookupPublicKey", mock.Anything, "buyer.com", "UKID1").
		Return(invalidKeyBase64, nil)

	authHeader := `keyId="buyer.com|UKID1|ed25519", signature="some_signature"`
	payload := []byte(`{"context":{"timestamp":"2024-01-15T10:00:00Z"}}`)

	err := service.VerifyRequestSignature(context.Background(), authHeader, payload)

	assert.Error(t, err)
	assert.True(t, errors.IsDomainError(err))
	domainErr, _ := err.(*errors.DomainError)
	assert.Equal(t, 65002, domainErr.Code)
	assert.Contains(t, domainErr.Details, "invalid registry public key size")
	mockRegistry.AssertExpectations(t)
}

func TestONDCAuthService_VerifyRequestSignature_WithOptionalFields(t *testing.T) {
	cfg := config.ONDCConfig{
		NetworkRegistryURL: "https://registry.ondc.org",
		TimestampWindow:    300,
		SubscriberID:       "seller.com",
		UkID:               "SELLER_UKID",
	}
	service, mockRegistry := createTestService(t, cfg)

	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)
	publicKeyBase64 := base64.StdEncoding.EncodeToString(publicKey)

	mockRegistry.On("LookupPublicKey", mock.Anything, "buyer.com", "UKID1").
		Return(publicKeyBase64, nil)

	payload := []byte(`{"context":{"timestamp":"2024-01-15T10:00:00Z"}}`)
	hash := blake2b.Sum256(payload)
	signature := ed25519.Sign(privateKey, hash[:])
	signatureBase64 := base64.StdEncoding.EncodeToString(signature)

	authHeader := fmt.Sprintf(`keyId="buyer.com|UKID1|ed25519", signature="%s", created="1234567890", expires="1234567999"`, signatureBase64)

	err = service.VerifyRequestSignature(context.Background(), authHeader, payload)

	assert.NoError(t, err)
	mockRegistry.AssertExpectations(t)
}

func TestONDCAuthService_VerifyTimestamp_Valid(t *testing.T) {
	service, _ := createTestService(t, config.ONDCConfig{
		TimestampWindow: 300,
	})

	timestamp := time.Now().UTC().Format(time.RFC3339)
	err := service.VerifyTimestamp(timestamp)

	assert.NoError(t, err)
}

func TestONDCAuthService_VerifyTimestamp_Expired(t *testing.T) {
	service, _ := createTestService(t, config.ONDCConfig{
		TimestampWindow: 300,
	})

	oldTimestamp := time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339)
	err := service.VerifyTimestamp(oldTimestamp)

	assert.Error(t, err)
	assert.True(t, errors.IsDomainError(err))
	domainErr, _ := err.(*errors.DomainError)
	assert.Equal(t, 65003, domainErr.Code)
}

func TestONDCAuthService_VerifyTimestamp_InvalidFormat(t *testing.T) {
	service, _ := createTestService(t, config.ONDCConfig{
		TimestampWindow: 300,
	})

	err := service.VerifyTimestamp("invalid-timestamp")

	assert.Error(t, err)
	assert.True(t, errors.IsDomainError(err))
	domainErr, _ := err.(*errors.DomainError)
	assert.Equal(t, 65003, domainErr.Code)
}

func TestONDCAuthService_SignResponse_Success(t *testing.T) {
	cfg := config.ONDCConfig{
		SubscriberID: "lsp.com",
		UkID:         "UKID1",
	}
	service, _ := createTestService(t, cfg)

	payload := []byte(`{"message":{"order_id":"123"}}`)

	authHeader, err := service.SignResponse(payload)

	assert.NoError(t, err)
	assert.NotEmpty(t, authHeader)
	assert.Contains(t, authHeader, "keyId")
	assert.Contains(t, authHeader, "lsp.com")
	assert.Contains(t, authHeader, "UKID1")
	assert.Contains(t, authHeader, "signature")
}

func TestONDCAuthService_SignResponse_VerifyRoundTrip(t *testing.T) {
	cfg := config.ONDCConfig{
		SubscriberID: "seller.com",
		UkID:         "SELLER_UKID",
	}
	service, mockRegistry := createTestService(t, cfg)

	payload := []byte(`{"message":{"order_id":"123"}}`)

	authHeader, err := service.SignResponse(payload)
	assert.NoError(t, err)

	publicKeyBase64 := base64.StdEncoding.EncodeToString(service.publicKey)
	mockRegistry.On("LookupPublicKey", mock.Anything, "seller.com", "SELLER_UKID").
		Return(publicKeyBase64, nil)

	err = service.VerifyRequestSignature(context.Background(), authHeader, payload)
	assert.NoError(t, err)
	mockRegistry.AssertExpectations(t)
}

func TestONDCAuthService_ParseKeyID_InvalidFormat(t *testing.T) {
	service, _ := createTestService(t, config.ONDCConfig{})

	_, _, _, err := service.parseKeyID("invalid")

	assert.Error(t, err)
}

func TestONDCAuthService_ParseKeyID_Valid(t *testing.T) {
	service, _ := createTestService(t, config.ONDCConfig{})

	subscriberID, ukID, algorithm, err := service.parseKeyID("subscriber|ukid|ed25519")

	assert.NoError(t, err)
	assert.Equal(t, "subscriber", subscriberID)
	assert.Equal(t, "ukid", ukID)
	assert.Equal(t, "ed25519", algorithm)
}
