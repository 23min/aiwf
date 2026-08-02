package verb_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// narrowWidthRunner commits a gap carrying a legacy narrow id, which is
// what gives rewidth a rename to plan.
func narrowWidthRunner(t *testing.T) (r *runner, legacyPath string) {
	t.Helper()
	r = newGapRunner(t)
	legacy := filepath.Join("work", "gaps", "G-02-narrow.md")
	body := "---\nid: G-02\ntitle: Narrow\nstatus: open\n---\n" +
		"## What's missing\n\nFixture prose.\n\n## Why it matters\n\nFixture prose.\n"
	if err := os.WriteFile(filepath.Join(r.root, legacy), []byte(body), 0o600); err != nil {
		t.Fatalf("writing the legacy-width gap: %v", err)
	}
	commitFixture(t, r.root, "fixture: a legacy-width id")
	return r, filepath.ToSlash(legacy)
}

// TestRewidth_DirtyRenameTarget_RefusedAtTheWriteSeam is the pin the
// recorded call for rewidth rests on.
//
// rewidth is out of the claim-side guard because a masked rewrite costs
// nothing — its body scan is independent of the rename set, so the next
// run re-emits whatever a working-copy edit hid. What must not be true is
// that a hand-edit rides into the rename commit, and that direction is
// caught where the write happens: the file is the source of a move, so
// the plan names it.
//
// Without this, nothing distinguishes "rewidth is deliberately unguarded"
// from "rewidth is unguarded and nobody checked" — a guard installed on
// rewidth breaks no test, which is the shape of an unpinned decision.
func TestRewidth_DirtyRenameTarget_RefusedAtTheWriteSeam(t *testing.T) {
	t.Parallel()
	r, legacy := narrowWidthRunner(t)

	// Hand-edit the very file the rename would carry.
	if err := os.WriteFile(filepath.Join(r.root, filepath.FromSlash(legacy)),
		[]byte("---\nid: G-02\ntitle: Narrow\nstatus: open\n---\n"+
			"## What's missing\n\nUNBLESSED EDIT.\n\n## Why it matters\n\nFixture prose.\n"), 0o600); err != nil {
		t.Fatalf("dirtying the legacy-width gap: %v", err)
	}
	before := headSHA(t, r.root)

	res, err := verb.Rewidth(r.ctx, r.root, testActor)
	if err != nil {
		t.Fatalf("Rewidth: %v", err)
	}
	if res.Plan == nil {
		t.Fatal("rewidth produced no plan; the narrow id should still be selected")
	}
	_, applyErr := verb.Apply(r.ctx, r.root, res.Plan)
	if applyErr == nil {
		t.Fatal("Apply succeeded; rewidth carried an edit no verb computed into its rename commit")
	}
	var conflictErr *verb.UncommittedConflictError
	if !errors.As(applyErr, &conflictErr) {
		t.Fatalf("error is not a *verb.UncommittedConflictError: %v", applyErr)
	}
	if !strings.Contains(applyErr.Error(), legacy) {
		t.Errorf("error does not name the blocking path %q:\n%v", legacy, applyErr)
	}
	if after := headSHA(t, r.root); after != before {
		t.Errorf("HEAD advanced to %s on a refusal", after)
	}
}

// TestRewidth_MaskedRewrite_IsReEmittedOnTheNextRun is the other half of
// the same call: the cost of leaving rewidth unguarded is a rewrite
// deferred, not a rewrite lost.
//
// planRewidthRewrites rescans every active markdown independently of the
// rename set, so once the working copy stops hiding the id, the next run
// emits what the masked one omitted. That is what separates rewidth from
// archive, whose skipped link rewrite could never be re-emitted because
// the archived target leaves the scan for good.
func TestRewidth_MaskedRewrite_IsReEmittedOnTheNextRun(t *testing.T) {
	t.Parallel()
	r, legacy := narrowWidthRunner(t)

	due, err := verb.Rewidth(r.ctx, r.root, testActor)
	if err != nil {
		t.Fatalf("Rewidth on a clean tree: %v", err)
	}
	if due.NoOp {
		t.Fatalf("fixture is wrong — no rewrite was due: %+v", due)
	}

	// Hide the narrow id from the scan without committing.
	abs := filepath.Join(r.root, filepath.FromSlash(legacy))
	if rmErr := os.Remove(abs); rmErr != nil {
		t.Fatalf("removing the legacy-width gap: %v", rmErr)
	}
	masked, err := verb.Rewidth(r.ctx, r.root, testActor)
	if err != nil {
		t.Fatalf("Rewidth over a masked rewrite: %v", err)
	}
	if !masked.NoOp {
		t.Errorf("rewidth planned a rewrite for an id the scan cannot see: %+v", masked)
	}

	// Restore the working copy; the rewrite returns.
	body := "---\nid: G-02\ntitle: Narrow\nstatus: open\n---\n" +
		"## What's missing\n\nFixture prose.\n\n## Why it matters\n\nFixture prose.\n"
	if wErr := os.WriteFile(abs, []byte(body), 0o600); wErr != nil {
		t.Fatalf("restoring the legacy-width gap: %v", wErr)
	}
	again, err := verb.Rewidth(r.ctx, r.root, testActor)
	if err != nil {
		t.Fatalf("Rewidth after restoring: %v", err)
	}
	if again.NoOp {
		t.Error("the masked rewrite was not re-emitted; the exemption rests on it being recoverable")
	}
}

// TestSweepNoOps_WriteNothing pins the premise both sweeps' recorded
// calls rest on: a converging sweep writes nothing. If either could
// commit while reporting nothing to do, "the converging path is harmless"
// would stop being an argument.
func TestSweepNoOps_WriteNothing(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		call func(r *runner) (*verb.Result, error)
	}{
		{"archive", func(r *runner) (*verb.Result, error) {
			return verb.Archive(r.ctx, r.root, testActor, "")
		}},
		{"rewidth", func(r *runner) (*verb.Result, error) {
			return verb.Rewidth(r.ctx, r.root, testActor)
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newRunner(t)
			r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Live gap", testActor,
				verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
			before := headSHA(t, r.root)

			res, err := tc.call(r)
			if err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if !res.NoOp {
				t.Fatalf("%s did not converge on a tree with nothing to sweep: %+v", tc.name, res)
			}
			if res.Plan != nil {
				t.Errorf("%s converged with a plan attached: %+v", tc.name, res.Plan)
			}
			if after := headSHA(t, r.root); after != before {
				t.Errorf("%s advanced HEAD to %s while converging", tc.name, after)
			}
		})
	}
}
