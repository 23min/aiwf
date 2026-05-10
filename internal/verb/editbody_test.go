package verb_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/verb"
)

// TestEditBody_RoundTrip is the M-058/AC-1 + AC-2 closure: edit-body
// replaces the markdown body of an existing entity in a single
// atomic commit, leaving frontmatter intact.
func TestEditBody_RoundTrip(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))

	newBody := "## Goal\n\nFreshly edited prose.\n\n## Scope\n\nUpdated scope.\n"
	r.must(verb.EditBody(r.ctx, r.tree(), "E-0001", []byte(newBody), testActor, ""))

	got, err := os.ReadFile(filepath.Join(r.root, "work", "epics", "E-0001-foundations", "epic.md"))
	if err != nil {
		t.Fatalf("read epic: %v", err)
	}
	_, body, ok := entity.Split(got)
	if !ok {
		t.Fatalf("epic file has no frontmatter:\n%s", got)
	}
	if string(body) != newBody {
		t.Errorf("body = %q, want %q", body, newBody)
	}
	// Frontmatter still has the original id/title/status.
	if !strings.Contains(string(got), "id: E-0001") {
		t.Errorf("frontmatter id missing: %s", got)
	}
	if !strings.Contains(string(got), "title: Foundations") {
		t.Errorf("frontmatter title missing: %s", got)
	}
	if !strings.Contains(string(got), "status: proposed") {
		t.Errorf("frontmatter status missing: %s", got)
	}
}

// TestEditBody_SingleOpWriteAndCommit pins AC-2: one OpWrite for the
// entity file regardless of body size — same single-commit guarantee
// as add/promote/cancel.
func TestEditBody_SingleOpWriteAndCommit(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Test gap", testActor, verb.AddOptions{}))

	res, err := verb.EditBody(r.ctx, r.tree(), "G-0001", []byte("## Body\n\nupdated\n"), testActor, "")
	if err != nil {
		t.Fatalf("EditBody: %v", err)
	}
	if res.Plan == nil {
		t.Fatal("no plan")
	}
	if len(res.Plan.Ops) != 1 {
		t.Fatalf("plan has %d ops, want 1", len(res.Plan.Ops))
	}
	if res.Plan.Ops[0].Type != verb.OpWrite {
		t.Errorf("op type = %v, want OpWrite", res.Plan.Ops[0].Type)
	}
}

// TestEditBody_TrailerSet pins AC-2's trailer half: the commit
// carries aiwf-verb edit-body + aiwf-entity + aiwf-actor. No
// aiwf-to: (no status change).
func TestEditBody_TrailerSet(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Epic", testActor, verb.AddOptions{}))
	r.must(verb.EditBody(r.ctx, r.tree(), "E-0001", []byte("body content\n"), testActor, ""))

	trailers, err := gitops.HeadTrailers(context.Background(), r.root)
	if err != nil {
		t.Fatal(err)
	}
	mustHaveTrailer(t, trailers, "aiwf-verb", "edit-body")
	mustHaveTrailer(t, trailers, "aiwf-entity", "E-0001")
	mustHaveTrailer(t, trailers, "aiwf-actor", testActor)
	for _, tr := range trailers {
		if tr.Key == "aiwf-to" {
			t.Errorf("edit-body should not emit aiwf-to (no status change); got %+v", tr)
		}
	}
}

// TestEditBody_WithReason: --reason lands in the commit body so the
// "why" is queryable via `aiwf history` / `git show`.
func TestEditBody_WithReason(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Epic", testActor, verb.AddOptions{}))
	r.must(verb.EditBody(r.ctx, r.tree(), "E-0001", []byte("body\n"), testActor,
		"reframing scope after the planning review"))

	body, err := gitops.HeadBody(context.Background(), r.root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "reframing scope after the planning review") {
		t.Errorf("commit body missing reason text: %q", body)
	}
}

// TestEditBody_RejectsFrontmatter: the shared validateUserBodyBytes
// helper applies — body content with leading `---` is refused so
// edit-body can't accidentally produce a double-frontmatter file.
func TestEditBody_RejectsFrontmatter(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Epic", testActor, verb.AddOptions{}))

	bad := []byte("---\nid: PRETEND\n---\n\n## body\n")
	_, err := verb.EditBody(r.ctx, r.tree(), "E-0001", bad, testActor, "")
	if err == nil || !strings.Contains(err.Error(), "frontmatter delimiter") {
		t.Errorf("expected frontmatter-delimiter error, got %v", err)
	}
}

// TestEditBody_RejectsCompositeID: AC sub-section editing is not
// supported in v1; the verb refuses with a message that points the
// user at editing the parent milestone instead.
func TestEditBody_RejectsCompositeID(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Epic", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Mile", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "criterion", testActor, nil))

	_, err := verb.EditBody(r.ctx, r.tree(), "M-0001/AC-1", []byte("body\n"), testActor, "")
	if err == nil || !strings.Contains(err.Error(), "composite ids") {
		t.Errorf("expected composite-id error, got %v", err)
	}
}

// TestEditBody_NonExistentID returns a Go error before any disk
// work — same shape as Promote/Cancel for missing ids.
func TestEditBody_NonExistentID(t *testing.T) {
	r := newRunner(t)
	_, err := verb.EditBody(r.ctx, r.tree(), "E-0099", []byte("body\n"), testActor, "")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found error, got %v", err)
	}
}

// TestEditBody_PostEditTreeIsClean: an edit-body commit leaves the
// tree clean — no projection findings introduced. Catches the
// regression where a malformed body produces a file the loader
// can't parse on the next aiwf check.
func TestEditBody_PostEditTreeIsClean(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Epic", testActor, verb.AddOptions{}))
	r.must(verb.EditBody(r.ctx, r.tree(), "E-0001", []byte("## Goal\n\nClean body.\n"), testActor, ""))

	if findings := check.Run(r.tree(), nil); check.HasErrors(findings) {
		t.Errorf("post-edit-body tree has errors: %+v", findings)
	}
}

// TestEditBody_PreservesFrontmatterFields: a milestone with
// references (parent epic, depends_on, acs[]) keeps every
// frontmatter field through an edit-body. The verb is body-only
// by contract; structured state is the domain of promote / rename
// / cancel.
func TestEditBody_PreservesFrontmatterFields(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Mile", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "first", testActor, nil))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "second", testActor, nil))

	r.must(verb.EditBody(r.ctx, r.tree(), "M-0001",
		[]byte("## Goal\n\nrewritten\n\n### AC-1 — first\n\n### AC-2 — second\n"),
		testActor, ""))

	m := r.tree().ByID("M-0001")
	if m == nil {
		t.Fatal("M-001 missing")
	}
	if m.Parent != "E-0001" {
		t.Errorf("parent = %q, want E-01", m.Parent)
	}
	if len(m.ACs) != 2 || m.ACs[0].ID != "AC-1" || m.ACs[1].ID != "AC-2" {
		t.Errorf("acs[] mangled: %+v", m.ACs)
	}
	if m.ACs[0].Title != "first" || m.ACs[1].Title != "second" {
		t.Errorf("AC titles mangled: %+v", m.ACs)
	}
}
