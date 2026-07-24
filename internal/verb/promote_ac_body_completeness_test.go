package verb_test

import (
	"os"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// stripHeading rewrites the file at path, dropping any line starting
// with "### AC-" — simulating a hand-edit that desyncs the body from
// frontmatter acs[] (used to test the missing-heading carve-out).
func stripHeading(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	var out []string
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "### AC-") {
			continue
		}
		out = append(out, line)
	}
	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// promote_ac_body_completeness_test.go pins M-0268/AC-2: a milestone
// with at least one AC whose body subsection carries no non-heading
// prose cannot start (draft -> in_progress). Per G-0216: a milestone
// whose AC is a title-only stub has no real contract for that
// criterion yet.
//
// Unlike AC-1's zero-AC guard, --force does NOT actually let this
// promote land: M-0268/AC-4 (acs-empty-body, error severity) flags
// exactly the resulting state, and Promote's projectionFindings check
// runs unconditionally regardless of force — force relaxes FSM-
// transition legality and this specific verb-time guard, never
// check-time coherence (see TestPromote_ForceStillFailsCoherence for
// the established precedent this follows). So the error message never
// claims a working override; the only path through is writing real
// prose via `aiwf edit-body`.

// TestPromote_EmptyACBodyRefusedAtDraftToInProgress is the headline
// case: a single AC with a heading but no body prose refuses the
// promote and names the composite id.
func TestPromote_EmptyACBodyRefusedAtDraftToInProgress(t *testing.T) {
	t.Parallel()
	r := setupACLessMilestoneOnEpicBranch(t)
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "Does the thing", testActor))

	_, err := verb.Promote(r.ctx, r.tree(), "M-0001", "in_progress", testActor, "", false, verb.PromoteOptions{})
	if err == nil {
		t.Fatal("expected error promoting a milestone with an empty AC body to in_progress; got nil")
	}
	if !strings.Contains(err.Error(), "M-0001/AC-1") {
		t.Errorf("error should name the composite AC id; got %v", err)
	}
	if strings.Contains(err.Error(), "--force") {
		t.Errorf("error must not claim --force overrides this (it doesn't, once AC-4 exists); got %v", err)
	}
	if m := r.tree().ByID("M-0001"); m == nil || m.Status != entity.StatusDraft {
		t.Errorf("refused promote must not mutate status; M-0001 = %+v", m)
	}
}

// TestPromote_EmptyACBodyForceSkipsVerbGuardButCheckStillBlocks pins
// the force/AC-4 interaction directly: --force does skip this specific
// Go-error refusal (the code path changes), but the commit still does
// not land — Promote returns a *Result carrying the acs-empty-body
// finding and a nil Plan, exactly like TestPromote_ForceStillFailsCoherence's
// status-valid case. Mirrors how cliutil.FinishVerb itself handles this
// shape (check findings before ever touching Plan).
func TestPromote_EmptyACBodyForceSkipsVerbGuardButCheckStillBlocks(t *testing.T) {
	t.Parallel()
	r := setupACLessMilestoneOnEpicBranch(t)
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "Does the thing", testActor))

	res, err := verb.Promote(r.ctx, r.tree(), "M-0001", "in_progress", testActor, "starting anyway", true, verb.PromoteOptions{})
	if err != nil {
		t.Fatalf("force should skip the verb-time Go-error refusal, not error again: %v", err)
	}
	if res.Plan != nil {
		t.Fatalf("expected nil Plan (check-time coherence still blocks); got %+v", res.Plan)
	}
	if !check.HasErrors(res.Findings) {
		t.Fatalf("expected an error-severity finding among %+v", res.Findings)
	}
	foundACEmptyBody := false
	for _, f := range res.Findings {
		if f.Code == check.CodeACsEmptyBodyOnStart {
			foundACEmptyBody = true
		}
	}
	if !foundACEmptyBody {
		t.Errorf("expected acs-empty-body among the blocking findings; got %+v", res.Findings)
	}

	// Nothing was committed; status is untouched.
	if m := r.tree().ByID("M-0001"); m == nil || m.Status != entity.StatusDraft {
		t.Errorf("refused promote must not mutate status; M-0001 = %+v", m)
	}
}

// TestPromote_PopulatedACBodyUnaffectedByEmptyBodyGuard is the
// regression companion: an AC whose body carries real prose promotes
// cleanly.
func TestPromote_PopulatedACBodyUnaffectedByEmptyBodyGuard(t *testing.T) {
	t.Parallel()
	r := setupACLessMilestoneOnEpicBranch(t)
	r.must(verb.AddACBatch(r.ctx, r.tree(), "M-0001", []string{"Does the thing"},
		[][]byte{[]byte("Real prose describing the criterion.")}, testActor))

	r.must(verb.Promote(r.ctx, r.tree(), "M-0001", "in_progress", testActor, "", false, verb.PromoteOptions{}))

	m := r.tree().ByID("M-0001")
	if m == nil || m.Status != entity.StatusInProgress {
		t.Fatalf("populated-AC-body milestone should promote cleanly; got %+v", m)
	}
}

// TestPromote_HeadingOnlySubHeadingACBodyStillRefused pins the "non-
// heading prose" wording precisely: a body section containing only a
// sub-heading (no real prose under it) still counts as empty, matching
// the pre-existing entity-body-empty/ac check rule's own leaf-level
// definition (sub-headings are not prose).
func TestPromote_HeadingOnlySubHeadingACBodyStillRefused(t *testing.T) {
	t.Parallel()
	r := setupACLessMilestoneOnEpicBranch(t)
	r.must(verb.AddACBatch(r.ctx, r.tree(), "M-0001", []string{"Does the thing"},
		[][]byte{[]byte("#### Notes")}, testActor))

	_, err := verb.Promote(r.ctx, r.tree(), "M-0001", "in_progress", testActor, "", false, verb.PromoteOptions{})
	if err == nil {
		t.Fatal("expected error promoting a milestone whose AC body is heading-only; got nil")
	}
	if !strings.Contains(err.Error(), "M-0001/AC-1") {
		t.Errorf("error should name the composite AC id; got %v", err)
	}
}

// TestPromote_MissingACHeadingNotTreatedAsEmptyBody pins the
// established precedent from the pre-existing entity-body-empty/ac
// check rule: an AC with NO `### AC-N` heading in the body at all is a
// different problem (acs-body-coherence/missing-heading), not this
// guard's concern — a hand-edited frontmatter/body mismatch should not
// double-fire here.
func TestPromote_MissingACHeadingNotTreatedAsEmptyBody(t *testing.T) {
	t.Parallel()
	r := setupACLessMilestoneOnEpicBranch(t)
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "Does the thing", testActor))

	// Strip the scaffolded heading from the body entirely, simulating a
	// hand-edit that desynced frontmatter acs[] from the body.
	m := r.tree()
	e := m.ByID("M-0001")
	bodyPath := r.root + "/" + e.Path
	stripHeading(t, bodyPath)

	r.must(verb.Promote(r.ctx, r.tree(), "M-0001", "in_progress", testActor, "", false, verb.PromoteOptions{}))

	got := r.tree().ByID("M-0001")
	if got == nil || got.Status != entity.StatusInProgress {
		t.Fatalf("missing-heading AC should not trip the empty-body guard; got %+v", got)
	}
}
