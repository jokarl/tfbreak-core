package rules

import "sync"

// Registry holds all registered rules
type Registry struct {
	mu    sync.RWMutex
	rules map[string]Rule
	order []string // preserve registration order
}

// NewRegistry creates a new empty Registry
func NewRegistry() *Registry {
	return &Registry{
		rules: make(map[string]Rule),
		order: make([]string, 0),
	}
}

// Register adds a rule to the registry
func (r *Registry) Register(rule Rule) {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := rule.ID()
	if _, exists := r.rules[id]; !exists {
		r.order = append(r.order, id)
	}
	r.rules[id] = rule
}

// Get returns a rule by ID
func (r *Registry) Get(id string) (Rule, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rule, ok := r.rules[id]
	return rule, ok
}

// All returns all registered rules in registration order
func (r *Registry) All() []Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Rule, 0, len(r.order))
	for _, id := range r.order {
		result = append(result, r.rules[id])
	}
	return result
}

// IDs returns all rule IDs in registration order
func (r *Registry) IDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, len(r.order))
	copy(result, r.order)
	return result
}

// DefaultRegistry is the global rule registry
var DefaultRegistry = NewRegistry()

// Register adds a rule to the default registry
func Register(rule Rule) {
	DefaultRegistry.Register(rule)
}
