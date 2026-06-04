package verb_test

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/scope"
	"github.com/23min/aiwf/internal/verb"
)

// TestAuthorize_Open_HappyPath: a human authorizes an agent to operate
// on an active epic. The opener commit lands with the full I2.5
// trailer set: aiwf-verb=authorize, aiwf-entity=E-01, aiwf-actor=human/...,
// aiwf-to=ai/claude, aiwf-scope=opened, plus the reason in the body
// and an aiwf-reason: trailer.
//
// CurrentBranch satisfies M-0103's preflight (ritual shape recognized by
// internal/branchparse/); the existing happy-path remains green once the
// preflight lands.
func TestAuthorize_Open_HappyPath(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:          verb.AuthorizeOpen,
		Agent:         "ai/claude",
		Reason:        "implement E-01",
		CurrentBranch: "epic/E-0001-engine",
	})
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	if res.Plan == nil {
		t.Fatalf("no plan; findings=%+v", res.Findings)
	}
	if !res.Plan.AllowEmpty {
		t.Error("Plan.AllowEmpty = false, want true (authorize commits have empty diffs)")
	}
	if len(res.Plan.Ops) != 0 {
		t.Errorf("Plan.Ops len = %d, want 0 (authorize never writes files)", len(res.Plan.Ops))
	}
	if got, want := res.Plan.Subject, "aiwf authorize E-0001 --to ai/claude"; got != want {
		t.Errorf("Subject = %q, want %q", got, want)
	}
	if res.Plan.Body != "implement E-01" {
		t.Errorf("Body = %q, want reason text", res.Plan.Body)
	}

	if applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: %v", applyErr)
	}
	tr, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatal(err)
	}
	mustHaveTrailer(t, tr, "aiwf-verb", "authorize")
	mustHaveTrailer(t, tr, "aiwf-entity", "E-0001")
	mustHaveTrailer(t, tr, "aiwf-actor", testActor)
	mustHaveTrailer(t, tr, "aiwf-to", "ai/claude")
	mustHaveTrailer(t, tr, "aiwf-scope", "opened")
	mustHaveTrailer(t, tr, "aiwf-reason", "implement E-01")
}

// TestAuthorize_Open_NoReason: --reason is optional for --to. The
// commit lands without an aiwf-reason: trailer when none is supplied.
// CurrentBranch satisfies M-0103's preflight.
func TestAuthorize_Open_NoReason(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:          verb.AuthorizeOpen,
		Agent:         "ai/claude",
		CurrentBranch: "epic/E-0001-engine",
	})
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	for _, tr := range res.Plan.Trailers {
		if tr.Key == "aiwf-reason" {
			t.Errorf("aiwf-reason trailer present without --reason: %q", tr.Value)
		}
	}
}

// TestAuthorize_Open_WithBranch_EmitsTrailer (M-0102/AC-3): when
// AuthorizeOpen carries a non-empty Branch, the resulting commit
// stamps an aiwf-branch: trailer with that value. BranchExists=true
// satisfies M-0103's preflight (the explicit Branch resolves under
// refs/heads/).
func TestAuthorize_Open_WithBranch_EmitsTrailer(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:         verb.AuthorizeOpen,
		Agent:        "ai/claude",
		Branch:       "epic/E-0001-engine",
		BranchExists: true,
		// M-0161/AC-2 (G-0201): the rung-pair predicate requires
		// (current, target) ∈ legal set. Set CurrentBranch + TrunkShort
		// so RungOf("main", "main")="trunk" + RungOf("epic/E-0001-engine", _)="epic"
		// → (trunk, epic) is legal. Without these the test ran on
		// pre-AC-2's loose BranchExists=true bypass; post-AC-2 the
		// rung-pair check applies regardless of BranchExists.
		CurrentBranch: "main",
		TrunkShort:    "main",
	})
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	if applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: %v", applyErr)
	}
	tr, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatal(err)
	}
	mustHaveTrailer(t, tr, "aiwf-branch", "epic/E-0001-engine")
}

// TestAuthorize_Open_NonAITarget_NoBranch_NoTrailer (M-0102/AC-3,
// preserved through M-0103): for non-ai/* targets, an empty Branch
// continues to emit no aiwf-branch: trailer — backward-compatible with
// the pre-M-0102 trailer set. M-0103's preflight does not fire on
// non-ai/* targets, so the implicit-from-current resolution is also
// gated to ai/*.
func TestAuthorize_Open_NonAITarget_NoBranch_NoTrailer(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:  verb.AuthorizeOpen,
		Agent: "bot/dependabot",
		// CurrentBranch is intentionally ritual-shape — assertion is
		// that the preflight does NOT fire (and so does not promote
		// CurrentBranch into opts.Branch) when the target is non-ai/*.
		CurrentBranch: "epic/E-0001-engine",
	})
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	for _, tr := range res.Plan.Trailers {
		if tr.Key == "aiwf-branch" {
			t.Errorf("aiwf-branch trailer present without Branch option (non-ai target): %q", tr.Value)
		}
	}
}

// TestAuthorize_Open_NonAITarget_BranchMissing_Accepted (M-0102/AC-8,
// preserved through M-0103): for non-ai/* targets, the M-0103 preflight
// does NOT fire — so --branch <name> with a non-existent local branch
// (BranchExists=false) is still accepted, and the trailer carries the
// passed name verbatim. Pins the M-0102 invariant that the preflight
// did NOT take over for non-AI agents; a regression that lifted the
// existence check out of the ai/*-gated block (or that wrapped the
// preflight in a non-AI gate) would silently break the M-0102 contract.
func TestAuthorize_Open_NonAITarget_BranchMissing_Accepted(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:   verb.AuthorizeOpen,
		Agent:  "bot/dependabot",
		Branch: "epic/E-9999-not-a-real-branch",
		// BranchExists deliberately false — the preflight would refuse
		// for an ai/* target, but for bot/* it must not fire.
		BranchExists:  false,
		CurrentBranch: "main",
	})
	if err != nil {
		t.Fatalf("Authorize refused non-AI target with missing branch (M-0103 preflight leaked outside ai/* gate): %v", err)
	}
	if applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: %v", applyErr)
	}
	tr, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatal(err)
	}
	mustHaveTrailer(t, tr, "aiwf-branch", "epic/E-9999-not-a-real-branch")
	mustHaveTrailer(t, tr, "aiwf-to", "bot/dependabot")
}

// TestAuthorize_Open_AITarget_ForceReasonBypassesPreflight
// (M-0103/AC-5): --force --reason "..." bypasses the AI-target
// preflight on a non-terminal scope-entity with no ritual branch
// context. The authorize commit lands and carries aiwf-force: with the
// reason text — the sovereign-override paper trail per ADR-0010 and
// docs/pocv3/design/provenance-model.md. Distinct from the existing
// TestAuthorize_Open_ForceOverridesTerminal: that test exercises Force
// against a terminal scope-entity (the terminal-status refusal); this
// one exercises Force against an active entity that would refuse on
// the preflight alone. Without the `!opts.Force` short-circuit on the
// preflight gate, this test would fail with branch-context-required.
func TestAuthorize_Open_AITarget_ForceReasonBypassesPreflight(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	const reason = "sovereign override: ad-hoc delegation outside ritual flow"
	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:   verb.AuthorizeOpen,
		Agent:  "ai/claude",
		Reason: reason,
		Force:  true,
		// CurrentBranch deliberately non-ritual; --branch deliberately
		// empty. The preflight would refuse with branch-context-required
		// without Force. With Force=true + non-empty Reason, the gate
		// is short-circuited and the verb proceeds.
		CurrentBranch: "main",
	})
	if err != nil {
		t.Fatalf("Authorize refused under --force --reason override: %v", err)
	}
	if applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: %v", applyErr)
	}
	tr, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatal(err)
	}
	mustHaveTrailer(t, tr, "aiwf-force", reason)
	mustHaveTrailer(t, tr, "aiwf-reason", reason)
	mustHaveTrailer(t, tr, "aiwf-to", "ai/claude")
	// The bypass does NOT promote CurrentBranch into an aiwf-branch
	// trailer — the preflight's implicit-to-explicit promotion is
	// inside the gate that Force skips. An explicit --branch would
	// still land its trailer (covered by AC-4 + AC-5 composition would
	// land later as part of M-0158); this test pins the no-trailer
	// shape of the override path.
	for _, e := range res.Plan.Trailers {
		if e.Key == "aiwf-branch" {
			t.Errorf("aiwf-branch trailer present under --force without --branch: %q (override path should not synthesize a branch binding)", e.Value)
		}
	}
}

// TestAuthorize_Open_AITarget_NoBranch_NoRitualCurrent_Refuses
// (M-0103/AC-1): opening a scope on ai/<agent> with no --branch and
// a current checkout that does not match a ritual shape refuses with
// PreflightBranchContextRequiredError. The error carries the
// branch-context-required code (entity.Coded), the message names the
// override path (--force --reason), and no commit is planned.
func TestAuthorize_Open_AITarget_NoBranch_NoRitualCurrent_Refuses(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:          verb.AuthorizeOpen,
		Agent:         "ai/claude",
		CurrentBranch: "main",
	})
	if err == nil {
		t.Fatalf("expected refusal for ai/* target on non-ritual branch; got plan=%+v", res.Plan)
	}
	if code, ok := entity.Code(err); !ok || code != verb.CodePreflightBranchContextRequired.ID {
		t.Errorf("entity.Code(err) = (%q, %v), want (%q, true)", code, ok, verb.CodePreflightBranchContextRequired.ID)
	}
	if !strings.Contains(err.Error(), "branch-context-required") {
		t.Errorf("error %q does not name the code", err.Error())
	}
	if !strings.Contains(err.Error(), "--force --reason") {
		t.Errorf("error %q does not name the override path (--force --reason)", err.Error())
	}
}

// TestAuthorize_Open_AITarget_DetachedHEAD_NoBranch_Refuses
// (M-0161/AC-7 / G-0207): opening a scope on ai/<agent> with
// no --branch from detached HEAD refuses via
// PreflightBranchContextRequiredError; the refined error text
// names "detached HEAD has no ritual context" so operators see
// the exact state. This unit test pins the
// CurrentBranch == "" branch of the error renderer at
// internal/verb/authorize.go:87-93 — without it, the refinement
// is dead-letter code (M-0161/AC-7 reviewer B1).
func TestAuthorize_Open_AITarget_DetachedHEAD_NoBranch_Refuses(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:          verb.AuthorizeOpen,
		Agent:         "ai/claude",
		CurrentBranch: "", // detached HEAD signal from the CLI
	})
	if err == nil {
		t.Fatalf("expected refusal on detached HEAD; got plan=%+v", res.Plan)
	}
	if code, ok := entity.Code(err); !ok || code != verb.CodePreflightBranchContextRequired.ID {
		t.Errorf("entity.Code(err) = (%q, %v); want (%q, true)", code, ok, verb.CodePreflightBranchContextRequired.ID)
	}
	if !strings.Contains(err.Error(), "detached HEAD has no ritual context") {
		t.Errorf("error %q does not name detached-HEAD state (M-0161/AC-7 refinement)", err.Error())
	}
	if !strings.Contains(err.Error(), "--force --reason") {
		t.Errorf("error %q does not name the override path (--force --reason)", err.Error())
	}
}

// TestAuthorize_Open_AITarget_DetachedHEAD_RitualBranch_RungPairError
// (M-0161/AC-7 / G-0207): opening a scope on ai/<agent> with
// --branch <ritual> from detached HEAD refuses via
// PreflightRungPairError; the refined text names "detached HEAD
// has no ritual context" rather than the generic
// "(non-ritual, epic) is not a legal rung pair" message. Pins
// the CurrentBranch == "" branch of PreflightRungPairError.Error()
// at internal/verb/authorize.go:163-167.
func TestAuthorize_Open_AITarget_DetachedHEAD_RitualBranch_RungPairError(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:          verb.AuthorizeOpen,
		Agent:         "ai/claude",
		Branch:        "epic/E-0001-engine",
		BranchExists:  false, // not yet created — future-branch carve-out
		CurrentBranch: "",    // detached HEAD signal
	})
	if err == nil {
		t.Fatalf("expected refusal on detached HEAD with ritual --branch; got plan=%+v", res.Plan)
	}
	if code, ok := entity.Code(err); !ok || code != verb.CodePreflightRungPair.ID {
		t.Errorf("entity.Code(err) = (%q, %v); want (%q, true)", code, ok, verb.CodePreflightRungPair.ID)
	}
	if !strings.Contains(err.Error(), "detached HEAD has no ritual context") {
		t.Errorf("error %q does not name detached-HEAD state (M-0161/AC-7 refinement)", err.Error())
	}
	if !strings.Contains(err.Error(), "--force --reason") {
		t.Errorf("error %q does not name the override path", err.Error())
	}
}

// TestAuthorize_Open_AITarget_BranchMissing_Refuses (M-0103/AC-2,
// narrowed by M-0104/AC-4 then again by M-0105/AC-6): opening a
// scope on ai/<agent> with --branch <name> where the named branch
// does not exist locally (BranchExists=false) refuses with
// PreflightBranchNotFoundError. The error carries the
// branch-not-found code; the message names --force --reason as the
// override.
//
// Two carve-outs have narrowed the refusal scope:
//   - M-0104/AC-4: CurrentBranch=="main" + ritual --branch → accept
//     (the step-7 sovereign authorize of aiwfx-start-epic).
//   - M-0105/AC-6: ritual CurrentBranch + ritual --branch → accept
//     (the step-4 sovereign authorize of aiwfx-start-milestone, on
//     the parent epic branch with a future milestone --branch).
//
// To keep this AC-2 test pinning the general "missing → refuse"
// rule OUTSIDE both carve-outs, CurrentBranch is pinned to a
// non-main, non-ritual shape (a plain feature branch). The
// missing-branch refusal stands regardless of --branch's shape.
//
// M-0161/AC-2 (G-0201) replaced the pre-AC-2 PreflightBranchNotFoundError
// refusal with PreflightRungPairError — the (non-ritual feature
// branch, epic) pair is now refused as ("", "epic") rung-pair-illegal,
// not as branch-not-found. The semantic is the same (verb refuses,
// names the override path, names the branches involved); the failure
// classification is finer.
func TestAuthorize_Open_AITarget_BranchMissing_Refuses(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:   verb.AuthorizeOpen,
		Agent:  "ai/claude",
		Branch: "epic/E-9999-typo",
		// BranchExists deliberately left false: the CLI's git
		// show-ref --verify would have set it true; this fixture
		// simulates the typo / missing-branch case.
		//
		// CurrentBranch is a non-main, non-ritual feature branch so
		// (RungOf("feature/test-fixture", _), RungOf("epic/E-9999-typo", _))
		// = ("", "epic") which is not in the legal set → AC-2's
		// rung-pair check refuses.
		CurrentBranch: "feature/test-fixture",
		TrunkShort:    "main",
	})
	if err == nil {
		t.Fatalf("expected refusal for ai/* target with non-ritual current + ritual target; got plan=%+v", res.Plan)
	}
	if code, ok := entity.Code(err); !ok || code != verb.CodePreflightRungPair.ID {
		t.Errorf("entity.Code(err) = (%q, %v), want (%q, true)", code, ok, verb.CodePreflightRungPair.ID)
	}
	if !strings.Contains(err.Error(), "rung-pair-illegal") {
		t.Errorf("error %q does not name the code", err.Error())
	}
	if !strings.Contains(err.Error(), "epic/E-9999-typo") {
		t.Errorf("error %q does not quote the target branch", err.Error())
	}
	if !strings.Contains(err.Error(), "feature/test-fixture") {
		t.Errorf("error %q does not quote the current branch", err.Error())
	}
	if !strings.Contains(err.Error(), "--force --reason") {
		t.Errorf("error %q does not name the override path (--force --reason)", err.Error())
	}
}

// TestAuthorize_Open_AITarget_ImplicitFromCurrent_AcceptsAndEmitsTrailer
// (M-0103/AC-3): opening a scope on ai/<agent> with no --branch but on
// a ritual-shape current checkout accepts. The trailer-emission
// promotes the implicit binding to explicit by stamping aiwf-branch:
// with the current branch name.
func TestAuthorize_Open_AITarget_ImplicitFromCurrent_AcceptsAndEmitsTrailer(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:          verb.AuthorizeOpen,
		Agent:         "ai/claude",
		CurrentBranch: "epic/E-0001-engine",
	})
	if err != nil {
		t.Fatalf("Authorize refused implicit ritual current-branch: %v", err)
	}
	if applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: %v", applyErr)
	}
	tr, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatal(err)
	}
	mustHaveTrailer(t, tr, "aiwf-branch", "epic/E-0001-engine")
}

// TestAuthorize_Open_AITarget_ExplicitBranchExists_AcceptsAndEmitsTrailer
// (M-0103/AC-4): opening a scope on ai/<agent> with --branch <name>
// where the named branch exists locally (BranchExists=true) accepts,
// regardless of the current checkout. Trailer carries the explicit
// --branch value.
func TestAuthorize_Open_AITarget_ExplicitBranchExists_AcceptsAndEmitsTrailer(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:         verb.AuthorizeOpen,
		Agent:        "ai/claude",
		Branch:       "epic/E-0001-engine",
		BranchExists: true,
		// M-0161/AC-2: trunk-classification requires TrunkShort.
		// CurrentBranch="main" + TrunkShort="main" → (trunk, epic)
		// is legal; the rung-pair check accepts.
		CurrentBranch: "main",
		TrunkShort:    "main",
	})
	if err != nil {
		t.Fatalf("Authorize refused explicit existing branch: %v", err)
	}
	if applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: %v", applyErr)
	}
	tr, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatal(err)
	}
	mustHaveTrailer(t, tr, "aiwf-branch", "epic/E-0001-engine")
}

// TestAuthorize_Open_AITarget_MainPlusRitualFutureBranch_Accepts
// (M-0104/AC-4): from CurrentBranch=="main", an explicit --branch
// naming a ritual-shape ref that does NOT yet exist
// (BranchExists=false) accepts. This is the well-formed step-4
// pattern of aiwfx-start-epic: the human stays on main, sovereign-
// authorizes the AI agent with --branch epic/E-NNNN-<slug>, then
// the branch is cut at step 5. The carve-out is narrow — only main
// (the trunk-naming convention this repo uses) is treated as the
// pre-cut staging ground; non-main current branches still refuse
// for missing --branch.
//
// Pins: preflight's main-only future-binding carve-out and the
// trailer-emission promotes opts.Branch to the future ref (the
// commit record carries the binding even though the ref doesn't
// resolve yet — the ritual cuts it at step 5).
func TestAuthorize_Open_AITarget_MainPlusRitualFutureBranch_Accepts(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:   verb.AuthorizeOpen,
		Agent:  "ai/claude",
		Branch: "epic/E-0001-engine",
		// BranchExists deliberately false: the ritual's step-4
		// authorize fires BEFORE step 5's branch cut, so at the
		// moment of authorize the named branch genuinely does not
		// exist yet. The carve-out must accept anyway.
		BranchExists:  false,
		CurrentBranch: "main",
		// M-0161/AC-1 (G-0200): the carve-out compares CurrentBranch
		// against the configured trunk short-name via opts.TrunkShort,
		// not the literal "main". Verb-level test populates TrunkShort
		// directly with "main" (the kernel default this test stages).
		// The CLI layer derives this value via
		// cliutil.ConfiguredTrunkBranchShortName under live usage.
		TrunkShort: "main",
	})
	if err != nil {
		t.Fatalf("Authorize refused main+ritual-future-branch: %v", err)
	}
	if applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: %v", applyErr)
	}
	tr, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatal(err)
	}
	mustHaveTrailer(t, tr, "aiwf-branch", "epic/E-0001-engine")
}

// TestAuthorize_Open_AITarget_RitualCurrentPlusRitualFutureBranch_Accepts
// (M-0105/AC-6): symmetric to M-0104/AC-4 one level down. From a
// ritual non-main current (e.g., the parent epic branch), an
// explicit --branch naming a ritual-shape ref that does NOT yet
// exist (BranchExists=false) accepts. This is the step-4 pattern
// of aiwfx-start-milestone: the operator is on epic/E-NN-<slug>,
// sovereign-authorizes the AI agent with --branch milestone/M-NN-<slug>,
// then cuts the milestone branch at step 5.
//
// The carve-out's union with M-0104/AC-4 (main OR ritual current)
// keeps the chokepoint useful — non-ritual non-main current still
// refuses missing --branch.
//
// Pins: the extended carve-out's ritual-current arm, and the
// trailer-emission promotes opts.Branch to the milestone future ref.
func TestAuthorize_Open_AITarget_RitualCurrentPlusRitualFutureBranch_Accepts(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache", testActor, verb.AddOptions{
		EpicID: "E-0001",
		TDD:    "required",
	}))

	res, err := verb.Authorize(r.ctx, r.tree(), "M-0001", testActor, verb.AuthorizeOptions{
		Mode:   verb.AuthorizeOpen,
		Agent:  "ai/claude",
		Branch: "milestone/M-0001-cache",
		// BranchExists deliberately false: the ritual's step-4
		// authorize fires BEFORE step 5's branch cut.
		BranchExists:  false,
		CurrentBranch: "epic/E-0001-engine",
	})
	if err != nil {
		t.Fatalf("Authorize refused ritual-current+ritual-future-branch: %v", err)
	}
	if applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: %v", applyErr)
	}
	tr, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatal(err)
	}
	mustHaveTrailer(t, tr, "aiwf-branch", "milestone/M-0001-cache")
}

// TestAuthorize_Open_AITarget_NonRitualNonMainCurrent_BranchMissing_Refuses
// (M-0105/AC-6 carve-out guard, lower bound): the M-0105 extension
// of the carve-out covers `main OR ritual` for CurrentBranch. The
// bottom bound — a non-ritual, non-main current (e.g., a plain
// `feature/x` branch) — must STILL refuse missing --branch, or the
// gate is a no-op for every branch.
//
// This guard is independent of the ritual-shape requirement on
// --branch (which is the M-0104 guard); it pins the CurrentBranch
// half of the carve-out.
func TestAuthorize_Open_AITarget_NonRitualNonMainCurrent_BranchMissing_Refuses(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:          verb.AuthorizeOpen,
		Agent:         "ai/claude",
		Branch:        "epic/E-9999-future",
		BranchExists:  false,
		CurrentBranch: "feature/scratch",
		TrunkShort:    "main",
	})
	if err == nil {
		t.Fatalf("expected refusal for non-ritual non-main current + ritual target; got plan=%+v", res.Plan)
	}
	// M-0161/AC-2: ("", "epic") is not legal; rung-pair-illegal
	// fires (subsumes the prior branch-not-found semantics).
	if code, ok := entity.Code(err); !ok || code != verb.CodePreflightRungPair.ID {
		t.Errorf("entity.Code(err) = (%q, %v), want (%q, true)", code, ok, verb.CodePreflightRungPair.ID)
	}
}

// TestAuthorize_Open_AITarget_MainPlusNonRitualMissingBranch_Refuses
// (M-0104/AC-4 carve-out guard): the M-0104/AC-4 carve-out is tight
// to ritual-shape --branch values. From CurrentBranch=="main" with
// an explicit --branch that does NOT parse as a ritual shape (i.e.,
// branchparse.ParseEntityFromBranch returns ""), the missing-branch
// refusal still fires. Without this guard the carve-out would be a
// gate-bypass — any string under --branch would authorize from main.
func TestAuthorize_Open_AITarget_MainPlusNonRitualMissingBranch_Refuses(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:          verb.AuthorizeOpen,
		Agent:         "ai/claude",
		Branch:        "feature/scratch",
		BranchExists:  false,
		CurrentBranch: "main",
		TrunkShort:    "main",
	})
	if err == nil {
		t.Fatalf("expected refusal for main+non-ritual-target; got plan=%+v", res.Plan)
	}
	// M-0161/AC-2: ("trunk", "") is not legal; rung-pair-illegal
	// fires. Subsumes the prior branch-not-found semantics.
	if code, ok := entity.Code(err); !ok || code != verb.CodePreflightRungPair.ID {
		t.Errorf("entity.Code(err) = (%q, %v), want (%q, true)", code, ok, verb.CodePreflightRungPair.ID)
	}
	if !strings.Contains(err.Error(), "feature/scratch") {
		t.Errorf("error %q does not quote the target branch", err.Error())
	}
}

// TestValidateAuthorizeTrailers_AiwfBranchShape pins the verb-layer
// seam between validateAuthorizeTrailers and gitops.ValidateTrailer
// for the aiwf-branch trailer specifically (M-0161/AC-2 reviewer
// S-2 follow-up). Pre-AC-2, TestAuthorize_Open_WithBranch_InvalidShapeRefused
// exercised this seam end-to-end via the verb (with BranchExists=true
// to bypass the missing-branch carve-out). Post-AC-2, that test now
// asserts rung-pair-illegal because the rung-pair check fires first
// for any malformed value (RungOf returns "" for non-ritual shapes).
//
// This unit test pins the seam DIRECTLY: it builds a trailer set
// containing a malformed aiwf-branch value and calls
// validateAuthorizeTrailers, asserting the gitops.ValidateTrailer
// shape rule fires through this seam (the error mentions
// "aiwf-branch", proving the trailer name reached validation). Pure
// unit-level, no entity/tree state needed.
//
// Without this test the seam between validateAuthorizeTrailers and
// gitops.ValidateTrailer is only covered by the gitops-layer
// trailers_test.go — that's the rule itself, but the verb wrapping
// (does Authorize correctly route every emitted trailer through
// validation?) is undocumented after the rung-pair predicate took
// over the integration path.
func TestValidateAuthorizeTrailers_AiwfBranchShape(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, branch string
	}{
		{"whitespace", "epic/with whitespace"},
		{"leading slash", "/epic/E-0001"},
		{"embedded double-dot", "epic/E-..-bad"},
		{"colon", "epic:E-0001"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Build the minimal trailer set the verb emits for an
			// ai-target authorize; substitute the malformed value
			// at aiwf-branch. validateAuthorizeTrailers should
			// route the value through gitops.ValidateTrailer and
			// the shape rule should refuse — the resulting error
			// must name aiwf-branch so a downstream operator can
			// diagnose.
			trailers := []gitops.Trailer{
				{Key: "aiwf-verb", Value: "authorize"},
				{Key: "aiwf-entity", Value: "E-0001"},
				{Key: "aiwf-to", Value: "ai/claude"},
				{Key: "aiwf-branch", Value: tc.branch},
			}
			err := verb.ValidateAuthorizeTrailersForTest(trailers)
			if err == nil {
				t.Fatalf("expected shape-validation refusal for malformed aiwf-branch %q; got nil", tc.branch)
			}
			if !strings.Contains(err.Error(), "aiwf-branch") {
				t.Errorf("error %q does not mention aiwf-branch (seam not exercised correctly)", err.Error())
			}
		})
	}
}

// TestAuthorize_Open_WithBranch_InvalidShapeRefused: defensive — when
// the Branch value violates ritual-shape rules (embedded whitespace,
// leading slash, embedded ".."), the verb refuses. The pre-M-0161/AC-2
// path landed at the trailer-shape validation pass (aiwf-branch
// mention); post-AC-2 the rung-pair check fires first because
// RungOf classifies malformed values as "" (no valid id segment) →
// (trunk, "") is not legal → PreflightRungPairError. The intent is
// preserved (malformed Branch refuses); the code path is finer.
// (The validateAuthorizeTrailers seam is now pinned directly by
// TestValidateAuthorizeTrailers_AiwfBranchShape above.)
func TestAuthorize_Open_WithBranch_InvalidShapeRefused(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, branch string
	}{
		{"whitespace", "epic/with whitespace"},
		{"leading slash", "/epic/E-0001"},
		{"embedded double-dot", "epic/E-..-bad"},
		{"colon", "epic:E-0001"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newRunner(t)
			r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
			r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

			_, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
				Mode:   verb.AuthorizeOpen,
				Agent:  "ai/claude",
				Branch: tc.branch,
				// BranchExists=true (immaterial post-AC-2; the
				// rung-pair check runs regardless of BranchExists).
				BranchExists:  true,
				CurrentBranch: "main",
				TrunkShort:    "main",
			})
			if err == nil {
				t.Fatalf("expected refusal for malformed branch %q; got nil", tc.branch)
			}
			// AC-2: malformed --branch classifies as targetRung=""
			// → (trunk, "") is not in the legal set → rung-pair-illegal.
			if code, ok := entity.Code(err); !ok || code != verb.CodePreflightRungPair.ID {
				t.Errorf("entity.Code(err) = (%q, %v), want (%q, true) for malformed branch %q",
					code, ok, verb.CodePreflightRungPair.ID, tc.branch)
			}
		})
	}
}

// TestAuthorize_Open_RefusesNonHumanActor: only human/... actors may
// authorize. An ai/... or bot/... actor is refused at the verb gate.
func TestAuthorize_Open_RefusesNonHumanActor(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	_, err := verb.Authorize(r.ctx, r.tree(), "E-0001", "ai/claude", verb.AuthorizeOptions{
		Mode:  verb.AuthorizeOpen,
		Agent: "ai/claude",
	})
	if err == nil {
		t.Fatal("expected refusal for non-human actor; got nil")
	}
	if !strings.Contains(err.Error(), "human/") {
		t.Errorf("error %q does not mention human/ requirement", err.Error())
	}
}

// TestAuthorize_Open_RefusesNonScopeEntityKind: per D-0007 only
// scope-entities (epic + milestone) carry autonomous-work scopes.
// Authorize on gap/decision/contract/adr is refused at the verb gate
// with the authorize-kind-not-allowed token in the error message.
// Closes the gap surfaced by M-0125/AC-2 dry-run; spec cells
// R-AUDIT-0122 / R-FP-0133.
func TestAuthorize_Open_RefusesNonScopeEntityKind(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		kind entity.Kind
		id   string
		add  func(r *runner)
	}{
		{
			name: "gap",
			kind: entity.KindGap,
			id:   "G-0001",
			add: func(r *runner) {
				r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Test Gap", testActor, verb.AddOptions{}))
			},
		},
		{
			name: "decision",
			kind: entity.KindDecision,
			id:   "D-0001",
			add: func(r *runner) {
				r.must(verb.Add(r.ctx, r.tree(), entity.KindDecision, "Test Decision", testActor, verb.AddOptions{}))
			},
		},
		{
			name: "contract",
			kind: entity.KindContract,
			id:   "C-0001",
			add: func(r *runner) {
				r.must(verb.Add(r.ctx, r.tree(), entity.KindContract, "Test Contract", testActor, verb.AddOptions{}))
			},
		},
		{
			name: "adr",
			kind: entity.KindADR,
			id:   "ADR-0001",
			add: func(r *runner) {
				r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "Test ADR", testActor, verb.AddOptions{}))
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newRunner(t)
			tc.add(r)
			_, err := verb.Authorize(r.ctx, r.tree(), tc.id, testActor, verb.AuthorizeOptions{
				Mode:  verb.AuthorizeOpen,
				Agent: "ai/claude",
			})
			if err == nil {
				t.Fatalf("expected refusal for kind %q; got nil", tc.kind)
			}
			if !strings.Contains(err.Error(), "authorize-kind-not-allowed") {
				t.Errorf("error %q does not name authorize-kind-not-allowed", err.Error())
			}
			// AC-3: the code is carried as structured data, extracted
			// via errors.As, not merely present in the message text.
			if code, ok := entity.Code(err); !ok || code != verb.CodeAuthorizeKindNotAllowed.ID {
				t.Errorf("entity.Code(err) = (%q, %v), want (%q, true)", code, ok, verb.CodeAuthorizeKindNotAllowed.ID)
			}
		})
	}
}

// TestAuthorize_Open_RefusesTerminalEntity: a `done` or `cancelled`
// epic refuses --to without --force.
func TestAuthorize_Open_RefusesTerminalEntity(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "done", testActor, "ship", false, verb.PromoteOptions{}))

	_, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:  verb.AuthorizeOpen,
		Agent: "ai/claude",
	})
	if err == nil {
		t.Fatal("expected refusal on terminal entity; got nil")
	}
	if !strings.Contains(err.Error(), "terminal") {
		t.Errorf("error %q does not mention terminal status", err.Error())
	}
}

// TestAuthorize_Open_ForceOverridesTerminal: --force --reason on a
// terminal entity opens a fresh scope and stamps an aiwf-force trailer.
func TestAuthorize_Open_ForceOverridesTerminal(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "done", testActor, "ship", false, verb.PromoteOptions{}))

	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:   verb.AuthorizeOpen,
		Agent:  "ai/claude",
		Reason: "resurrect for follow-up",
		Force:  true,
	})
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	if applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: %v", applyErr)
	}
	tr, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatal(err)
	}
	mustHaveTrailer(t, tr, "aiwf-force", "resurrect for follow-up")
}

// TestAuthorize_Open_ForceRequiresReason: --force without --reason
// (or with whitespace-only reason) is refused. The terminal-entity
// shape exercises the existing rule's interaction with the terminal
// gate; the non-terminal + no-ritual-branch shape below
// (TestAuthorize_Open_AITarget_ForceWithoutReason_RefusesWithReasonError)
// pins AC-6's ordering invariant — the force-requires-reason check
// fires BEFORE the M-0103 preflight.
func TestAuthorize_Open_ForceRequiresReason(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		reason string
	}{
		{"empty-reason", ""},
		{"whitespace-only-reason", "   \t  "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newRunner(t)
			r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
			r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))
			r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "done", testActor, "ship", false, verb.PromoteOptions{}))

			_, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
				Mode:   verb.AuthorizeOpen,
				Agent:  "ai/claude",
				Reason: tc.reason,
				Force:  true,
			})
			if err == nil || !strings.Contains(err.Error(), "--reason") {
				t.Errorf("expected --reason refusal; got %v", err)
			}
		})
	}
}

// TestAuthorize_Open_AITarget_ForceWithoutReason_RefusesWithReasonError
// (M-0103/AC-6): on an ai/* target with --force but no --reason, the
// error message names "--reason" (not branch-context-required) even
// though the current branch is non-ritual and the preflight would
// otherwise fire.
//
// Pins the error-message-identity invariant for the operator-visible
// surface: when (Force=true, Reason=""), the user sees the --reason
// error, not branch-context-required. The preflight's `!opts.Force`
// short-circuit (verb/authorize.go around the preflight gate)
// guarantees this regardless of literal source order between the
// force-requires-reason check and the preflight — a reorder that
// preserved `!opts.Force` would NOT break this test. What WOULD break
// it: dropping the `!opts.Force` clause from the preflight (the
// preflight then fires for ai/* + non-ritual branch and the operator
// sees branch-context-required instead of --reason).
func TestAuthorize_Open_AITarget_ForceWithoutReason_RefusesWithReasonError(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	_, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:  verb.AuthorizeOpen,
		Agent: "ai/claude",
		// Reason deliberately empty. CurrentBranch non-ritual; --branch
		// empty. The preflight would refuse with branch-context-required
		// if it ran first; the assertion that --reason is named in the
		// error is what pins ordering.
		Force:         true,
		CurrentBranch: "main",
	})
	if err == nil {
		t.Fatal("expected refusal for --force without --reason; got nil")
	}
	if !strings.Contains(err.Error(), "--reason") {
		t.Errorf("error %q does not name --reason — gate-ordering may be inverted (preflight fired before force-requires-reason)", err.Error())
	}
	if strings.Contains(err.Error(), "branch-context-required") {
		t.Errorf("error %q names branch-context-required — preflight fired before force-requires-reason check (gate-ordering inverted)", err.Error())
	}
}

// TestAuthorize_Open_RequiresAgent: --to without an agent argument
// (or with whitespace-only) is refused.
func TestAuthorize_Open_RequiresAgent(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	_, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode: verb.AuthorizeOpen,
	})
	if err == nil || !strings.Contains(err.Error(), "agent") {
		t.Errorf("expected missing-agent refusal; got %v", err)
	}
}

// TestAuthorize_Open_AgentMustBeRoleSlashID: an agent value that
// isn't <role>/<id> is refused.
func TestAuthorize_Open_AgentMustBeRoleSlashID(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	_, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:  verb.AuthorizeOpen,
		Agent: "claude", // missing /
	})
	if err == nil || !strings.Contains(err.Error(), "<role>/<id>") {
		t.Errorf("expected role/id refusal; got %v", err)
	}
}

// TestAuthorize_Pause_RefusesWithoutActiveScope: --pause with no
// active scope on the entity is refused.
func TestAuthorize_Pause_RefusesWithoutActiveScope(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	_, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:   verb.AuthorizePause,
		Reason: "blocked",
	})
	if err == nil || !strings.Contains(err.Error(), "no active scope") {
		t.Errorf("expected no-active-scope refusal; got %v", err)
	}
}

// TestAuthorize_PauseResume_DoNotTriggerPreflight (M-0103/AC-7):
// the AI-target preflight is structurally gated to AuthorizeOpen — it
// lives inside authorizeOpen, not at the verb's top-level entry. Pause
// and resume modes route through authorizeTransition and never consume
// opts.CurrentBranch / opts.BranchExists, so a non-ritual current
// branch with no --branch passed does NOT refuse on those modes.
//
// Pins the structural separation: a future refactor that moved the
// preflight to a shared helper called by all modes (or to the top of
// verb.Authorize before the mode switch) would silently break this
// test on the pause+non-ritual-branch arm. The test sets opts.Force
// to false and CurrentBranch to a non-ritual name; under those
// conditions an AuthorizeOpen on ai/* would refuse.
func TestAuthorize_PauseResume_DoNotTriggerPreflight(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		mode        verb.AuthorizeMode
		wantSubject string
		wantScope   string
		sourceState scope.State
	}{
		{
			name:        "pause-on-non-ritual-branch",
			mode:        verb.AuthorizePause,
			wantSubject: "aiwf authorize E-0001 --pause",
			wantScope:   "paused",
			sourceState: scope.StateActive,
		},
		{
			name:        "resume-on-non-ritual-branch",
			mode:        verb.AuthorizeResume,
			wantSubject: "aiwf authorize E-0001 --resume",
			wantScope:   "resumed",
			sourceState: scope.StatePaused,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newRunner(t)
			r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
			r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

			scopes := []*scope.Scope{
				{AuthSHA: "deadbee", Entity: "E-0001", Agent: "ai/claude", Principal: testActor, State: tc.sourceState},
			}
			res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
				Mode:   tc.mode,
				Reason: "preflight-irrelevant on " + tc.name,
				Scopes: scopes,
				// These three would refuse with branch-context-required
				// if the preflight applied to non-Open modes. The
				// assertion is the verb succeeds — proving the gate is
				// Open-only.
				CurrentBranch: "main",
				BranchExists:  false,
				Force:         false,
			})
			if err != nil {
				t.Fatalf("Authorize %s refused on non-ritual branch — preflight may have leaked outside AuthorizeOpen: %v", tc.name, err)
			}
			if got := res.Plan.Subject; got != tc.wantSubject {
				t.Errorf("Subject = %q, want %q", got, tc.wantSubject)
			}
			var sawScope bool
			for _, e := range res.Plan.Trailers {
				if e.Key == "aiwf-scope" {
					sawScope = true
					if e.Value != tc.wantScope {
						t.Errorf("aiwf-scope = %q, want %q", e.Value, tc.wantScope)
					}
				}
				if e.Key == "aiwf-branch" {
					t.Errorf("aiwf-branch trailer present on %s commit — preflight's implicit-to-explicit promotion may have leaked: %q", tc.name, e.Value)
				}
				// Belt-and-suspenders: pause/resume never read opts.Force
				// (authorizeTransition's emission path doesn't add the
				// trailer), so aiwf-force should be absent. A refactor
				// that leaked Force into pause/resume's emission path
				// would surface here.
				if e.Key == "aiwf-force" {
					t.Errorf("aiwf-force trailer present on %s commit — pause/resume emission may have leaked Force: %q", tc.name, e.Value)
				}
			}
			if !sawScope {
				t.Errorf("aiwf-scope trailer missing; trailers = %+v", res.Plan.Trailers)
			}
		})
	}
}

// TestAuthorize_Pause_HappyPath: with one active scope on the entity,
// --pause produces an authorize commit with aiwf-scope=paused and
// aiwf-reason=<text>.
func TestAuthorize_Pause_HappyPath(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	scopes := []*scope.Scope{
		{AuthSHA: "deadbee", Entity: "E-0001", Agent: "ai/claude", Principal: testActor, State: scope.StateActive},
	}
	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:   verb.AuthorizePause,
		Reason: "blocked by E-09",
		Scopes: scopes,
	})
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	if got, want := res.Plan.Subject, "aiwf authorize E-0001 --pause"; got != want {
		t.Errorf("Subject = %q, want %q", got, want)
	}
	if applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: %v", applyErr)
	}
	tr, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatal(err)
	}
	mustHaveTrailer(t, tr, "aiwf-scope", "paused")
	mustHaveTrailer(t, tr, "aiwf-reason", "blocked by E-09")
	// --pause never carries aiwf-to (no agent argument).
	for _, x := range tr {
		if x.Key == "aiwf-to" {
			t.Errorf("aiwf-to present on --pause commit: %q", x.Value)
		}
	}
}

// TestAuthorize_Pause_RequiresReason: --pause with no reason (or
// whitespace-only) is refused before scope state is even consulted.
func TestAuthorize_Pause_RequiresReason(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	_, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:   verb.AuthorizePause,
		Reason: "   ",
		Scopes: []*scope.Scope{
			{AuthSHA: "deadbee", Entity: "E-0001", State: scope.StateActive},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "non-empty reason") {
		t.Errorf("expected non-empty-reason refusal; got %v", err)
	}
}

// TestAuthorize_Resume_HappyPath: with a paused scope on the entity,
// --resume produces an authorize commit with aiwf-scope=resumed.
func TestAuthorize_Resume_HappyPath(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	scopes := []*scope.Scope{
		{AuthSHA: "deadbee", Entity: "E-0001", Agent: "ai/claude", Principal: testActor, State: scope.StatePaused},
	}
	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:   verb.AuthorizeResume,
		Reason: "back to it",
		Scopes: scopes,
	})
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	if applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: %v", applyErr)
	}
	tr, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatal(err)
	}
	mustHaveTrailer(t, tr, "aiwf-scope", "resumed")
	mustHaveTrailer(t, tr, "aiwf-reason", "back to it")
}

// TestAuthorize_Resume_RefusesWithoutPausedScope: --resume with no
// paused scope on the entity is refused.
func TestAuthorize_Resume_RefusesWithoutPausedScope(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	scopes := []*scope.Scope{
		{AuthSHA: "deadbee", Entity: "E-0001", State: scope.StateActive},
	}
	_, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:   verb.AuthorizeResume,
		Reason: "any",
		Scopes: scopes,
	})
	if err == nil || !strings.Contains(err.Error(), "no paused scope") {
		t.Errorf("expected no-paused-scope refusal; got %v", err)
	}
}

// TestAuthorize_Pause_PicksMostRecentlyOpenedActive: when multiple
// scopes exist on the entity, --pause picks the most-recently-opened
// active one. Ended/paused scopes are skipped over.
func TestAuthorize_Pause_PicksMostRecentlyOpenedActive(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	// Scopes ordered oldest-first: ended, paused, active.
	scopes := []*scope.Scope{
		{AuthSHA: "first11", Entity: "E-0001", State: scope.StateEnded},
		{AuthSHA: "second2", Entity: "E-0001", State: scope.StatePaused},
		{AuthSHA: "third33", Entity: "E-0001", State: scope.StateActive},
	}
	res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:   verb.AuthorizePause,
		Reason: "context switch",
		Scopes: scopes,
	})
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	// The pick is implicit (we don't write the auth-sha into the
	// commit), but if no active existed the verb would've refused.
	// Add a regression on subject + trailers to lock the current
	// behaviour.
	if got, want := res.Plan.Subject, "aiwf authorize E-0001 --pause"; got != want {
		t.Errorf("Subject = %q, want %q", got, want)
	}
}

// TestAuthorize_Open_RefusesUnknownEntity: an id that doesn't resolve
// to a tree entity is rejected before any other rule fires.
func TestAuthorize_Open_RefusesUnknownEntity(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	_, err := verb.Authorize(r.ctx, r.tree(), "E-0099", testActor, verb.AuthorizeOptions{
		Mode:  verb.AuthorizeOpen,
		Agent: "ai/claude",
	})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found refusal; got %v", err)
	}
}

// TestAuthorize_Open_PauseResumeCycleE2E: full cycle with the cmd-
// loaded scope state in mind. Open → pause → resume → pause → resume.
// Each transition reads the scopes the previous transition left
// behind, simulated here by manually flipping the state on the in-
// memory scope.
func TestAuthorize_Open_PauseResumeCycleE2E(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "begin", false, verb.PromoteOptions{}))

	// Open. CurrentBranch satisfies M-0103's preflight.
	open, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
		Mode:          verb.AuthorizeOpen,
		Agent:         "ai/claude",
		Reason:        "implement E-01",
		CurrentBranch: "epic/E-0001-engine",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := verb.Apply(r.ctx, r.root, open.Plan); err != nil {
		t.Fatal(err)
	}

	// Simulate a fresh scope (cmd would loadEntityScopes from git here).
	s := &scope.Scope{AuthSHA: "openSHA", Entity: "E-0001", State: scope.StateActive}
	scopes := []*scope.Scope{s}

	for _, step := range []struct {
		mode   verb.AuthorizeMode
		reason string
		next   scope.State
	}{
		{verb.AuthorizePause, "pause-1", scope.StatePaused},
		{verb.AuthorizeResume, "resume-1", scope.StateActive},
		{verb.AuthorizePause, "pause-2", scope.StatePaused},
		{verb.AuthorizeResume, "resume-2", scope.StateActive},
	} {
		res, err := verb.Authorize(r.ctx, r.tree(), "E-0001", testActor, verb.AuthorizeOptions{
			Mode:   step.mode,
			Reason: step.reason,
			Scopes: scopes,
		})
		if err != nil {
			t.Fatalf("step %s: %v", step.reason, err)
		}
		if err := verb.Apply(r.ctx, r.root, res.Plan); err != nil {
			t.Fatalf("step %s apply: %v", step.reason, err)
		}
		s.State = step.next
	}
}
