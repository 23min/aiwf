package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupPathAreaRepo initializes a repo whose areas declare `paths:` globs (the
// object form) — the oracle `aiwf add --path-hint` derivation (M-0182) reads.
func setupPathAreaRepo(t *testing.T) string {
	t.Helper()
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	patched := string(raw) +
		"areas:\n" +
		"  members:\n" +
		"    - {name: platform, paths: [projects/platform/**]}\n" +
		"    - {name: billing, paths: [projects/billing/**]}\n"
	if err := os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
	return root
}

// TestRunAdd_PathHintDerivesArea pins M-0182/AC-4: with --area omitted, a
// --path-hint falling under exactly one declared area's paths derives that
// area into the created entity's frontmatter; a hint matching no area leaves
// the entity untagged. Driven through the real dispatcher so the
// deriveAreaFromHint seam (not just areamatch.Derive) is exercised.
func TestRunAdd_PathHintDerivesArea(t *testing.T) {
	t.Run("single unambiguous hint derives area", func(t *testing.T) {
		root := setupPathAreaRepo(t)
		mustRun(t, "add", "epic", "--title", "Platform work",
			"--path-hint", "projects/platform/auth/login.go",
			"--actor", "human/test", "--root", root)
		fm := frontmatterOf(readOne(t, root, "work/epics/E-*/epic.md"))
		if !strings.Contains(fm, "area: platform") {
			t.Errorf("epic frontmatter missing derived `area: platform`:\n%s", fm)
		}
	})

	t.Run("hint matching no area leaves entity untagged", func(t *testing.T) {
		root := setupPathAreaRepo(t)
		mustRun(t, "add", "epic", "--title", "Orphan work",
			"--path-hint", "services/unmapped/x.go",
			"--actor", "human/test", "--root", root)
		fm := frontmatterOf(readOne(t, root, "work/epics/E-*/epic.md"))
		if strings.Contains(fm, "area:") {
			t.Errorf("expected untagged epic for a non-matching hint, got area:\n%s", fm)
		}
	})
}
