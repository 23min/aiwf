package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDiscoverabilityChannels_ExcludeDevelopmentGuidance is M-0333 AC-5:
// neither host entry point nor an on-demand development document — one
// the project router links to — is a discoverability channel, so a
// finding code or config field cannot be made discoverable by a line in
// the guidance. An ordinary document under docs/ still is.
func TestDiscoverabilityChannels_ExcludeDevelopmentGuidance(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	files := map[string]string{
		"internal/cli/root.go":          "package cli\n\nfunc printHelp() {}\n",
		"internal/skills/embedded/x.md": "skill\n",
		"docs/other.md":                 "token-in-docs\n",
		"CLAUDE.md":                     "token-in-claude\n",
		"AGENTS.md":                     "token-in-agents\n",
		".guidance/project.md":          "Read [testing](../docs/dev/testing.md).\n",
		"docs/dev/testing.md":           "token-on-demand\n",
	}
	for rel, content := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	haystack, err := readDiscoverabilityChannels(root)
	if err != nil {
		t.Fatalf("readDiscoverabilityChannels: %v", err)
	}
	if !strings.Contains(string(haystack), "token-in-docs") {
		t.Error("an ordinary docs/ document must stay a channel")
	}
	for _, excluded := range []string{"token-in-claude", "token-in-agents", "token-on-demand"} {
		if strings.Contains(string(haystack), excluded) {
			t.Errorf("the haystack reads %q, but development guidance is not a discoverability channel", excluded)
		}
	}
}

// TestDiscoverabilityChannels_MissingBannerIsAnError pins that a tree
// without the banner source cannot be judged: the haystack would silently
// lose a channel.
func TestDiscoverabilityChannels_MissingBannerIsAnError(t *testing.T) {
	t.Parallel()
	if _, err := readDiscoverabilityChannels(t.TempDir()); err == nil {
		t.Fatal("want an error when the banner source is missing, got nil")
	}
}
