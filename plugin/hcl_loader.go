// Package plugin provides plugin discovery, loading, and execution for tfbreak.
//
// This file provides utilities for loading raw HCL files from Terraform directories,
// which are then passed to plugins via the Runner interface.
package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// LoadHCLFiles loads all .tf files from the given directory and returns them
// as a map from filename to parsed HCL file.
func LoadHCLFiles(dir string) (map[string]*hcl.File, error) {
	// Convert to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Check if directory exists
	info, err := os.Stat(absDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory does not exist: %s", absDir)
		}
		return nil, fmt.Errorf("failed to access directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", absDir)
	}

	// Read directory entries
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	parser := hclparse.NewParser()
	files := make(map[string]*hcl.File)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".tf") {
			continue
		}

		filePath := filepath.Join(absDir, name)
		file, diags := parser.ParseHCLFile(filePath)
		if diags.HasErrors() {
			// Log warning but continue - don't fail on parse errors for individual files
			continue
		}

		files[filePath] = file
	}

	return files, nil
}

// LoadHCLFilesWithFilter loads HCL files from the given directory,
// optionally applying a filter function to determine which files to include.
func LoadHCLFilesWithFilter(dir string, filter func(filename string) bool) (map[string]*hcl.File, error) {
	// Convert to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Check if directory exists
	info, err := os.Stat(absDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory does not exist: %s", absDir)
		}
		return nil, fmt.Errorf("failed to access directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", absDir)
	}

	// Read directory entries
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	parser := hclparse.NewParser()
	files := make(map[string]*hcl.File)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".tf") {
			continue
		}

		// Apply filter if provided
		if filter != nil && !filter(name) {
			continue
		}

		filePath := filepath.Join(absDir, name)
		file, diags := parser.ParseHCLFile(filePath)
		if diags.HasErrors() {
			// Log warning but continue - don't fail on parse errors for individual files
			continue
		}

		files[filePath] = file
	}

	return files, nil
}
