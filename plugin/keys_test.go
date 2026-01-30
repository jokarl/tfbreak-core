package plugin

import "testing"

func TestGetBuiltinSigningKey_ExactMatch(t *testing.T) {
	// Register a test key
	RegisterBuiltinSigningKey("github.com/testorg", "test-key-exact")
	defer UnregisterBuiltinSigningKey("github.com/testorg")

	key := GetBuiltinSigningKey("github.com/testorg")
	if key != "test-key-exact" {
		t.Errorf("expected 'test-key-exact', got %q", key)
	}
}

func TestGetBuiltinSigningKey_PrefixMatch(t *testing.T) {
	// Register a test key for an organization
	RegisterBuiltinSigningKey("github.com/testorg", "test-key-prefix")
	defer UnregisterBuiltinSigningKey("github.com/testorg")

	// Should match repos under that organization
	key := GetBuiltinSigningKey("github.com/testorg/tfbreak-ruleset-test")
	if key != "test-key-prefix" {
		t.Errorf("expected 'test-key-prefix', got %q", key)
	}
}

func TestGetBuiltinSigningKey_NoMatch(t *testing.T) {
	key := GetBuiltinSigningKey("github.com/unknownorg/repo")
	if key != "" {
		t.Errorf("expected empty string, got %q", key)
	}
}

func TestHasBuiltinSigningKey(t *testing.T) {
	// Register a test key
	RegisterBuiltinSigningKey("github.com/haskey", "has-key")
	defer UnregisterBuiltinSigningKey("github.com/haskey")

	if !HasBuiltinSigningKey("github.com/haskey/repo") {
		t.Error("expected HasBuiltinSigningKey to return true")
	}

	if HasBuiltinSigningKey("github.com/nokey/repo") {
		t.Error("expected HasBuiltinSigningKey to return false")
	}
}

func TestRegisterUnregisterBuiltinSigningKey(t *testing.T) {
	// Should not exist initially
	if HasBuiltinSigningKey("github.com/temporg") {
		t.Error("key should not exist before registration")
	}

	// Register
	RegisterBuiltinSigningKey("github.com/temporg", "temp-key")

	// Should exist now
	if !HasBuiltinSigningKey("github.com/temporg") {
		t.Error("key should exist after registration")
	}

	// Unregister
	UnregisterBuiltinSigningKey("github.com/temporg")

	// Should not exist anymore
	if HasBuiltinSigningKey("github.com/temporg") {
		t.Error("key should not exist after unregistration")
	}
}
