package annotation

// RegistryResolver resolves rule names to IDs using a name-to-ID map
type RegistryResolver struct {
	nameToID map[string]string
}

// NewRegistryResolver creates a new RegistryResolver from a name-to-ID map
func NewRegistryResolver(nameToID map[string]string) *RegistryResolver {
	return &RegistryResolver{nameToID: nameToID}
}

// ResolveRuleID resolves a rule name to a canonical rule ID
// Only rule names are accepted - legacy rule codes (BC001, etc.) are not supported
func (r *RegistryResolver) ResolveRuleID(name string) (string, bool) {
	if id, ok := r.nameToID[name]; ok {
		return id, true
	}
	// Not found
	return "", false
}
