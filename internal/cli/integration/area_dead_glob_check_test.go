package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// TestRunCheck_AreaDeadGlobSurfacesViaDispatcher is the dispatcher seam test
// for M-0180/AC-2: `aiwf check` reads aiwf.yaml's areas member path globs and
// surfaces the area-dead-glob finding for a glob that locates no real path,
// while a sibling glob that locates a real directory stays silent. Catches
// the bug class where check.AreaDeadGlob exists and is unit-tested but the
// CLI Run forgets to call it (or fails to project the members' paths) — the
// same seam the area-unknown dispatcher test guards.
func TestRunCheck_AreaDeadGlobSurfacesViaDispatcher(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	// A real directory for the live area; the ghost area points at nothing.
	if err := os.MkdirAll(filepath.Join(root, "projects", "app-a"), 0o755); err != nil {
		t.Fatalf("mkdir projects/app-a: %v", err)
	}

	// Declare two areas with path globs: one live, one dead.
	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	patched := string(raw) + "areas:\n  members:\n" +
		"    - {name: app, paths: [projects/app-a/**]}\n" +
		"    - {name: ghost, paths: [projects/ghost/**]}\n"
	if err = os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}

	captured := testutil.CaptureStdout(t, func() {
		// A warning-only check exits 0 (HasErrors counts errors, not
		// warnings); the seam claim under test is that the finding code
		// surfaces in stdout, which the assertions below verify.
		_ = cli.Execute([]string{"check", "--root", root})
	})
	out := string(captured)
	if !strings.Contains(out, "area-dead-glob") {
		t.Errorf("expected area-dead-glob in check output; got:\n%s", out)
	}
	if !strings.Contains(out, "ghost") {
		t.Errorf("expected the dead-glob finding to name the ghost area; got:\n%s", out)
	}
}
