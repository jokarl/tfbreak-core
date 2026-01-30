package git

import (
	"errors"
	"testing"
)

func TestGitError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *GitError
		contains []string
	}{
		{
			name: "includes command name",
			err: &GitError{
				Command:  []string{"status"},
				ExitCode: 1,
				Stderr:   "fatal: not a git repository",
			},
			contains: []string{"git", "status", "exit 1", "fatal: not a git repository"},
		},
		{
			name: "includes exit code",
			err: &GitError{
				Command:  []string{"rev-parse", "--verify", "HEAD"},
				ExitCode: 128,
				Stderr:   "fatal: bad revision",
			},
			contains: []string{"128", "bad revision"},
		},
		{
			name: "empty stderr",
			err: &GitError{
				Command:  []string{"fetch"},
				ExitCode: 1,
				Stderr:   "",
			},
			contains: []string{"fetch", "exit 1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()
			for _, s := range tt.contains {
				if !containsString(errMsg, s) {
					t.Errorf("Error() = %q, want to contain %q", errMsg, s)
				}
			}
		})
	}
}

func TestErrRefNotFound_Error(t *testing.T) {
	tests := []struct {
		name      string
		err       *ErrRefNotFound
		contains  []string
		notContains []string
	}{
		{
			name: "simple ref not found",
			err: &ErrRefNotFound{
				Ref:       "nonexistent-branch",
				IsShallow: false,
			},
			contains:  []string{"nonexistent-branch", "not found"},
			notContains: []string{"shallow"},
		},
		{
			name: "ref not found in shallow clone",
			err: &ErrRefNotFound{
				Ref:       "v1.0.0",
				IsShallow: true,
			},
			contains: []string{"v1.0.0", "not found", "shallow", "fetch"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()
			for _, s := range tt.contains {
				if !containsString(errMsg, s) {
					t.Errorf("Error() = %q, want to contain %q", errMsg, s)
				}
			}
			for _, s := range tt.notContains {
				if containsString(errMsg, s) {
					t.Errorf("Error() = %q, should not contain %q", errMsg, s)
				}
			}
		})
	}
}

func TestErrNotARepository_Error(t *testing.T) {
	err := &ErrNotARepository{Dir: "/tmp/not-a-repo"}
	errMsg := err.Error()

	if !containsString(errMsg, "/tmp/not-a-repo") {
		t.Errorf("Error() = %q, want to contain directory path", errMsg)
	}
	if !containsString(errMsg, "not a git repository") {
		t.Errorf("Error() = %q, want to contain 'not a git repository'", errMsg)
	}
}

func TestIsNotFound_True(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{
			name: "exit code 128 with unknown revision",
			err: &GitError{
				Command:  []string{"rev-parse", "--verify", "nonexistent"},
				ExitCode: 128,
				Stderr:   "fatal: Needed a single revision",
			},
			expect: true,
		},
		{
			name: "exit code 128 with bad object",
			err: &GitError{
				Command:  []string{"rev-parse", "--verify", "bad-ref"},
				ExitCode: 128,
				Stderr:   "fatal: bad object bad-ref",
			},
			expect: true,
		},
		{
			name: "ls-remote exit code 2 (ref not found)",
			err: &GitError{
				Command:  []string{"ls-remote", "--exit-code", "origin", "nonexistent"},
				ExitCode: 2,
				Stderr:   "",
			},
			expect: true,
		},
		{
			name: "pathspec did not match",
			err: &GitError{
				Command:  []string{"show", "nonexistent:file.txt"},
				ExitCode: 128,
				Stderr:   "fatal: pathspec 'nonexistent:file.txt' did not match any file(s) known to git",
			},
			expect: true,
		},
		{
			name: "ErrRefNotFound error type",
			err:  &ErrRefNotFound{Ref: "missing"},
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotFound(tt.err)
			if result != tt.expect {
				t.Errorf("IsNotFound() = %v, want %v", result, tt.expect)
			}
		})
	}
}

func TestIsNotFound_False(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "generic error",
			err:  errors.New("some error"),
		},
		{
			name: "permission denied",
			err: &GitError{
				Command:  []string{"fetch", "origin"},
				ExitCode: 128,
				Stderr:   "fatal: could not read Username: terminal prompts disabled",
			},
		},
		{
			name: "network error",
			err: &GitError{
				Command:  []string{"fetch", "origin"},
				ExitCode: 128,
				Stderr:   "fatal: unable to access 'https://github.com/org/repo.git/': Could not resolve host",
			},
		},
		{
			name: "nil error",
			err:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotFound(tt.err)
			if result {
				t.Errorf("IsNotFound() = true, want false for %q", tt.name)
			}
		})
	}
}

func TestIsAuthError_SSH(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{
			name: "SSH permission denied",
			err: &GitError{
				Command:  []string{"fetch", "origin"},
				ExitCode: 128,
				Stderr:   "git@github.com: Permission denied (publickey).",
			},
			expect: true,
		},
		{
			name: "SSH host key verification failed",
			err: &GitError{
				Command:  []string{"clone", "git@github.com:org/repo.git"},
				ExitCode: 128,
				Stderr:   "Host key verification failed.",
			},
			expect: true,
		},
		{
			name: "SSH connection refused",
			err: &GitError{
				Command:  []string{"push", "origin", "main"},
				ExitCode: 128,
				Stderr:   "ssh: connect to host github.com port 22: Connection refused",
			},
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAuthError(tt.err)
			if result != tt.expect {
				t.Errorf("IsAuthError() = %v, want %v", result, tt.expect)
			}
		})
	}
}

func TestIsAuthError_HTTPS(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{
			name: "HTTPS 401 unauthorized",
			err: &GitError{
				Command:  []string{"fetch", "origin"},
				ExitCode: 128,
				Stderr:   "fatal: Authentication failed for 'https://github.com/org/repo.git/'",
			},
			expect: true,
		},
		{
			name: "HTTPS 403 forbidden",
			err: &GitError{
				Command:  []string{"push", "origin", "main"},
				ExitCode: 128,
				Stderr:   "remote: Permission to org/repo.git denied to user.",
			},
			expect: true,
		},
		{
			name: "terminal prompts disabled",
			err: &GitError{
				Command:  []string{"clone", "https://github.com/org/repo.git"},
				ExitCode: 128,
				Stderr:   "fatal: could not read Username for 'https://github.com': terminal prompts disabled",
			},
			expect: true,
		},
		{
			name: "invalid credentials",
			err: &GitError{
				Command:  []string{"fetch"},
				ExitCode: 128,
				Stderr:   "fatal: Invalid credentials",
			},
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAuthError(tt.err)
			if result != tt.expect {
				t.Errorf("IsAuthError() = %v, want %v", result, tt.expect)
			}
		})
	}
}

func TestIsAuthError_False(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "not found error",
			err: &GitError{
				Command:  []string{"rev-parse", "nonexistent"},
				ExitCode: 128,
				Stderr:   "fatal: bad revision 'nonexistent'",
			},
		},
		{
			name: "network error",
			err: &GitError{
				Command:  []string{"fetch"},
				ExitCode: 128,
				Stderr:   "fatal: unable to access: Could not resolve host: github.com",
			},
		},
		{
			name: "generic error",
			err:  errors.New("some error"),
		},
		{
			name: "nil error",
			err:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAuthError(tt.err)
			if result {
				t.Errorf("IsAuthError() = true, want false for %q", tt.name)
			}
		})
	}
}

func TestErrGitNotFound(t *testing.T) {
	// Verify ErrGitNotFound is a sentinel error
	if ErrGitNotFound == nil {
		t.Fatal("ErrGitNotFound should not be nil")
	}

	errMsg := ErrGitNotFound.Error()
	if !containsString(errMsg, "git") {
		t.Errorf("ErrGitNotFound.Error() = %q, want to contain 'git'", errMsg)
	}
}

func TestGitError_Unwrap(t *testing.T) {
	// If GitError wraps an underlying error, test Unwrap
	// This test checks that GitError implements error interface correctly
	var err error = &GitError{
		Command:  []string{"status"},
		ExitCode: 1,
		Stderr:   "error",
	}

	// Type assertion should work
	var gitErr *GitError
	if !errors.As(err, &gitErr) {
		t.Error("errors.As should match *GitError")
	}
}

// Helper function for string containment check
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchString(s, substr)))
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
