package verb_test

import (
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
	"github.com/23min/ai-workflow-v2/tools/internal/scope"
	"github.com/23min/ai-workflow-v2/tools/internal/verb"
)

// TestAuthorize_Open_HappyPath: a human authorizes an agent to operate
// on an active epic. The opener commit lands with the full I2.5
// trailer set: aiwf-verb=authorize, aiwf-entity=E-01, aiwf-actor=human/...,
// aiwf-to=ai/claude, aiwf-scope=opened, plus the reason in the body
// and an aiwf-reason: trailer.
func TestAuthorize_Open_HappyPath(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "active", testActor, "begin", false))

	res, err := verb.Authorize(r.ctx, r.tree(), "E-01", testActor, verb.AuthorizeOptions{
		Mode:   verb.AuthorizeOpen,
		Agent:  "ai/claude",
		Reason: "implement E-01",
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
	if got, want := res.Plan.Subject, "aiwf authorize E-01 --to ai/claude"; got != want {
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
	mustHaveTrailer(t, tr, "aiwf-entity", "E-01")
	mustHaveTrailer(t, tr, "aiwf-actor", testActor)
	mustHaveTrailer(t, tr, "aiwf-to", "ai/claude")
	mustHaveTrailer(t, tr, "aiwf-scope", "opened")
	mustHaveTrailer(t, tr, "aiwf-reason", "implement E-01")
}

// TestAuthorize_Open_NoReason: --reason is optional for --to. The
// commit lands without an aiwf-reason: trailer when none is supplied.
func TestAuthorize_Open_NoReason(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "active", testActor, "begin", false))

	res, err := verb.Authorize(r.ctx, r.tree(), "E-01", testActor, verb.AuthorizeOptions{
		Mode:  verb.AuthorizeOpen,
		Agent: "ai/claude",
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

// TestAuthorize_Open_RefusesNonHumanActor: only human/... actors may
// authorize. An ai/... or bot/... actor is refused at the verb gate.
func TestAuthorize_Open_RefusesNonHumanActor(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "active", testActor, "begin", false))

	_, err := verb.Authorize(r.ctx, r.tree(), "E-01", "ai/claude", verb.AuthorizeOptions{
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

// TestAuthorize_Open_RefusesTerminalEntity: a `done` or `cancelled`
// epic refuses --to without --force.
func TestAuthorize_Open_RefusesTerminalEntity(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "active", testActor, "begin", false))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "done", testActor, "ship", false))

	_, err := verb.Authorize(r.ctx, r.tree(), "E-01", testActor, verb.AuthorizeOptions{
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
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "active", testActor, "begin", false))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "done", testActor, "ship", false))

	res, err := verb.Authorize(r.ctx, r.tree(), "E-01", testActor, verb.AuthorizeOptions{
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
// (or with whitespace-only reason) is refused.
func TestAuthorize_Open_ForceRequiresReason(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "active", testActor, "begin", false))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "done", testActor, "ship", false))

	_, err := verb.Authorize(r.ctx, r.tree(), "E-01", testActor, verb.AuthorizeOptions{
		Mode:  verb.AuthorizeOpen,
		Agent: "ai/claude",
		Force: true,
	})
	if err == nil || !strings.Contains(err.Error(), "--reason") {
		t.Errorf("expected --reason refusal; got %v", err)
	}
}

// TestAuthorize_Open_RequiresAgent: --to without an agent argument
// (or with whitespace-only) is refused.
func TestAuthorize_Open_RequiresAgent(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "active", testActor, "begin", false))

	_, err := verb.Authorize(r.ctx, r.tree(), "E-01", testActor, verb.AuthorizeOptions{
		Mode: verb.AuthorizeOpen,
	})
	if err == nil || !strings.Contains(err.Error(), "agent") {
		t.Errorf("expected missing-agent refusal; got %v", err)
	}
}

// TestAuthorize_Open_AgentMustBeRoleSlashID: an agent value that
// isn't <role>/<id> is refused.
func TestAuthorize_Open_AgentMustBeRoleSlashID(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "active", testActor, "begin", false))

	_, err := verb.Authorize(r.ctx, r.tree(), "E-01", testActor, verb.AuthorizeOptions{
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
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "active", testActor, "begin", false))

	_, err := verb.Authorize(r.ctx, r.tree(), "E-01", testActor, verb.AuthorizeOptions{
		Mode:   verb.AuthorizePause,
		Reason: "blocked",
	})
	if err == nil || !strings.Contains(err.Error(), "no active scope") {
		t.Errorf("expected no-active-scope refusal; got %v", err)
	}
}

// TestAuthorize_Pause_HappyPath: with one active scope on the entity,
// --pause produces an authorize commit with aiwf-scope=paused and
// aiwf-reason=<text>.
func TestAuthorize_Pause_HappyPath(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "active", testActor, "begin", false))

	scopes := []*scope.Scope{
		{AuthSHA: "deadbee", Entity: "E-01", Agent: "ai/claude", Principal: testActor, State: scope.StateActive},
	}
	res, err := verb.Authorize(r.ctx, r.tree(), "E-01", testActor, verb.AuthorizeOptions{
		Mode:   verb.AuthorizePause,
		Reason: "blocked by E-09",
		Scopes: scopes,
	})
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	if got, want := res.Plan.Subject, "aiwf authorize E-01 --pause"; got != want {
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
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "active", testActor, "begin", false))

	_, err := verb.Authorize(r.ctx, r.tree(), "E-01", testActor, verb.AuthorizeOptions{
		Mode:   verb.AuthorizePause,
		Reason: "   ",
		Scopes: []*scope.Scope{
			{AuthSHA: "deadbee", Entity: "E-01", State: scope.StateActive},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "non-empty reason") {
		t.Errorf("expected non-empty-reason refusal; got %v", err)
	}
}

// TestAuthorize_Resume_HappyPath: with a paused scope on the entity,
// --resume produces an authorize commit with aiwf-scope=resumed.
func TestAuthorize_Resume_HappyPath(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "active", testActor, "begin", false))

	scopes := []*scope.Scope{
		{AuthSHA: "deadbee", Entity: "E-01", Agent: "ai/claude", Principal: testActor, State: scope.StatePaused},
	}
	res, err := verb.Authorize(r.ctx, r.tree(), "E-01", testActor, verb.AuthorizeOptions{
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
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "active", testActor, "begin", false))

	scopes := []*scope.Scope{
		{AuthSHA: "deadbee", Entity: "E-01", State: scope.StateActive},
	}
	_, err := verb.Authorize(r.ctx, r.tree(), "E-01", testActor, verb.AuthorizeOptions{
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
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "active", testActor, "begin", false))

	// Scopes ordered oldest-first: ended, paused, active.
	scopes := []*scope.Scope{
		{AuthSHA: "first11", Entity: "E-01", State: scope.StateEnded},
		{AuthSHA: "second2", Entity: "E-01", State: scope.StatePaused},
		{AuthSHA: "third33", Entity: "E-01", State: scope.StateActive},
	}
	res, err := verb.Authorize(r.ctx, r.tree(), "E-01", testActor, verb.AuthorizeOptions{
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
	if got, want := res.Plan.Subject, "aiwf authorize E-01 --pause"; got != want {
		t.Errorf("Subject = %q, want %q", got, want)
	}
}

// TestAuthorize_Open_RefusesUnknownEntity: an id that doesn't resolve
// to a tree entity is rejected before any other rule fires.
func TestAuthorize_Open_RefusesUnknownEntity(t *testing.T) {
	r := newRunner(t)
	_, err := verb.Authorize(r.ctx, r.tree(), "E-99", testActor, verb.AuthorizeOptions{
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
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "active", testActor, "begin", false))

	// Open.
	open, err := verb.Authorize(r.ctx, r.tree(), "E-01", testActor, verb.AuthorizeOptions{
		Mode:   verb.AuthorizeOpen,
		Agent:  "ai/claude",
		Reason: "implement E-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := verb.Apply(r.ctx, r.root, open.Plan); err != nil {
		t.Fatal(err)
	}

	// Simulate a fresh scope (cmd would loadEntityScopes from git here).
	s := &scope.Scope{AuthSHA: "openSHA", Entity: "E-01", State: scope.StateActive}
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
		res, err := verb.Authorize(r.ctx, r.tree(), "E-01", testActor, verb.AuthorizeOptions{
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
