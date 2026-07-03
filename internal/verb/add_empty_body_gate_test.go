package verb_test

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestAdd_EmptyBodyGate_BornCompleteKindsDefaultTemplateRefused pins
// G-0326/AC-1: `aiwf add {gap,decision,adr,contract}` with no
// --body/--body-file at all refuses to create the entity, because the
// default per-kind template is headings-only — exactly the shape the
// gate targets. Table-driven over the four born-complete kinds; the
// error must name every empty section.
func TestAdd_EmptyBodyGate_BornCompleteKindsDefaultTemplateRefused(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		kind         entity.Kind
		wantSections []string
	}{
		{"gap", entity.KindGap, []string{"What's missing", "Why it matters"}},
		{"decision", entity.KindDecision, []string{"Question", "Decision", "Reasoning"}},
		{"adr", entity.KindADR, []string{"Context", "Decision", "Consequences"}},
		{"contract", entity.KindContract, []string{"Purpose", "Stability"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newRunner(t)
			_, err := verb.Add(r.ctx, r.tree(), tc.kind, "Untitled entity", testActor, verb.AddOptions{})
			if err == nil {
				t.Fatalf("expected refusal for %s with no body content, got nil error", tc.kind)
			}
			for _, want := range tc.wantSections {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("%s: error %q should name empty section %q", tc.kind, err, want)
				}
			}
			if !strings.Contains(err.Error(), "--force") || !strings.Contains(err.Error(), "--reason") {
				t.Errorf("%s: error %q should mention the --force --reason escape hatch", tc.kind, err)
			}
		})
	}
}

// TestAdd_EmptyBodyGate_NonEmptyBodyOverrideAllowed pins G-0326/AC-2:
// a born-complete kind with real prose under every required heading
// (as --body / --body-file both route through opts.BodyOverride) is
// allowed — the gate does not block substantive content.
func TestAdd_EmptyBodyGate_NonEmptyBodyOverrideAllowed(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	res, err := verb.Add(r.ctx, r.tree(), entity.KindGap, "Retry loop spins forever", testActor, verb.AddOptions{
		BodyOverride: bornCompleteFixtureBody(entity.KindGap),
	})
	if err != nil {
		t.Fatalf("Add: unexpected error: %v", err)
	}
	if res.Plan == nil {
		t.Fatal("expected a Plan, got nil (findings-only result)")
	}
}

// TestAdd_EmptyBodyGate_WhitespaceOnlyBodyOverrideRefused pins
// G-0326/AC-3: a --body/--body-file whose required sections are
// present but all-whitespace is refused exactly like the bare
// default template — the gate's emptiness test is content-based, not
// presence-based.
func TestAdd_EmptyBodyGate_WhitespaceOnlyBodyOverrideRefused(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	whitespaceBody := []byte("## What's missing\n\n   \n\n## Why it matters\n\n\t\n")
	_, err := verb.Add(r.ctx, r.tree(), entity.KindGap, "Whitespace only", testActor, verb.AddOptions{
		BodyOverride: whitespaceBody,
	})
	if err == nil {
		t.Fatal("expected refusal for whitespace-only body content")
	}
	if !strings.Contains(err.Error(), "What's missing") || !strings.Contains(err.Error(), "Why it matters") {
		t.Errorf("error %q should name both empty sections", err)
	}
}

// TestAdd_EmptyBodyGate_ForceBypassesGate pins G-0326/AC-4: opts.Force
// bypasses the gate even with the bare default (headings-only)
// template, and the create commit's trailers carry `aiwf-force:
// <reason>`. The verb itself does not require opts.Reason to be
// non-empty — that usage-shape validation is the CLI dispatcher's job
// (internal/cli/add/add.go), mirroring `aiwf promote`'s split between
// verb and dispatcher.
func TestAdd_EmptyBodyGate_ForceBypassesGate(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	res, err := verb.Add(r.ctx, r.tree(), entity.KindGap, "Forced through", testActor, verb.AddOptions{
		Force:  true,
		Reason: "deliberately deferring the writeup",
	})
	if err != nil {
		t.Fatalf("Add with Force=true: unexpected error: %v", err)
	}
	if res.Plan == nil {
		t.Fatal("expected a Plan under Force, got nil (findings-only result)")
	}
	var forceTrailer string
	found := false
	for _, tr := range res.Plan.Trailers {
		if tr.Key == "aiwf-force" {
			forceTrailer = tr.Value
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an aiwf-force trailer on the forced create commit; got %+v", res.Plan.Trailers)
	}
	if forceTrailer != "deliberately deferring the writeup" {
		t.Errorf("aiwf-force trailer = %q, want the reason verbatim", forceTrailer)
	}
	if res.Plan.Body != "deliberately deferring the writeup" {
		t.Errorf("Plan.Body = %q, want the reason verbatim (the commit body should carry the same override justification as the trailer)", res.Plan.Body)
	}
}

// TestAdd_EmptyBodyGate_ForceNoOpOnNonBornCompleteKindNoTrailer pins
// the reviewer-flagged fix: `aiwf add epic --force --reason "x"`
// succeeds (epic has no draft-phase gate to bypass — --force is
// inert there) but must NOT stamp an `aiwf-force:` trailer or carry
// the reason in the commit body, because nothing was actually
// overridden. A trailer here would be a false provenance record —
// exactly what the kernel's "sovereign override" audit trail must not
// produce for a no-op flag.
func TestAdd_EmptyBodyGate_ForceNoOpOnNonBornCompleteKindNoTrailer(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	res, err := verb.Add(r.ctx, r.tree(), entity.KindEpic, "Untouched epic", testActor, verb.AddOptions{
		Force:  true,
		Reason: "x",
	})
	if err != nil {
		t.Fatalf("Add with Force=true on kind=epic: unexpected error: %v", err)
	}
	for _, tr := range res.Plan.Trailers {
		if tr.Key == "aiwf-force" {
			t.Errorf("unexpected aiwf-force trailer on a no-op --force (kind=epic has no gate to bypass): %+v", tr)
		}
	}
	if res.Plan.Body != "" {
		t.Errorf("Plan.Body = %q, want empty — a no-op --force must not carry the reason into the commit", res.Plan.Body)
	}
}

// TestAdd_EmptyBodyGate_ForceNoOpWhenBodyAlreadyNonEmptyNoTrailer:
// the companion no-op case on a born-complete kind — --force with a
// body that was already non-empty (nothing to bypass) also stamps no
// aiwf-force trailer.
func TestAdd_EmptyBodyGate_ForceNoOpWhenBodyAlreadyNonEmptyNoTrailer(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	res, err := verb.Add(r.ctx, r.tree(), entity.KindGap, "Already has real prose", testActor, verb.AddOptions{
		BodyOverride: bornCompleteFixtureBody(entity.KindGap),
		Force:        true,
		Reason:       "x",
	})
	if err != nil {
		t.Fatalf("Add with Force=true and a non-empty body: unexpected error: %v", err)
	}
	for _, tr := range res.Plan.Trailers {
		if tr.Key == "aiwf-force" {
			t.Errorf("unexpected aiwf-force trailer when the body was already non-empty (nothing to bypass): %+v", tr)
		}
	}
	if res.Plan.Body != "" {
		t.Errorf("Plan.Body = %q, want empty — a no-op --force must not carry the reason into the commit", res.Plan.Body)
	}
}

// TestAdd_EmptyBodyGate_NoForceNoTrailer pins the complementary
// branch: a normal (non-forced) create commit carries no aiwf-force
// trailer at all.
func TestAdd_EmptyBodyGate_NoForceNoTrailer(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	res, err := verb.Add(r.ctx, r.tree(), entity.KindGap, "Ordinary gap", testActor, verb.AddOptions{
		BodyOverride: bornCompleteFixtureBody(entity.KindGap),
	})
	if err != nil {
		t.Fatalf("Add: unexpected error: %v", err)
	}
	for _, tr := range res.Plan.Trailers {
		if tr.Key == "aiwf-force" {
			t.Errorf("unexpected aiwf-force trailer on a non-forced create commit: %+v", tr)
		}
	}
}

// TestAdd_EmptyBodyGate_DraftBearingKindsUnaffected pins G-0326's
// scope line: epic and milestone keep today's behavior exactly — the
// bare default (headings-only) template is still accepted with no
// --body/--body-file and no --force, because the gate applies only to
// entity.IsBornComplete kinds.
func TestAdd_EmptyBodyGate_DraftBearingKindsUnaffected(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Untouched epic", testActor, verb.AddOptions{}))
	if _, err := verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Untouched milestone", testActor, verb.AddOptions{
		EpicID: "E-0001", TDD: "none",
	}); err != nil {
		t.Fatalf("milestone Add with default (empty) body should still succeed post-G-0326; got: %v", err)
	}
}
