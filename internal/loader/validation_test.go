package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseValidationBlocks(t *testing.T) {
	// Create a temporary directory with test files
	dir := t.TempDir()

	// Write a test Terraform file with validations
	tfContent := `
variable "environment" {
  type        = string
  description = "The deployment environment"

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be dev, staging, or prod."
  }
}

variable "instance_count" {
  type = number

  validation {
    condition     = var.instance_count > 0
    error_message = "Instance count must be positive."
  }

  validation {
    condition     = var.instance_count <= 100
    error_message = "Instance count must not exceed 100."
  }
}

variable "name" {
  type        = string
  description = "No validations here"
}
`

	err := os.WriteFile(filepath.Join(dir, "variables.tf"), []byte(tfContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Parse validation blocks
	validationMap, err := parseValidationBlocks(dir)
	if err != nil {
		t.Fatalf("parseValidationBlocks failed: %v", err)
	}

	// Check environment variable (1 validation)
	envValidations := validationMap["environment"]
	if len(envValidations) != 1 {
		t.Errorf("expected 1 validation for 'environment', got %d", len(envValidations))
	}
	if len(envValidations) > 0 {
		if envValidations[0].Condition == "" {
			t.Error("expected non-empty condition for environment validation")
		}
		if envValidations[0].ErrorMessage != "Environment must be dev, staging, or prod." {
			t.Errorf("unexpected error_message: %q", envValidations[0].ErrorMessage)
		}
	}

	// Check instance_count variable (2 validations)
	countValidations := validationMap["instance_count"]
	if len(countValidations) != 2 {
		t.Errorf("expected 2 validations for 'instance_count', got %d", len(countValidations))
	}

	// Check name variable (0 validations)
	nameValidations := validationMap["name"]
	if len(nameValidations) != 0 {
		t.Errorf("expected 0 validations for 'name', got %d", len(nameValidations))
	}
}

func TestParseValidationBlocks_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	validationMap, err := parseValidationBlocks(dir)
	if err != nil {
		t.Fatalf("parseValidationBlocks failed on empty dir: %v", err)
	}

	if len(validationMap) != 0 {
		t.Errorf("expected empty map for empty dir, got %d entries", len(validationMap))
	}
}

func TestParseValidationBlocks_NoVariables(t *testing.T) {
	dir := t.TempDir()

	// Write a Terraform file with no variables
	tfContent := `
resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
}
`
	err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(tfContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	validationMap, err := parseValidationBlocks(dir)
	if err != nil {
		t.Fatalf("parseValidationBlocks failed: %v", err)
	}

	if len(validationMap) != 0 {
		t.Errorf("expected empty map for file with no variables, got %d entries", len(validationMap))
	}
}

func TestLoad_WithValidations(t *testing.T) {
	dir := t.TempDir()

	// Write a test Terraform file with validations
	tfContent := `
variable "environment" {
  type        = string
  description = "The deployment environment"

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be dev, staging, or prod."
  }

  validation {
    condition     = var.environment != ""
    error_message = "Environment cannot be empty."
  }
}

variable "name" {
  type = string
}
`
	err := os.WriteFile(filepath.Join(dir, "variables.tf"), []byte(tfContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Load the module
	snapshot, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check environment variable
	envVar, exists := snapshot.Variables["environment"]
	if !exists {
		t.Fatal("expected 'environment' variable in snapshot")
	}
	if envVar.ValidationCount != 2 {
		t.Errorf("expected ValidationCount=2 for 'environment', got %d", envVar.ValidationCount)
	}
	if len(envVar.Validations) != 2 {
		t.Errorf("expected 2 Validations for 'environment', got %d", len(envVar.Validations))
	}

	// Check name variable (no validations)
	nameVar, exists := snapshot.Variables["name"]
	if !exists {
		t.Fatal("expected 'name' variable in snapshot")
	}
	if nameVar.ValidationCount != 0 {
		t.Errorf("expected ValidationCount=0 for 'name', got %d", nameVar.ValidationCount)
	}
	if len(nameVar.Validations) != 0 {
		t.Errorf("expected 0 Validations for 'name', got %d", len(nameVar.Validations))
	}
}
