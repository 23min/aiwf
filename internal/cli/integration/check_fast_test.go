package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// M-0193 AC-1: `aiwf check --fast` is the content-only check mode. It
// loads the tree without the trunk read and runs the in-memory content
// rules (check.Run + the cheap config tree rules), skipping the
// trunk-collision / provenance / FSM-history / metrics layer that makes
// a full `aiwf check` seconds-to-minutes scale on a large tree.
//
// The two halves of the AC:
//   - --fast catches content findings that --shape-only is blind to
//     (refs-resolve here), and a clean tree exits 0.
//   - --fast is a strict subset of the full check: a git-history-layer
//     finding (provenance-untrailered-entity-commit) the full check
//     emits is absent under --fast.

// initFastFixture inits an aiwf repo, commits a clean base, then writes
// a shape-valid gap whose addressed_by points at a non-existent
// milestone (a refs-resolve content finding) via a plain `git commit`
// with no aiwf trailers (a provenance-untrailered finding). Returns the
// repo root and the base SHA to pass as --since for the provenance
// audit.
func initFastFixture(t *testing.T) (root, baseSHA string) {
	t.Helper()
	root = setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	// Clean, trailer-free base commit so --since has a reachable ref
	// regardless of whether init already committed. The base touches
	// no entity file, so the provenance audit does not flag it.
	if _, err := testutil.RunGit(root, "add", "-A"); err != nil {
		t.Fatalf("git add base: %v", err)
	}
	if _, err := testutil.RunGit(root, "commit", "-m", "base", "--allow-empty"); err != nil {
		t.Fatalf("git commit base: %v", err)
	}
	base, err := testutil.RunGit(root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse base: %v", err)
	}
	baseSHA = strings.TrimSpace(base)

	// Shape-valid gap with an unresolved addressed_by ref. Written
	// directly + committed without aiwf trailers so the same commit
	// trips both refs-resolve (on-disk state) and the provenance
	// untrailered-entity audit (the commit shape).
	gapDir := filepath.Join(root, "work", "gaps")
	if err := os.MkdirAll(gapDir, 0o755); err != nil {
		t.Fatal(err)
	}
	gap := "---\nid: G-0001\ntitle: ref fixture\nstatus: addressed\naddressed_by:\n    - M-9999\n---\nbody\n"
	if err := os.WriteFile(filepath.Join(gapDir, "G-0001-ref-fixture.md"), []byte(gap), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := testutil.RunGit(root, "add", "-A"); err != nil {
		t.Fatalf("git add gap: %v", err)
	}
	if _, err := testutil.RunGit(root, "commit", "-m", "plain untrailered entity edit"); err != nil {
		t.Fatalf("git commit gap: %v", err)
	}
	return root, baseSHA
}

// TestRun_CheckFast_CatchesContentFindingShapeOnlyMisses pins the first
// half of AC-1: --shape-only is blind to a content finding that --fast
// (and the full check) report.
func TestRun_CheckFast_CatchesContentFindingShapeOnlyMisses(t *testing.T) {
	root, base := initFastFixture(t)

	// --shape-only: clean exit, blind to the bad ref.
	shapeRC, shapeOut, _ := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"check", "--shape-only", "--root", root})
	})
	if shapeRC != cliutil.ExitOK {
		t.Errorf("shape-only rc = %d, want %d (blind to content findings)", shapeRC, cliutil.ExitOK)
	}
	if strings.Contains(shapeOut, check.CodeRefsResolve) {
		t.Errorf("shape-only should not report %q:\n%s", check.CodeRefsResolve, shapeOut)
	}

	// --fast: reports the refs-resolve error and exits with findings.
	fastRC, fastOut, _ := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"check", "--fast", "--root", root})
	})
	if fastRC != cliutil.ExitFindings {
		t.Errorf("fast rc = %d, want %d (refs-resolve is an error)", fastRC, cliutil.ExitFindings)
	}
	if !strings.Contains(fastOut, check.CodeRefsResolve) {
		t.Errorf("fast should report %q:\n%s", check.CodeRefsResolve, fastOut)
	}

	// full check: also reports refs-resolve.
	fullRC, fullOut, _ := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"check", "--root", root, "--since", base})
	})
	if fullRC != cliutil.ExitFindings {
		t.Errorf("full rc = %d, want %d", fullRC, cliutil.ExitFindings)
	}
	if !strings.Contains(fullOut, check.CodeRefsResolve) {
		t.Errorf("full check should report %q:\n%s", check.CodeRefsResolve, fullOut)
	}
}

// TestRun_CheckFast_SkipsGitHistoryLayer pins the second half of AC-1:
// a git-history-layer finding the full check emits
// (provenance-untrailered-entity-commit) is absent under --fast, proving
// --fast is a strict subset that skips the expensive provenance walk.
func TestRun_CheckFast_SkipsGitHistoryLayer(t *testing.T) {
	root, base := initFastFixture(t)

	_, fullOut, _ := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"check", "--root", root, "--since", base})
	})
	if !strings.Contains(fullOut, check.CodeProvenanceUntrailedEntityCommit) {
		t.Fatalf("full check should emit %q (scope-proof precondition):\n%s",
			check.CodeProvenanceUntrailedEntityCommit, fullOut)
	}

	fastRC, fastOut, _ := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"check", "--fast", "--root", root, "--since", base})
	})
	// --fast must actually run the content rules (not error out): it
	// reports the refs-resolve error here. Asserting this keeps the
	// scope check below non-vacuous — an absent or broken --fast would
	// trivially "not contain" the provenance code for the wrong reason.
	if fastRC != cliutil.ExitFindings {
		t.Fatalf("fast rc = %d, want %d (must run the content rules)", fastRC, cliutil.ExitFindings)
	}
	if !strings.Contains(fastOut, check.CodeRefsResolve) {
		t.Fatalf("fast should report %q:\n%s", check.CodeRefsResolve, fastOut)
	}
	if strings.Contains(fastOut, check.CodeProvenanceUntrailedEntityCommit) {
		t.Errorf("--fast must skip the provenance/git-history layer; got %q:\n%s",
			check.CodeProvenanceUntrailedEntityCommit, fastOut)
	}
}

// TestRun_CheckFast_CleanTreeExitsOK: a freshly-initialized tree with no
// content findings exits 0 under --fast.
func TestRun_CheckFast_CleanTreeExitsOK(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	rc, out, _ := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"check", "--fast", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Errorf("fast on clean tree rc = %d, want %d:\n%s", rc, cliutil.ExitOK, out)
	}
}

// TestRun_CheckFast_WarningsOnlyExitsOK pins the benign-warning linchpin of the
// health glyph: a tree with only warnings (no errors) exits 0, so the statusline
// never lights ⚠ on the repo's always-present advisory warnings.
func TestRun_CheckFast_WarningsOnlyExitsOK(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	// A gap marked addressed with no resolver: gap-addressed-has-resolver and
	// terminal-entity-not-archived are warnings, not errors — and with no
	// addressed_by there is no refs-resolve error.
	gapDir := filepath.Join(root, "work", "gaps")
	if err := os.MkdirAll(gapDir, 0o755); err != nil {
		t.Fatal(err)
	}
	gap := "---\nid: G-0001\ntitle: warn only\nstatus: addressed\n---\nbody\n"
	if err := os.WriteFile(filepath.Join(gapDir, "G-0001-warn-only.md"), []byte(gap), 0o644); err != nil {
		t.Fatal(err)
	}

	rc, out, _ := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"check", "--fast", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Errorf("warnings-only tree must exit %d (no errors), got %d:\n%s", cliutil.ExitOK, rc, out)
	}
	// Sanity: there ARE warnings, so this is not vacuously a clean tree.
	if !strings.Contains(out, "warning") {
		t.Errorf("expected at least one warning in output (else the test is vacuous):\n%s", out)
	}
}

// TestRun_CheckFast_JSONEnvelope: `--fast --format=json` emits the standard
// envelope carrying the content findings, with metadata.fast set. Parsed
// structurally rather than substring-matched.
func TestRun_CheckFast_JSONEnvelope(t *testing.T) {
	root, _ := initFastFixture(t)

	_, out, _ := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"check", "--fast", "--format=json", "--root", root})
	})
	var env struct {
		Status   string `json:"status"`
		Findings []struct {
			Code     string `json:"code"`
			Severity string `json:"severity"`
		} `json:"findings"`
		Metadata map[string]any `json:"metadata"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("--fast --format=json must emit a parseable envelope: %v\n out: %s", err, out)
	}
	if env.Status != "findings" {
		t.Errorf("envelope status = %q, want \"findings\"", env.Status)
	}
	gotRefErr := false
	for _, f := range env.Findings {
		if f.Code == check.CodeRefsResolve && f.Severity == "error" {
			gotRefErr = true
		}
	}
	if !gotRefErr {
		t.Errorf("envelope findings must include a %q error\n out: %s", check.CodeRefsResolve, out)
	}
	if env.Metadata["fast"] != true {
		t.Errorf("envelope metadata.fast = %v, want true", env.Metadata["fast"])
	}
}
