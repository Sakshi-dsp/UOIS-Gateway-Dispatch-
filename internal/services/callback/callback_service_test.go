package callback

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"uois-gateway/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockSigner is a mock implementation of Signer interface
type MockSigner struct {
	mock.Mock
}

func (m *MockSigner) SignRequest(ctx context.Context, method, url string, body []byte, headers map[string]string) (string, error) {
	args := m.Called(ctx, method, url, body, headers)
	return args.String(0), args.Error(1)
}

func TestCallbackService_SendCallback_Success(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.CallbackConfig{
		HTTPTimeoutSeconds: 5,
		MaxConcurrent:      100,
	}

	mockSigner := new(MockSigner)
	mockSigner.On("SignRequest", mock.Anything, http.MethodPost, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return("Signature keyId=\"test-bpp|key1|ed25519\",headers=\"(created) (expires) digest content-type\",signature=\"test-signature\"", nil)

	// Create a test server that returns 200 OK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify ONDC-required headers
		digest := r.Header.Get("Digest")
		assert.NotEmpty(t, digest, "Digest header must be present")
		assert.True(t, strings.HasPrefix(digest, "SHA-256="), "Digest must use SHA-256")

		auth := r.Header.Get("Authorization")
		assert.NotEmpty(t, auth, "Authorization header must be present")
		assert.True(t, strings.HasPrefix(auth, "Signature"), "Authorization must use Signature scheme")
		assert.Contains(t, auth, "keyId=", "Authorization must contain keyId")
		assert.Contains(t, auth, "headers=", "Authorization must contain headers")
		assert.Contains(t, auth, "signature=", "Authorization must contain signature")

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := NewService(cfg, mockSigner, logger)

	payload := map[string]interface{}{
		"message": "test",
	}

	err := service.SendCallback(context.Background(), server.URL, payload)
	assert.NoError(t, err)
	mockSigner.AssertExpectations(t)
}

func TestCallbackService_SendCallback_WithDigest(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.CallbackConfig{
		HTTPTimeoutSeconds: 5,
		MaxConcurrent:      100,
	}

	mockSigner := new(MockSigner)

	var receivedDigest string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedDigest = r.Header.Get("Digest")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mockSigner.On("SignRequest", mock.Anything, http.MethodPost, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return("Signature keyId=\"test-bpp|key1|ed25519\",headers=\"(created) (expires) digest content-type\",signature=\"test-signature\"", nil)

	service := NewService(cfg, mockSigner, logger)

	payload := map[string]interface{}{
		"order": map[string]interface{}{
			"id": "ORDER123",
		},
	}

	err := service.SendCallback(context.Background(), server.URL, payload)
	assert.NoError(t, err)

	// Verify digest matches body
	expectedBody, _ := json.Marshal(payload)
	hash := sha256.Sum256(expectedBody)
	expectedDigest := "SHA-256=" + base64.StdEncoding.EncodeToString(hash[:])
	assert.Equal(t, expectedDigest, receivedDigest, "Digest must match SHA-256 hash of body")
}

func TestCallbackService_SendCallback_ServerError(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.CallbackConfig{
		HTTPTimeoutSeconds: 5,
		MaxConcurrent:      100,
	}

	mockSigner := new(MockSigner)
	mockSigner.On("SignRequest", mock.Anything, http.MethodPost, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return("Signature keyId=\"test-bpp|key1|ed25519\",headers=\"(created) (expires) digest content-type\",signature=\"test-signature\"", nil)

	// Create a test server that returns 500 Internal Server Error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	service := NewService(cfg, mockSigner, logger)

	payload := map[string]interface{}{
		"message": "test",
	}

	err := service.SendCallback(context.Background(), server.URL, payload)
	assert.Error(t, err)
	mockSigner.AssertExpectations(t)
}

func TestCallbackService_SendCallback_SignerError(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.CallbackConfig{
		HTTPTimeoutSeconds: 5,
		MaxConcurrent:      100,
	}

	mockSigner := new(MockSigner)
	mockSigner.On("SignRequest", mock.Anything, http.MethodPost, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return("", assert.AnError)

	service := NewService(cfg, mockSigner, logger)

	payload := map[string]interface{}{
		"message": "test",
	}

	err := service.SendCallback(context.Background(), "https://example.com/callback", payload)
	assert.Error(t, err)
	mockSigner.AssertExpectations(t)
}

func TestCallbackService_SendCallback_Timeout(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.CallbackConfig{
		HTTPTimeoutSeconds: 1,
		MaxConcurrent:      100,
	}

	mockSigner := new(MockSigner)
	mockSigner.On("SignRequest", mock.Anything, http.MethodPost, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return("Signature keyId=\"test-bpp|key1|ed25519\",headers=\"(created) (expires) digest content-type\",signature=\"test-signature\"", nil)

	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := NewService(cfg, mockSigner, logger)

	payload := map[string]interface{}{
		"message": "test",
	}

	err := service.SendCallback(context.Background(), server.URL, payload)
	assert.Error(t, err)
	mockSigner.AssertExpectations(t)
}

func TestCallbackService_SendCallback_InvalidURL(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.CallbackConfig{
		HTTPTimeoutSeconds: 5,
		MaxConcurrent:      100,
	}

	mockSigner := new(MockSigner)
	mockSigner.On("SignRequest", mock.Anything, http.MethodPost, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return("Signature keyId=\"test-bpp|key1|ed25519\",headers=\"(created) (expires) digest content-type\",signature=\"test-signature\"", nil)

	service := NewService(cfg, mockSigner, logger)

	payload := map[string]interface{}{
		"message": "test",
	}

	err := service.SendCallback(context.Background(), "invalid-url", payload)
	assert.Error(t, err)
	mockSigner.AssertExpectations(t)
}

func TestCallbackService_SendCallback_SerializationFailure(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.CallbackConfig{
		HTTPTimeoutSeconds: 5,
		MaxConcurrent:      100,
	}

	service := NewService(cfg, nil, logger)

	// Create a payload that cannot be serialized (circular reference)
	type Circular struct {
		Self *Circular
	}
	circular := &Circular{}
	circular.Self = circular

	err := service.SendCallback(context.Background(), "https://example.com/callback", circular)
	assert.Error(t, err)
}

func TestCallbackService_SendCallback_ValidJSON(t *testing.T) {
	logger := zap.NewNop()
	cfg := config.CallbackConfig{
		HTTPTimeoutSeconds: 5,
		MaxConcurrent:      100,
	}

	mockSigner := new(MockSigner)
	mockSigner.On("SignRequest", mock.Anything, http.MethodPost, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return("Signature keyId=\"test-bpp|key1|ed25519\",headers=\"(created) (expires) digest content-type\",signature=\"test-signature\"", nil)

	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&receivedPayload)
		assert.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := NewService(cfg, mockSigner, logger)

	payload := map[string]interface{}{
		"order": map[string]interface{}{
			"id":    "ORDER123",
			"state": "CONFIRMED",
		},
	}

	err := service.SendCallback(context.Background(), server.URL, payload)
	assert.NoError(t, err)
	assert.Equal(t, "ORDER123", receivedPayload["order"].(map[string]interface{})["id"])
	mockSigner.AssertExpectations(t)
}
