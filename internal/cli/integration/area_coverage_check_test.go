package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
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

// TestRunCheck_AreaUnslottedEscalatesUnderRequired pins M-0185/AC-5 end-to-end:
// with areas.required: true, area-unslotted surfaces at ERROR severity so
// `aiwf check` exits ExitFindings — proving the ApplyAreaRequiredStrict
// escalation seam actually applies in production, mirroring the required-true
// coverage the area-required / dead-glob / overlap checks carry. The repo is
// clean (no entities), so area-unslotted on the unclaimed child is the sole
// error rather than being masked by an unrelated finding.
func TestRunCheck_AreaUnslottedEscalatesUnderRequired(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

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
		"  required: true\n" +
		"  members:\n" +
		"    - {name: app-a, paths: [projects/app-a/**]}\n" +
		"  coverage_roots:\n" +
		"    - projects\n"
	if err = os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}

	rc, stdout, _ := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"check", "--root", root})
	})
	if rc != cliutil.ExitFindings {
		t.Errorf("rc = %d, want ExitFindings (%d) — area-unslotted must escalate to error under areas.required", rc, cliutil.ExitFindings)
	}
	if !strings.Contains(stdout, "area-unslotted") {
		t.Fatalf("expected area-unslotted in output; got:\n%s", stdout)
	}
	// The finding must render at error severity (not warning) on the unclaimed
	// child. The clean repo guarantees area-unslotted is the only finding.
	if !strings.Contains(stdout, "error") || strings.Contains(stdout, "warning area-unslotted") {
		t.Errorf("expected area-unslotted at ERROR severity under required; got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "projects/orphan") {
		t.Errorf("expected the finding to name projects/orphan; got:\n%s", stdout)
	}
}
