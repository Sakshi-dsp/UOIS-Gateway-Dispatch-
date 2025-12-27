package callback

import (
	"context"
	"time"
)

// Signer provides HTTP signature generation for ONDC callbacks
// ONDC requires all Seller NP callbacks to be HTTP-signed with:
// - Authorization header containing signature
// - keyId (BPP ID + key ID)
// - (created) timestamp
// - (expires) timestamp
// - digest (SHA-256 of body)
// - content-type
type Signer interface {
	// SignRequest generates an HTTP signature for the given request
	// Returns the Authorization header value and any error
	SignRequest(ctx context.Context, method, url string, body []byte, headers map[string]string) (string, error)
}

// SignatureParams contains parameters for signature generation
type SignatureParams struct {
	KeyID     string    // Format: "bpp_id|key_id|ed25519"
	Created   time.Time // Signature creation time
	Expires   time.Time // Signature expiration time
	Digest    string    // SHA-256 digest of body (base64 encoded)
	Headers   []string  // Headers to include in signature: ["(created)", "(expires)", "digest", "content-type"]
	Signature string    // Base64 encoded signature
}
