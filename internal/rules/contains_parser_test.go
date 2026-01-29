package rules

import (
	"reflect"
	"testing"
)

func TestParseContainsPattern(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		wantValues []string
		wantVar    string
		wantNil    bool
	}{
		{
			name:       "simple contains with double quotes",
			condition:  `contains(["dev", "staging", "prod"], var.environment)`,
			wantValues: []string{"dev", "staging", "prod"},
			wantVar:    "environment",
		},
		{
			name:       "simple contains with single quotes",
			condition:  `contains(['a', 'b', 'c'], var.tier)`,
			wantValues: []string{"a", "b", "c"},
			wantVar:    "tier",
		},
		{
			name:       "contains with extra whitespace",
			condition:  `contains(  [ "one" ,  "two"  ]  ,  var.region  )`,
			wantValues: []string{"one", "two"},
			wantVar:    "region",
		},
		{
			name:       "contains with newlines",
			condition:  "contains([\n  \"dev\",\n  \"prod\"\n], var.env)",
			wantValues: []string{"dev", "prod"},
			wantVar:    "env",
		},
		{
			name:       "empty list",
			condition:  `contains([], var.x)`,
			wantValues: []string{},
			wantVar:    "x",
		},
		{
			name:       "single value",
			condition:  `contains(["only"], var.choice)`,
			wantValues: []string{"only"},
			wantVar:    "choice",
		},
		{
			name:      "dynamic list - should not match",
			condition: `contains(var.allowed_envs, var.environment)`,
			wantNil:   true,
		},
		{
			name:      "function call in list - should not match",
			condition: `contains(concat(["a"], ["b"]), var.x)`,
			wantNil:   true,
		},
		{
			name:      "not a contains pattern",
			condition: `var.environment != ""`,
			wantNil:   true,
		},
		{
			name:      "length check - not contains",
			condition: `length(var.name) > 0`,
			wantNil:   true,
		},
		{
			name:      "regex check - not contains",
			condition: `can(regex("^[a-z]+$", var.name))`,
			wantNil:   true,
		},
		{
			name:      "contains with local reference - should not match",
			condition: `contains(local.valid_envs, var.environment)`,
			wantNil:   true,
		},
		{
			name:       "values with spaces",
			condition:  `contains(["hello world", "foo bar"], var.greeting)`,
			wantValues: []string{"hello world", "foo bar"},
			wantVar:    "greeting",
		},
		{
			name:       "values with escaped quotes",
			condition:  `contains(["say \"hello\"", "normal"], var.msg)`,
			wantValues: []string{`say "hello"`, "normal"},
			wantVar:    "msg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseContainsPattern(tt.condition)

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil result, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatalf("expected non-nil result")
			}

			if !reflect.DeepEqual(result.Values, tt.wantValues) {
				t.Errorf("values mismatch: got %v, want %v", result.Values, tt.wantValues)
			}

			if result.VarName != tt.wantVar {
				t.Errorf("var name mismatch: got %q, want %q", result.VarName, tt.wantVar)
			}
		})
	}
}

func TestFindRemovedValues(t *testing.T) {
	tests := []struct {
		name        string
		oldValues   []string
		newValues   []string
		wantRemoved []string
	}{
		{
			name:        "one value removed",
			oldValues:   []string{"a", "b", "c"},
			newValues:   []string{"a", "b"},
			wantRemoved: []string{"c"},
		},
		{
			name:        "multiple values removed",
			oldValues:   []string{"a", "b", "c", "d"},
			newValues:   []string{"b"},
			wantRemoved: []string{"a", "c", "d"},
		},
		{
			name:        "all values removed",
			oldValues:   []string{"a", "b"},
			newValues:   []string{},
			wantRemoved: []string{"a", "b"},
		},
		{
			name:        "no values removed (same)",
			oldValues:   []string{"a", "b"},
			newValues:   []string{"a", "b"},
			wantRemoved: nil,
		},
		{
			name:        "no values removed (added)",
			oldValues:   []string{"a", "b"},
			newValues:   []string{"a", "b", "c"},
			wantRemoved: nil,
		},
		{
			name:        "complete replacement",
			oldValues:   []string{"a", "b"},
			newValues:   []string{"c", "d"},
			wantRemoved: []string{"a", "b"},
		},
		{
			name:        "empty old list",
			oldValues:   []string{},
			newValues:   []string{"a"},
			wantRemoved: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldPattern := &ContainsPattern{Values: tt.oldValues, VarName: "x"}
			newPattern := &ContainsPattern{Values: tt.newValues, VarName: "x"}

			removed := FindRemovedValues(oldPattern, newPattern)

			if !reflect.DeepEqual(removed, tt.wantRemoved) {
				t.Errorf("got %v, want %v", removed, tt.wantRemoved)
			}
		})
	}
}

func TestFindRemovedValues_NilPatterns(t *testing.T) {
	pattern := &ContainsPattern{Values: []string{"a"}, VarName: "x"}

	if removed := FindRemovedValues(nil, pattern); removed != nil {
		t.Errorf("expected nil for nil old pattern, got %v", removed)
	}

	if removed := FindRemovedValues(pattern, nil); removed != nil {
		t.Errorf("expected nil for nil new pattern, got %v", removed)
	}

	if removed := FindRemovedValues(nil, nil); removed != nil {
		t.Errorf("expected nil for both nil patterns, got %v", removed)
	}
}
