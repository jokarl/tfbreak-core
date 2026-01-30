package git

import (
	"os"
	"path/filepath"
)

// FindGitRoot finds the root directory of the git repository containing dir.
// Returns the path to the repository root, or an error if dir is not in a git repository.
func FindGitRoot(dir string) (string, error) {
	// Resolve to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	// Use git rev-parse to find the root
	root, err := Run([]string{"rev-parse", "--show-toplevel"}, &RunOptions{Dir: absDir})
	if err != nil {
		return "", &ErrNotARepository{Dir: dir}
	}

	return root, nil
}

// IsGitRepository returns true if dir is inside a git repository.
func IsGitRepository(dir string) bool {
	_, err := FindGitRoot(dir)
	return err == nil
}

// IsShallowClone returns true if the repository at dir is a shallow clone.
// A shallow clone has a .git/shallow file.
func IsShallowClone(dir string) (bool, error) {
	gitDir, err := getGitDir(dir)
	if err != nil {
		return false, err
	}

	shallowFile := filepath.Join(gitDir, "shallow")
	_, err = os.Stat(shallowFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// getGitDir returns the path to the .git directory for a repository.
func getGitDir(dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	// git rev-parse --git-dir returns the path to .git
	gitDir, err := Run([]string{"rev-parse", "--git-dir"}, &RunOptions{Dir: absDir})
	if err != nil {
		return "", &ErrNotARepository{Dir: dir}
	}

	// If the result is relative, make it absolute
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(absDir, gitDir)
	}

	return gitDir, nil
}

// GetCurrentBranch returns the current branch name, or empty string if in detached HEAD state.
func GetCurrentBranch(dir string) (string, error) {
	branch, err := Run([]string{"rev-parse", "--abbrev-ref", "HEAD"}, &RunOptions{Dir: dir})
	if err != nil {
		return "", err
	}

	// HEAD is returned for detached HEAD state
	if branch == "HEAD" {
		return "", nil
	}

	return branch, nil
}

// GetHEAD returns the current HEAD commit SHA.
func GetHEAD(dir string) (string, error) {
	return Run([]string{"rev-parse", "HEAD"}, &RunOptions{Dir: dir})
}
