package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestVersionCmd_Exists(t *testing.T) {
	// Verify the version command is registered
	if versionCmd == nil {
		t.Fatal("versionCmd is nil")
	}

	if versionCmd.Use != "version" {
		t.Errorf("versionCmd.Use = %q, want %q", versionCmd.Use, "version")
	}
}

func TestVersionCmd_OutputsVersion(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set version info
	SetVersionInfo("1.2.3", "abc123", "2024-01-01")

	// Run the version command
	versionCmd.Run(versionCmd, []string{})

	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output contains version
	if !strings.Contains(output, "1.2.3") {
		t.Errorf("version output doesn't contain version: %q", output)
	}

	// Verify output contains commit
	if !strings.Contains(output, "abc123") {
		t.Errorf("version output doesn't contain commit: %q", output)
	}

	// Verify output contains date
	if !strings.Contains(output, "2024-01-01") {
		t.Errorf("version output doesn't contain date: %q", output)
	}
}

func TestVersionCmd_SkipsEmptyCommit(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set version info with empty commit
	SetVersionInfo("1.0.0", "", "2024-01-01")

	// Run the version command
	versionCmd.Run(versionCmd, []string{})

	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output doesn't contain "commit:"
	if strings.Contains(output, "commit:") {
		t.Errorf("version output contains commit when empty: %q", output)
	}
}

func TestVersionCmd_SkipsNoneCommit(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set version info with "none" commit
	SetVersionInfo("1.0.0", "none", "unknown")

	// Run the version command
	versionCmd.Run(versionCmd, []string{})

	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output doesn't contain "commit:" or "built:"
	if strings.Contains(output, "commit:") {
		t.Errorf("version output contains commit when 'none': %q", output)
	}
	if strings.Contains(output, "built:") {
		t.Errorf("version output contains built when 'unknown': %q", output)
	}
}

func TestSetVersionInfo(t *testing.T) {
	// Test that SetVersionInfo sets the variables correctly
	SetVersionInfo("v2.0.0", "def456", "2025-06-15")

	if versionStr != "v2.0.0" {
		t.Errorf("versionStr = %q, want %q", versionStr, "v2.0.0")
	}
	if commitStr != "def456" {
		t.Errorf("commitStr = %q, want %q", commitStr, "def456")
	}
	if dateStr != "2025-06-15" {
		t.Errorf("dateStr = %q, want %q", dateStr, "2025-06-15")
	}
}
