package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// TestRunCheck_AreaUnslottedSurfacesViaDispatcher is the dispatcher seam test
// for M-0185/AC-3: `aiwf check` reads aiwf.yaml's areas.coverage_roots, walks
// each declared root's immediate child directories, and surfaces the
// area-unslotted finding for a child claimed by no area's glob, while a sibling
// child claimed by a `**` glob stays silent. Catches the bug class where
// check.AreaCoverage exists and is unit-tested but the CLI Run forgets to call
// it (or fails to source coverage_roots) — the same seam the area-dead-glob /
// area-overlap dispatcher tests guard.
func TestRunCheck_AreaUnslottedSurfacesViaDispatcher(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	// Two project dirs under the coverage root: app-a is claimed by an area
	// glob, orphan is claimed by none.
	for _, d := range []string{"projects/app-a", "projects/orphan"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	patched := string(raw) + "areas:\n" +
		"  members:\n" +
		"    - {name: app-a, paths: [projects/app-a/**]}\n" +
		"  coverage_roots:\n" +
		"    - projects\n"
	if err = os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}

	captured := testutil.CaptureStdout(t, func() {
		// Warning-only check exits 0 (HasErrors counts errors, not warnings);
		// the seam claim under test is that the finding code surfaces in stdout.
		_ = cli.Execute([]string{"check", "--root", root})
	})
	out := string(captured)
	if !strings.Contains(out, "area-unslotted") {
		t.Errorf("expected area-unslotted in check output; got:\n%s", out)
	}
	if !strings.Contains(out, "projects/orphan") {
		t.Errorf("expected the unslotted finding to name projects/orphan; got:\n%s", out)
	}
	if strings.Contains(out, "projects/app-a is claimed by no area") {
		t.Errorf("app-a is claimed by a glob and must not fire unslotted; got:\n%s", out)
	}
}
