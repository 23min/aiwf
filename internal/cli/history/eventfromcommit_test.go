package history

import (
	"testing"

	"github.com/23min/aiwf/internal/gitops"
)

// TestEventFromCommit_FullPromote covers the happy path: every mapped
// field, the prose body derived from %B with the trailer block stripped,
// and the parsed test metrics.
func TestEventFromCommit_FullPromote(t *testing.T) {
	t.Parallel()
	body := "feat(x): a subject\n\nsome prose reason\n\naiwf-verb: promote\naiwf-actor: human/peter\n"
	trailers := []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "promote"},
		{Key: gitops.TrailerActor, Value: "human/peter"},
		{Key: gitops.TrailerTo, Value: "done"},
		{Key: gitops.TrailerForce, Value: "legacy migration"},
		{Key: gitops.TrailerAuditOnly, Value: "backfill"},
		{Key: gitops.TrailerPrincipal, Value: "human/peter"},
		{Key: gitops.TrailerOnBehalfOf, Value: "human/peter"},
		{Key: gitops.TrailerReason, Value: "manual recovery"},
		{Key: gitops.TrailerTests, Value: "pass=12 fail=0 skip=1 total=13"},
	}
	ev, ok := EventFromCommit("abcdef1234567890", "2026-07-03T00:00:00Z", "feat(x): a subject", body, trailers)
	if !ok {
		t.Fatal("EventFromCommit ok=false, want true for a real promote event")
	}
	if ev.Commit != "abcdef1" {
		t.Errorf("Commit = %q, want short hash abcdef1", ev.Commit)
	}
	if ev.Date != "2026-07-03T00:00:00Z" || ev.Detail != "feat(x): a subject" {
		t.Errorf("Date/Detail = (%q, %q)", ev.Date, ev.Detail)
	}
	if ev.Verb != "promote" || ev.Actor != "human/peter" || ev.To != "done" {
		t.Errorf("verb/actor/to = (%q, %q, %q)", ev.Verb, ev.Actor, ev.To)
	}
	if ev.Force != "legacy migration" || ev.AuditOnly != "backfill" ||
		ev.Principal != "human/peter" || ev.OnBehalfOf != "human/peter" || ev.Reason != "manual recovery" {
		t.Errorf("force/audit/principal/onbehalf/reason = (%q, %q, %q, %q, %q)",
			ev.Force, ev.AuditOnly, ev.Principal, ev.OnBehalfOf, ev.Reason)
	}
	if ev.Body != "some prose reason" {
		t.Errorf("Body = %q, want %q (subject dropped, trailers stripped)", ev.Body, "some prose reason")
	}
	if ev.Tests == nil || ev.Tests.Pass != 12 || ev.Tests.Skip != 1 || ev.Tests.TotalOrDerive() != 13 {
		t.Errorf("Tests = %+v, want pass=12 skip=1 total=13", ev.Tests)
	}
}

// TestEventFromCommit_ProseMentionSkipped pins the G30 skip: a commit
// with an aiwf-entity trailer but neither aiwf-verb nor aiwf-actor is a
// grep false-positive, not a real event — ok=false.
func TestEventFromCommit_ProseMentionSkipped(t *testing.T) {
	t.Parallel()
	trailers := []gitops.Trailer{{Key: gitops.TrailerEntity, Value: "M-0001"}}
	if ev, ok := EventFromCommit("sha", "2026-07-03T00:00:00Z", "prose mentioning M-0001", "body", trailers); ok {
		t.Errorf("EventFromCommit ok=true for a prose-mention commit; want false (got %+v)", ev)
	}
}

// TestEventFromCommit_ScopeEndsAndNoBody covers two edge arms together:
// repeating aiwf-scope-ends collect into the slice in order, and a
// subject-only message (no blank line) yields an empty body.
func TestEventFromCommit_ScopeEndsAndNoBody(t *testing.T) {
	t.Parallel()
	trailers := []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "promote"},
		{Key: gitops.TrailerActor, Value: "human/peter"},
		{Key: gitops.TrailerScopeEnds, Value: "authA"},
		{Key: gitops.TrailerScopeEnds, Value: "  "}, // blank value → skipped
		{Key: gitops.TrailerScopeEnds, Value: "authB"},
	}
	// Subject only, no blank line → no body.
	ev, ok := EventFromCommit("sha", "2026-07-03T00:00:00Z", "promote M-0001 done", "promote M-0001 done", trailers)
	if !ok {
		t.Fatal("ok=false, want true")
	}
	if ev.Body != "" {
		t.Errorf("Body = %q, want empty (subject-only message)", ev.Body)
	}
	if len(ev.ScopeEnds) != 2 || ev.ScopeEnds[0] != "authA" || ev.ScopeEnds[1] != "authB" {
		t.Errorf("ScopeEnds = %v, want [authA authB] in order", ev.ScopeEnds)
	}
	if ev.Tests != nil {
		t.Errorf("Tests = %+v, want nil (no aiwf-tests trailer)", ev.Tests)
	}
}
