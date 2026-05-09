package verb_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/check"
	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/gitops"
	"github.com/23min/ai-workflow-v2/internal/verb"
)

// TestEditBody_Bless_RoundTrip is the M-060/AC-1 closure: with no
// `body` argument (nil), edit-body reads the working-copy edit and
// commits it under the standard edit-body trailers, leaving
// frontmatter untouched.
func TestEditBody_Bless_RoundTrip(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))

	// Simulate the user opening the file in $EDITOR and rewriting
	// the body in place.
	epicPath := filepath.Join(r.root, "work", "epics", "E-0001-foundations", "epic.md")
	original, err := os.ReadFile(epicPath)
	if err != nil {
		t.Fatal(err)
	}
	fm, _, ok := entity.Split(original)
	if !ok {
		t.Fatal("test setup: epic file lacks frontmatter")
	}
	edited := append(append([]byte("---\n"), fm...), []byte("---\n\n## Goal\n\nUser-edited prose, in place.\n")...)
	if writeErr := os.WriteFile(epicPath, edited, 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}

	r.must(verb.EditBody(r.ctx, r.tree(), "E-0001", nil, testActor, ""))

	// The committed file matches the user's working-copy edit byte-for-byte.
	got, err := os.ReadFile(epicPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, edited) {
		t.Errorf("post-bless file differs from working-copy edit:\nwant:\n%s\ngot:\n%s", edited, got)
	}

	// Trailer set is the standard edit-body triple.
	trailers, err := gitops.HeadTrailers(context.Background(), r.root)
	if err != nil {
		t.Fatal(err)
	}
	mustHaveTrailer(t, trailers, "aiwf-verb", "edit-body")
	mustHaveTrailer(t, trailers, "aiwf-entity", "E-0001")
	mustHaveTrailer(t, trailers, "aiwf-actor", testActor)
}

// TestEditBody_Bless_RefusesWhenNoChanges pins M-060/AC-3: when
// there's no working-copy diff, bless mode refuses cleanly without
// producing an empty commit.
func TestEditBody_Bless_RefusesWhenNoChanges(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Quiet gap", testActor, verb.AddOptions{}))

	_, err := verb.EditBody(r.ctx, r.tree(), "G-0001", nil, testActor, "")
	if err == nil || !strings.Contains(err.Error(), "no changes to commit") {
		t.Errorf("expected no-changes error in bless mode; got %v", err)
	}
}

// TestEditBody_Bless_RefusesFrontmatterChange pins M-060/AC-2: when
// the working-copy diff includes frontmatter changes, bless mode
// refuses with a pointer to the structured-state verbs (promote,
// rename, cancel, reallocate). Frontmatter is body-only's
// counterpart and stays the domain of those verbs.
func TestEditBody_Bless_RefusesFrontmatterChange(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Frontmatter test", testActor, verb.AddOptions{}))

	// Hand-edit frontmatter (e.g., flip status) — the kind of edit
	// that should go through aiwf promote, not aiwf edit-body.
	epicPath := filepath.Join(r.root, "work", "epics", "E-0001-frontmatter-test", "epic.md")
	raw, err := os.ReadFile(epicPath)
	if err != nil {
		t.Fatal(err)
	}
	tampered := strings.Replace(string(raw), "status: proposed", "status: active", 1)
	if writeErr := os.WriteFile(epicPath, []byte(tampered), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}

	_, err = verb.EditBody(r.ctx, r.tree(), "E-0001", nil, testActor, "")
	if err == nil {
		t.Fatal("expected frontmatter-changed refusal in bless mode")
	}
	if !strings.Contains(err.Error(), "frontmatter changed") {
		t.Errorf("expected 'frontmatter changed' message; got %v", err)
	}
	if !strings.Contains(err.Error(), "promote") {
		t.Errorf("expected pointer at structured-state verbs; got %v", err)
	}
}

// TestEditBody_Bless_RefusesNewEntity: bless mode applies to entities
// that already have a HEAD version. A file added to the working
// tree but not yet committed has no HEAD bytes to diff against, so
// the verb refuses and points at `aiwf add` for the create-time path.
func TestEditBody_Bless_RefusesNewEntity(t *testing.T) {
	r := newRunner(t)

	// Hand-write a gap file directly without committing — this
	// simulates a user starting to draft an entity outside the verb
	// path, which bless mode is not designed for.
	gapDir := filepath.Join(r.root, "work", "gaps")
	if err := os.MkdirAll(gapDir, 0o755); err != nil {
		t.Fatal(err)
	}
	gapPath := filepath.Join(gapDir, "G-0001-untracked.md")
	content := []byte("---\nid: G-001\ntitle: Untracked\nstatus: open\n---\n\n## Body\n\nPre-aiwf draft.\n")
	if err := os.WriteFile(gapPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := verb.EditBody(r.ctx, r.tree(), "G-0001", nil, testActor, "")
	if err == nil {
		t.Fatal("expected refusal on new (uncommitted) entity in bless mode")
	}
	if !strings.Contains(err.Error(), "no committed version") {
		t.Errorf("expected 'no committed version' message; got %v", err)
	}
	if !strings.Contains(err.Error(), "aiwf add") {
		t.Errorf("expected pointer at aiwf add for new entities; got %v", err)
	}
}

// TestEditBody_Bless_PreservesYAMLFormatting: bless mode commits
// the working-copy bytes verbatim — no re-serialization through
// entity.Serialize. If the user's frontmatter has an unusual key
// order or extra whitespace that round-trips cleanly through the
// loader, bless mode preserves it; the explicit-content path
// (which re-serializes) would canonicalize.
func TestEditBody_Bless_PreservesYAMLFormatting(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Format gap", testActor, verb.AddOptions{}))

	gapPath := filepath.Join(r.root, "work", "gaps", "G-0001-format-gap.md")
	original, err := os.ReadFile(gapPath)
	if err != nil {
		t.Fatal(err)
	}
	// Replace only the body, preserving the frontmatter bytes
	// verbatim so we can compare frontmatter byte-for-byte after
	// bless mode commits.
	fm, _, ok := entity.Split(original)
	if !ok {
		t.Fatal("test setup: gap file lacks frontmatter")
	}
	edited := append(append([]byte("---\n"), fm...), []byte("---\n\nNew body prose.\n")...)
	if writeErr := os.WriteFile(gapPath, edited, 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}

	r.must(verb.EditBody(r.ctx, r.tree(), "G-0001", nil, testActor, ""))

	got, err := os.ReadFile(gapPath)
	if err != nil {
		t.Fatal(err)
	}
	gotFM, _, ok := entity.Split(got)
	if !ok {
		t.Fatal("post-bless file lacks frontmatter")
	}
	if !bytes.Equal(gotFM, fm) {
		t.Errorf("bless mode altered frontmatter bytes:\nwant:\n%s\ngot:\n%s", fm, gotFM)
	}
}

// TestEditBody_Bless_ACSubSectionEdit covers M-060/AC-5: editing
// the prose under a single AC heading in a milestone body, then
// running bless on the parent milestone, commits exactly the AC
// sub-section change. The verb doesn't need composite-id support
// — it commits whatever changed in the milestone file.
func TestEditBody_Bless_ACSubSectionEdit(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Mile", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddACBatch(r.ctx, r.tree(), "M-0001", []string{"first criterion", "second criterion"}, nil, testActor, nil))

	mPath := filepath.Join(r.root, "work", "epics", "E-0001-platform", "M-0001-mile.md")
	raw, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatal(err)
	}
	// Add a paragraph under AC-1's heading. The acsBodyCoherence
	// validator requires `### AC-N` headings to remain present and
	// match acs[]; appending prose after the heading is fine.
	withProse := strings.Replace(
		string(raw),
		"### AC-1 — first criterion",
		"### AC-1 — first criterion\n\nDetailed prose for the first AC, written after the heading.",
		1,
	)
	if writeErr := os.WriteFile(mPath, []byte(withProse), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}

	r.must(verb.EditBody(r.ctx, r.tree(), "M-0001", nil, testActor, ""))

	got, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "Detailed prose for the first AC") {
		t.Errorf("AC-1 sub-section edit not committed:\n%s", got)
	}
	// AC headings still intact (acs-body-coherence check would have
	// fired otherwise).
	if !strings.Contains(string(got), "### AC-1 — first criterion") {
		t.Errorf("AC-1 heading missing after bless:\n%s", got)
	}
	if !strings.Contains(string(got), "### AC-2 — second criterion") {
		t.Errorf("AC-2 heading missing after bless:\n%s", got)
	}
	// Tree validates clean.
	if findings := check.Run(r.tree(), nil); check.HasErrors(findings) {
		t.Errorf("post-AC-sub-section bless tree has errors: %+v", findings)
	}
}

// TestEditBody_Bless_RejectsCompositeID: composite ids are still
// refused in bless mode (same as explicit mode). The user routes
// AC sub-section edits through the parent milestone, per AC-5.
func TestEditBody_Bless_RejectsCompositeID(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Epic", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Mile", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "criterion", testActor, nil))

	_, err := verb.EditBody(r.ctx, r.tree(), "M-0001/AC-1", nil, testActor, "")
	if err == nil || !strings.Contains(err.Error(), "composite ids") {
		t.Errorf("expected composite-id refusal in bless mode; got %v", err)
	}
}

// TestEditBody_Bless_AcceptsBodyShapeWarnings_PrePushIsChokepoint:
// bless mode does NOT refuse on body-shape issues like a missing
// `### AC-N` heading — body-shape validators (acs-body-coherence)
// read disk bytes, and `projectionFindings` subtracts pre-existing
// findings, so a user-introduced shape problem already reflected on
// disk shows up identically before and after projection (introduced
// = empty). This is a pre-existing limitation of the projection
// mechanism for body-shape rules; bless mode inherits it.
//
// The chokepoint for body-shape issues is the pre-push hook
// (`aiwf check`), not the verb. Bless mode commits the user's edit;
// the standing check then surfaces any tree-level concerns. Treat
// this test as the explicit boundary contract — if a future change
// makes the projection check body-shape-aware, this test needs to
// flip from "succeeds, surfaces warning later" to "verb refuses with
// a finding."
func TestEditBody_Bless_AcceptsBodyShapeWarnings_PrePushIsChokepoint(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Mile", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "stays", testActor, nil))

	mPath := filepath.Join(r.root, "work", "epics", "E-0001-platform", "M-0001-mile.md")
	raw, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(raw), "### AC-1 — stays", "", 1)
	if writeErr := os.WriteFile(mPath, []byte(broken), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}

	// Bless mode commits successfully — the verb does not pre-flight
	// body-shape coherence.
	r.must(verb.EditBody(r.ctx, r.tree(), "M-0001", nil, testActor, ""))

	// `aiwf check` (the chokepoint) surfaces the body-shape warning
	// for the user to act on. Caller-side responsibility, not verb-
	// side gating.
	postCheck := check.Run(r.tree(), nil)
	foundCoherence := false
	for _, f := range postCheck {
		if f.Code == "acs-body-coherence" {
			foundCoherence = true
		}
	}
	if !foundCoherence {
		t.Errorf("expected acs-body-coherence warning post-bless on broken heading; got %+v", postCheck)
	}
}
