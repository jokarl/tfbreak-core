package types

import "testing"

func TestNewModuleSnapshot(t *testing.T) {
	path := "/path/to/module"
	snap := NewModuleSnapshot(path)

	if snap.Path != path {
		t.Errorf("Path = %q, want %q", snap.Path, path)
	}
	if snap.Variables == nil {
		t.Error("Variables map is nil")
	}
	if snap.Outputs == nil {
		t.Error("Outputs map is nil")
	}
	if snap.Resources == nil {
		t.Error("Resources map is nil")
	}
	if snap.Modules == nil {
		t.Error("Modules map is nil")
	}
	if snap.MovedBlocks == nil {
		t.Error("MovedBlocks slice is nil")
	}
	if snap.RequiredProviders == nil {
		t.Error("RequiredProviders map is nil")
	}
}

func TestVariableSignature_HasDefault(t *testing.T) {
	tests := []struct {
		name     string
		required bool
		want     bool
	}{
		{"with default", false, true},
		{"without default", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &VariableSignature{Required: tt.required}
			if got := v.HasDefault(); got != tt.want {
				t.Errorf("HasDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsResourceAddress(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{"aws_s3_bucket.main", true},
		{"null_resource.test", true},
		{"resource_type.name", true},
		{"module.vpc", false},
		{"module.nested.resource", false},
		{"", false},
		{"nodot", false},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			if got := IsResourceAddress(tt.addr); got != tt.want {
				t.Errorf("IsResourceAddress(%q) = %v, want %v", tt.addr, got, tt.want)
			}
		})
	}
}

func TestIsModuleAddress(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{"module.vpc", true},
		{"module.nested", true},
		{"module.", true},
		{"aws_s3_bucket.main", false},
		{"", false},
		{"modul", false},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			if got := IsModuleAddress(tt.addr); got != tt.want {
				t.Errorf("IsModuleAddress(%q) = %v, want %v", tt.addr, got, tt.want)
			}
		})
	}
}
