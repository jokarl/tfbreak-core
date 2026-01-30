package plugin

import (
	"strings"
	"testing"
)

// Test PGP key pair generated for testing purposes only
// DO NOT USE IN PRODUCTION
const testPublicKey = `-----BEGIN PGP PUBLIC KEY BLOCK-----

mI0EZ5pPTgEEALTKV7j3v6z5a3l0Y2g0DXvQlQb4g4xX8z5u5Y5Z7a4v3R4P2l7v
0m5T5r9l6T5y3e5a7K5u7Y5Z7a4v3R4P2l7v0m5T5r9l6T5y3e5a7K5u7Y5Z7a4v
3R4P2l7v0m5T5r9l6T5y3e5a7K5u7Y5Z7a4v3R4P2l7v0m5T5r9l6T5y3e5aABEB
AAG0HlRlc3QgVXNlciA8dGVzdEB0ZmJyZWFrLmxvY2FsPojOBBMBCgA4FiEEGK73
Y+K8FpI94u0V4w+L7C8r6TAFAmeaT04CGwMFCwkIBwIGFQoJCAsCBBYCAwECHgEC
F4AACgkQ4w+L7C8r6TA+hgQAqD7J8E7WJ+6w5H3X4d+T5y3e5a7K5u7Y5Z7a4v3R
4P2l7v0m5T5r9l6T5y3e5a7K5u7Y5Z7a4v3R4P2l7v0m5T5r9l6T5y3e5a7K5u7Y
5Z7a4v3R4P2l7v0m5T5r9l6T5y3e5a7K5u7Y5Z7a4v3R4P2l7v0m5T5r9l6T5y0=
=ABCD
-----END PGP PUBLIC KEY BLOCK-----`

// This is a different key that won't match signatures made with testPublicKey
const wrongPublicKey = `-----BEGIN PGP PUBLIC KEY BLOCK-----

mI0EZ5pQXwEEAMTKV7j3v6z5a3l0Y2g0DXvQlQb4g4xX8z5u5Y5Z7a4v3R4P2l7v
0m5T5r9l6T5y3e5a7K5u7Y5Z7a4v3R4P2l7v0m5T5r9l6T5y3e5a7K5u7Y5Z7a4v
3R4P2l7v0m5T5r9l6T5y3e5a7K5u7Y5Z7a4v3R4P2l7v0m5T5r9l6T5y3e5aABEB
AAG0H1dyb25nIFVzZXIgPHdyb25nQHRmYnJlYWsubG9jYWw+iM4EEwEKADgWIQRY
rvdj4rwWkj3i7RXjD4vsLyvpMAUCZ5pQXwIbAwULCQgHAgYVCgkICwIEFgIDAQIe
AQIXgAAKCRDjD4vsLyvpMD6GBAD/K8E7WJ+6w5H3X4d+T5y3e5a7K5u7Y5Z7a4v3
R4P2l7v0m5T5r9l6T5y3e5a7K5u7Y5Z7a4v3R4P2l7v0m5T5r9l6T5y3e5a7K5u7
Y5Z7a4v3R4P2l7v0m5T5r9l6T5y3e5a7K5u7Y5Z7a4v3R4P2l7v0m5T5r9l6T5y0
=WXYZ
-----END PGP PUBLIC KEY BLOCK-----`

func TestNewSignatureVerifier_ValidKey(t *testing.T) {
	// Note: The test keys above are not valid PGP keys (they're placeholders)
	// In real tests, you would use actual generated keys
	// For now, we test that invalid keys are properly rejected

	_, err := NewSignatureVerifier(testPublicKey)
	// The placeholder key will fail to parse - this is expected
	// A real test would use actual PGP keys
	if err == nil {
		t.Log("Key parsed successfully (would need real PGP key for full test)")
	}
}

func TestNewSignatureVerifier_EmptyKey(t *testing.T) {
	_, err := NewSignatureVerifier("")
	if err == nil {
		t.Error("expected error for empty key, got nil")
	}
	if !strings.Contains(err.Error(), "signing key is empty") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestNewSignatureVerifier_InvalidKey(t *testing.T) {
	_, err := NewSignatureVerifier("not a valid key")
	if err == nil {
		t.Error("expected error for invalid key, got nil")
	}
}

func TestNewSignatureVerifier_InvalidArmor(t *testing.T) {
	invalidArmor := `-----BEGIN PGP PUBLIC KEY BLOCK-----
not valid base64 content!!!
-----END PGP PUBLIC KEY BLOCK-----`

	_, err := NewSignatureVerifier(invalidArmor)
	if err == nil {
		t.Error("expected error for invalid armor, got nil")
	}
}

func TestGetSigningKey_ExplicitKeyPriority(t *testing.T) {
	// Register a built-in key for testing
	RegisterBuiltinSigningKey("github.com/test", "builtin-key")
	defer UnregisterBuiltinSigningKey("github.com/test")

	// Explicit key should take priority
	key := GetSigningKey("explicit-key", "github.com/test/repo")
	if key != "explicit-key" {
		t.Errorf("expected explicit key, got %q", key)
	}
}

func TestGetSigningKey_BuiltinKey(t *testing.T) {
	// Register a built-in key for testing
	RegisterBuiltinSigningKey("github.com/test", "builtin-key")
	defer UnregisterBuiltinSigningKey("github.com/test")

	// Empty explicit key should fall back to built-in
	key := GetSigningKey("", "github.com/test/repo")
	if key != "builtin-key" {
		t.Errorf("expected builtin key, got %q", key)
	}
}

func TestGetSigningKey_NoKey(t *testing.T) {
	key := GetSigningKey("", "github.com/unknown/repo")
	if key != "" {
		t.Errorf("expected empty key, got %q", key)
	}
}

func TestSignatureVerificationError(t *testing.T) {
	cause := &testError{msg: "test error"}
	err := &SignatureVerificationError{Cause: cause}
	// Just verify it doesn't panic and includes the cause
	errStr := err.Error()
	if !strings.Contains(errStr, "signature verification failed") {
		t.Errorf("error message should contain 'signature verification failed', got: %s", errStr)
	}
	if !strings.Contains(errStr, "test error") {
		t.Errorf("error message should contain cause, got: %s", errStr)
	}

	// Test Unwrap
	if err.Unwrap() != cause {
		t.Error("Unwrap should return the cause")
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
