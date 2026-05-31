package check

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/scope"
)

// commitWithVerb builds a scope.Commit carrying a single aiwf-verb
// trailer plus the minimum trailers the test needs. Other trailers
// can be appended.
func commitWithVerb(sha, verb string, extra ...gitops.Trailer) scope.Commit {
	t := []gitops.Trailer{}
	if verb != "" {
		t = append(t, gitops.Trailer{Key: gitops.TrailerVerb, Value: verb})
	}
	t = append(t, extra...)
	return scope.Commit{SHA: sha, Trailers: t}
}

// TestRunTrailerVerbUnknown_FiresOnFabricatedVerb pins the gap's
// canonical failure mode: a hand-rolled commit carrying an
// `aiwf-verb: <not-a-real-verb>` trailer must surface a
// trailer-verb-unknown warning. The gap's worked example was
// `aiwf-verb: implement`.
//
// Closes G-0150.
func TestRunTrailerVerbUnknown_FiresOnFabricatedVerb(t *testing.T) {
	t.Parallel()
	registered := map[string]struct{}{
		"add":     {},
		"promote": {},
	}
	commits := []scope.Commit{
		commitWithVerb("aaa1111", "implement"), // fabricated
	}
	got := RunTrailerVerbUnknown(commits, registered, nil)
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1", len(got))
	}
	f := got[0]
	if f.Code != CodeTrailerVerbUnknown {
		t.Errorf("Code = %q, want %q", f.Code, CodeTrailerVerbUnknown)
	}
	if f.Severity != SeverityWarning {
		t.Errorf("Severity = %q, want %q (advisory per gap)", f.Severity, SeverityWarning)
	}
	if !strings.Contains(f.Message, "implement") {
		t.Errorf("Message must name the offending value; got %q", f.Message)
	}
}

// TestRunTrailerVerbUnknown_SilentOnRegisteredVerbs asserts the
// happy path: every commit's aiwf-verb is a registered verb,
// so no finding fires.
func TestRunTrailerVerbUnknown_SilentOnRegisteredVerbs(t *testing.T) {
	t.Parallel()
	registered := map[string]struct{}{
		"add":                  {},
		"promote":              {},
		"edit-body":            {},
		"milestone-depends-on": {}, // sub-command, hyphen-joined
		"render-roadmap":       {}, // sub-command
	}
	commits := []scope.Commit{
		commitWithVerb("aaa1", "add"),
		commitWithVerb("aaa2", "promote"),
		commitWithVerb("aaa3", "edit-body"),
		commitWithVerb("aaa4", "milestone-depends-on"),
		commitWithVerb("aaa5", "render-roadmap"),
	}
	if got := RunTrailerVerbUnknown(commits, registered, nil); len(got) != 0 {
		for i := range got {
			t.Logf("unexpected: %s — %s", got[i].Code, got[i].Message)
		}
		t.Fatalf("findings = %d, want 0", len(got))
	}
}

// TestRunTrailerVerbUnknown_SkipsCommitsWithoutAiwfVerb asserts that
// pre-aiwf or plain Conventional-Commits commits (no aiwf-verb
// trailer) are silent — the rule only fires when the key is present.
func TestRunTrailerVerbUnknown_SkipsCommitsWithoutAiwfVerb(t *testing.T) {
	t.Parallel()
	registered := map[string]struct{}{"add": {}}
	commits := []scope.Commit{
		{SHA: "noaiwf", Trailers: []gitops.Trailer{
			{Key: gitops.TrailerActor, Value: "human/peter"},
		}},
		{SHA: "plain"}, // no trailers at all
	}
	if got := RunTrailerVerbUnknown(commits, registered, nil); len(got) != 0 {
		t.Fatalf("findings = %d, want 0 (no aiwf-verb trailer present)", len(got))
	}
}

// TestRunTrailerVerbUnknown_EmptyValueIsSilent asserts that a
// commit with an empty aiwf-verb value doesn't fire — that's a
// different shape pinned by other rules; we don't double-report.
func TestRunTrailerVerbUnknown_EmptyValueIsSilent(t *testing.T) {
	t.Parallel()
	registered := map[string]struct{}{"add": {}}
	commits := []scope.Commit{
		{SHA: "empty", Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: ""},
		}},
	}
	if got := RunTrailerVerbUnknown(commits, registered, nil); len(got) != 0 {
		t.Fatalf("findings = %d, want 0 (empty value is a different rule's domain)", len(got))
	}
}

// TestRunTrailerVerbUnknown_EmptyRegisteredSetIsSilent guards
// against a misconfigured caller that fails to enumerate the verb
// tree: the rule must produce no findings rather than flag every
// commit as unknown. This is a load-bearing safety: the verb
// enumeration runs at RunE time and could in principle return an
// empty set if the cobra tree isn't wired up; we'd rather skip
// than flood.
func TestRunTrailerVerbUnknown_EmptyRegisteredSetIsSilent(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{commitWithVerb("aaa", "implement")}
	if got := RunTrailerVerbUnknown(commits, nil, nil); len(got) != 0 {
		t.Fatalf("findings = %d, want 0 (empty registry → skip rather than flood)", len(got))
	}
	if got := RunTrailerVerbUnknown(commits, map[string]struct{}{}, nil); len(got) != 0 {
		t.Fatalf("findings = %d, want 0 (empty registry → skip rather than flood)", len(got))
	}
}

// TestRunTrailerVerbUnknown_MultipleCommitsOneFindingEach pins the
// per-commit emission shape — N commits with N distinct bogus
// values produce N findings.
func TestRunTrailerVerbUnknown_MultipleCommitsOneFindingEach(t *testing.T) {
	t.Parallel()
	registered := map[string]struct{}{"add": {}, "promote": {}}
	commits := []scope.Commit{
		commitWithVerb("aaa", "implement"),
		commitWithVerb("bbb", "feat"),
		commitWithVerb("ccc", "add"), // valid; not counted
		commitWithVerb("ddd", "test"),
	}
	got := RunTrailerVerbUnknown(commits, registered, nil)
	if len(got) != 3 {
		t.Fatalf("findings = %d, want 3 (implement, feat, test)", len(got))
	}
	wantValues := map[string]bool{"implement": false, "feat": false, "test": false}
	for _, f := range got {
		for v := range wantValues {
			if strings.Contains(f.Message, "\""+v+"\"") {
				wantValues[v] = true
			}
		}
	}
	for v, seen := range wantValues {
		if !seen {
			t.Errorf("missing finding for fabricated value %q", v)
		}
	}
}

// TestRunTrailerVerbUnknown_SilentOnRitualVerbs pins G-0180: ritual
// lifecycle verbs that the aiwf-extensions rituals stamp as
// `aiwf-verb:` values are recognized even though they are not kernel
// Cobra verbs — so a trailered epic-wrap merge does not trip the
// finding. A genuinely-fabricated verb alongside them still fires
// (the allowlist does not weaken the fabrication guard).
//
// Per G-0190, the allowlist is derived from the embedded ritual
// snapshot, so the verb set under test mirrors what the rituals
// actually stamp today (`wrap-epic` is the only literal --trailer
// "aiwf-verb: ..." present in the snapshot). The fixture passes the
// verb in to assert the silent path; the auto-derive is exercised in
// skills.TestRitualTrailerVerbs_DerivedFromEmbedded.
func TestRunTrailerVerbUnknown_SilentOnRitualVerbs(t *testing.T) {
	t.Parallel()
	registered := map[string]struct{}{"add": {}, "promote": {}}
	rituals := map[string]struct{}{"wrap-epic": {}}
	commits := []scope.Commit{
		commitWithVerb("rit1", "wrap-epic"), // ritual verb — derived from embedded
		commitWithVerb("kvb1", "promote"),   // kernel verb
		commitWithVerb("bad1", "implement"), // fabricated — must still fire
	}
	got := RunTrailerVerbUnknown(commits, registered, rituals)
	if len(got) != 1 {
		for i := range got {
			t.Logf("finding: %s", got[i].Message)
		}
		t.Fatalf("findings = %d, want 1 (only the fabricated `implement`)", len(got))
	}
	if !strings.Contains(got[0].Message, "implement") {
		t.Errorf("the surviving finding must be the fabricated verb; got %q", got[0].Message)
	}
}
