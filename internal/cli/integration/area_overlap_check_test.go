package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// TestRunCheck_AreaOverlapSurfacesViaDispatcher is the dispatcher seam test
// for M-0180/AC-3: `aiwf check` reads aiwf.yaml's areas member path globs and
// surfaces the area-overlap finding when two areas claim a shared directory,
// while disjoint areas stay silent. Catches the bug class where
// check.AreaOverlap exists and is unit-tested but the CLI Run forgets to call
// it — the same seam the dead-glob dispatcher test guards.
func TestRunCheck_AreaOverlapSurfacesViaDispatcher(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	// A real shared directory both areas will claim.
	if err := os.MkdirAll(filepath.Join(root, "projects", "shared"), 0o755); err != nil {
		t.Fatalf("mkdir projects/shared: %v", err)
	}

	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	patched := string(raw) + "areas:\n  members:\n" +
		"    - {name: left, paths: [projects/shared/**]}\n" +
		"    - {name: right, paths: [projects/shared/**]}\n"
	if err = os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}

	captured := testutil.CaptureStdout(t, func() {
		// rc=1 expected: the warning fires (check exits 1 on any findings).
		_ = cli.Execute([]string{"check", "--root", root})
	})
	out := string(captured)
	if !strings.Contains(out, "area-overlap") {
		t.Errorf("expected area-overlap in check output; got:\n%s", out)
	}
	for _, want := range []string{"left", "right"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected the overlap finding to name area %q; got:\n%s", want, out)
		}
	}
}
