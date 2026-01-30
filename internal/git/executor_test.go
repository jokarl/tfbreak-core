package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAvailable_GitInstalled(t *testing.T) {
	// Skip if git is not installed (allows test to pass in environments without git)
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	if !Available() {
		t.Error("Available() = false, want true when git is installed")
	}
}

func TestAvailable_GitMissing(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// Set PATH to empty to simulate git not being available
	os.Setenv("PATH", "")

	if Available() {
		t.Error("Available() = true, want false when git is not in PATH")
	}
}

func TestRun_Success(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	output, err := Run([]string{"--version"}, nil)
	if err != nil {
		t.Fatalf("Run(['--version']) returned error: %v", err)
	}

	if !strings.HasPrefix(output, "git version") {
		t.Errorf("Run(['--version']) = %q, want output starting with 'git version'", output)
	}
}

func TestRun_Failure(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Run an invalid git command
	_, err := Run([]string{"invalid-command-that-does-not-exist"}, nil)
	if err == nil {
		t.Fatal("Run(['invalid-command-that-does-not-exist']) should return error")
	}

	// Verify it's a GitError
	gitErr, ok := err.(*GitError)
	if !ok {
		t.Fatalf("expected *GitError, got %T", err)
	}

	// Verify error contains command info
	if len(gitErr.Command) == 0 {
		t.Error("GitError.Command should not be empty")
	}

	// Verify exit code is set
	if gitErr.ExitCode == 0 {
		t.Error("GitError.ExitCode should be non-zero for failed command")
	}
}

func TestRun_WorkingDir(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a temporary git repository
	repoDir := t.TempDir()
	initGitRepo(t, repoDir)

	// Create a subdirectory
	subDir := filepath.Join(repoDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	// Run git rev-parse --show-toplevel from the subdirectory
	output, err := Run([]string{"rev-parse", "--show-toplevel"}, &RunOptions{Dir: subDir})
	if err != nil {
		t.Fatalf("Run with Dir option failed: %v", err)
	}

	// The output should be the repo root, not the subdirectory
	// Resolve symlinks for comparison (macOS /tmp -> /private/tmp)
	expectedRoot, err := filepath.EvalSymlinks(repoDir)
	if err != nil {
		t.Fatalf("failed to eval symlinks: %v", err)
	}
	actualRoot, err := filepath.EvalSymlinks(output)
	if err != nil {
		t.Fatalf("failed to eval symlinks for output: %v", err)
	}

	if actualRoot != expectedRoot {
		t.Errorf("git rev-parse --show-toplevel = %q, want %q", actualRoot, expectedRoot)
	}
}

func TestRun_Environment(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a temporary git repository
	repoDir := t.TempDir()
	initGitRepo(t, repoDir)

	// Set custom author via environment
	customAuthor := "Test Author <test@example.com>"

	// Create a file and commit with custom author
	testFile := filepath.Join(repoDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Stage the file
	_, err := Run([]string{"add", "test.txt"}, &RunOptions{Dir: repoDir})
	if err != nil {
		t.Fatalf("git add failed: %v", err)
	}

	// Commit with custom environment
	_, err = Run([]string{"commit", "-m", "test commit"}, &RunOptions{
		Dir: repoDir,
		Env: []string{
			"GIT_AUTHOR_NAME=Test Author",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test Author",
			"GIT_COMMITTER_EMAIL=test@example.com",
		},
	})
	if err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Verify the author
	output, err := Run([]string{"log", "-1", "--format=%an <%ae>"}, &RunOptions{Dir: repoDir})
	if err != nil {
		t.Fatalf("git log failed: %v", err)
	}

	if output != customAuthor {
		t.Errorf("commit author = %q, want %q", output, customAuthor)
	}
}

func TestRun_InheritsEnvironment(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Set a custom environment variable
	testEnvKey := "TFBREAK_TEST_VAR"
	testEnvValue := "test-value-12345"
	os.Setenv(testEnvKey, testEnvValue)
	defer os.Unsetenv(testEnvKey)

	// Run git with no additional env - it should inherit current environment
	// We can't directly verify env inheritance from git output,
	// but we can verify the command runs successfully
	_, err := Run([]string{"--version"}, nil)
	if err != nil {
		t.Errorf("Run should succeed when inheriting environment: %v", err)
	}
}

func TestRun_StderrCapture(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Run a command that writes to stderr
	_, err := Run([]string{"rev-parse", "--verify", "nonexistent-ref-12345"}, nil)
	if err == nil {
		t.Fatal("expected error for nonexistent ref")
	}

	gitErr, ok := err.(*GitError)
	if !ok {
		t.Fatalf("expected *GitError, got %T", err)
	}

	// Stderr should contain the error message
	if gitErr.Stderr == "" {
		t.Error("GitError.Stderr should contain error output")
	}
}

func TestRun_OutputTrimmed(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	output, err := Run([]string{"--version"}, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Output should not have trailing newline
	if strings.HasSuffix(output, "\n") {
		t.Error("Run output should have trailing whitespace trimmed")
	}
}

func TestRun_EmptyArgs(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Running git with no arguments should show usage (and may return error)
	// We just want to ensure it doesn't panic
	_, _ = Run([]string{}, nil)
}

func TestRun_NilOptions(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Passing nil options should work
	output, err := Run([]string{"--version"}, nil)
	if err != nil {
		t.Fatalf("Run with nil options failed: %v", err)
	}

	if output == "" {
		t.Error("Run with nil options should return output")
	}
}

func TestRunOptions_EmptyDir(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Empty Dir should use current directory
	output, err := Run([]string{"--version"}, &RunOptions{Dir: ""})
	if err != nil {
		t.Fatalf("Run with empty Dir failed: %v", err)
	}

	if output == "" {
		t.Error("Run with empty Dir should return output")
	}
}

func TestRunOptions_EmptyEnv(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Empty Env slice should work (inherits current env)
	output, err := Run([]string{"--version"}, &RunOptions{Env: []string{}})
	if err != nil {
		t.Fatalf("Run with empty Env failed: %v", err)
	}

	if output == "" {
		t.Error("Run with empty Env should return output")
	}
}

func TestRun_NonExistentDir(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Running in a non-existent directory should fail
	_, err := Run([]string{"status"}, &RunOptions{Dir: "/nonexistent/directory/that/does/not/exist"})
	if err == nil {
		t.Error("Run in non-existent directory should return error")
	}
}

// Helper function to initialize a git repository for testing
func initGitRepo(t *testing.T, dir string) {
	t.Helper()

	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, output)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git config email failed: %v\n%s", err, output)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git config name failed: %v\n%s", err, output)
	}
}
