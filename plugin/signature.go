package plugin

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
)

// SignatureVerifier verifies PGP signatures against a public key.
type SignatureVerifier struct {
	keyring openpgp.EntityList
}

// NewSignatureVerifier creates a new verifier from an ASCII-armored PGP public key.
func NewSignatureVerifier(armoredKey string) (*SignatureVerifier, error) {
	if armoredKey == "" {
		return nil, fmt.Errorf("signing key is empty")
	}

	// Parse ASCII-armored key
	block, err := armor.Decode(strings.NewReader(armoredKey))
	if err != nil {
		return nil, fmt.Errorf("failed to decode armored key: %w", err)
	}

	if block.Type != openpgp.PublicKeyType {
		return nil, fmt.Errorf("invalid key type: expected %s, got %s", openpgp.PublicKeyType, block.Type)
	}

	keyring, err := openpgp.ReadKeyRing(block.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}

	if len(keyring) == 0 {
		return nil, fmt.Errorf("no keys found in keyring")
	}

	return &SignatureVerifier{keyring: keyring}, nil
}

// Verify checks that the signature is valid for the given data.
// The signature should be an ASCII-armored detached signature.
func (v *SignatureVerifier) Verify(data io.Reader, armoredSignature io.Reader) error {
	// Read data into buffer (needed for verification)
	dataBytes, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	// Decode armored signature
	block, err := armor.Decode(armoredSignature)
	if err != nil {
		return fmt.Errorf("failed to decode armored signature: %w", err)
	}

	// Verify detached signature
	_, err = openpgp.CheckDetachedSignature(v.keyring, bytes.NewReader(dataBytes), block.Body, nil)
	if err != nil {
		return &SignatureVerificationError{Cause: err}
	}

	return nil
}

// VerifyBytes is a convenience method that verifies signature against byte slices.
func (v *SignatureVerifier) VerifyBytes(data, signature []byte) error {
	return v.Verify(bytes.NewReader(data), bytes.NewReader(signature))
}

// SignatureVerificationError indicates a signature verification failure.
type SignatureVerificationError struct {
	Cause error
}

func (e *SignatureVerificationError) Error() string {
	return fmt.Sprintf("signature verification failed: %v", e.Cause)
}

func (e *SignatureVerificationError) Unwrap() error {
	return e.Cause
}

// GetSigningKey returns the signing key for a plugin, checking:
// 1. Explicit signing_key in config (highest priority)
// 2. Built-in signing key for the source organization
// Returns empty string if no key is configured.
func GetSigningKey(configKey, source string) string {
	// Explicit key takes priority
	if configKey != "" {
		return configKey
	}

	// Check for built-in key
	return GetBuiltinSigningKey(source)
}
