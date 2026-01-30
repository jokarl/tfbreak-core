package git

import (
	"fmt"
	"os"
)

// Clone represents a shallow clone that will be cleaned up.
type Clone struct {
	// Path is the filesystem path to the clone.
	Path string

	// URL is the repository URL that was cloned.
	URL string

	// Ref is the ref that was checked out.
	Ref string

	// SHA is the resolved commit SHA.
	SHA string
}

// ShallowClone creates a shallow clone of a remote repository at a specific ref.
// The clone is created in a temporary directory and should be cleaned up
// by calling Remove() when done.
//
// This uses --depth 1 --single-branch for efficiency, downloading only the
// necessary data for the specified ref.
func ShallowClone(url, ref string) (*Clone, error) {
	// Pre-flight: validate ref exists remotely (fast failure without cloning)
	sha, fullRef, err := ResolveRemoteRef(url, ref)
	if err != nil {
		return nil, err
	}

	// Create temp directory for clone
	tmpDir, err := os.MkdirTemp("", "tfbreak-clone-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Shallow clone with minimal data
	// --depth 1: only fetch the single commit
	// --branch: specify the ref to clone
	// --single-branch: don't fetch other branches
	_, err = Run([]string{
		"clone",
		"--depth", "1",
		"--branch", ref,
		"--single-branch",
		url,
		tmpDir,
	}, nil)
	if err != nil {
		os.RemoveAll(tmpDir) // Clean up temp dir on failure
		return nil, fmt.Errorf("failed to clone %s at %s: %w", url, ref, err)
	}

	// Use the SHA from ls-remote since shallow clone might not have full ref info
	_ = fullRef // fullRef available if needed

	return &Clone{
		Path: tmpDir,
		URL:  url,
		Ref:  ref,
		SHA:  sha,
	}, nil
}

// Remove cleans up the clone by removing the directory.
func (c *Clone) Remove() error {
	if c.Path == "" {
		return nil
	}

	err := os.RemoveAll(c.Path)

	// Clear the path to prevent double-cleanup
	c.Path = ""

	return err
}

// CloneForComparison creates two shallow clones for comparing two refs.
// Both clones are returned, and the caller is responsible for calling Remove() on both.
// If creating either clone fails, any created clone is cleaned up before returning.
func CloneForComparison(url, baseRef, headRef string) (baseClone, headClone *Clone, err error) {
	// Clone base ref first
	baseClone, err = ShallowClone(url, baseRef)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to clone base ref %q: %w", baseRef, err)
	}

	// Clone head ref
	headClone, err = ShallowClone(url, headRef)
	if err != nil {
		// Clean up base clone on failure
		baseClone.Remove()
		return nil, nil, fmt.Errorf("failed to clone head ref %q: %w", headRef, err)
	}

	return baseClone, headClone, nil
}
