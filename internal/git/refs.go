package git

import (
	"fmt"
)

// RefExists checks if a ref exists in a local repository.
// Returns true if the ref exists, false if it doesn't, or an error if the check fails.
func RefExists(dir, ref string) (bool, error) {
	_, err := Run([]string{"rev-parse", "--verify", "--quiet", ref}, &RunOptions{Dir: dir})
	if err != nil {
		if IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check ref %q: %w", ref, err)
	}
	return true, nil
}

// ResolveRef resolves a ref to its commit SHA in a local repository.
func ResolveRef(dir, ref string) (string, error) {
	sha, err := Run([]string{"rev-parse", ref}, &RunOptions{Dir: dir})
	if err != nil {
		if IsNotFound(err) {
			isShallow, _ := IsShallowClone(dir)
			return "", &ErrRefNotFound{Ref: ref, IsShallow: isShallow}
		}
		return "", fmt.Errorf("failed to resolve ref %q: %w", ref, err)
	}
	return sha, nil
}

// RemoteRefExists checks if a ref exists in a remote repository without cloning.
// This uses git ls-remote which only fetches ref information, not content.
func RemoteRefExists(url, ref string) (bool, error) {
	// git ls-remote --exit-code returns exit code 2 if ref not found
	_, err := Run([]string{"ls-remote", "--exit-code", url, ref}, nil)
	if err != nil {
		if IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check remote ref %q in %q: %w", ref, url, err)
	}
	return true, nil
}

// ResolveRemoteRef resolves a ref to its commit SHA in a remote repository.
// Returns the SHA and the full ref name (e.g., "refs/tags/v1.0.0").
func ResolveRemoteRef(url, ref string) (sha string, fullRef string, err error) {
	// git ls-remote returns lines like:
	// abc123def456... refs/heads/main
	// abc123def456... refs/tags/v1.0.0
	out, err := Run([]string{"ls-remote", url, ref}, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve remote ref %q: %w", ref, err)
	}

	if out == "" {
		return "", "", &ErrRefNotFound{Ref: ref, Remote: url}
	}

	// Parse the first line (there might be multiple matches)
	var resolvedSHA, resolvedRef string
	_, err = fmt.Sscanf(out, "%s %s", &resolvedSHA, &resolvedRef)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse ls-remote output: %w", err)
	}

	return resolvedSHA, resolvedRef, nil
}

// ListRemoteRefs lists all refs in a remote repository.
// If pattern is provided, only matching refs are returned.
func ListRemoteRefs(url string, patterns ...string) (map[string]string, error) {
	args := []string{"ls-remote", url}
	args = append(args, patterns...)

	out, err := Run(args, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list remote refs: %w", err)
	}

	refs := make(map[string]string)
	if out == "" {
		return refs, nil
	}

	// Parse lines: "sha\trefs/..."
	lines := splitLines(out)
	for _, line := range lines {
		var sha, ref string
		if _, err := fmt.Sscanf(line, "%s %s", &sha, &ref); err == nil {
			refs[ref] = sha
		}
	}

	return refs, nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			if i > start {
				lines = append(lines, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
