package integration

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/history"
	"github.com/23min/aiwf/internal/cli/show"
	"github.com/23min/aiwf/internal/scope"
)

// scope_grep_guard_test.go — E-0054 / M-0223. Pins the read-verb guard on
// the repo-wide authorize-opener grep for `aiwf history` and `aiwf show`.
//
// The fixture is a real git repo whose commits carry the exact trailer sets
// the mutating verbs emit (opener: aiwf-verb: authorize + aiwf-scope: opened;
// source-(a) work: aiwf-authorized-by + aiwf-on-behalf-of; terminal promote:
// aiwf-scope-ends). The read paths under test consume `git log` output, so
// hand-authoring the commit messages exercises exactly the boundary that
// matters while keeping every scope state deterministic. The real authorize
// verb's trailer format is separately pinned by
// TestRunAuthorize_OpenPauseResumeRoundTrip.

// scopeGuardRepo is a fixture repo covering every scope shape M-0223's guard
// must preserve. openerE0002 is the active direct-scope opener's SHA (E-0003
// is worked under it); the returned root holds all entities below.
type scopeGuardRepo struct {
	root        string
	openerE0002 string // active opener on E-0002 (source (a) auth SHA for E-0003)
	openerE0004 string // opener on E-0004, later ended
	openerE0005 string // self-scope opener on E-0005 (E-0005 also works under it)
	openerE14   string // narrow-width opener (aiwf-entity: E-14)
}

func buildScopeGuardRepo(t *testing.T) scopeGuardRepo {
	t.Helper()
	root := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
		{"config", "commit.gpgsign", "false"},
	} {
		if out, err := testutil.RunGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	commit := func(msg string) string {
		if out, err := testutil.RunGit(root, "commit", "--allow-empty", "-m", msg); err != nil {
			t.Fatalf("git commit: %v\n%s", err, out)
		}
		out, err := testutil.RunGit(root, "rev-parse", "HEAD")
		if err != nil {
			t.Fatalf("git rev-parse: %v\n%s", err, out)
		}
		return strings.TrimSpace(out)
	}

	// (i) E-0001 — scopeless entity (guard skips the grep).
	commit("add E-0001\n\naiwf-verb: add\naiwf-entity: E-0001\naiwf-actor: human/peter\n")

	// (iii-support / active opener) E-0002 — active direct-scope opener.
	commit("add E-0002\n\naiwf-verb: add\naiwf-entity: E-0002\naiwf-actor: human/peter\n")
	openerE0002 := commit("authorize E-0002 --to ai/claude\n\n" +
		"aiwf-verb: authorize\naiwf-entity: E-0002\naiwf-actor: human/peter\n" +
		"aiwf-to: ai/claude\naiwf-scope: opened\n")

	// (ii) E-0003 — worked under E-0002's scope: has aiwf-authorized-by
	// (source (a); the global grep must run for show).
	commit("add E-0003\n\naiwf-verb: add\naiwf-entity: E-0003\naiwf-actor: human/peter\n")
	commit(fmt.Sprintf("promote E-0003 active\n\n"+
		"aiwf-verb: promote\naiwf-entity: E-0003\naiwf-actor: ai/claude\n"+
		"aiwf-to: active\naiwf-on-behalf-of: human/peter\naiwf-authorized-by: %s\n", openerE0002))

	// (iv) E-0004 — scope opened then ended (aiwf-scope-ends present, no
	// aiwf-authorized-by: history needs the grep, show does not).
	commit("add E-0004\n\naiwf-verb: add\naiwf-entity: E-0004\naiwf-actor: human/peter\n")
	openerE0004 := commit("authorize E-0004 --to ai/claude\n\n" +
		"aiwf-verb: authorize\naiwf-entity: E-0004\naiwf-actor: human/peter\n" +
		"aiwf-to: ai/claude\naiwf-scope: opened\n")
	commit(fmt.Sprintf("promote E-0004 done\n\n"+
		"aiwf-verb: promote\naiwf-entity: E-0004\naiwf-actor: human/peter\n"+
		"aiwf-to: done\naiwf-scope-ends: %s\n", openerE0004))

	// E-0005 — self-scope: opens a scope on itself, then does work under
	// that same scope (aiwf-authorized-by points at its own opener). This
	// is the common case where an opener also works within its own scope;
	// it exercises the `ent == id` guard in LoadEntityScopeViews that keeps
	// the own scope from being counted twice (source (b) + source (a)).
	commit("add E-0005\n\naiwf-verb: add\naiwf-entity: E-0005\naiwf-actor: human/peter\n")
	openerE0005 := commit("authorize E-0005 --to ai/claude\n\n" +
		"aiwf-verb: authorize\naiwf-entity: E-0005\naiwf-actor: human/peter\n" +
		"aiwf-to: ai/claude\naiwf-scope: opened\n")
	commit(fmt.Sprintf("promote E-0005 active\n\n"+
		"aiwf-verb: promote\naiwf-entity: E-0005\naiwf-actor: ai/claude\n"+
		"aiwf-to: active\naiwf-on-behalf-of: human/peter\naiwf-authorized-by: %s\n", openerE0005))

	// Width fix — E-14 opener authored at legacy narrow width.
	commit("add E-14\n\naiwf-verb: add\naiwf-entity: E-14\naiwf-actor: human/peter\n")
	openerE14 := commit("authorize E-14 --to ai/claude\n\n" +
		"aiwf-verb: authorize\naiwf-entity: E-14\naiwf-actor: human/peter\n" +
		"aiwf-to: ai/claude\naiwf-scope: opened\n")

	return scopeGuardRepo{
		root:        root,
		openerE0002: openerE0002,
		openerE0004: openerE0004,
		openerE0005: openerE0005,
		openerE14:   openerE14,
	}
}

// TestScopeGuard_ShowDerivationForActiveOpener is M-0223 AC-1's show seam:
// for an active direct-scope opener (an authorize event in its own stream
// with no aiwf-authorized-by), the global grep is skipped — HasAuthorizedBy
// returns false — AND the direct-scope derivation (cliutil.LoadEntityScopes)
// still returns the opener's own active scope. Omitting either half is how
// an AuthorizedBy-only guard would silently drop a live scope table.
func TestScopeGuard_ShowDerivationForActiveOpener(t *testing.T) {
	t.Parallel()
	repo := buildScopeGuardRepo(t)
	ctx := context.Background()

	events, err := history.ReadHistory(ctx, repo.root, "E-0002")
	if err != nil {
		t.Fatalf("ReadHistory(E-0002): %v", err)
	}
	if history.HasAuthorizedBy(events) {
		t.Errorf("HasAuthorizedBy(E-0002 events) = true, want false — an active opener has no aiwf-authorized-by, so show must not run the global grep")
	}

	scopes, err := cliutil.LoadEntityScopes(ctx, repo.root, "E-0002")
	if err != nil {
		t.Fatalf("LoadEntityScopes(E-0002): %v", err)
	}
	if len(scopes) != 1 {
		t.Fatalf("LoadEntityScopes(E-0002) len = %d, want 1 (the direct derivation must find the opener's own scope)", len(scopes))
	}
	if scopes[0].State != scope.StateActive {
		t.Errorf("scope state = %q, want %q", scopes[0].State, scope.StateActive)
	}
	if scopes[0].AuthSHA != repo.openerE0002 {
		t.Errorf("scope AuthSHA = %q, want opener %q", scopes[0].AuthSHA, repo.openerE0002)
	}
}

// TestScopeGuard_ShowSkipsGrepForScopeEndedEntity pins the other half of the
// asymmetry M-0223 AC-1 names: a scope-ended entity (aiwf-scope-ends present,
// no aiwf-authorized-by) still skips show's global grep — HasAuthorizedBy is
// false — while its own ended scope comes from the direct derivation. history
// treats the same entity differently (HasScopeData is true for scope-ends),
// which the predicate tests in the history package pin.
func TestScopeGuard_ShowSkipsGrepForScopeEndedEntity(t *testing.T) {
	t.Parallel()
	repo := buildScopeGuardRepo(t)
	ctx := context.Background()

	events, err := history.ReadHistory(ctx, repo.root, "E-0004")
	if err != nil {
		t.Fatalf("ReadHistory(E-0004): %v", err)
	}
	if history.HasAuthorizedBy(events) {
		t.Errorf("HasAuthorizedBy(E-0004 events) = true, want false — scope-ends alone must not trigger show's global grep")
	}
	if !history.HasScopeData(events) {
		t.Errorf("HasScopeData(E-0004 events) = false, want true — history resolves the [ended] chip via the grep")
	}

	scopes, err := cliutil.LoadEntityScopes(ctx, repo.root, "E-0004")
	if err != nil {
		t.Fatalf("LoadEntityScopes(E-0004): %v", err)
	}
	if len(scopes) != 1 || scopes[0].State != scope.StateEnded {
		t.Fatalf("LoadEntityScopes(E-0004) = %+v, want one ended scope", scopes)
	}
}

// unguardedScopeViews is a faithful copy of the pre-M-0223
// show.LoadEntityScopeViews algorithm: it runs the global authorize-opener
// grep unconditionally and derives source (b) — scopes opened on id — by
// filtering that global map to `ent == id` (the raw-id comparison that
// carried the width bug). It is the differential oracle for
// TestScopeGuard_ShowViewsEquivalence: run before the production rewrite it
// proves this copy is faithful (production == oracle); run after, it proves
// the guarded rewrite preserved behavior (new == oracle) for canonical ids.
// It is test-only — the shipped path is the guarded one.
func unguardedScopeViews(t *testing.T, ctx context.Context, root, id string) []show.ScopeView {
	t.Helper()
	if !cliutil.HasCommits(ctx, root) {
		return nil
	}
	events, err := history.ReadHistory(ctx, root, id)
	if err != nil {
		t.Fatalf("ReadHistory(%s): %v", id, err)
	}
	scopeEntityByAuthSHA, err := cliutil.AuthorizeOpeners(ctx, root)
	if err != nil {
		t.Fatalf("AuthorizeOpeners: %v", err)
	}
	interested := map[string]struct{}{}
	for i := range events {
		if events[i].AuthorizedBy != "" {
			interested[events[i].AuthorizedBy] = struct{}{}
		}
	}
	for sha, ent := range scopeEntityByAuthSHA {
		if ent == id {
			interested[sha] = struct{}{}
		}
	}
	if len(interested) == 0 {
		return nil
	}
	scopeEntitiesNeeded := map[string]struct{}{}
	for sha := range interested {
		if ent, ok := scopeEntityByAuthSHA[sha]; ok {
			scopeEntitiesNeeded[ent] = struct{}{}
		}
	}
	var allScopes []*scope.Scope
	for ent := range scopeEntitiesNeeded {
		scopes, err := cliutil.LoadEntityScopes(ctx, root, ent)
		if err != nil {
			t.Fatalf("LoadEntityScopes(%s): %v", ent, err)
		}
		allScopes = append(allScopes, scopes...)
	}
	dateCache := map[string]string{}
	var views []show.ScopeView
	for _, s := range allScopes {
		if _, ok := interested[s.AuthSHA]; !ok {
			continue
		}
		opened := show.LookupCommitDateCached(ctx, root, s.AuthSHA, dateCache)
		var ended string
		if s.State == scope.StateEnded {
			if last := show.LastEventSHA(s, scope.StateEnded); last != "" {
				ended = show.LookupCommitDateCached(ctx, root, last, dateCache)
			}
		}
		views = append(views, show.ScopeView{
			AuthSHA:    s.AuthSHA,
			Entity:     s.Entity,
			Agent:      s.Agent,
			Principal:  s.Principal,
			State:      string(s.State),
			Opened:     opened,
			EndedAt:    ended,
			EventCount: len(s.Events),
		})
	}
	sort.Slice(views, func(i, j int) bool { return views[i].Opened < views[j].Opened })
	return views
}

// TestScopeGuard_ShowViewsEquivalence is M-0223 AC-2(a) for show: the guarded
// LoadEntityScopeViews returns byte-identical scope views to the unguarded
// oracle for every canonical-id fixture entity — scopeless (i), worked under
// a foreign scope (ii), active direct-scope opener (iii), scope-ended (iv).
// Case (iii) is load-bearing: its show scope table must be non-empty, or an
// AuthorizedBy-only guard would silently drop it and the test would pass
// vacuously.
func TestScopeGuard_ShowViewsEquivalence(t *testing.T) {
	t.Parallel()
	repo := buildScopeGuardRepo(t)
	ctx := context.Background()
	for _, id := range []string{"E-0001", "E-0002", "E-0003", "E-0004", "E-0005"} {
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			want := unguardedScopeViews(t, ctx, repo.root, id)
			got, err := show.LoadEntityScopeViews(ctx, repo.root, id)
			if err != nil {
				t.Fatalf("LoadEntityScopeViews(%s): %v", id, err)
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("guarded vs unguarded scope views for %s (-unguarded +guarded):\n%s", id, diff)
			}
		})
	}
	// Case (iii) non-vacuity guard: the active opener's table is non-empty.
	got, err := show.LoadEntityScopeViews(ctx, repo.root, "E-0002")
	if err != nil {
		t.Fatalf("LoadEntityScopeViews(E-0002): %v", err)
	}
	if len(got) == 0 {
		t.Fatal("active direct-scope opener E-0002 has an empty scope table — the equivalence assertion would be vacuous")
	}
	// Self-scope guard: E-0005 opened a scope AND worked under it. Its own
	// scope must appear exactly once, not doubled via source (a) — the
	// `ent == id` guard. cmp.Diff above would catch a duplicate, but pin the
	// count explicitly so the intent is legible.
	selfViews, err := show.LoadEntityScopeViews(ctx, repo.root, "E-0005")
	if err != nil {
		t.Fatalf("LoadEntityScopeViews(E-0005): %v", err)
	}
	if len(selfViews) != 1 {
		t.Errorf("self-scope E-0005 returned %d views, want 1 (no double-count across source (a)/(b))", len(selfViews))
	}
}

// TestAuthorizeOpeners_SkipsOpenerMissingEntity covers AuthorizeOpeners'
// parse guard: a commit matching the opener grep (aiwf-verb: authorize +
// aiwf-scope: opened) but missing aiwf-entity yields a blank entity id and
// must be skipped, not mapped to a blank. A well-formed opener alongside it
// is mapped, so the map holds exactly the valid entry.
func TestAuthorizeOpeners_SkipsOpenerMissingEntity(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
		{"config", "commit.gpgsign", "false"},
	} {
		if out, err := testutil.RunGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	commit := func(msg string) string {
		if out, err := testutil.RunGit(root, "commit", "--allow-empty", "-m", msg); err != nil {
			t.Fatalf("git commit: %v\n%s", err, out)
		}
		out, err := testutil.RunGit(root, "rev-parse", "HEAD")
		if err != nil {
			t.Fatalf("git rev-parse: %v\n%s", err, out)
		}
		return strings.TrimSpace(out)
	}
	// Malformed opener: matches the grep but carries no aiwf-entity.
	commit("authorize (broken) --to ai/claude\n\n" +
		"aiwf-verb: authorize\naiwf-actor: human/peter\naiwf-to: ai/claude\naiwf-scope: opened\n")
	// Well-formed opener.
	valid := commit("authorize E-0009 --to ai/claude\n\n" +
		"aiwf-verb: authorize\naiwf-entity: E-0009\naiwf-actor: human/peter\naiwf-to: ai/claude\naiwf-scope: opened\n")

	got, err := cliutil.AuthorizeOpeners(context.Background(), root)
	if err != nil {
		t.Fatalf("AuthorizeOpeners: %v", err)
	}
	if len(got) != 1 || got[valid] != "E-0009" {
		t.Errorf("AuthorizeOpeners = %v, want exactly {%s: E-0009} (the entity-less opener must be skipped)", got, valid)
	}
}

// TestScopeMapFor_GuardsTheGrep covers the history text path's guard seam
// in-process (M-0223): ScopeMapFor builds the authorize-opener map only when
// the loaded events carry scope data, and returns nil (skipping the grep)
// otherwise. The RunBin wiring test proves the same behavior end-to-end but
// its subprocess coverage isn't captured; this is the in-process branch
// coverage of the guard.
func TestScopeMapFor_GuardsTheGrep(t *testing.T) {
	t.Parallel()
	repo := buildScopeGuardRepo(t)
	ctx := context.Background()

	// Scoped events → the grep runs and the map is built.
	scoped := []history.HistoryEvent{{Verb: "promote", AuthorizedBy: repo.openerE0002}}
	m := history.ScopeMapFor(ctx, repo.root, scoped)
	if m == nil {
		t.Fatal("ScopeMapFor returned nil for events with scope data; want the built map")
	}
	if m[repo.openerE0002] != "E-0002" {
		t.Errorf("built map[%s] = %q, want E-0002", repo.openerE0002, m[repo.openerE0002])
	}

	// Scopeless events → nil, the grep is skipped.
	if got := history.ScopeMapFor(ctx, repo.root, []history.HistoryEvent{{Verb: "add"}}); got != nil {
		t.Errorf("ScopeMapFor returned %v for scopeless events; want nil (grep skipped)", got)
	}
}

// TestScopeGuard_HistoryTextWiring drives the real `aiwf history` text path
// end-to-end (M-0223 AC-2 integration seam): the guard block in history.Run
// runs the authorize grep only for an entity whose events carry scope data.
// A scoped entity (E-0003, worked under E-0002) renders the authorizing-scope
// chip; a scopeless entity (E-0001) renders no chip. This is the wiring proof
// the RenderScopeChips-level differential can't give.
func TestScopeGuard_HistoryTextWiring(t *testing.T) {
	t.Parallel()
	repo := buildScopeGuardRepo(t)
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)

	scopedOut, err := testutil.RunBin(t, repo.root, binDir, nil, "history", "E-0003")
	if err != nil {
		t.Fatalf("aiwf history E-0003: %v\n%s", err, scopedOut)
	}
	if !strings.Contains(scopedOut, "[E-0002 ") {
		t.Errorf("aiwf history E-0003 text missing the authorizing-scope chip [E-0002 ...]:\n%s", scopedOut)
	}

	scopelessOut, err := testutil.RunBin(t, repo.root, binDir, nil, "history", "E-0001")
	if err != nil {
		t.Fatalf("aiwf history E-0001: %v\n%s", err, scopelessOut)
	}
	if strings.Contains(scopelessOut, "[") {
		t.Errorf("aiwf history E-0001 (scopeless) rendered an unexpected chip:\n%s", scopelessOut)
	}
}

// TestScopeGuard_HistoryChipsEquivalence is M-0223 AC-2(a) for history text:
// the only thing the guard changes is the scope-entity map handed to
// RenderScopeChips. For every fixture entity's events, the chip block must be
// identical whether the map is the guarded one (empty when HasScopeData is
// false) or the full one (grep always run). Verifying this across the fixture
// is the empirical proof of the guard's core invariant: when the guard skips
// the grep, no event reads the map.
func TestScopeGuard_HistoryChipsEquivalence(t *testing.T) {
	t.Parallel()
	repo := buildScopeGuardRepo(t)
	ctx := context.Background()
	full, err := cliutil.AuthorizeOpeners(ctx, repo.root)
	if err != nil {
		t.Fatalf("AuthorizeOpeners: %v", err)
	}
	for _, id := range []string{"E-0001", "E-0002", "E-0003", "E-0004"} {
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			events, err := history.ReadHistory(ctx, repo.root, id)
			if err != nil {
				t.Fatalf("ReadHistory(%s): %v", id, err)
			}
			// The guarded map is exactly what history.Run builds: the grep
			// runs only when the loaded events carry scope data.
			var guarded map[string]string
			if history.HasScopeData(events) {
				guarded = full
			}
			for i := range events {
				for _, showAuth := range []bool{false, true} {
					wantChip := history.RenderScopeChips(events[i], full, showAuth)
					gotChip := history.RenderScopeChips(events[i], guarded, showAuth)
					if wantChip != gotChip {
						t.Errorf("%s event %d (showAuth=%v): guarded chip %q != full chip %q",
							id, i, showAuth, gotChip, wantChip)
					}
				}
			}
		})
	}
}

// TestScopeGuard_ShowWidthFix is M-0223 AC-2(b): querying a legacy
// narrow-width opener by its narrow id must render the scope table. The
// pre-M-0223 raw-id path compared a canonicalized map value (E-0014) against
// the raw query id (E-14) and silently dropped the table; deriving source (b)
// from the width-tolerant cliutil.LoadEntityScopes corrects it. Asserted
// one-sided (the unguarded raw-id path is the bug, not the oracle).
func TestScopeGuard_ShowWidthFix(t *testing.T) {
	t.Parallel()
	repo := buildScopeGuardRepo(t)
	ctx := context.Background()
	got, err := show.LoadEntityScopeViews(ctx, repo.root, "E-14")
	if err != nil {
		t.Fatalf("LoadEntityScopeViews(E-14): %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("LoadEntityScopeViews(E-14) len = %d, want 1 — narrow-id query must render the opener's scope table", len(got))
	}
	if got[0].AuthSHA != repo.openerE14 {
		t.Errorf("scope AuthSHA = %q, want narrow opener %q", got[0].AuthSHA, repo.openerE14)
	}
}
