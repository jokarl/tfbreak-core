package types

import (
	"encoding/json"
	"testing"
)

func TestSeverityString(t *testing.T) {
	tests := []struct {
		severity Severity
		want     string
	}{
		{SeverityError, "ERROR"},
		{SeverityWarning, "WARNING"},
		{SeverityNotice, "NOTICE"},
		{Severity(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.severity.String()
			if got != tt.want {
				t.Errorf("Severity.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSeverityComparison(t *testing.T) {
	tests := []struct {
		name     string
		s        Severity
		other    Severity
		expected bool
	}{
		{"error at least error", SeverityError, SeverityError, true},
		{"error at least warning", SeverityError, SeverityWarning, true},
		{"error at least notice", SeverityError, SeverityNotice, true},
		{"warning at least error", SeverityWarning, SeverityError, false},
		{"warning at least warning", SeverityWarning, SeverityWarning, true},
		{"warning at least notice", SeverityWarning, SeverityNotice, true},
		{"notice at least error", SeverityNotice, SeverityError, false},
		{"notice at least warning", SeverityNotice, SeverityWarning, false},
		{"notice at least notice", SeverityNotice, SeverityNotice, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.AtLeast(tt.other)
			if got != tt.expected {
				t.Errorf("%s.AtLeast(%s) = %v, want %v", tt.s, tt.other, got, tt.expected)
			}
		})
	}
}

func TestParseSeverity(t *testing.T) {
	tests := []struct {
		input   string
		want    Severity
		wantErr bool
	}{
		{"ERROR", SeverityError, false},
		{"error", SeverityError, false},
		{"Error", SeverityError, false},
		{"WARNING", SeverityWarning, false},
		{"warning", SeverityWarning, false},
		{"NOTICE", SeverityNotice, false},
		{"notice", SeverityNotice, false},
		{"invalid", SeverityNotice, true},
		{"", SeverityNotice, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseSeverity(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSeverity(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseSeverity(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSeverityJSON(t *testing.T) {
	type wrapper struct {
		Level Severity `json:"level"`
	}

	t.Run("marshal", func(t *testing.T) {
		w := wrapper{Level: SeverityError}
		data, err := json.Marshal(w)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		want := `{"level":"ERROR"}`
		if string(data) != want {
			t.Errorf("Marshal = %s, want %s", data, want)
		}
	})

	t.Run("unmarshal", func(t *testing.T) {
		input := `{"level":"WARNING"}`
		var w wrapper
		if err := json.Unmarshal([]byte(input), &w); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if w.Level != SeverityWarning {
			t.Errorf("Unmarshal level = %v, want %v", w.Level, SeverityWarning)
		}
	})

	t.Run("unmarshal invalid JSON type", func(t *testing.T) {
		// JSON is valid but type is wrong (number instead of string)
		input := `{"level":123}`
		var w wrapper
		err := json.Unmarshal([]byte(input), &w)
		if err == nil {
			t.Error("expected error for invalid JSON type, got nil")
		}
	})

	t.Run("unmarshal invalid severity value", func(t *testing.T) {
		// JSON string that's not a valid severity
		input := `{"level":"CRITICAL"}`
		var w wrapper
		err := json.Unmarshal([]byte(input), &w)
		if err == nil {
			t.Error("expected error for invalid severity value, got nil")
		}
	})
}
