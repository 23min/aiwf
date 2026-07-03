package render

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/history"
	"github.com/23min/aiwf/internal/cli/show"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
)

// TestHistoryBucketKeys pins the bucket-key derivation directly: canonical
// width, the composite M-NNNN/AC-N fold into the parent milestone, dedup
// when two trailers canonicalize to one key, and the empty-value skip.
func TestHistoryBucketKeys(t *testing.T) {
	t.Parallel()
	tr := func(k, v string) gitops.Trailer { return gitops.Trailer{Key: k, Value: v} }
	tests := []struct {
		name     string
		trailers []gitops.Trailer
		want     []string
	}{
		{"bare entity", []gitops.Trailer{tr(gitops.TrailerEntity, "E-0001")}, []string{"E-0001"}},
		{"narrow width canonicalizes", []gitops.Trailer{tr(gitops.TrailerEntity, "E-22")}, []string{"E-0022"}},
		{
			"composite folds into milestone",
			[]gitops.Trailer{tr(gitops.TrailerEntity, "M-0001/AC-3")},
			[]string{"M-0001/AC-3", "M-0001"},
		},
		{
			"entity and prior-entity canonicalize alike dedup to one key",
			[]gitops.Trailer{tr(gitops.TrailerEntity, "E-22"), tr(gitops.TrailerPriorEntity, "E-0022")},
			[]string{"E-0022"},
		},
		{
			"empty trailer value is skipped",
			[]gitops.Trailer{tr(gitops.TrailerEntity, ""), tr(gitops.TrailerEntity, "E-0002")},
			[]string{"E-0002"},
		},
		{"no entity trailers", []gitops.Trailer{tr(gitops.TrailerVerb, "add")}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if diff := cmp.Diff(tt.want, historyBucketKeys(tt.trailers)); diff != "" {
				t.Errorf("historyBucketKeys (-want +got):\n%s", diff)
			}
		})
	}
}

// singlepass_test.go — E-0054 / M-0221 AC-3 data-level differential.
//
// The single-pass index must reproduce the exact per-entity views the old
// per-entity greps produced. This runs BOTH projections over one synthetic
// fixture — new (buildHistoryIndex buckets) vs old (the untouched
// ReadHistoryChain / LoadEntityScopeViews oracles) — and asserts equal for
// every trap the spec enumerates: a composite M-NNNN/AC-N fold, repeating
// aiwf-scope-ends, an active-scope opener, a scope-ended entity, a
// pathless (--allow-empty) acknowledge commit, a prior-entity reallocate
// lineage, and both narrow (E-22) and canonical (E-0022) id widths. The
// oracles are the source of truth; a bucketing drift fails here.

// trapRepo is a synthetic git repo covering every M-0221 bucketing trap.
// It carries only commits (with the trailer sets the mutating verbs emit);
// the data differential works off git history, so no markdown tree is
// needed here (the site-level wiring is pinned separately).
type trapRepo struct {
	root       string
	queryIDs   []string // every id the differential asserts bucket == ReadHistory for
	milestones []string // milestone ids the scope-view differential covers
}

func buildTrapRepo(t *testing.T) trapRepo {
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
	// Deterministic, strictly-increasing author dates so scope views sort
	// unambiguously (the old and new paths build allScopes via map
	// iteration; distinct Opened dates make the sort order canonical).
	n := 0
	commit := func(msg string) string {
		n++
		date := fmt.Sprintf("2026-01-01T00:%02d:00Z", n)
		env := []string{"GIT_AUTHOR_DATE=" + date, "GIT_COMMITTER_DATE=" + date}
		if out, err := testutil.RunGitWithExtraEnv(root, env, "commit", "--allow-empty", "-m", msg); err != nil {
			t.Fatalf("git commit: %v\n%s", err, out)
		}
		out, err := testutil.RunGit(root, "rev-parse", "HEAD")
		if err != nil {
			t.Fatalf("git rev-parse: %v\n%s", err, out)
		}
		return strings.TrimSpace(out)
	}

	// (1) Scopeless entity.
	commit("add E-0001\n\naiwf-verb: add\naiwf-entity: E-0001\naiwf-actor: human/peter\n")
	commit("promote E-0001 active\n\naiwf-verb: promote\naiwf-entity: E-0001\naiwf-actor: human/peter\naiwf-to: active\n")
	// A forced/audit-only promote carrying a real prose BODY plus the
	// force / audit-only / principal / on-behalf-of / reason trailer set —
	// so the differential's full-struct equality exercises those fields (not
	// just the trailer-only commits above) and the %B→%b body derivation.
	commit("promote E-0001 done (recovery)\n\n" +
		"Backfilled after a manual commit; recorded for the audit trail.\n\n" +
		"aiwf-verb: promote\naiwf-entity: E-0001\naiwf-actor: human/peter\naiwf-to: done\n" +
		"aiwf-force: legacy migration\naiwf-audit-only: backfill\n" +
		"aiwf-principal: human/peter\naiwf-on-behalf-of: human/peter\naiwf-reason: manual recovery\n")
	// Prose-mention (G30): an aiwf-entity trailer with NO aiwf-verb/aiwf-actor.
	// ReadHistoryChain skips it (verb+actor empty) and so must the single pass
	// (EventFromCommit returns ok=false) — it belongs to no bucket.
	commit("chore: touch up notes\n\naiwf-entity: E-0001\n")

	// (2) Active direct-scope opener E-0002 + E-0003 worked under it.
	commit("add E-0002\n\naiwf-verb: add\naiwf-entity: E-0002\naiwf-actor: human/peter\n")
	openerE0002 := commit("authorize E-0002 --to ai/claude\n\n" +
		"aiwf-verb: authorize\naiwf-entity: E-0002\naiwf-actor: human/peter\naiwf-to: ai/claude\naiwf-scope: opened\n")
	commit("add E-0003\n\naiwf-verb: add\naiwf-entity: E-0003\naiwf-actor: human/peter\n")
	commit(fmt.Sprintf("promote E-0003 active\n\n"+
		"aiwf-verb: promote\naiwf-entity: E-0003\naiwf-actor: ai/claude\naiwf-to: active\n"+
		"aiwf-on-behalf-of: human/peter\naiwf-authorized-by: %s\n", openerE0002))

	// (3) Two openers on E-0004 ended by ONE terminal commit — repeating
	// aiwf-scope-ends.
	commit("add E-0004\n\naiwf-verb: add\naiwf-entity: E-0004\naiwf-actor: human/peter\n")
	openA := commit("authorize E-0004 --to ai/claude\n\naiwf-verb: authorize\naiwf-entity: E-0004\naiwf-actor: human/peter\naiwf-to: ai/claude\naiwf-scope: opened\n")
	openB := commit("authorize E-0004 --to ai/bot\n\naiwf-verb: authorize\naiwf-entity: E-0004\naiwf-actor: human/peter\naiwf-to: ai/bot\naiwf-scope: opened\n")
	commit(fmt.Sprintf("promote E-0004 done\n\n"+
		"aiwf-verb: promote\naiwf-entity: E-0004\naiwf-actor: human/peter\naiwf-to: done\n"+
		"aiwf-scope-ends: %s\naiwf-scope-ends: %s\n", openA, openB))

	// (4) Milestone M-0001 with AC composites — the fold: M-0001/AC-N events
	// must appear in BOTH the AC bucket and the M-0001 bucket.
	commit("add M-0001\n\naiwf-verb: add\naiwf-entity: M-0001\naiwf-actor: human/peter\n")
	commit("promote M-0001 in_progress\n\naiwf-verb: promote\naiwf-entity: M-0001\naiwf-actor: human/peter\naiwf-to: in_progress\n")
	commit("add ac M-0001/AC-1\n\naiwf-verb: add\naiwf-entity: M-0001/AC-1\naiwf-actor: human/peter\n")
	commit("promote M-0001/AC-1 green\n\naiwf-verb: promote\naiwf-entity: M-0001/AC-1\naiwf-actor: human/peter\naiwf-to: green\naiwf-tests: pass=5 fail=0 skip=0 total=5\n")
	commit("add ac M-0001/AC-2\n\naiwf-verb: add\naiwf-entity: M-0001/AC-2\naiwf-actor: human/peter\n")

	// (5) Pathless acknowledge: an --allow-empty commit carrying aiwf-entity
	// but touching no file (already --allow-empty here). Verb = acknowledge.
	commit("acknowledge illegal E-0001\n\naiwf-verb: acknowledge-illegal\naiwf-entity: E-0001\naiwf-actor: human/peter\naiwf-reason: retroactive\n")

	// (6) Reallocate lineage: E-0007 was E-0006 (prior-entity). A query for
	// the OLD id must still find the reallocate commit (prior-entity grep);
	// the new id finds it via aiwf-entity.
	commit("add E-0006\n\naiwf-verb: add\naiwf-entity: E-0006\naiwf-actor: human/peter\n")
	commit("reallocate E-0006 -> E-0007\n\naiwf-verb: reallocate\naiwf-entity: E-0007\naiwf-prior-entity: E-0006\naiwf-actor: human/peter\n")

	// (7) Width: the SAME entity authored at narrow (E-22) then canonical
	// (E-0022) width. A query for either must find both commits.
	commit("add E-22\n\naiwf-verb: add\naiwf-entity: E-22\naiwf-actor: human/peter\n")
	commit("promote E-0022 active\n\naiwf-verb: promote\naiwf-entity: E-0022\naiwf-actor: human/peter\naiwf-to: active\n")

	// (8) Doubled aiwf-entity trailer on a scope opener: a single commit
	// carrying `aiwf-entity: E-0008` twice. The `^aiwf-entity: E-0008$` grep
	// matches the commit once, so the scope replay must too — the per-bucket
	// SHA dedup (exactSeen) is what keeps the index from replaying the opener
	// twice and inventing a second scope.
	commit("add E-0008\n\naiwf-verb: add\naiwf-entity: E-0008\naiwf-actor: human/peter\n")
	commit("authorize E-0008 (doubled entity trailer)\n\n" +
		"aiwf-verb: authorize\naiwf-entity: E-0008\naiwf-entity: E-0008\n" +
		"aiwf-actor: human/peter\naiwf-to: ai/claude\naiwf-scope: opened\n")

	return trapRepo{
		root: root,
		queryIDs: []string{
			"E-0001", "E-0002", "E-0003", "E-0004",
			"M-0001", "M-0001/AC-1", "M-0001/AC-2",
			"E-0006", "E-0007", // reallocate lineage, both ids
			"E-22", "E-0022", // width, both forms
			"E-0008", // doubled aiwf-entity trailer
		},
		milestones: []string{"E-0001", "E-0002", "E-0003", "E-0004", "M-0001", "E-0008"},
	}
}

// TestSinglePass_EventsMatchReadHistory is AC-3's core: for every query id,
// the single-pass bucket equals ReadHistoryChain([id]) — the untouched
// oracle — byte-for-byte, across every trap.
func TestSinglePass_EventsMatchReadHistory(t *testing.T) {
	t.Parallel()
	repo := buildTrapRepo(t)
	ctx := context.Background()
	head, err := check.WalkHeadCommits(ctx, repo.root)
	if err != nil {
		t.Fatalf("WalkHeadCommits: %v", err)
	}
	idx := buildHistoryIndex(head)

	for _, id := range repo.queryIDs {
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			want, err := history.ReadHistory(ctx, repo.root, id)
			if err != nil {
				t.Fatalf("ReadHistory(%s): %v", id, err)
			}
			got := idx.events[entity.Canonicalize(id)]
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("single-pass bucket != ReadHistory(%s) (-oracle +bucket):\n%s", id, diff)
			}
			// Non-vacuity: the fixture ids all have at least one event.
			if len(want) == 0 {
				t.Errorf("oracle ReadHistory(%s) is empty — the differential would be vacuous", id)
			}
		})
	}
}

// TestSinglePass_MilestoneFoldsACEvents pins the composite fold explicitly:
// M-0001's bucket must contain its AC-1 and AC-2 events (bare-milestone
// path-prefix match), and the AC buckets must NOT contain M-0001's own
// status events.
func TestSinglePass_MilestoneFoldsACEvents(t *testing.T) {
	t.Parallel()
	repo := buildTrapRepo(t)
	head, err := check.WalkHeadCommits(context.Background(), repo.root)
	if err != nil {
		t.Fatalf("WalkHeadCommits: %v", err)
	}
	idx := buildHistoryIndex(head)

	milestoneDetails := map[string]bool{}
	for _, e := range idx.events["M-0001"] {
		milestoneDetails[e.Detail] = true
	}
	for _, want := range []string{"promote M-0001 in_progress", "add ac M-0001/AC-1", "promote M-0001/AC-1 green", "add ac M-0001/AC-2"} {
		if !milestoneDetails[want] {
			t.Errorf("M-0001 bucket missing folded event %q; has %v", want, milestoneDetails)
		}
	}
	// The AC bucket holds only its own events, not the milestone's status.
	for _, e := range idx.events["M-0001/AC-1"] {
		if strings.HasPrefix(e.Detail, "promote M-0001 ") {
			t.Errorf("AC-1 bucket leaked milestone status event %q", e.Detail)
		}
	}
}

// TestSinglePass_ScopeViewsMatchLoadEntityScopeViews is AC-3's provenance
// half: for every scope-bearing entity, the index-derived scope views equal
// show.LoadEntityScopeViews — the untouched oracle — including the repeating
// scope-ends (E-0004) and the foreign-scope resolution (E-0003 under E-0002).
func TestSinglePass_ScopeViewsMatchLoadEntityScopeViews(t *testing.T) {
	t.Parallel()
	repo := buildTrapRepo(t)
	ctx := context.Background()
	head, err := check.WalkHeadCommits(ctx, repo.root)
	if err != nil {
		t.Fatalf("WalkHeadCommits: %v", err)
	}
	r := &Resolver{index: buildHistoryIndex(head)}

	for _, id := range repo.milestones {
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			want, err := show.LoadEntityScopeViews(ctx, repo.root, id)
			if err != nil {
				t.Fatalf("LoadEntityScopeViews(%s): %v", id, err)
			}
			got := r.scopeViewsFor(id)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("index scope views != LoadEntityScopeViews(%s) (-oracle +index):\n%s", id, diff)
			}
		})
	}

	// Non-vacuity: E-0002 (active opener) and E-0004 (two ended scopes) must
	// produce non-empty tables, or the equivalence is vacuous.
	if got := r.scopeViewsFor("E-0002"); len(got) != 1 {
		t.Errorf("E-0002 scope views = %d, want 1 (active opener)", len(got))
	}
	if got := r.scopeViewsFor("E-0004"); len(got) != 2 {
		t.Errorf("E-0004 scope views = %d, want 2 (both ended via repeating scope-ends)", len(got))
	}
	// E-0008's opener carries a DOUBLED aiwf-entity trailer: the scope table
	// must hold exactly one scope, not two — the exactSeen per-bucket dedup.
	if got := r.scopeViewsFor("E-0008"); len(got) != 1 {
		t.Errorf("E-0008 scope views = %d, want 1 (doubled aiwf-entity must not double the scope)", len(got))
	}
}
