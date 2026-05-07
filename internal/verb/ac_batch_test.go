package verb_test

import (
	"context"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/check"
	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/gitops"
	"github.com/23min/ai-workflow-v2/internal/verb"
)

// TestAddACBatch_RoundTrip is the load-bearing M-057/AC-1 + AC-2
// check: passing N titles in one call appends N consecutive ACs to
// the milestone's frontmatter and walks the on-disk file forward in
// one commit. Asserts the ids are AC-1..AC-N (allocation order),
// titles are preserved per-position, and statuses default to open.
func TestAddACBatch_RoundTrip(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Batch", testActor, verb.AddOptions{EpicID: "E-01", TDD: "none"}))

	titles := []string{"first criterion", "second criterion", "third criterion"}
	r.must(verb.AddACBatch(r.ctx, r.tree(), "M-001", titles, testActor, nil))

	m := r.tree().ByID("M-001")
	if m == nil {
		t.Fatal("M-001 missing")
	}
	if len(m.ACs) != 3 {
		t.Fatalf("acs[] len = %d, want 3 (after batch)", len(m.ACs))
	}
	for i, want := range titles {
		ac := m.ACs[i]
		wantID := []string{"AC-1", "AC-2", "AC-3"}[i]
		if ac.ID != wantID {
			t.Errorf("acs[%d].id = %q, want %q", i, ac.ID, wantID)
		}
		if ac.Title != want {
			t.Errorf("acs[%d].title = %q, want %q", i, ac.Title, want)
		}
		if ac.Status != entity.StatusOpen {
			t.Errorf("acs[%d].status = %q, want %q", i, ac.Status, entity.StatusOpen)
		}
	}
}

// TestAddACBatch_AppendsToExistingACs: when ACs already exist, the
// batch allocates ids starting at len(parent.ACs)+1, position-stable
// per the rest of the kernel. AC-2's contract presumes mid-list
// allocation works.
func TestAddACBatch_AppendsToExistingACs(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Batch", testActor, verb.AddOptions{EpicID: "E-01", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "first existing", testActor, nil))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "second existing", testActor, nil))

	r.must(verb.AddACBatch(r.ctx, r.tree(), "M-001", []string{"third batch", "fourth batch"}, testActor, nil))

	m := r.tree().ByID("M-001")
	if len(m.ACs) != 4 {
		t.Fatalf("acs[] len = %d, want 4", len(m.ACs))
	}
	want := []struct{ id, title string }{
		{"AC-1", "first existing"},
		{"AC-2", "second existing"},
		{"AC-3", "third batch"},
		{"AC-4", "fourth batch"},
	}
	for i, w := range want {
		if m.ACs[i].ID != w.id || m.ACs[i].Title != w.title {
			t.Errorf("acs[%d] = %+v, want id=%q title=%q", i, m.ACs[i], w.id, w.title)
		}
	}
}

// TestAddACBatch_SingleOpWriteAndCommit pins M-057/AC-4: regardless
// of N, the plan produces exactly one OpWrite (the milestone file).
// One verb invocation = one git commit.
func TestAddACBatch_SingleOpWriteAndCommit(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Batch", testActor, verb.AddOptions{EpicID: "E-01", TDD: "none"}))

	res, err := verb.AddACBatch(r.ctx, r.tree(), "M-001", []string{"a", "b", "c", "d"}, testActor, nil)
	if err != nil {
		t.Fatalf("AddACBatch: %v", err)
	}
	if res.Plan == nil {
		t.Fatal("no plan")
	}
	if len(res.Plan.Ops) != 1 {
		t.Fatalf("plan has %d ops, want 1; ops=%+v", len(res.Plan.Ops), res.Plan.Ops)
	}
	if res.Plan.Ops[0].Type != verb.OpWrite {
		t.Errorf("op type = %v, want OpWrite", res.Plan.Ops[0].Type)
	}
}

// TestAddACBatch_EmitsOneEntityTrailerPerAC pins M-057/AC-3: the
// commit produced by the batch carries one aiwf-entity trailer per
// created composite id, in allocation order. `aiwf history M-NNN/AC-X`
// finds the commit because git's --grep matches any trailer line.
func TestAddACBatch_EmitsOneEntityTrailerPerAC(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Batch", testActor, verb.AddOptions{EpicID: "E-01", TDD: "none"}))
	r.must(verb.AddACBatch(r.ctx, r.tree(), "M-001", []string{"a", "b", "c"}, testActor, nil))

	trailers, err := gitops.HeadTrailers(context.Background(), r.root)
	if err != nil {
		t.Fatalf("HeadTrailers: %v", err)
	}
	var seen []string
	for _, tr := range trailers {
		if tr.Key == "aiwf-entity" {
			seen = append(seen, tr.Value)
		}
	}
	want := []string{"M-001/AC-1", "M-001/AC-2", "M-001/AC-3"}
	if len(seen) != len(want) {
		t.Fatalf("aiwf-entity count = %d, want %d; trailers=%+v", len(seen), len(want), trailers)
	}
	for i, w := range want {
		if seen[i] != w {
			t.Errorf("aiwf-entity[%d] = %q, want %q", i, seen[i], w)
		}
	}
}

// TestAddACBatch_SingleTitleUnchanged is the M-057/AC-5 backward-
// compat regression: a length-1 batch produces exactly the same
// trailer set, subject shape, and disk effect as the legacy
// AddAC(parentID, title, ...) entry point. If this test starts
// failing, it means the batch path silently changed semantics for
// the most common single-title case.
func TestAddACBatch_SingleTitleUnchanged(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Single", testActor, verb.AddOptions{EpicID: "E-01", TDD: "none"}))
	r.must(verb.AddACBatch(r.ctx, r.tree(), "M-001", []string{"only one"}, testActor, nil))

	subject, err := gitops.HeadSubject(context.Background(), r.root)
	if err != nil {
		t.Fatal(err)
	}
	if subject != `aiwf add ac M-001/AC-1 "only one"` {
		t.Errorf("single-title batch subject = %q, want pre-batch shape", subject)
	}

	trailers, err := gitops.HeadTrailers(context.Background(), r.root)
	if err != nil {
		t.Fatal(err)
	}
	var entityTrailers []string
	for _, tr := range trailers {
		if tr.Key == "aiwf-entity" {
			entityTrailers = append(entityTrailers, tr.Value)
		}
	}
	if len(entityTrailers) != 1 || entityTrailers[0] != "M-001/AC-1" {
		t.Errorf("single-title batch should emit exactly one aiwf-entity; got %v", entityTrailers)
	}
}

// TestAddACBatch_RejectsEmptyTitleInBatch: AC-2 atomicity — if any
// title is invalid, the whole batch aborts before any disk work.
// Pre-fix, an empty title might create N-1 ACs and surprise the
// user with an inconsistent state.
func TestAddACBatch_RejectsEmptyTitleInBatch(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Batch", testActor, verb.AddOptions{EpicID: "E-01", TDD: "none"}))

	_, err := verb.AddACBatch(r.ctx, r.tree(), "M-001", []string{"good", "  ", "also good"}, testActor, nil)
	if err == nil || !strings.Contains(err.Error(), "empty title") {
		t.Errorf("expected empty-title-in-batch error; got %v", err)
	}
	// No ACs were created — the milestone is still empty.
	m := r.tree().ByID("M-001")
	if m != nil && len(m.ACs) != 0 {
		t.Errorf("expected 0 ACs after refused batch; got %d", len(m.ACs))
	}
}

// TestAddACBatch_RejectsTestsWithMultipleTitles: --tests with N>1
// is rejected because a single set of metrics can't apply
// unambiguously to multiple ACs. An LLM batching N criteria with
// one --tests value would otherwise silently apply the same
// metrics to every AC.
func TestAddACBatch_RejectsTestsWithMultipleTitles(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Batch", testActor, verb.AddOptions{EpicID: "E-01", TDD: "none"}))

	_, err := verb.AddACBatch(r.ctx, r.tree(), "M-001", []string{"a", "b"}, testActor,
		&gitops.TestMetrics{Total: 1, Pass: 1})
	if err == nil || !strings.Contains(err.Error(), "single AC at a time") {
		t.Errorf("expected --tests/N>1 error; got %v", err)
	}
}

// TestAddACBatch_AtomicReversionOnProjectionFailure: AC-2 says
// "failure reverts entire batch". A prosey title on the second AC
// of a 3-batch should leave the milestone with zero ACs, not one.
// (Prosey-title is the reachable failure mode under the current
// rule set; once the title shape passes, projection is well-defined
// for any number of new ACs against an empty milestone.)
func TestAddACBatch_AtomicReversionOnProjectionFailure(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Batch", testActor, verb.AddOptions{EpicID: "E-01", TDD: "none"}))

	// A title that exceeds the prose threshold makes title validation
	// reject the whole batch before disk work. This is the path most
	// likely to be tripped by an LLM batching auto-generated content.
	prosey := strings.Repeat("Long sentence that exceeds the prose threshold so the title validator refuses it. ", 3)
	_, err := verb.AddACBatch(r.ctx, r.tree(), "M-001",
		[]string{"good first", prosey, "would-be-third"}, testActor, nil)
	if err == nil {
		t.Fatal("expected refusal on prosey title in batch")
	}

	m := r.tree().ByID("M-001")
	if m != nil && len(m.ACs) != 0 {
		t.Errorf("batch should be all-or-nothing; got %d ACs after refusal", len(m.ACs))
	}

	// Tree validates clean — no half-batch artifacts.
	if findings := check.Run(r.tree(), nil); check.HasErrors(findings) {
		t.Errorf("tree errors after refused batch: %+v", findings)
	}
}

// TestAddACBatch_BodyHeadingsAppendedInOrder: every new AC gets
// its `### AC-N — <title>` heading in the milestone body, in
// allocation order. This is what makes the milestone's rendered
// page show all the new criteria; the frontmatter alone isn't
// enough.
func TestAddACBatch_BodyHeadingsAppendedInOrder(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Body", testActor, verb.AddOptions{EpicID: "E-01", TDD: "none"}))
	r.must(verb.AddACBatch(r.ctx, r.tree(), "M-001",
		[]string{"alpha criterion", "beta criterion", "gamma criterion"}, testActor, nil))

	res, err := verb.AddACBatch(r.ctx, r.tree(), "M-001", []string{"delta criterion"}, testActor, nil)
	if err != nil {
		t.Fatalf("AddACBatch: %v", err)
	}
	body := string(res.Plan.Ops[0].Content)
	wantInOrder := []string{
		"### AC-1 — alpha criterion",
		"### AC-2 — beta criterion",
		"### AC-3 — gamma criterion",
		"### AC-4 — delta criterion",
	}
	prev := -1
	for _, want := range wantInOrder {
		idx := strings.Index(body, want)
		if idx < 0 {
			t.Errorf("body missing %q:\n%s", want, body)
			continue
		}
		if idx <= prev {
			t.Errorf("body heading order wrong: %q at %d, prev at %d", want, idx, prev)
		}
		prev = idx
	}
}
