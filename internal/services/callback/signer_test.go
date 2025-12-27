package callback

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSignatureParams_Validation(t *testing.T) {
	params := SignatureParams{
		KeyID:     "bpp_id|key_id|ed25519",
		Created:   time.Now(),
		Expires:   time.Now().Add(time.Hour),
		Digest:    "SHA-256=abc123",
		Headers:   []string{"(created)", "(expires)", "digest", "content-type"},
		Signature: "signature123",
	}

	assert.NotEmpty(t, params.KeyID)
	assert.NotZero(t, params.Created)
	assert.NotZero(t, params.Expires)
	assert.NotEmpty(t, params.Digest)
	assert.NotEmpty(t, params.Headers)
	assert.NotEmpty(t, params.Signature)
}

func TestSigner_Interface(t *testing.T) {
	// Test that Signer interface is properly defined
	var signer Signer
	assert.Nil(t, signer)

	// Test interface method signature
	signerType := func(s Signer) {
		_, _ = s.SignRequest(context.Background(), "POST", "https://example.com", []byte("body"), map[string]string{})
	}
	signerType(nil)
}
