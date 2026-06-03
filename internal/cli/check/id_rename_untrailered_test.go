package check

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	clicontract "github.com/23min/aiwf/internal/cli/contract"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// id_rename_untrailered_test.go — M-0160/AC-4 REFACTOR: hint-flow
// pin for the new id-rename-untrailered rule, mirroring the
// M-0106/AC-12 + M-0159/AC-9 patterns at
// internal/cli/check/isolation_escape_test.go.
//
// The unit test at internal/check/id_rename_untrailered_test.go
// asserts the rule emits a finding with the expected message
// substring; this test asserts the hint flows from the hint
// table through `clicontract.ApplyHintsLikeRun` onto the
// finding's Hint field, end-to-end via RunProvenanceCheck.
//
// Without this seam, a future refactor that detached the
// hint-application chain (e.g., the gather no longer flows
// through `Run`'s composition) would silently ship findings
// with empty Hints — readable in CI but missing the
// canonical-resolution suggestion the operator needs.

// TestRunProvenanceCheck_IDRenameUntrailered_FindingCarriesHint
// constructs the canonical untrailered-rename fixture, drives
// RunProvenanceCheck end-to-end, applies hints exactly as the
// outer `Run` orchestrator does, and asserts the resulting
// finding's Hint field is non-empty AND names both the
// canonical resolution (`aiwf reallocate`) AND the sovereign-
// human override (`aiwf acknowledge-illegal`).
func TestRunProvenanceCheck_IDRenameUntrailered_FindingCarriesHint(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ctx := context.Background()

	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if out, err := exec.CommandContext(ctx, "git", "-C", root, "branch", "-M", "main").CombinedOutput(); err != nil {
		t.Fatalf("git branch -M main: %v\n%s", err, out)
	}

	// Trunk-side seed: an id-bearing gap entity at the original
	// slug. The file's content is minimal — the walker only
	// needs the path-shape to recognize G-0001 via
	// entity.PathKind + entity.IDFromPath.
	originalRel := "work/gaps/G-0001-original-slug.md"
	originalAbs := filepath.Join(root, originalRel)
	if err := os.MkdirAll(filepath.Dir(originalAbs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(originalAbs, []byte("---\nid: G-0001\nkind: gap\ntitle: orig\nstatus: open\n---\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := gitops.Add(ctx, root, originalRel); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(ctx, root, "seed: add G-0001", "", nil); err != nil {
		t.Fatal(err)
	}
	// Snapshot a local trunk ref at the post-seed tip. The
	// walker computes merge-base(HEAD, trunk-ref) and reads
	// from this ref to enumerate the range.
	if out, err := exec.CommandContext(ctx, "git", "-C", root, "branch", "trunk-ref").CombinedOutput(); err != nil {
		t.Fatalf("git branch trunk-ref: %v\n%s", err, out)
	}

	// Inline `git mv` rename — the failure mode the rule polices.
	// No aiwf-verb trailer, so the walker emits a record.
	newRel := "work/gaps/G-0001-via-inline-mv.md"
	if out, err := exec.CommandContext(ctx, "git", "-C", root, "mv", originalRel, newRel).CombinedOutput(); err != nil {
		t.Fatalf("git mv: %v\n%s", err, out)
	}
	if err := gitops.Commit(ctx, root, "chore: rename G-0001 slug", "", nil); err != nil {
		t.Fatal(err)
	}

	// Drive RunProvenanceCheck with a tree.Tree carrying the
	// TrunkRef. The rule's wire-up at provenance.go:97 reads
	// this field; an empty TrunkRef short-circuits the walker.
	tr := &tree.Tree{TrunkRef: "refs/heads/trunk-ref"}
	registered := map[string]struct{}{}
	findings, err := RunProvenanceCheck(ctx, root, tr, "", registered, nil)
	if err != nil {
		t.Fatalf("RunProvenanceCheck: %v", err)
	}

	// Apply hints exactly as the outer `Run` orchestrator does
	// (see internal/cli/check/check.go:158, 224). The
	// finding's Hint is empty until ApplyHintsLikeRun populates
	// it from the hint table.
	clicontract.ApplyHintsLikeRun(findings)

	var found *check.Finding
	for i := range findings {
		if findings[i].Code == check.CodeIDRenameUntrailered.ID {
			found = &findings[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("id-rename-untrailered finding not present; envelope:\n%+v", findings)
	}
	if found.Hint == "" {
		t.Fatal("id-rename-untrailered finding has empty Hint after ApplyHintsLikeRun — hint-table-to-finding flow broken")
	}
	// The hint names two paths an operator can take:
	//   - `aiwf reallocate` (canonical resolution)
	//   - `aiwf acknowledge-illegal` (sovereign-human override)
	// Both must surface so the operator's editor / CI summary
	// renders the actionable choice inline. A regression that
	// dropped either path's mention would degrade the chokepoint's
	// AI-discoverability claim.
	if !strings.Contains(found.Hint, "aiwf reallocate") {
		t.Errorf("id-rename-untrailered Hint %q does not name `aiwf reallocate` (canonical resolution)", found.Hint)
	}
	if !strings.Contains(found.Hint, "aiwf acknowledge-illegal") {
		t.Errorf("id-rename-untrailered Hint %q does not name `aiwf acknowledge-illegal` (sovereign-human override)", found.Hint)
	}
}
