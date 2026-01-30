package plugin

import "strings"

// builtinSigningKeys maps source organization prefixes to their PGP public keys.
// This allows official plugins to be verified without explicit signing_key configuration.
//
// Keys should be added for organizations that publish official tfbreak plugins
// and have established trust relationships.
var builtinSigningKeys = map[string]string{
	// Example: official tfbreak plugins from jokarl organization
	// "github.com/jokarl": `-----BEGIN PGP PUBLIC KEY BLOCK-----
	// ...key content...
	// -----END PGP PUBLIC KEY BLOCK-----`,
}

// GetBuiltinSigningKey returns the built-in signing key for a source URL.
// It checks if any registered organization prefix matches the source.
// Returns empty string if no built-in key exists.
func GetBuiltinSigningKey(source string) string {
	// Check exact matches first
	if key, ok := builtinSigningKeys[source]; ok {
		return key
	}

	// Check organization prefixes (e.g., "github.com/jokarl" matches "github.com/jokarl/tfbreak-ruleset-azurerm")
	for prefix, key := range builtinSigningKeys {
		if strings.HasPrefix(source, prefix+"/") || source == prefix {
			return key
		}
	}

	return ""
}

// HasBuiltinSigningKey checks if a source has a built-in signing key.
func HasBuiltinSigningKey(source string) bool {
	return GetBuiltinSigningKey(source) != ""
}

// RegisterBuiltinSigningKey adds or updates a built-in signing key.
// This is primarily useful for testing.
func RegisterBuiltinSigningKey(orgPrefix, armoredKey string) {
	builtinSigningKeys[orgPrefix] = armoredKey
}

// UnregisterBuiltinSigningKey removes a built-in signing key.
// This is primarily useful for testing.
func UnregisterBuiltinSigningKey(orgPrefix string) {
	delete(builtinSigningKeys, orgPrefix)
}
