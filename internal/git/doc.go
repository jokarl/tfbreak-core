// Package git provides a safe wrapper around git command execution.
//
// This package delegates all git operations to the system git binary,
// leveraging the user's existing git configuration for authentication.
// It does not implement any platform-specific code (GitHub, GitLab, etc.)
// and does not store or manage credentials.
//
// Key features:
//   - Command execution with proper stderr capture for error diagnostics
//   - Git version detection and validation (minimum: 2.5)
//   - Ref existence checking for local and remote repositories
//   - Repository state detection (shallow clone, git root)
//   - Structured error types with actionable messages
//
// Example usage:
//
//	// Check if git is available
//	if !git.Available() {
//	    return git.ErrGitNotFound
//	}
//
//	// Check git version
//	if err := git.CheckVersion(2, 5); err != nil {
//	    return err
//	}
//
//	// Check if a ref exists locally
//	exists, err := git.RefExists("/path/to/repo", "main")
//
//	// Check if a ref exists in a remote repository
//	exists, err := git.RemoteRefExists("https://github.com/org/repo", "v1.0.0")
package git
