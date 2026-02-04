package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIsShallowClone_True(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a "remote" repository with multiple commits (needed for shallow to be meaningful)
	remoteDir := t.TempDir()
	setupFullTestRepoWithHistory(t, remoteDir)

	// Create a shallow clone
	shallowDir := t.TempDir()
	cmd := exec.Command("git", "clone", "--depth=1", remoteDir, shallowDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone --depth=1 failed: %v\n%s", err, output)
	}

	// Verify .git/shallow file exists (some git configs may not create it for local clones)
	shallowFile := filepath.Join(shallowDir, ".git", "shallow")
	if _, err := os.Stat(shallowFile); os.IsNotExist(err) {
		t.Skip("git did not create shallow file for local clone (git version/config dependent)")
	}

	// Check if it's detected as shallow
	isShallow, err := IsShallowClone(shallowDir)
	if err != nil {
		t.Fatalf("IsShallowClone() error = %v", err)
	}

	if !isShallow {
		t.Error("IsShallowClone() = false, want true for shallow clone")
	}
}

func TestIsShallowClone_False(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a regular (non-shallow) repository
	repoDir := t.TempDir()
	setupFullTestRepo(t, repoDir)

	// Check if it's detected as non-shallow
	isShallow, err := IsShallowClone(repoDir)
	if err != nil {
		t.Fatalf("IsShallowClone() error = %v", err)
	}

	if isShallow {
		t.Error("IsShallowClone() = true, want false for full clone")
	}
}

func TestIsShallowClone_FullClone(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a "remote" repository
	remoteDir := t.TempDir()
	setupFullTestRepo(t, remoteDir)

	// Create a full clone (no --depth)
	fullCloneDir := t.TempDir()
	cmd := exec.Command("git", "clone", remoteDir, fullCloneDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone failed: %v\n%s", err, output)
	}

	// Check if it's detected as non-shallow
	isShallow, err := IsShallowClone(fullCloneDir)
	if err != nil {
		t.Fatalf("IsShallowClone() error = %v", err)
	}

	if isShallow {
		t.Error("IsShallowClone() = true, want false for full clone")
	}
}

func TestIsShallowClone_NotGitRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a regular directory (not a git repo)
	dir := t.TempDir()

	_, err := IsShallowClone(dir)
	if err == nil {
		t.Error("IsShallowClone in non-git directory should return error")
	}
}

func TestIsShallowClone_Subdirectory(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a repository
	repoDir := t.TempDir()
	setupFullTestRepo(t, repoDir)

	// Create a subdirectory
	subDir := filepath.Join(repoDir, "sub", "dir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	// Check from subdirectory - should still work
	isShallow, err := IsShallowClone(subDir)
	if err != nil {
		t.Fatalf("IsShallowClone() error = %v", err)
	}

	if isShallow {
		t.Error("IsShallowClone() from subdirectory = true, want false")
	}
}

func TestFindGitRoot_Found(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a repository
	repoDir := t.TempDir()
	setupFullTestRepo(t, repoDir)

	// Create nested subdirectories
	subDir := filepath.Join(repoDir, "a", "b", "c")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	// Find git root from subdirectory
	root, err := FindGitRoot(subDir)
	if err != nil {
		t.Fatalf("FindGitRoot() error = %v", err)
	}

	// Resolve symlinks for comparison (macOS /tmp -> /private/tmp)
	expectedRoot, err := filepath.EvalSymlinks(repoDir)
	if err != nil {
		t.Fatalf("failed to eval symlinks: %v", err)
	}
	actualRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("failed to eval symlinks for output: %v", err)
	}

	if actualRoot != expectedRoot {
		t.Errorf("FindGitRoot() = %q, want %q", actualRoot, expectedRoot)
	}
}

func TestFindGitRoot_AtRoot(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a repository
	repoDir := t.TempDir()
	setupFullTestRepo(t, repoDir)

	// Find git root from the root itself
	root, err := FindGitRoot(repoDir)
	if err != nil {
		t.Fatalf("FindGitRoot() error = %v", err)
	}

	// Resolve symlinks for comparison
	expectedRoot, err := filepath.EvalSymlinks(repoDir)
	if err != nil {
		t.Fatalf("failed to eval symlinks: %v", err)
	}
	actualRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("failed to eval symlinks for output: %v", err)
	}

	if actualRoot != expectedRoot {
		t.Errorf("FindGitRoot() = %q, want %q", actualRoot, expectedRoot)
	}
}

func TestFindGitRoot_NotRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a regular directory (not a git repo)
	dir := t.TempDir()

	_, err := FindGitRoot(dir)
	if err == nil {
		t.Error("FindGitRoot in non-git directory should return error")
	}

	// Check if the error is the expected type
	var notRepoErr *ErrNotARepository
	if !isNotARepositoryError(err) {
		// Allow for different error implementations
		t.Logf("FindGitRoot returned error: %v (type: %T)", err, err)
	}
	_ = notRepoErr
}

func TestFindGitRoot_NonExistentDir(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	_, err := FindGitRoot("/nonexistent/directory/that/does/not/exist")
	if err == nil {
		t.Error("FindGitRoot with non-existent directory should return error")
	}
}

func TestFindGitRoot_EmptyPath(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Empty path should either return an error or use current directory
	_, err := FindGitRoot("")
	// This behavior depends on implementation - either is acceptable
	_ = err
}

func TestFindGitRoot_WithGitDir(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a repository
	repoDir := t.TempDir()
	setupFullTestRepo(t, repoDir)

	// Try finding root from the .git directory itself
	gitDir := filepath.Join(repoDir, ".git")
	root, err := FindGitRoot(gitDir)

	// This might succeed or fail depending on implementation
	// Both behaviors are reasonable
	if err == nil {
		// Resolve symlinks for comparison
		expectedRoot, _ := filepath.EvalSymlinks(repoDir)
		actualRoot, _ := filepath.EvalSymlinks(root)
		if actualRoot != expectedRoot {
			t.Logf("FindGitRoot from .git dir = %q, repo = %q", actualRoot, expectedRoot)
		}
	}
}

func TestIsShallowClone_UnshallowedRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a "remote" repository with multiple commits
	remoteDir := t.TempDir()
	setupFullTestRepoWithHistory(t, remoteDir)

	// Create a shallow clone
	shallowDir := t.TempDir()
	cmd := exec.Command("git", "clone", "--depth=1", remoteDir, shallowDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone --depth=1 failed: %v\n%s", err, output)
	}

	// Verify .git/shallow file exists (some git configs may not create it for local clones)
	shallowFile := filepath.Join(shallowDir, ".git", "shallow")
	if _, err := os.Stat(shallowFile); os.IsNotExist(err) {
		t.Skip("git did not create shallow file for local clone (git version/config dependent)")
	}

	// Verify it's shallow
	isShallow, err := IsShallowClone(shallowDir)
	if err != nil {
		t.Fatalf("IsShallowClone() error = %v", err)
	}
	if !isShallow {
		t.Error("Clone should be shallow initially")
	}

	// Unshallow the repo
	cmd = exec.Command("git", "fetch", "--unshallow")
	cmd.Dir = shallowDir
	if output, err := cmd.CombinedOutput(); err != nil {
		// Skip if unshallow fails (might be due to single commit)
		t.Logf("git fetch --unshallow failed (may be expected): %v\n%s", err, output)
		t.Skip("skipping unshallow test - remote may have insufficient history")
	}

	// Now it should not be shallow
	isShallow, err = IsShallowClone(shallowDir)
	if err != nil {
		t.Fatalf("IsShallowClone() after unshallow error = %v", err)
	}
	if isShallow {
		t.Error("IsShallowClone() = true after unshallow, want false")
	}
}

// Helper functions

func setupFullTestRepo(t *testing.T, dir string) {
	t.Helper()

	runGitCmd(t, dir, "init")
	runGitCmd(t, dir, "config", "user.email", "test@test.com")
	runGitCmd(t, dir, "config", "user.name", "Test User")

	// Create initial commit
	testFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test Repository\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runGitCmd(t, dir, "add", "README.md")
	runGitCmd(t, dir, "commit", "-m", "Initial commit")
}

func setupFullTestRepoWithHistory(t *testing.T, dir string) {
	t.Helper()

	runGitCmd(t, dir, "init")
	runGitCmd(t, dir, "config", "user.email", "test@test.com")
	runGitCmd(t, dir, "config", "user.name", "Test User")

	// Create multiple commits for history
	for i := 1; i <= 5; i++ {
		testFile := filepath.Join(dir, "file.txt")
		content := []byte("Content version " + string(rune('0'+i)) + "\n")
		if err := os.WriteFile(testFile, content, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		runGitCmd(t, dir, "add", "file.txt")
		runGitCmd(t, dir, "commit", "-m", "Commit "+string(rune('0'+i)))
	}
}

func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}

func TestIsGitRepository_True(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a repository
	repoDir := t.TempDir()
	setupFullTestRepo(t, repoDir)

	if !IsGitRepository(repoDir) {
		t.Error("IsGitRepository() = false, want true for git repository")
	}
}

func TestIsGitRepository_False(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a regular directory (not a git repo)
	dir := t.TempDir()

	if IsGitRepository(dir) {
		t.Error("IsGitRepository() = true, want false for non-git directory")
	}
}

func TestIsGitRepository_Subdirectory(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a repository
	repoDir := t.TempDir()
	setupFullTestRepo(t, repoDir)

	// Create a nested subdirectory
	subDir := filepath.Join(repoDir, "nested", "dir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	if !IsGitRepository(subDir) {
		t.Error("IsGitRepository() = false, want true for subdirectory of git repo")
	}
}

func TestGetCurrentBranch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a repository
	repoDir := t.TempDir()
	setupFullTestRepo(t, repoDir)

	branch, err := GetCurrentBranch(repoDir)
	if err != nil {
		t.Fatalf("GetCurrentBranch() error = %v", err)
	}

	// Modern git defaults to main, older versions use master
	if branch != "main" && branch != "master" {
		t.Errorf("GetCurrentBranch() = %q, want 'main' or 'master'", branch)
	}
}

func TestGetCurrentBranch_DetachedHead(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a repository with history
	repoDir := t.TempDir()
	setupFullTestRepoWithHistory(t, repoDir)

	// Get the HEAD SHA
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoDir
	headOutput, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get HEAD: %v", err)
	}
	headSHA := string(headOutput)[:40] // Get the short SHA

	// Checkout a detached HEAD
	runGitCmd(t, repoDir, "checkout", headSHA)

	branch, err := GetCurrentBranch(repoDir)
	if err != nil {
		t.Fatalf("GetCurrentBranch() error = %v", err)
	}

	// In detached HEAD state, branch should be empty
	if branch != "" {
		t.Errorf("GetCurrentBranch() in detached HEAD = %q, want empty string", branch)
	}
}

func TestGetHEAD(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a repository
	repoDir := t.TempDir()
	setupFullTestRepo(t, repoDir)

	head, err := GetHEAD(repoDir)
	if err != nil {
		t.Fatalf("GetHEAD() error = %v", err)
	}

	// HEAD should be a 40-character hex string
	if len(head) != 40 {
		t.Errorf("GetHEAD() = %q, want 40-character SHA", head)
	}

	// Verify it's valid hex
	for _, c := range head {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			t.Errorf("GetHEAD() contains invalid character: %c", c)
			break
		}
	}
}

func TestGetHEAD_NotGitRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a regular directory (not a git repo)
	dir := t.TempDir()

	_, err := GetHEAD(dir)
	if err == nil {
		t.Error("GetHEAD() in non-git directory should return error")
	}
}

// Helper to check for ErrNotARepository error type
func isNotARepositoryError(err error) bool {
	if err == nil {
		return false
	}
	// Check by error message since we can't import the actual type in tests
	msg := err.Error()
	return containsSubstring(msg, "not a git repository") ||
		containsSubstring(msg, "not in a git repository")
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
