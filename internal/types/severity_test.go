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
		{SeverityBreaking, "BREAKING"},
		{SeverityRisky, "RISKY"},
		{SeverityInfo, "INFO"},
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
		{"breaking at least breaking", SeverityBreaking, SeverityBreaking, true},
		{"breaking at least risky", SeverityBreaking, SeverityRisky, true},
		{"breaking at least info", SeverityBreaking, SeverityInfo, true},
		{"risky at least breaking", SeverityRisky, SeverityBreaking, false},
		{"risky at least risky", SeverityRisky, SeverityRisky, true},
		{"risky at least info", SeverityRisky, SeverityInfo, true},
		{"info at least breaking", SeverityInfo, SeverityBreaking, false},
		{"info at least risky", SeverityInfo, SeverityRisky, false},
		{"info at least info", SeverityInfo, SeverityInfo, true},
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
		{"BREAKING", SeverityBreaking, false},
		{"breaking", SeverityBreaking, false},
		{"Breaking", SeverityBreaking, false},
		{"RISKY", SeverityRisky, false},
		{"risky", SeverityRisky, false},
		{"INFO", SeverityInfo, false},
		{"info", SeverityInfo, false},
		{"invalid", SeverityInfo, true},
		{"", SeverityInfo, true},
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
		w := wrapper{Level: SeverityBreaking}
		data, err := json.Marshal(w)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		want := `{"level":"BREAKING"}`
		if string(data) != want {
			t.Errorf("Marshal = %s, want %s", data, want)
		}
	})

	t.Run("unmarshal", func(t *testing.T) {
		input := `{"level":"RISKY"}`
		var w wrapper
		if err := json.Unmarshal([]byte(input), &w); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if w.Level != SeverityRisky {
			t.Errorf("Unmarshal level = %v, want %v", w.Level, SeverityRisky)
		}
	})
}
