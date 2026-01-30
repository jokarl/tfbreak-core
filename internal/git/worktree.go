package git

import (
	"fmt"
	"os"
	"path/filepath"
)

// Worktree represents a git worktree that will be cleaned up.
type Worktree struct {
	// Path is the filesystem path to the worktree.
	Path string

	// RepoDir is the path to the main repository.
	RepoDir string

	// Ref is the ref that was checked out.
	Ref string

	// SHA is the resolved commit SHA.
	SHA string
}

// CreateWorktree creates a detached worktree at the specified ref.
// The worktree is created in a temporary directory and should be cleaned up
// by calling Remove() when done.
func CreateWorktree(repoDir, ref string) (*Worktree, error) {
	// Pre-flight: validate we're in a git repository
	repoRoot, err := FindGitRoot(repoDir)
	if err != nil {
		return nil, err
	}

	// Pre-flight: validate ref exists
	sha, err := ResolveRef(repoRoot, ref)
	if err != nil {
		return nil, err
	}

	// Create temp directory for worktree
	tmpDir, err := os.MkdirTemp("", "tfbreak-worktree-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Create detached worktree
	// Using --detach prevents accidental commits in the worktree
	_, err = Run([]string{"worktree", "add", "--detach", tmpDir, ref}, &RunOptions{Dir: repoRoot})
	if err != nil {
		os.RemoveAll(tmpDir) // Clean up temp dir on failure
		return nil, fmt.Errorf("failed to create worktree at %q: %w", ref, err)
	}

	return &Worktree{
		Path:    tmpDir,
		RepoDir: repoRoot,
		Ref:     ref,
		SHA:     sha,
	}, nil
}

// Remove cleans up the worktree.
// It removes the worktree from git's tracking and deletes the directory.
// Errors during removal are logged but not returned to ensure cleanup completes.
func (w *Worktree) Remove() error {
	if w.Path == "" {
		return nil
	}

	// First try to remove via git worktree remove
	// Use --force in case there are untracked files
	_, err := Run([]string{"worktree", "remove", "--force", w.Path}, &RunOptions{Dir: w.RepoDir})
	if err != nil {
		// If git worktree remove fails, manually clean up
		// This can happen if the worktree was already partially deleted
		_ = os.RemoveAll(w.Path)

		// Also try to prune any stale worktree entries
		_, _ = Run([]string{"worktree", "prune"}, &RunOptions{Dir: w.RepoDir})
	}

	// Clear the path to prevent double-cleanup
	w.Path = ""

	return nil
}

// WorktreeList returns a list of all worktrees for the repository.
func WorktreeList(repoDir string) ([]string, error) {
	out, err := Run([]string{"worktree", "list", "--porcelain"}, &RunOptions{Dir: repoDir})
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var worktrees []string
	lines := splitLines(out)
	for _, line := range lines {
		if len(line) > 9 && line[:9] == "worktree " {
			worktrees = append(worktrees, line[9:])
		}
	}

	return worktrees, nil
}

// CleanupOrphanedWorktrees removes any tfbreak worktrees that weren't properly cleaned up.
// This is a best-effort operation and errors are ignored.
func CleanupOrphanedWorktrees(repoDir string) {
	// Prune any worktrees whose directories no longer exist
	_, _ = Run([]string{"worktree", "prune"}, &RunOptions{Dir: repoDir})

	// Find and remove any worktrees in temp directories that look like ours
	worktrees, err := WorktreeList(repoDir)
	if err != nil {
		return
	}

	tmpDir := os.TempDir()
	for _, wt := range worktrees {
		// Check if this worktree is in the temp directory and matches our prefix
		// Use strings.HasPrefix on cleaned paths instead of deprecated filepath.HasPrefix
		cleanWt := filepath.Clean(wt)
		cleanTmp := filepath.Clean(tmpDir)
		baseName := filepath.Base(cleanWt)
		if len(baseName) >= 16 && baseName[:16] == "tfbreak-worktree" &&
			len(cleanWt) > len(cleanTmp) && cleanWt[:len(cleanTmp)] == cleanTmp {
			// Check if the directory actually exists
			if _, err := os.Stat(wt); err == nil {
				// Remove it
				_, _ = Run([]string{"worktree", "remove", "--force", wt}, &RunOptions{Dir: repoDir})
			}
		}
	}
}
