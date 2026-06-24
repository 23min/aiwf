package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// TestRunCheck_AreaUnknownSurfacesViaDispatcher is the dispatcher seam
// test for M-0172/AC-6: `aiwf check` reads aiwf.yaml's areas.members and
// surfaces the area-unknown finding for an entity carrying an undeclared
// area. Catches the bug class where check.AreaUnknown exists and is unit-
// tested but the CLI Run forgets to call it (or passes the wrong config
// field) — the same seam the tests-metrics dispatcher test guards.
func TestRunCheck_AreaUnknownSurfacesViaDispatcher(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--title", "Leak", "--actor", "human/test", "--root", root)

	// Declare an areas block in aiwf.yaml (the single source of truth for
	// the member set).
	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	patched := string(raw) + "areas:\n  members:\n    - platform\n    - billing\n"
	if err = os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}

	// Hand-edit the gap to carry an undeclared area (a typo of "platform")
	// — exactly the drift a creation-time flag alone can't catch.
	matches, err := filepath.Glob(filepath.Join(root, "work", "gaps", "G-0001-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("locate gap file: matches=%v err=%v", matches, err)
	}
	gapRaw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read gap: %v", err)
	}
	gapPatched := strings.Replace(string(gapRaw), "status: open\n", "status: open\narea: platfrm\n", 1)
	if gapPatched == string(gapRaw) {
		t.Fatalf("failed to inject area into gap frontmatter:\n%s", gapRaw)
	}
	if err = os.WriteFile(matches[0], []byte(gapPatched), 0o644); err != nil {
		t.Fatalf("write gap: %v", err)
	}

	captured := testutil.CaptureStdout(t, func() {
		// rc=1 expected: the warning fires (check exits 1 on any findings).
		_ = cli.Execute([]string{"check", "--root", root})
	})
	if !strings.Contains(string(captured), "area-unknown") {
		t.Errorf("expected area-unknown in check output; got:\n%s", captured)
	}
}
