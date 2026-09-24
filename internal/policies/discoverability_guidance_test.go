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

// TestEngineeringPrinciples_NameNoInstructionFileChannel is the principle
// half of M-0333 AC-5, an absence assertion D-0091 permits: the
// AI-discoverability principle does not offer CLAUDE.md ("this file") as
// a channel, since the policies no longer read it. The section is
// asserted to exist first, so the absence cannot pass vacuously.
func TestEngineeringPrinciples_NameNoInstructionFileChannel(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	section := markdownSection(string(data), "## Engineering principles")
	if section == "" {
		t.Fatal(`CLAUDE.md has no "## Engineering principles" section; the absence below would be vacuous`)
	}
	if strings.Contains(section, "this file") {
		t.Error(`§"Engineering principles" still names "this file" as a discoverability channel`)
	}
}
