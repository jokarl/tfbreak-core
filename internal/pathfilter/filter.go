// Package pathfilter provides glob-based file filtering using doublestar patterns.
package pathfilter

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// Filter holds the include and exclude patterns for file filtering
type Filter struct {
	include []string
	exclude []string
}

// New creates a new Filter with the given include and exclude patterns
func New(include, exclude []string) *Filter {
	return &Filter{
		include: include,
		exclude: exclude,
	}
}

// FilterFiles returns a list of files in dir that match the include patterns
// and do not match any exclude patterns.
// The returned paths are relative to dir.
func (f *Filter) FilterFiles(dir string) ([]string, error) {
	fsys := os.DirFS(dir)
	seen := make(map[string]bool)
	var result []string

	// Apply include patterns
	for _, pattern := range f.include {
		matches, err := doublestar.Glob(fsys, pattern)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			if !seen[match] {
				seen[match] = true
				result = append(result, match)
			}
		}
	}

	// Filter out excluded patterns
	if len(f.exclude) > 0 {
		filtered := make([]string, 0, len(result))
		for _, path := range result {
			excluded := false
			for _, pattern := range f.exclude {
				match, err := doublestar.Match(pattern, path)
				if err != nil {
					return nil, err
				}
				if match {
					excluded = true
					break
				}
			}
			if !excluded {
				filtered = append(filtered, path)
			}
		}
		result = filtered
	}

	return result, nil
}

// FilterFilesAbs returns absolute paths of filtered files
func (f *Filter) FilterFilesAbs(dir string) ([]string, error) {
	relPaths, err := f.FilterFiles(dir)
	if err != nil {
		return nil, err
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	absPaths := make([]string, len(relPaths))
	for i, relPath := range relPaths {
		absPaths[i] = filepath.Join(absDir, relPath)
	}

	return absPaths, nil
}

// MatchFile checks if a single file path matches the filter criteria
func (f *Filter) MatchFile(path string) (bool, error) {
	// Check if it matches any include pattern
	included := false
	for _, pattern := range f.include {
		match, err := doublestar.Match(pattern, path)
		if err != nil {
			return false, err
		}
		if match {
			included = true
			break
		}
	}

	if !included {
		return false, nil
	}

	// Check if it matches any exclude pattern
	for _, pattern := range f.exclude {
		match, err := doublestar.Match(pattern, path)
		if err != nil {
			return false, err
		}
		if match {
			return false, nil
		}
	}

	return true, nil
}

// DefaultFilter returns a filter with default patterns
func DefaultFilter() *Filter {
	return New(
		[]string{"**/*.tf"},
		[]string{".terraform/**"},
	)
}

// WalkDir walks the directory applying the filter and calling fn for each matching file
func (f *Filter) WalkDir(dir string, fn func(path string, d fs.DirEntry) error) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// Check if entire directory should be excluded
			relPath, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			// Normalize to forward slashes for pattern matching
			relPath = strings.ReplaceAll(relPath, string(filepath.Separator), "/")

			for _, pattern := range f.exclude {
				// Check if this directory matches an exclude pattern
				dirPattern := strings.TrimSuffix(pattern, "/**")
				if relPath == dirPattern || strings.HasPrefix(relPath, dirPattern+"/") {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Get relative path for matching
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		// Normalize to forward slashes for pattern matching
		relPath = strings.ReplaceAll(relPath, string(filepath.Separator), "/")

		match, err := f.MatchFile(relPath)
		if err != nil {
			return err
		}

		if match {
			return fn(path, d)
		}

		return nil
	})
}
