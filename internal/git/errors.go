package git

import (
	"errors"
	"fmt"
	"strings"
)

// ErrGitNotFound is returned when git is not installed or not in PATH.
var ErrGitNotFound = errors.New("git is not installed or not in PATH")

// GitError wraps errors from git command execution with full context.
type GitError struct {
	Command  []string
	ExitCode int
	Stderr   string
}

func (e *GitError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("git %s failed (exit %d): %s", e.Command[0], e.ExitCode, strings.TrimSpace(e.Stderr))
	}
	return fmt.Sprintf("git %s failed (exit %d)", e.Command[0], e.ExitCode)
}

// ErrNotARepository is returned when the directory is not inside a git repository.
type ErrNotARepository struct {
	Dir string
}

func (e *ErrNotARepository) Error() string {
	return fmt.Sprintf("'%s' is not a git repository (or any parent directory)", e.Dir)
}

// ErrRefNotFound is returned when a ref doesn't exist.
type ErrRefNotFound struct {
	Ref       string
	Remote    string // Empty for local refs
	IsShallow bool
}

func (e *ErrRefNotFound) Error() string {
	var msg string
	if e.Remote != "" {
		msg = fmt.Sprintf("ref '%s' not found in '%s'", e.Ref, e.Remote)
	} else {
		msg = fmt.Sprintf("ref '%s' not found", e.Ref)
	}

	if e.IsShallow {
		msg += "\n\nThis repository is a shallow clone. The ref may exist but is not in the local history.\n"
		msg += "To fix, fetch the ref:\n\n"
		msg += fmt.Sprintf("  git fetch origin %s\n\n", e.Ref)
		msg += "Or fetch full history:\n\n"
		msg += "  git fetch --unshallow"
	}

	return msg
}

// ErrVersionTooOld is returned when git version is below the minimum required.
type ErrVersionTooOld struct {
	Current  string
	Required string
}

func (e *ErrVersionTooOld) Error() string {
	return fmt.Sprintf("git version %s is below minimum required %s\n\n"+
		"Please upgrade git: https://git-scm.com/downloads", e.Current, e.Required)
}

// IsNotFound returns true if the error indicates a ref was not found.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	var refErr *ErrRefNotFound
	if errors.As(err, &refErr) {
		return true
	}

	var gitErr *GitError
	if errors.As(err, &gitErr) {
		stderr := strings.ToLower(gitErr.Stderr)

		// First check if this is an auth error - those are not "not found"
		if isAuthErrorStderr(stderr) {
			return false
		}

		// git rev-parse --verify --quiet returns exit code 1 for non-existent refs
		// git ls-remote --exit-code returns exit code 2 specifically for ref not found
		if gitErr.ExitCode == 1 || gitErr.ExitCode == 2 {
			return true
		}

		// For exit code 128, check for specific "not found" patterns in stderr
		if gitErr.ExitCode == 128 {
			if strings.Contains(stderr, "unknown revision") ||
				strings.Contains(stderr, "bad object") ||
				strings.Contains(stderr, "pathspec") ||
				strings.Contains(stderr, "does not exist") ||
				strings.Contains(stderr, "needed a single revision") ||
				strings.Contains(stderr, "bad revision") {
				return true
			}
		}
	}

	return false
}

// IsAuthError returns true if the error indicates an authentication failure.
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}

	var gitErr *GitError
	if !errors.As(err, &gitErr) {
		return false
	}

	return isAuthErrorStderr(strings.ToLower(gitErr.Stderr))
}

// isAuthErrorStderr checks stderr content for authentication error patterns.
func isAuthErrorStderr(stderr string) bool {
	// SSH authentication failures
	if strings.Contains(stderr, "permission denied") ||
		strings.Contains(stderr, "publickey") ||
		strings.Contains(stderr, "authentication failed") ||
		strings.Contains(stderr, "could not read from remote repository") ||
		strings.Contains(stderr, "host key verification failed") ||
		strings.Contains(stderr, "connection refused") {
		return true
	}

	// HTTPS authentication failures
	if strings.Contains(stderr, "401") ||
		strings.Contains(stderr, "403") ||
		strings.Contains(stderr, "authentication") ||
		strings.Contains(stderr, "invalid credentials") ||
		strings.Contains(stderr, "could not authenticate") ||
		strings.Contains(stderr, "terminal prompts disabled") ||
		strings.Contains(stderr, "permission to") { // e.g. "Permission to org/repo.git denied"
		return true
	}

	return false
}
