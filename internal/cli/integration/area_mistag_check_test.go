package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// TestRunCheck_AreaMistagSurfacesViaDispatcher is the dispatcher seam test for
// M-0181/AC-2: `aiwf check` gathers an entity's commits via the aiwf-entity
// trailer (GatherEntityPaths) and surfaces area-mistag when the entity's
// area-claimed work landed entirely in a foreign area's path territory. Catches
// the bug class where check.AreaMistag and GatherEntityPaths exist and are
// unit-tested but the CLI Run forgets to gather + compose them.
func TestRunCheck_AreaMistagSurfacesViaDispatcher(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	// Both area directories exist on disk so neither glob reads as dead;
	// billing is where the foreign work lands.
	for _, dir := range []string{"projects/app-a", "projects/billing"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "projects", "app-a", "keep.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatalf("write keep: %v", err)
	}

	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	patched := string(raw) + "areas:\n  members:\n" +
		"    - {name: app-a, paths: [projects/app-a/**]}\n" +
		"    - {name: billing, paths: [projects/billing/**]}\n"
	if err = os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}

	// A gap tagged app-a, then a commit trailered to it that touches ONLY
	// billing — work that landed entirely in a foreign area.
	mustRun(t, "add", "gap", "--root", root, "--actor", "human/test", "--area", "app-a", "--title", "login timeout fix")
	if err = os.WriteFile(filepath.Join(root, "projects", "billing", "invoice.go"), []byte("package billing\n"), 0o644); err != nil {
		t.Fatalf("write invoice: %v", err)
	}
	if err = osExec(t, root, "git", "add", "projects/billing/invoice.go"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err = osExec(t, root, "git", "commit", "-q", "-m", "billing work", "--trailer", "aiwf-entity: G-0001"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	captured := testutil.CaptureStdout(t, func() {
		// A warning-only check exits 0 (HasErrors counts errors, not
		// warnings); the seam claim under test is that the code surfaces.
		_ = cli.Execute([]string{"check", "--root", root})
	})
	out := string(captured)
	if !strings.Contains(out, "area-mistag") {
		t.Errorf("expected area-mistag in check output; got:\n%s", out)
	}
	if !strings.Contains(out, "G-0001") {
		t.Errorf("expected the finding to name the entity G-0001; got:\n%s", out)
	}
	if !strings.Contains(out, "billing") {
		t.Errorf("expected the finding to name the foreign billing area; got:\n%s", out)
	}
}

// TestRunCheck_AreaMistag_InertWhenNoAreaDeclaresPaths pins M-0181/AC-4 at the
// seam: with areas declared but NONE carrying `paths:` (label-only / legacy
// string form), the AnyAreaHasPaths gate skips the gather entirely, so
// area-mistag can never surface — even for an entity whose commits land in
// another area's would-be territory. The unit complement is
// TestAreaMistag_NoFinding ("no area declares paths").
func TestRunCheck_AreaMistag_InertWhenNoAreaDeclaresPaths(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	patched := string(raw) + "areas:\n  members:\n    - app-a\n    - billing\n" // label-only, no paths
	if err = os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}

	mustRun(t, "add", "gap", "--root", root, "--actor", "human/test", "--area", "app-a", "--title", "login fix")
	if err = os.MkdirAll(filepath.Join(root, "projects", "billing"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err = os.WriteFile(filepath.Join(root, "projects", "billing", "z.go"), []byte("package billing\n"), 0o644); err != nil {
		t.Fatalf("write z.go: %v", err)
	}
	if err = osExec(t, root, "git", "add", "projects/billing/z.go"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err = osExec(t, root, "git", "commit", "-q", "-m", "billing work", "--trailer", "aiwf-entity: G-0001"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	captured := testutil.CaptureStdout(t, func() {
		_ = cli.Execute([]string{"check", "--root", root})
	})
	if strings.Contains(string(captured), "area-mistag") {
		t.Errorf("area-mistag must be inert when no area declares paths; got:\n%s", string(captured))
	}
}

// TestRunCheck_AreaMistag_SkipsGlobalTaggedEntity pins M-0181/AC-4 at the seam:
// an entity tagged the reserved `global` sentinel is inherently cross-cutting
// (ADR-0021), so even with paths declared and its commits landing entirely in a
// foreign area, area-mistag never fires for it. The unit complement is
// TestAreaMistag_NoFinding ("global sentinel is inherently cross-cutting").
func TestRunCheck_AreaMistag_SkipsGlobalTaggedEntity(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	for _, dir := range []string{"projects/app-a", "projects/billing"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	patched := string(raw) + "areas:\n  members:\n" +
		"    - {name: app-a, paths: [projects/app-a/**]}\n" +
		"    - {name: billing, paths: [projects/billing/**]}\n"
	if err = os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}

	mustRun(t, "add", "gap", "--root", root, "--actor", "human/test", "--area", "global", "--title", "shared migration")
	if err = os.WriteFile(filepath.Join(root, "projects", "billing", "shared.go"), []byte("package billing\n"), 0o644); err != nil {
		t.Fatalf("write shared.go: %v", err)
	}
	if err = osExec(t, root, "git", "add", "projects/billing/shared.go"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err = osExec(t, root, "git", "commit", "-q", "-m", "shared work", "--trailer", "aiwf-entity: G-0001"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	captured := testutil.CaptureStdout(t, func() {
		_ = cli.Execute([]string{"check", "--root", root})
	})
	if strings.Contains(string(captured), "area-mistag") {
		t.Errorf("global-tagged entity must never fire area-mistag; got:\n%s", string(captured))
	}
}
