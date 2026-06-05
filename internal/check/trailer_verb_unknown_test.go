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
	got := RunTrailerVerbUnknown(commits, registered, nil, nil, nil)
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1", len(got))
	}
	f := got[0]
	if f.Code != CodeTrailerVerbUnknown {
		t.Errorf("Code = %q, want %q", f.Code, CodeTrailerVerbUnknown)
	}
	if f.Severity != SeverityWarning {
		t.Errorf("Severity = %q, want %q (advisory per gap, nil postCutoffSHAs)", f.Severity, SeverityWarning)
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
	if got := RunTrailerVerbUnknown(commits, registered, nil, nil, nil); len(got) != 0 {
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
	if got := RunTrailerVerbUnknown(commits, registered, nil, nil, nil); len(got) != 0 {
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
	if got := RunTrailerVerbUnknown(commits, registered, nil, nil, nil); len(got) != 0 {
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
	if got := RunTrailerVerbUnknown(commits, nil, nil, nil, nil); len(got) != 0 {
		t.Fatalf("findings = %d, want 0 (empty registry → skip rather than flood)", len(got))
	}
	if got := RunTrailerVerbUnknown(commits, map[string]struct{}{}, nil, nil, nil); len(got) != 0 {
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
	got := RunTrailerVerbUnknown(commits, registered, nil, nil, nil)
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
	got := RunTrailerVerbUnknown(commits, registered, rituals, nil, nil)
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

// TestRunTrailerVerbUnknown_PostCutoffEmitsError pins G-0218 Patch 2:
// a fabricated `aiwf-verb:` value on a commit whose SHA is in the
// postCutoffSHAs set (i.e. descends from HookInstallSHA) emits at
// SeverityError with a remediation hint. The commit-msg hook would
// have refused this at composition time; landing it requires
// `--no-verify` or git plumbing, which is a policy violation.
func TestRunTrailerVerbUnknown_PostCutoffEmitsError(t *testing.T) {
	t.Parallel()
	registered := map[string]struct{}{"add": {}, "promote": {}}
	commits := []scope.Commit{
		commitWithVerb("postcut1", "implement"), // fabricated AND post-hook
	}
	postCutoff := map[string]bool{
		"postcut1": true,
	}
	got := RunTrailerVerbUnknown(commits, registered, nil, nil, postCutoff)
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1", len(got))
	}
	f := got[0]
	if f.Code != CodeTrailerVerbUnknown {
		t.Errorf("Code = %q, want %q", f.Code, CodeTrailerVerbUnknown)
	}
	if f.Severity != SeverityError {
		t.Errorf("Severity = %q, want %q (post-cutoff per G-0218 Patch 2)", f.Severity, SeverityError)
	}
	if f.Hint == "" {
		t.Error("post-cutoff finding must carry a remediation Hint naming the commit-msg hook")
	}
	if !strings.Contains(f.Hint, "commit-msg hook") {
		t.Errorf("Hint must name the commit-msg hook; got %q", f.Hint)
	}
	if !strings.Contains(f.Hint, "--no-verify") {
		t.Errorf("Hint must reference the bypass mechanism (--no-verify or plumbing); got %q", f.Hint)
	}
}

// TestRunTrailerVerbUnknown_PreCutoffStaysWarning pins the
// backward-compat half of G-0218 Patch 2: a fabricated `aiwf-verb:`
// value on a commit whose SHA is NOT in the postCutoffSHAs set stays
// at SeverityWarning with no Hint — same shape G-0150 shipped, so
// pre-hook trunk history isn't retroactively broken.
func TestRunTrailerVerbUnknown_PreCutoffStaysWarning(t *testing.T) {
	t.Parallel()
	registered := map[string]struct{}{"add": {}, "promote": {}}
	commits := []scope.Commit{
		commitWithVerb("oldsha", "implement"),
	}
	// Non-empty postCutoff that does NOT include the offending commit.
	// Pins that the predicate is per-SHA, not a global on/off switch.
	postCutoff := map[string]bool{
		"unrelated-post-cutoff-sha": true,
	}
	got := RunTrailerVerbUnknown(commits, registered, nil, nil, postCutoff)
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1", len(got))
	}
	f := got[0]
	if f.Severity != SeverityWarning {
		t.Errorf("Severity = %q, want %q (pre-cutoff stays at warning)", f.Severity, SeverityWarning)
	}
	if f.Hint != "" {
		t.Errorf("pre-cutoff finding must NOT carry the post-cutoff Hint; got %q", f.Hint)
	}
}

// TestRunTrailerVerbUnknown_AckedSilencesEvenPostCutoff guards the
// composition order documented in the rule's docstring: an explicit
// `aiwf acknowledge-illegal <sha>` silences the finding regardless
// of post-cutoff status. The ack is sovereign; severity tightening
// does not bypass it.
func TestRunTrailerVerbUnknown_AckedSilencesEvenPostCutoff(t *testing.T) {
	t.Parallel()
	registered := map[string]struct{}{"add": {}, "promote": {}}
	commits := []scope.Commit{
		commitWithVerb("postcut2", "implement"),
	}
	ack := map[string]bool{"postcut2": true}
	postCutoff := map[string]bool{"postcut2": true}
	got := RunTrailerVerbUnknown(commits, registered, nil, ack, postCutoff)
	if len(got) != 0 {
		t.Fatalf("findings = %d, want 0 (ack overrides post-cutoff severity transition); got %+v", len(got), got)
	}
}

// TestRunTrailerVerbUnknown_NilPostCutoffFallsBackToWarning pins the
// graceful-degrade contract for shallow clones, forks that diverged
// before HookInstallSHA, or any state where the gather layer's
// `git rev-list HookInstallSHA..HEAD` returns an empty set: every
// finding emits at SeverityWarning (the G-0150 baseline). Without
// this fallback, every operator working with a clone where the
// cutoff SHA is unreachable would see retroactive errors on
// pre-existing fabrications.
func TestRunTrailerVerbUnknown_NilPostCutoffFallsBackToWarning(t *testing.T) {
	t.Parallel()
	registered := map[string]struct{}{"add": {}, "promote": {}}
	commits := []scope.Commit{
		commitWithVerb("anysha", "implement"),
	}
	got := RunTrailerVerbUnknown(commits, registered, nil, nil, nil)
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1", len(got))
	}
	if got[0].Severity != SeverityWarning {
		t.Errorf("Severity = %q, want %q (nil postCutoff → warning baseline)", got[0].Severity, SeverityWarning)
	}
	// Empty map (different from nil but same semantically) — same shape.
	got = RunTrailerVerbUnknown(commits, registered, nil, nil, map[string]bool{})
	if len(got) != 1 || got[0].Severity != SeverityWarning {
		t.Errorf("empty postCutoff map should also degrade to warning; got %+v", got)
	}
}
