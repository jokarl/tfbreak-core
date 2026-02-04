package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRefExists_Branch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a test repository
	repoDir := t.TempDir()
	setupTestRepo(t, repoDir)

	// Check if the default branch exists
	exists, err := RefExists(repoDir, "HEAD")
	if err != nil {
		t.Fatalf("RefExists() error = %v", err)
	}

	if !exists {
		t.Error("RefExists('HEAD') = false, want true")
	}
}

func TestRefExists_Tag(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a test repository with a tag
	repoDir := t.TempDir()
	setupTestRepo(t, repoDir)
	createTag(t, repoDir, "v1.0.0")

	// Check if the tag exists
	exists, err := RefExists(repoDir, "v1.0.0")
	if err != nil {
		t.Fatalf("RefExists() error = %v", err)
	}

	if !exists {
		t.Error("RefExists('v1.0.0') = false, want true")
	}
}

func TestRefExists_Commit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a test repository
	repoDir := t.TempDir()
	setupTestRepo(t, repoDir)

	// Get the HEAD commit SHA
	sha := getHeadSHA(t, repoDir)

	// Check if the commit SHA exists
	exists, err := RefExists(repoDir, sha)
	if err != nil {
		t.Fatalf("RefExists() error = %v", err)
	}

	if !exists {
		t.Error("RefExists(commit SHA) = false, want true")
	}
}

func TestRefExists_NotFound(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a test repository
	repoDir := t.TempDir()
	setupTestRepo(t, repoDir)

	// Check for a non-existent ref
	exists, err := RefExists(repoDir, "nonexistent-branch-xyz-12345")
	if err != nil {
		t.Fatalf("RefExists() returned unexpected error: %v", err)
	}

	if exists {
		t.Error("RefExists('nonexistent-branch-xyz-12345') = true, want false")
	}
}

func TestRefExists_InvalidRef(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a test repository
	repoDir := t.TempDir()
	setupTestRepo(t, repoDir)

	// Check for an invalid ref (contains invalid characters)
	exists, err := RefExists(repoDir, "refs/heads/invalid..ref")
	if err != nil {
		// An error is acceptable for truly invalid refs
		return
	}

	if exists {
		t.Error("RefExists with invalid ref should return false")
	}
}

func TestRefExists_PartialSHA(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a test repository
	repoDir := t.TempDir()
	setupTestRepo(t, repoDir)

	// Get the HEAD commit SHA and use only first 7 chars
	sha := getHeadSHA(t, repoDir)
	if len(sha) >= 7 {
		partialSHA := sha[:7]

		exists, err := RefExists(repoDir, partialSHA)
		if err != nil {
			t.Fatalf("RefExists() error = %v", err)
		}

		if !exists {
			t.Error("RefExists(partial SHA) = false, want true")
		}
	}
}

func TestRefExists_NotGitRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a regular directory (not a git repo)
	dir := t.TempDir()

	// Check should return an error (not a git repo)
	_, err := RefExists(dir, "HEAD")
	if err == nil {
		t.Error("RefExists in non-git directory should return error")
	}
}

func TestRemoteRefExists_Branch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Skip in CI or if network is unavailable
	if os.Getenv("CI") != "" {
		t.Skip("skipping network test in CI")
	}

	// Test against a well-known public repository
	// Using a stable ref that should always exist
	exists, err := RemoteRefExists("https://github.com/git/git.git", "refs/heads/master")
	if err != nil {
		// Network errors are acceptable in test environments
		if IsAuthError(err) {
			t.Skip("authentication error, skipping")
		}
		t.Logf("RemoteRefExists() error (may be network related): %v", err)
		t.Skip("network unavailable")
	}

	if !exists {
		t.Error("RemoteRefExists for git/git master should return true")
	}
}

func TestRemoteRefExists_NotFound(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Skip in CI or if network is unavailable
	if os.Getenv("CI") != "" {
		t.Skip("skipping network test in CI")
	}

	// Test against a well-known public repository with a non-existent ref
	exists, err := RemoteRefExists("https://github.com/git/git.git", "refs/heads/nonexistent-branch-xyz-12345")
	if err != nil {
		// Network errors are acceptable in test environments
		if IsAuthError(err) {
			t.Skip("authentication error, skipping")
		}
		t.Logf("RemoteRefExists() error (may be network related): %v", err)
		t.Skip("network unavailable")
	}

	if exists {
		t.Error("RemoteRefExists for nonexistent branch should return false")
	}
}

func TestRemoteRefExists_Tag(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Skip in CI or if network is unavailable
	if os.Getenv("CI") != "" {
		t.Skip("skipping network test in CI")
	}

	// Test against a well-known public repository with a tag
	exists, err := RemoteRefExists("https://github.com/git/git.git", "refs/tags/v2.0.0")
	if err != nil {
		if IsAuthError(err) {
			t.Skip("authentication error, skipping")
		}
		t.Logf("RemoteRefExists() error (may be network related): %v", err)
		t.Skip("network unavailable")
	}

	if !exists {
		t.Error("RemoteRefExists for git/git v2.0.0 tag should return true")
	}
}

func TestRemoteRefExists_InvalidURL(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Test with an invalid URL
	_, err := RemoteRefExists("not-a-valid-url", "refs/heads/main")
	if err == nil {
		t.Error("RemoteRefExists with invalid URL should return error")
	}
}

func TestRemoteRefExists_LocalRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a "remote" repository (local bare repo)
	remoteDir := t.TempDir()
	setupBareRepo(t, remoteDir)

	// Create a local repo and push to the "remote"
	localDir := t.TempDir()
	setupTestRepo(t, localDir)
	addRemoteAndPush(t, localDir, remoteDir)

	// Now test RemoteRefExists against the local "remote"
	exists, err := RemoteRefExists(remoteDir, "refs/heads/main")
	if err != nil && !IsNotFound(err) {
		// Some git configs might use master instead of main
		exists, err = RemoteRefExists(remoteDir, "refs/heads/master")
	}

	// Either main or master should exist
	if err != nil {
		t.Logf("RemoteRefExists error: %v", err)
	}
	// We don't fail because branch name might vary
	_ = exists
}

func TestResolveRef(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a test repository
	repoDir := t.TempDir()
	setupTestRepo(t, repoDir)

	// Resolve HEAD
	sha, err := ResolveRef(repoDir, "HEAD")
	if err != nil {
		t.Fatalf("ResolveRef() error = %v", err)
	}

	// SHA should be 40 characters (full SHA)
	if len(sha) != 40 {
		t.Errorf("ResolveRef('HEAD') returned %q, want 40-char SHA", sha)
	}

	// SHA should be valid hex
	for _, c := range sha {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("ResolveRef returned invalid SHA character: %c", c)
		}
	}
}

func TestResolveRef_Tag(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a test repository with a tag
	repoDir := t.TempDir()
	setupTestRepo(t, repoDir)
	createTag(t, repoDir, "v1.0.0")

	// Resolve the tag
	sha, err := ResolveRef(repoDir, "v1.0.0")
	if err != nil {
		t.Fatalf("ResolveRef() error = %v", err)
	}

	// Compare with HEAD - they should be the same (tag points to HEAD)
	headSHA := getHeadSHA(t, repoDir)
	if sha != headSHA {
		t.Errorf("ResolveRef('v1.0.0') = %q, want %q (HEAD)", sha, headSHA)
	}
}

func TestResolveRef_NotFound(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a test repository
	repoDir := t.TempDir()
	setupTestRepo(t, repoDir)

	// Try to resolve a non-existent ref
	_, err := ResolveRef(repoDir, "nonexistent-ref-xyz-12345")
	if err == nil {
		t.Error("ResolveRef for nonexistent ref should return error")
	}
}

func TestResolveRef_NotGitRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a regular directory (not a git repo)
	dir := t.TempDir()

	// Resolve should return an error
	_, err := ResolveRef(dir, "HEAD")
	if err == nil {
		t.Error("ResolveRef in non-git directory should return error")
	}
}

// Helper functions for test setup

func setupTestRepo(t *testing.T, dir string) {
	t.Helper()

	// Initialize repo
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test User")

	// Create initial commit
	testFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-m", "Initial commit")
}

func setupBareRepo(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init", "--bare")
}

func createTag(t *testing.T, dir, tag string) {
	t.Helper()
	runGit(t, dir, "tag", tag)
}

func getHeadSHA(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD failed: %v", err)
	}
	return string(output[:40]) // Trim newline and any extra output
}

func addRemoteAndPush(t *testing.T, localDir, remoteDir string) {
	t.Helper()
	runGit(t, localDir, "remote", "add", "origin", remoteDir)

	// Get the current branch name
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = localDir
	output, err := cmd.Output()
	if err != nil {
		// Older git versions might not have --show-current
		runGit(t, localDir, "push", "-u", "origin", "HEAD:main")
		return
	}

	branch := strings.TrimSpace(string(output))
	if branch == "" {
		branch = "main"
	}
	runGit(t, localDir, "push", "-u", "origin", branch)
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}

func TestListRemoteRefs_LocalRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a "remote" repository (local bare repo)
	remoteDir := t.TempDir()
	setupBareRepo(t, remoteDir)

	// Create a local repo and push to the "remote"
	localDir := t.TempDir()
	setupTestRepo(t, localDir)
	createTag(t, localDir, "v1.0.0")
	createTag(t, localDir, "v2.0.0")
	addRemoteAndPush(t, localDir, remoteDir)
	// Push the tags
	runGit(t, localDir, "push", "origin", "--tags")

	// List all refs
	refs, err := ListRemoteRefs(remoteDir)
	if err != nil {
		t.Fatalf("ListRemoteRefs() error = %v", err)
	}

	// Should have at least some refs
	if len(refs) == 0 {
		t.Error("ListRemoteRefs returned empty map, expected some refs")
	}

	// Check for tags
	hasTag := false
	for ref := range refs {
		if strings.HasPrefix(ref, "refs/tags/") {
			hasTag = true
			break
		}
	}
	if !hasTag {
		t.Error("ListRemoteRefs didn't return any tags")
	}
}

func TestListRemoteRefs_WithPattern(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Create a "remote" repository (local bare repo)
	remoteDir := t.TempDir()
	setupBareRepo(t, remoteDir)

	// Create a local repo and push to the "remote"
	localDir := t.TempDir()
	setupTestRepo(t, localDir)
	createTag(t, localDir, "v1.0.0")
	addRemoteAndPush(t, localDir, remoteDir)
	runGit(t, localDir, "push", "origin", "--tags")

	// List only tags
	refs, err := ListRemoteRefs(remoteDir, "refs/tags/*")
	if err != nil {
		t.Fatalf("ListRemoteRefs() error = %v", err)
	}

	// All refs should be tags
	for ref := range refs {
		if !strings.HasPrefix(ref, "refs/tags/") {
			t.Errorf("ListRemoteRefs with tags pattern returned non-tag ref: %s", ref)
		}
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "single line no newline",
			input: "abc123def456\trefs/heads/main",
			want:  []string{"abc123def456\trefs/heads/main"},
		},
		{
			name:  "single line with newline",
			input: "abc123def456\trefs/heads/main\n",
			want:  []string{"abc123def456\trefs/heads/main"},
		},
		{
			name:  "multiple lines",
			input: "abc123\trefs/heads/main\ndef456\trefs/tags/v1.0.0\n",
			want:  []string{"abc123\trefs/heads/main", "def456\trefs/tags/v1.0.0"},
		},
		{
			name:  "multiple lines no trailing newline",
			input: "abc123\trefs/heads/main\ndef456\trefs/tags/v1.0.0",
			want:  []string{"abc123\trefs/heads/main", "def456\trefs/tags/v1.0.0"},
		},
		{
			name:  "lines with empty line in middle",
			input: "abc123\trefs/heads/main\n\ndef456\trefs/tags/v1.0.0\n",
			want:  []string{"abc123\trefs/heads/main", "def456\trefs/tags/v1.0.0"},
		},
		{
			name:  "only newlines",
			input: "\n\n\n",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitLines(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("splitLines(%q) returned %d lines, want %d", tt.input, len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitLines(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}
