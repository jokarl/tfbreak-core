---
status: accepted
date: 2026-01-30
decision-makers: [project maintainers]
consulted: []
informed: []
---

# Git Ref Comparison for Remote and Local Repository Analysis

## Context and Problem Statement

tfbreak currently requires users to provide two local directories for comparison (`tfbreak check <old_dir> <new_dir>`). This creates friction in CI/CD workflows where users must manually check out specific git refs (tags, branches, commits) into temporary directories before running the comparison.

Users need the ability to compare:
1. The current working directory against a git ref (e.g., `main` branch, `v1.0.0` tag)
2. Two arbitrary git refs without manual checkout
3. Against remote repositories (their own private repos or public modules)

The key challenge is authentication: how should tfbreak handle private repositories that require credentials, while maintaining a seamless experience for public repositories?

## Decision Drivers

* Must be platform-agnostic - work with any git remote (self-hosted, cloud-hosted)
* Must delegate authentication to the user's existing git configuration
* Must handle authentication securely without storing or managing credentials
* Must work transparently for public repositories without configuration
* Must maintain backwards compatibility with existing directory-based comparison
* Must follow established patterns from similar tools (tflint, terraform)
* Should minimize external dependencies and complexity
* Should enable community-built wrappers (GitHub Actions, GitLab CI templates) without requiring platform-specific code in tfbreak

## Considered Options

* **Option 1: Shell out to system git with credential delegation**
* **Option 2: Embedded go-git library with custom credential handling**
* **Option 3: Virtual filesystem with sparse checkout**
* **Option 4: GitHub/GitLab API-first approach**

## Decision Outcome

Chosen option: "Option 1: Shell out to system git with credential delegation", because it leverages the user's existing git configuration and credential helpers, requires no additional authentication implementation, and works identically to how users already interact with git.

### Consequences

* Good, because users' existing SSH keys, credential helpers, and tokens work automatically
* Good, because supports all git hosting platforms without platform-specific code
* Good, because minimal implementation complexity and maintenance burden
* Good, because familiar behavior - same as running `git clone` or `git worktree`
* Good, because CI/CD environments already have git credentials configured
* Neutral, because requires git to be installed on the system (reasonable assumption)
* Bad, because cannot provide custom error messages for authentication failures
* Bad, because dependent on system git version for advanced features

### Confirmation

This decision will be confirmed by:
1. Successfully comparing against public repositories without authentication
2. Successfully comparing against private repositories using SSH keys
3. Successfully comparing against private repositories using HTTPS with credential helper
4. Git errors (authentication failures, network issues) are surfaced clearly to the user
5. No platform-specific code exists in tfbreak - all git operations use standard git commands

## Pros and Cons of the Options

### Option 1: Shell out to system git with credential delegation

Use the system's installed git binary via `os/exec` to perform all git operations. tfbreak acts as a thin wrapper that invokes git commands and reads the resulting files.

```
User's git config        tfbreak              git binary
┌────────────────┐      ┌─────────┐          ┌─────────┐
│ ~/.gitconfig   │      │         │  exec    │         │
│ SSH keys       │─────▶│ tfbreak │─────────▶│  git    │
│ credential     │      │         │  clone/  │         │
│ helpers        │      │         │  worktree│         │
└────────────────┘      └─────────┘          └─────────┘
```

**Authentication flow:**
1. tfbreak invokes `git clone` or `git worktree add`
2. Git uses the user's configured credential helpers (macOS Keychain, Windows Credential Manager, `git-credential-cache`, etc.)
3. For SSH URLs, git uses the user's SSH agent and keys
4. For HTTPS URLs with tokens, git uses `GIT_ASKPASS` or credential helpers
5. Environment variables like `GITHUB_TOKEN` work via git's built-in support

* Good, because leverages decades of git credential management development
* Good, because SSH keys, GPG signing, credential helpers all work automatically
* Good, because users can debug issues using standard git commands
* Good, because supports any git remote (self-hosted, cloud-hosted, any provider)
* Good, because sparse checkout and partial clone supported natively
* Good, because platform-agnostic - no provider-specific code required
* Neutral, because requires git >= 2.25 for optimal sparse checkout support
* Bad, because git must be installed (reasonable assumption for Terraform users)
* Bad, because error messages come from git, not tfbreak

### Option 2: Embedded go-git library with custom credential handling

Use the pure-Go [go-git](https://github.com/go-git/go-git) library to perform git operations without shelling out.

* Good, because no external git dependency
* Good, because single binary distribution maintained
* Good, because consistent behavior across platforms
* Bad, because must implement credential handling from scratch
* Bad, because SSH key handling is complex (agent, key files, passphrases)
* Bad, because credential helper integration requires significant work
* Bad, because go-git doesn't support all git features (sparse checkout limited)
* Bad, because increases binary size significantly (~10MB)
* Bad, because would need to maintain credential handling code

### Option 3: Virtual filesystem with sparse checkout

Implement a virtual filesystem that fetches only the `.tf` files needed for comparison, using git's sparse checkout and partial clone features.

* Good, because minimal data transfer (only .tf files)
* Good, because faster for large repositories
* Bad, because complex implementation
* Bad, because sparse checkout support varies by git version
* Bad, because still requires authentication solution
* Bad, because harder to debug issues

### Option 4: Platform API-first approach

Use platform-specific APIs (GitHub API, GitLab API, etc.) to fetch files directly, with git as a fallback.

* Good, because can provide platform-specific optimizations
* Good, because fine-grained access control via API tokens
* Bad, because requires platform-specific code for each provider
* Bad, because violates platform-agnostic design principle
* Bad, because API rate limits affect usability
* Bad, because doesn't support self-hosted git servers without additional code
* Bad, because maintenance burden grows with each supported platform
* Bad, because inconsistent behavior across platforms

## More Information

### CLI Interface Design

The proposed CLI interface extends the existing `check` command:

```bash
# Current: directory-based (unchanged)
tfbreak check ./old ./new

# New: git ref mode
tfbreak check --base main ./                    # Compare current dir against main branch
tfbreak check --base v1.0.0 --head v2.0.0       # Compare two tags
tfbreak check --base HEAD~5 ./                  # Compare against 5 commits ago
tfbreak check --base origin/main ./             # Compare against remote branch

# New: ref:path syntax for monorepos (like git show REVISION:path)
tfbreak check --base main:modules/vpc ./modules/vpc      # Compare subdirectory at ref
tfbreak check --base v1:src --head v2:src                # Compare src/ between tags
tfbreak check --base v1:modules/old-vpc --head v2:modules/vpc  # Handle renames

# New: remote repository mode
tfbreak check --repo https://github.com/org/module --base v1.0.0 --head v2.0.0
tfbreak check --repo git@github.com:org/module.git --base main --head feature
tfbreak check --repo https://github.com/org/mod --base v1:terraform --head v2:terraform
```

Flags:
- `--base <ref[:path]>`: Git ref for the "old" configuration (tag, branch, commit SHA), optionally with a subdirectory path
- `--head <ref[:path]>`: Git ref for the "new" configuration (defaults to working directory), optionally with a subdirectory path
- `--repo <url>`: Remote repository URL (when comparing remote refs)

### Ref:Path Syntax (Monorepo Support)

The `ref:path` syntax follows git's convention (as used in `git show REVISION:path`). This enables comparison of specific subdirectories within a repository, which is essential for monorepos containing multiple Terraform modules.

**Syntax:** `<ref>:<path>` where:
- `<ref>` is any valid git ref (branch, tag, commit SHA, relative ref like HEAD~5)
- `<path>` is a path relative to the repository root

**Examples:**
```bash
# Compare modules/vpc at main branch against local modules/vpc
tfbreak check --base main:modules/vpc ./modules/vpc

# Compare src directory between two tags
tfbreak check --base v1.0.0:src --head v2.0.0:src

# Handle module rename between versions
tfbreak check --base v1.0.0:modules/old-name --head v2.0.0:modules/new-name

# Remote monorepo comparison
tfbreak check --repo https://github.com/org/infra --base v1:terraform/prod --head v2:terraform/prod
```

**Behavior:**
- If no `:path` is specified, the repository root is used (default behavior)
- The path is applied after the worktree/clone is created
- URL-like refs (containing `://`) are not parsed for paths
- Each ref can have its own independent path, enabling comparison across renames

### Authentication Model

tfbreak delegates all authentication to git. This means:

1. **Public repositories**: Work without any configuration
2. **Private repositories (SSH)**: Use the user's SSH keys via ssh-agent
3. **Private repositories (HTTPS)**: Use git credential helpers configured by the user

tfbreak does not implement, manage, or store any credentials. Users configure their git environment once, and tfbreak benefits from that configuration automatically.

**Supported authentication methods (via git):**
- SSH keys (`~/.ssh/id_rsa`, `~/.ssh/id_ed25519`, etc.)
- SSH agent forwarding
- Git credential helpers (`git-credential-cache`, `git-credential-store`, OS-specific helpers)
- Environment variables that git recognizes (`GIT_SSH_COMMAND`, `GIT_ASKPASS`, etc.)

**Out of scope:**
- Platform-specific token handling (e.g., `GITHUB_TOKEN`, `CI_JOB_TOKEN`)
- CI/CD-specific credential injection
- OAuth flows or interactive authentication

Community contributors may create platform-specific wrappers (GitHub Actions, GitLab CI templates) that handle token injection before invoking tfbreak.

### Implementation Approach

1. **Temporary worktree strategy**: Use `git worktree add` for comparing refs within the same repository
2. **Shallow clone for remote**: Use `git clone --depth 1 --branch <ref>` for remote repository comparison
3. **Cleanup**: Automatic cleanup of temporary directories on exit (via `defer`)

### Community Wrappers (Out of Scope)

tfbreak provides the core CLI capability. Platform-specific integrations are intentionally out of scope and left to community contributors:

- **GitHub Actions**: A community action could handle `GITHUB_TOKEN` injection and provide inputs for base/head refs
- **GitLab CI templates**: A community template could handle `CI_JOB_TOKEN` configuration
- **Azure DevOps tasks**: A community task could integrate with Azure Repos authentication

This separation keeps tfbreak simple and platform-agnostic while enabling rich integrations through the ecosystem.

### Security Considerations

1. **Credential exposure**: tfbreak never reads, stores, or logs credentials - all credential handling is delegated to git
2. **Temporary files**: Git worktrees/clones created in system temp directory with restricted permissions
3. **Environment variables**: tfbreak passes through the environment to git but does not interpret credential-related variables
4. **Principle of least privilege**: Users should configure their git credentials with minimal required permissions (read-only where possible)

### Error Handling

**Fail-fast principle**: tfbreak validates prerequisites before expensive operations:

1. Check git is installed (via `exec.LookPath`)
2. Check we're in a git repository (when using `--base` without `--repo`)
3. Validate refs exist before creating worktrees or clones
4. Check git version meets minimum requirements

When git operations fail, tfbreak will:
1. Display the git error message verbatim (no interpretation or transformation)
2. Provide generic troubleshooting suggestions for common issues
3. Exit with a non-zero status code (exit code 2 for pre-flight failures)

Example error:
```
Error: failed to fetch git ref 'main' from 'git@example.com:org/private-repo.git'

Git error: Permission denied (publickey).

Troubleshooting:
- For SSH: Ensure your SSH key is loaded (ssh-add -l) and the remote host is known
- For HTTPS: Verify your credential helper is configured (git config credential.helper)
- Test manually: git ls-remote <url>
```

tfbreak does not attempt to diagnose authentication issues beyond surfacing git's error messages. Users should use standard git debugging techniques (`GIT_SSH_COMMAND="ssh -vvv"`, `GIT_TRACE=1`, etc.).

### Phased Implementation

**Phase 1: Git Infrastructure (CR-0015)**
- Create `internal/git` package with command execution wrapper
- Implement error handling with stderr capture
- Add git version detection and validation (minimum: git 2.5)
- Add `git ls-remote` support for ref validation

**Phase 2: Local Repository Git Refs (CR-0016)**
- Add `--base` flag to compare working directory against local refs
- Uses `git worktree add --detach` for efficient comparison
- Automatic cleanup of temporary worktrees
- No network access required for local refs

**Phase 3: Remote Repository Comparison (CR-0017)**
- Add `--repo` flag for remote repository comparison
- Uses `git clone --depth 1 --branch <ref> --single-branch` for efficiency
- Pre-flight validation with `git ls-remote`
- Automatic cleanup of temporary clones

**Phase 4: Optimizations (Future)**
- Sparse checkout to fetch only `.tf` files (requires git 2.25+)
- Caching of remote clones for repeated comparisons
- Parallel fetching for faster comparison

### Shallow Clone Considerations

tfbreak assumes the local repository has sufficient history to resolve the requested refs. This is the typical case for developer workstations.

**CI/CD environments** often use shallow clones by default for performance. When using `--base` or `--head` with local refs:

1. Refs must be present in the local history
2. Shallow clones may not have the requested refs
3. tfbreak will detect shallow clones and provide remediation guidance

**Remediation options:**
- Fetch specific ref: `git fetch origin tag v1.0.0 --no-tags`
- Fetch full history: `git fetch --unshallow`
- Configure CI for full depth (e.g., `fetch-depth: 0` in GitHub Actions)

tfbreak will NOT automatically fetch refs - this could have unintended side effects. Users must ensure refs are available.

### Git Version Requirements

| Feature | Minimum Git Version | Notes |
|---------|---------------------|-------|
| Basic worktree | 2.5 | Introduced in July 2015 |
| Worktree lock/unlock | 2.15 | Prevents accidental deletion |
| Worktree prune | 2.17 | Cleans up stale metadata |
| Sparse checkout (cone mode) | 2.25 | Efficient partial checkouts |

tfbreak will require **git 2.5** minimum and warn if advanced features require newer versions.

### Ref Validation Strategy

Before performing expensive operations (clone, worktree), tfbreak will validate refs using `git ls-remote`:

```bash
# Validate ref exists in local repository
git rev-parse --verify <ref>

# Validate ref exists in remote repository (without cloning)
git ls-remote --exit-code <url> <ref>
```

This provides fast failure with clear error messages when refs don't exist.

### Go Implementation Patterns

The `internal/git` package will use `os/exec` with these patterns:

```go
// Capture both stdout and stderr separately for proper error reporting
var stdout, stderr bytes.Buffer
cmd := exec.Command("git", args...)
cmd.Stdout = &stdout
cmd.Stderr = &stderr

err := cmd.Run()
if err != nil {
    // Include stderr in error for debugging
    var exitErr *exec.ExitError
    if errors.As(err, &exitErr) {
        exitErr.Stderr = stderr.Bytes()
    }
    return fmt.Errorf("git %s failed: %w\nstderr: %s", args[0], err, stderr.String())
}
```

Key patterns:
- Always capture stderr separately for error diagnostics
- Pass through environment variables (credentials, SSH config)
- Use `exec.ExitError` to access exit codes
- Create temporary directories with `os.MkdirTemp` and defer cleanup

### Related Documents

- Original specification: `spec/001_IDEA.md` (Section 10, Phase 4 mentions git ref mode)
- ADR-0001: Project inception and technology stack
- ADR-0002: Plugin architecture
- CR-0015: Git infrastructure package
- CR-0016: Local git ref comparison
- CR-0017: Remote repository comparison
