package plugin

import (
	"testing"
)

func TestNewSource_GitHub(t *testing.T) {
	source, err := NewSource("github.com/jokarl/tfbreak-ruleset-azurerm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := source.(*GitHubSource); !ok {
		t.Errorf("expected GitHubSource, got %T", source)
	}

	if source.String() != "github.com/jokarl/tfbreak-ruleset-azurerm" {
		t.Errorf("unexpected String(): %s", source.String())
	}
}

func TestNewSource_GitLab(t *testing.T) {
	source, err := NewSource("gitlab.com/company/tfbreak-ruleset-internal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := source.(*GitLabSource); !ok {
		t.Errorf("expected GitLabSource, got %T", source)
	}
}

func TestNewSource_GitLabSelfHosted(t *testing.T) {
	source, err := NewSource("gitlab.company.com/team/tfbreak-ruleset-private")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := source.(*GitLabSource); !ok {
		t.Errorf("expected GitLabSource, got %T", source)
	}
}

func TestNewSource_HTTP(t *testing.T) {
	source, err := NewSource("https://plugins.example.com/tfbreak-ruleset-custom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := source.(*HTTPSource); !ok {
		t.Errorf("expected HTTPSource, got %T", source)
	}
}

func TestNewSource_File(t *testing.T) {
	source, err := NewSource("file:///var/tfbreak-plugins/tfbreak-ruleset-local")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := source.(*FileSource); !ok {
		t.Errorf("expected FileSource, got %T", source)
	}
}

func TestNewSource_Unknown(t *testing.T) {
	_, err := NewSource("ftp://example.com/plugin")
	if err == nil {
		t.Error("expected error for unknown source type")
	}
}

func TestNewSource_BitbucketNotSupported(t *testing.T) {
	// Bitbucket is not supported yet
	_, err := NewSource("bitbucket.org/company/repo")
	if err == nil {
		t.Error("expected error for unsupported source type")
	}
}

func TestSourceRelease_FindAsset(t *testing.T) {
	release := &SourceRelease{
		Version: "0.1.0",
		Assets: []SourceAsset{
			{Name: "checksums.txt", URL: "https://example.com/checksums.txt"},
			{Name: "plugin_darwin_arm64.zip", URL: "https://example.com/plugin_darwin_arm64.zip"},
			{Name: "plugin_linux_amd64.zip", URL: "https://example.com/plugin_linux_amd64.zip"},
		},
	}

	// Find existing asset
	asset := release.FindAsset("checksums.txt")
	if asset == nil {
		t.Fatal("expected to find checksums.txt")
	}
	if asset.Name != "checksums.txt" {
		t.Errorf("unexpected asset name: %s", asset.Name)
	}

	// Find non-existing asset
	asset = release.FindAsset("not-found.txt")
	if asset != nil {
		t.Error("expected nil for non-existing asset")
	}
}

func TestIsGitLabSource(t *testing.T) {
	tests := []struct {
		source   string
		expected bool
	}{
		{"gitlab.com/org/repo", true},
		{"gitlab.company.com/org/repo", true},
		{"my-gitlab.example.com/org/repo", true},
		{"github.com/org/repo", false},
		{"bitbucket.org/org/repo", false},
		{"example.com/org/repo", false},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			result := isGitLabSource(tt.source)
			if result != tt.expected {
				t.Errorf("isGitLabSource(%q) = %v, expected %v", tt.source, result, tt.expected)
			}
		})
	}
}
