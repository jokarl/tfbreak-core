package git

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

// RunOptions configures how a git command is executed.
type RunOptions struct {
	// Dir is the working directory for the command.
	// If empty, the current working directory is used.
	Dir string

	// Env contains additional environment variables.
	// These are appended to the current environment.
	Env []string
}

// Available returns true if git is installed and in PATH.
func Available() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// Run executes a git command and returns the stdout output.
// If the command fails, a *GitError is returned with stderr context.
func Run(args []string, opts *RunOptions) (string, error) {
	if !Available() {
		return "", ErrGitNotFound
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set working directory if specified
	if opts != nil && opts.Dir != "" {
		cmd.Dir = opts.Dir
	}

	// Inherit environment for credentials, SSH config, etc.
	cmd.Env = os.Environ()
	if opts != nil && len(opts.Env) > 0 {
		cmd.Env = append(cmd.Env, opts.Env...)
	}

	err := cmd.Run()
	if err != nil {
		exitCode := 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		return "", &GitError{
			Command:  args,
			ExitCode: exitCode,
			Stderr:   stderr.String(),
		}
	}

	return strings.TrimSpace(stdout.String()), nil
}

// RunSilent executes a git command without capturing output.
// It returns an error if the command fails.
func RunSilent(args []string, opts *RunOptions) error {
	_, err := Run(args, opts)
	return err
}
