package verb_test

import (
	"os"
	"strings"
	"testing"

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
// prose cannot start (draft -> in_progress) without --force. Per
// G-0216: a milestone whose AC is a title-only stub has no real
// contract for that criterion yet.

// TestPromote_EmptyACBodyRefusedAtDraftToInProgress is the headline
// case: a single AC with a heading but no body prose refuses the
// promote and names the composite id.
func TestPromote_EmptyACBodyRefusedAtDraftToInProgress(t *testing.T) {
	t.Parallel()
	r := setupACLessMilestoneOnEpicBranch(t)
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "Does the thing", testActor, nil))

	_, err := verb.Promote(r.ctx, r.tree(), "M-0001", "in_progress", testActor, "", false, verb.PromoteOptions{})
	if err == nil {
		t.Fatal("expected error promoting a milestone with an empty AC body to in_progress; got nil")
	}
	if !strings.Contains(err.Error(), "M-0001/AC-1") {
		t.Errorf("error should name the composite AC id; got %v", err)
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error should point at --force as the override; got %v", err)
	}
	if m := r.tree().ByID("M-0001"); m == nil || m.Status != entity.StatusDraft {
		t.Errorf("refused promote must not mutate status; M-0001 = %+v", m)
	}
}

// TestPromote_EmptyACBodyForceOverridesRefusal mirrors AC-1's own
// --force stance: this is a soft precondition, not an unconditional
// structural guard.
func TestPromote_EmptyACBodyForceOverridesRefusal(t *testing.T) {
	t.Parallel()
	r := setupACLessMilestoneOnEpicBranch(t)
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "Does the thing", testActor, nil))

	r.must(verb.Promote(r.ctx, r.tree(), "M-0001", "in_progress", testActor, "starting anyway", true, verb.PromoteOptions{}))

	m := r.tree().ByID("M-0001")
	if m == nil || m.Status != entity.StatusInProgress {
		t.Fatalf("force-promote should have landed in_progress; got %+v", m)
	}
}

// TestPromote_PopulatedACBodyUnaffectedByEmptyBodyGuard is the
// regression companion: an AC whose body carries real prose promotes
// cleanly.
func TestPromote_PopulatedACBodyUnaffectedByEmptyBodyGuard(t *testing.T) {
	t.Parallel()
	r := setupACLessMilestoneOnEpicBranch(t)
	r.must(verb.AddACBatch(r.ctx, r.tree(), "M-0001", []string{"Does the thing"},
		[][]byte{[]byte("Real prose describing the criterion.")}, testActor, nil))

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
		[][]byte{[]byte("#### Notes")}, testActor, nil))

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
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "Does the thing", testActor, nil))

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
