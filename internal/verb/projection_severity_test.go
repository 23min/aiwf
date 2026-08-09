package verb_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/severity"
	"github.com/23min/aiwf/internal/verb"
)

// writeRunnerConfig plants an aiwf.yaml in the runner's root and
// commits it, so the tree the verb under test loads is clean and the
// config the projection guard reads is the one this test declares.
func writeRunnerConfig(t *testing.T, r *runner, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(r.root, "aiwf.yaml"), []byte(body), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
	commitFixture(t, r.root, "chore: declare aiwf.yaml")
}

// findingSeverity returns the severity reported for code, and whether
// it was reported at all.
func findingSeverity(fs []check.Finding, code string) (check.Severity, bool) {
	for i := range fs {
		if fs[i].Code == code {
			return fs[i].Severity, true
		}
	}
	return "", false
}

// TestProjectionGuard_TDDStrictRefusesAnUndeclaredMilestone is
// G-0573's defect in the one shape the projection guard can actually
// see. Measured before the shared seam, in a repo with
// `tdd.strict: true`:
//
//	aiwf import seed.yaml    exit 0, commits
//	aiwf check               error milestone-tdd-undeclared, exit 1
//
// The guard projected the post-mutation tree, read the finding at its
// unescalated warning default, and let the write land — so the verb
// reported success for a state the pre-push hook refuses.
//
// `milestone-tdd-undeclared` is the code that exercises this because it
// reads frontmatter the projection carries in memory. Its sibling under
// the same knob, entity-body-empty, reads body bytes off disk, so for a
// file the verb has not written yet the rule returns nothing to
// escalate and no severity policy can reach it. Closing that one needs
// a verb-time scan of the planned-write bytes, the shape add.go already
// runs for body-prose-id — a different fix, outside what this test
// pins.
func TestProjectionGuard_TDDStrictRefusesAnUndeclaredMilestone(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	writeRunnerConfig(t, r, "tdd:\n  strict: true\n")
	src := `version: 1
entities:
  - kind: epic
    id: E-0001
    frontmatter:
      title: "Host epic"
      status: active
    body: |
      ## Goal

      Real prose.

      ## Scope

      Real prose.

      ## Out of scope

      Real prose.
  - kind: milestone
    id: M-0001
    frontmatter:
      title: "Imported milestone declaring no tdd policy"
      status: draft
      parent: E-0001
    body: "## Goal\n\nReal prose.\n"
`
	res, err := verb.Import(r.ctx, r.tree(), loadManifest(t, src), testActor, verb.ImportOptions{})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	got, found := findingSeverity(res.Findings, check.CodeMilestoneTDDUndeclared)
	if !found {
		t.Fatalf("no milestone-tdd-undeclared finding; got %+v", res.Findings)
	}
	if got != check.SeverityError {
		t.Errorf("milestone-tdd-undeclared severity = %q, want %q — the guard read the finding at its config-agnostic default, so `tdd.strict` never reached the write path",
			got, check.SeverityError)
	}
	if len(res.Plans) != 0 {
		t.Errorf("len(Plans) = %d, want 0 — the verb must refuse rather than commit a state `aiwf check` then blocks the push for", len(res.Plans))
	}
}

// TestProjectionGuard_TDDStrictOffLeavesTheImportAlone pins the
// complementary branch: without the knob, the same manifest imports
// cleanly at the rule's warning default. The escalation is the
// operator's declared policy, never the guard's own opinion.
func TestProjectionGuard_TDDStrictOffLeavesTheImportAlone(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	src := `version: 1
entities:
  - kind: epic
    id: E-0001
    frontmatter:
      title: "Host epic"
      status: active
    body: "## Goal\n\nReal prose.\n"
  - kind: milestone
    id: M-0001
    frontmatter:
      title: "Imported milestone declaring no tdd policy"
      status: draft
      parent: E-0001
    body: "## Goal\n\nReal prose.\n"
`
	res, err := verb.Import(r.ctx, r.tree(), loadManifest(t, src), testActor, verb.ImportOptions{})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if check.HasErrors(res.Findings) {
		t.Fatalf("unexpected error-severity findings with tdd.strict unset: %+v", res.Findings)
	}
	if len(res.Plans) == 0 {
		t.Fatal("expected a plan with tdd.strict unset")
	}
}

// TestProjectionGuard_ArchiveSweepCeilingDoesNotBlockATerminalPromote
// pins the exclusion the seam ships with, and is the test that fails if
// archive-sweep-pending is dropped from skipDuringProjection.
//
// The aggregate finding names its own pending count in its message, and
// the guard's diff keys findings by message among other fields. So every
// verb that changes the count re-keys the finding, and a tree already
// past its ceiling makes each subsequent terminal promote look like the
// mutation that introduced the breach. Measured with
// `archive.sweep_threshold: 1`, escalating it at the guard refused
// `aiwf promote G-0002 wontfix` — a legal FSM transition blocked for a
// tree-level drift condition whose remedy is a different verb entirely.
//
// The sweep ceiling is `aiwf check`'s to enforce, which sees the
// condition whichever verb preceded it. This is the same
// unattributability ADR-0041 settles for the cross-branch findings the
// guard already excludes.
func TestProjectionGuard_ArchiveSweepCeilingDoesNotBlockATerminalPromote(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	writeRunnerConfig(t, r, "archive:\n  sweep_threshold: 1\n")
	for _, title := range []string{"First probe gap", "Second probe gap"} {
		r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, title, testActor, verb.AddOptions{
			BodyOverride: bornCompleteFixtureBody(entity.KindGap),
		}))
	}
	// The first promote takes the tree to one pending sweep — at the
	// ceiling, not past it. The second crosses it, which is the case
	// that re-keys the aggregate and would be blamed on this verb.
	r.must(verb.Promote(r.ctx, r.tree(), "G-0001", "wontfix", testActor, "", false, verb.PromoteOptions{}))
	res, err := verb.Promote(r.ctx, r.tree(), "G-0002", "wontfix", testActor, "", false, verb.PromoteOptions{})
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if check.HasErrors(res.Findings) {
		t.Fatalf("the promote that crosses `archive.sweep_threshold` was refused: %+v\nthe ceiling is a tree-level drift control, not this verb's responsibility", res.Findings)
	}
	if res.Plan == nil {
		t.Fatal("expected a plan for a legal terminal transition")
	}

	// Positive control: every assertion above is a negative, so a fixture
	// that had stopped producing the breach at all would pin nothing
	// while still passing. Apply the plan and confirm the gate does raise
	// the aggregate to error — the verb declined to own the condition, it
	// did not make it go away.
	if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: %v", applyErr)
	}
	gate := check.Run(r.tree(), nil)
	severity.Apply(gate, severity.Load(r.root), r.tree())
	got, found := findingSeverity(gate, check.CodeArchiveSweepPending)
	if !found || got != check.SeverityError {
		t.Errorf("after the promote, archive-sweep-pending = (%q, found=%v); want an error at the gate — the fixture must actually breach its ceiling for this test's refusal assertions to mean anything",
			got, found)
	}
}
