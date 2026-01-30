---
id: "CR-0020"
status: "implemented"
date: 2026-01-30
requestor: jokarl
stakeholders: [jokarl, tfbreak-core maintainers]
priority: "low"
target-version: v0.5.0
implements: ADR-0004
phase: 3
---

# Plugin Source Extensibility (Phase 3)

## Change Summary

Abstract the plugin source interface to support multiple backends beyond GitHub. This enables plugin distribution from GitLab, Bitbucket, self-hosted Git forges, and custom HTTP endpoints while maintaining the same user-facing configuration syntax.

## Motivation and Background

CR-0018 and CR-0019 implement plugin distribution via GitHub releases. While GitHub covers the majority of open-source plugin distribution needs, enterprise users often require:

1. **GitLab**: Many enterprises use GitLab as their primary Git platform
2. **Self-hosted forges**: Gitea, Gogs, or self-hosted GitHub/GitLab instances
3. **Internal registries**: Custom HTTP endpoints for internal plugin distribution
4. **Air-gapped environments**: Local mirrors without internet access

By abstracting the source interface, tfbreak can support these scenarios without changing the user-facing configuration syntax.

## Change Drivers

* Enterprise adoption: GitLab and self-hosted forges are common in enterprises
* Air-gapped environments: Security-sensitive environments need local sources
* Platform flexibility: Avoid locking users into GitHub
* Future-proofing: New platforms can be added without user-facing changes

## Current State

After CR-0018/CR-0019, plugin sources are hard-coded to GitHub:

```go
// plugin/install.go
func (c *InstallConfig) Install(pluginDir string) error {
    // Hard-coded GitHub API calls
    release := githubClient.GetRelease(owner, repo, tag)
    // ...
}
```

Source URL parsing assumes GitHub format:

```go
// github.com/owner/repo -> owner, repo
```

## Proposed Change

Introduce a `Source` interface that abstracts release metadata and asset download:

```go
// plugin/source.go
type Source interface {
    // GetRelease returns metadata for a specific version
    GetRelease(version string) (*Release, error)

    // DownloadAsset downloads a release asset to the writer
    DownloadAsset(asset *Asset, dest io.Writer) error

    // String returns the source identifier for logging
    String() string
}

type Release struct {
    Version string
    Assets  []Asset
}

type Asset struct {
    Name string
    URL  string
    Size int64
}
```

### Source Detection

Sources are detected from the URL pattern:

| Pattern | Source Type |
|---------|-------------|
| `github.com/owner/repo` | GitHub |
| `gitlab.com/owner/repo` | GitLab |
| `*.gitlab.com/owner/repo` | GitLab (self-hosted) |
| `https://example.com/plugins/foo` | HTTP |
| `file:///path/to/plugins` | Local filesystem |

### Configuration Syntax

The user-facing syntax remains unchanged:

```hcl
# GitHub (existing)
plugin "azurerm" {
  source  = "github.com/jokarl/tfbreak-ruleset-azurerm"
  version = "0.1.0"
}

# GitLab
plugin "internal" {
  source  = "gitlab.com/company/tfbreak-ruleset-internal"
  version = "2.0.0"
}

# Self-hosted GitLab
plugin "private" {
  source  = "gitlab.company.com/team/tfbreak-ruleset-private"
  version = "1.0.0"
}

# Custom HTTP endpoint
plugin "custom" {
  source  = "https://plugins.company.com/tfbreak-ruleset-custom"
  version = "3.0.0"
}
```

### GitLab Release Structure

GitLab releases follow a similar pattern to GitHub:

```
https://gitlab.com/api/v4/projects/{project}/releases/{tag}
```

Asset naming convention is the same:

```
tfbreak-ruleset-{name}_{GOOS}_{GOARCH}.zip
checksums.txt
checksums.txt.sig  (optional)
```

### HTTP Source Protocol

For custom HTTP endpoints, a simple protocol:

```
GET {source}/releases/{version}/metadata.json
{
  "version": "3.0.0",
  "assets": [
    {"name": "tfbreak-ruleset-custom_darwin_arm64.zip", "url": "..."},
    {"name": "checksums.txt", "url": "..."}
  ]
}

GET {asset.url}
-> Binary content
```

### Local Filesystem Source

For air-gapped environments:

```
file:///var/tfbreak-plugins/
└── tfbreak-ruleset-internal/
    └── 1.0.0/
        ├── tfbreak-ruleset-internal_darwin_arm64.zip
        ├── tfbreak-ruleset-internal_linux_amd64.zip
        └── checksums.txt
```

## Requirements

### Functional Requirements

1. The system **MUST** support GitHub as a plugin source (existing)
2. The system **MUST** support GitLab as a plugin source
3. The system **MUST** support self-hosted GitLab instances
4. The system **MUST** support custom HTTP endpoints
5. The system **MUST** support local filesystem sources
6. The system **MUST** auto-detect source type from URL pattern
7. The system **MUST** use the same asset naming convention across all sources
8. The system **MUST** support checksum verification for all sources
9. The system **MUST** support signature verification for all sources (if configured)
10. The system **MUST** support authentication tokens for GitLab (`GITLAB_TOKEN`)

### Non-Functional Requirements

1. The system **MUST** maintain backwards compatibility with existing GitHub sources
2. The system **MUST** provide clear errors when source type cannot be detected
3. The system **MUST** handle network errors gracefully for all source types

## Affected Components

* `plugin/source.go` - New file: Source interface and factory
* `plugin/source_github.go` - Refactored GitHub implementation
* `plugin/source_gitlab.go` - New GitLab implementation
* `plugin/source_http.go` - New HTTP implementation
* `plugin/source_file.go` - New filesystem implementation
* `plugin/install.go` - Use Source interface

## Scope Boundaries

### In Scope

* Source interface abstraction
* GitHub source (refactored from existing)
* GitLab source implementation
* HTTP source implementation
* Local filesystem source
* Source auto-detection
* `GITLAB_TOKEN` support

### Out of Scope ("Here, But Not Further")

* Bitbucket support - can be added later following same pattern
* S3/GCS bucket sources - future if demanded
* OCI registry sources - significant complexity
* Plugin search/discovery - users must know exact source
* Source fallback chains - single source per plugin

## Implementation Approach

### Phase 3.1: Source Interface

```go
// plugin/source.go
type Source interface {
    GetRelease(version string) (*Release, error)
    DownloadAsset(asset *Asset, dest io.Writer) error
    String() string
}

func NewSource(sourceURL string) (Source, error) {
    switch {
    case strings.HasPrefix(sourceURL, "github.com/"):
        return NewGitHubSource(sourceURL)
    case strings.Contains(sourceURL, "gitlab"):
        return NewGitLabSource(sourceURL)
    case strings.HasPrefix(sourceURL, "https://"):
        return NewHTTPSource(sourceURL)
    case strings.HasPrefix(sourceURL, "file://"):
        return NewFileSource(sourceURL)
    default:
        return nil, fmt.Errorf("unknown source type: %s", sourceURL)
    }
}
```

### Phase 3.2: Refactor GitHub Source

```go
// plugin/source_github.go
type GitHubSource struct {
    owner  string
    repo   string
    client *GitHubClient
}

func (s *GitHubSource) GetRelease(version string) (*Release, error)
func (s *GitHubSource) DownloadAsset(asset *Asset, dest io.Writer) error
```

### Phase 3.3: GitLab Source

```go
// plugin/source_gitlab.go
type GitLabSource struct {
    host      string
    projectID string
    client    *GitLabClient
}

func NewGitLabSource(sourceURL string) (*GitLabSource, error)
func (s *GitLabSource) GetRelease(version string) (*Release, error)
func (s *GitLabSource) DownloadAsset(asset *Asset, dest io.Writer) error
```

### Phase 3.4: HTTP and File Sources

```go
// plugin/source_http.go
type HTTPSource struct {
    baseURL string
    client  *http.Client
}

// plugin/source_file.go
type FileSource struct {
    basePath string
}
```

## Test Strategy

### Tests to Add

| Test File | Test Name | Description | Inputs | Expected Output |
|-----------|-----------|-------------|--------|-----------------|
| `plugin/source_test.go` | `TestNewSource_GitHub` | Detect GitHub source | `github.com/org/repo` | GitHubSource |
| `plugin/source_test.go` | `TestNewSource_GitLab` | Detect GitLab source | `gitlab.com/org/repo` | GitLabSource |
| `plugin/source_test.go` | `TestNewSource_GitLabSelfHosted` | Detect self-hosted GitLab | `gitlab.company.com/org/repo` | GitLabSource |
| `plugin/source_test.go` | `TestNewSource_HTTP` | Detect HTTP source | `https://example.com/plugin` | HTTPSource |
| `plugin/source_test.go` | `TestNewSource_File` | Detect file source | `file:///path/to/plugins` | FileSource |
| `plugin/source_test.go` | `TestNewSource_Unknown` | Reject unknown source | `ftp://example.com` | Error |
| `plugin/source_gitlab_test.go` | `TestGitLabSource_GetRelease` | Fetch GitLab release | Mock API | Release struct |
| `plugin/source_gitlab_test.go` | `TestGitLabSource_DownloadAsset` | Download GitLab asset | Mock asset | Downloaded bytes |
| `plugin/source_gitlab_test.go` | `TestGitLabSource_WithToken` | Use GITLAB_TOKEN | Token set | Auth header |
| `plugin/source_http_test.go` | `TestHTTPSource_GetRelease` | Fetch HTTP metadata | Mock server | Release struct |
| `plugin/source_file_test.go` | `TestFileSource_GetRelease` | Read local release | Local files | Release struct |

### Tests to Modify

| Test File | Test Name | Current Behavior | New Behavior | Reason for Change |
|-----------|-----------|------------------|--------------|-------------------|
| `plugin/install_test.go` | `TestInstall_*` | Direct GitHub calls | Use Source interface | Abstraction change |

### Tests to Remove

Not applicable.

## Acceptance Criteria

### AC-1: GitLab plugin installation

```gherkin
Given a plugin with source "gitlab.com/company/tfbreak-ruleset-internal"
  And version "1.0.0"
  And the GitLab release exists with correct assets
When the user runs "tfbreak --init"
Then the plugin is downloaded from GitLab
  And checksums are verified
  And the plugin is installed correctly
```

### AC-2: Self-hosted GitLab

```gherkin
Given a plugin with source "gitlab.company.com/team/tfbreak-ruleset-private"
  And GITLAB_TOKEN is set
When the user runs "tfbreak --init"
Then the GitLab API calls use the correct host
  And authentication is included
  And the plugin is installed
```

### AC-3: HTTP source

```gherkin
Given a plugin with source "https://plugins.company.com/tfbreak-ruleset-custom"
  And the HTTP endpoint returns valid metadata.json
When the user runs "tfbreak --init"
Then the plugin is downloaded from the HTTP endpoint
  And checksums are verified
  And the plugin is installed
```

### AC-4: Local filesystem source

```gherkin
Given a plugin with source "file:///var/tfbreak-plugins/tfbreak-ruleset-airgap"
  And the local directory contains the plugin files
When the user runs "tfbreak --init"
Then the plugin is copied from the local filesystem
  And checksums are verified
  And the plugin is installed
```

### AC-5: Unknown source type

```gherkin
Given a plugin with source "ftp://example.com/plugin"
When the user runs "tfbreak --init"
Then an error is displayed: "unknown source type: ftp://example.com/plugin"
  And no download is attempted
```

### AC-6: Backwards compatibility

```gherkin
Given existing .tfbreak.hcl files with GitHub sources
When the user upgrades tfbreak and runs "tfbreak --init"
Then all existing GitHub source configurations continue to work
  And no configuration changes are required
```

## Quality Standards Compliance

### Verification Commands

```bash
go build ./...
golangci-lint run
go test ./...
```

## Risks and Mitigation

### Risk 1: GitLab API differences

**Likelihood:** Medium
**Impact:** Medium
**Mitigation:** Thorough testing against GitLab.com and self-hosted instances. Document known limitations.

### Risk 2: HTTP protocol complexity

**Likelihood:** Low
**Impact:** Low
**Mitigation:** Keep protocol simple. Document specification for plugin hosts.

### Risk 3: Source detection ambiguity

**Likelihood:** Low
**Impact:** Medium
**Mitigation:** Clear detection rules. Allow explicit source type hints if needed.

## Dependencies

* CR-0018: Plugin auto-download baseline (prerequisite)
* CR-0019: Plugin signature verification (for signature support across sources)

## Estimated Effort

| Component | Effort |
|-----------|--------|
| Source interface | 2 hours |
| GitHub refactor | 2 hours |
| GitLab implementation | 4 hours |
| HTTP implementation | 2 hours |
| File implementation | 2 hours |
| Testing | 6 hours |
| Documentation | 2 hours |
| **Total** | **20 hours** |

## Decision Outcome

Chosen approach: "Abstract Source interface with auto-detection", because it provides maximum flexibility while maintaining a simple user-facing configuration syntax.

## Related Items

* ADR-0004: Plugin distribution and installation
* CR-0018: Plugin auto-download baseline
* CR-0019: Plugin signature verification
