package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	clicheck "github.com/23min/aiwf/internal/cli/check"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/show"
)

// TestResolveUntrailedRange_NoUpstream: a fresh repo with no
// upstream configured returns no range and an advisory finding
// so step-7b's audit pass is skipped (the previous "all of HEAD"
// fallback flooded long-lived branches with commits already
// merged in from trunk — see issue #5 sub-item 2).
func TestResolveUntrailedRange_NoUpstream(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q", "-b", "main"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	rangeArg, advisory, err := clicheck.ResolveUntrailedRange(context.Background(), root, "")
	if err != nil {
		t.Fatalf("resolveUntrailedRange: %v", err)
	}
	if rangeArg != "" {
		t.Errorf("rangeArg = %q, want empty (skipped)", rangeArg)
	}
	if advisory == nil {
		t.Fatal("advisory is nil; want a scope-undefined warning")
	}
	if advisory.Code != check.CodeProvenanceUntrailedScopeUndefined {
		t.Errorf("advisory.Code = %q, want provenance-untrailered-scope-undefined", advisory.Code)
	}
}

// TestResolveUntrailedRange_WithUpstream: when @{u} resolves, the
// helper returns "@{u}..HEAD" and no advisory.
func TestResolveUntrailedRange_WithUpstream(t *testing.T) {
	t.Parallel()
	upstream := t.TempDir()
	if out, err := testutil.RunGit(upstream, "init", "--bare", "-q"); err != nil {
		t.Fatalf("git init bare: %v\n%s", err, out)
	}
	root := t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q", "-b", "main"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
		{"remote", "add", "origin", upstream},
	} {
		if out, err := testutil.RunGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := testutil.RunGit(root, "commit", "--allow-empty", "-m", "seed"); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	// Push and set upstream tracking.
	if out, err := testutil.RunGit(root, "push", "-u", "origin", "HEAD:main"); err != nil {
		t.Fatalf("git push: %v\n%s", err, out)
	}
	rangeArg, advisory, err := clicheck.ResolveUntrailedRange(context.Background(), root, "")
	if err != nil {
		t.Fatalf("resolveUntrailedRange: %v", err)
	}
	if rangeArg != "@{u}..HEAD" {
		t.Errorf("rangeArg = %q, want @{u}..HEAD", rangeArg)
	}
	if advisory != nil {
		t.Errorf("advisory = %+v, want nil", advisory)
	}
}

// TestResolveUntrailedRange_SinceWins: an explicit --since <ref>
// overrides upstream detection. The ref is verified via
// `git rev-parse`; an unknown ref returns an advisory finding
// (audit skipped) rather than failing the whole check.
func TestResolveUntrailedRange_SinceWins(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q", "-b", "main"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := testutil.RunGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := testutil.RunGit(root, "commit", "--allow-empty", "-m", "seed"); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	// Valid ref → "<ref>..HEAD" range, no advisory.
	rangeArg, advisory, err := clicheck.ResolveUntrailedRange(context.Background(), root, "HEAD")
	if err != nil {
		t.Fatalf("resolveUntrailedRange (valid since): %v", err)
	}
	if rangeArg != "HEAD..HEAD" || advisory != nil {
		t.Errorf("valid since: rangeArg=%q advisory=%+v; want HEAD..HEAD, nil", rangeArg, advisory)
	}
	// Unknown ref → empty range + advisory.
	rangeArg, advisory, err = clicheck.ResolveUntrailedRange(context.Background(), root, "no-such-ref")
	if err != nil {
		t.Fatalf("resolveUntrailedRange (bad since): %v", err)
	}
	if rangeArg != "" {
		t.Errorf("bad since: rangeArg = %q, want empty", rangeArg)
	}
	if advisory == nil || advisory.Code != check.CodeProvenanceUntrailedScopeUndefined {
		t.Errorf("bad since: advisory = %+v, want scope-undefined", advisory)
	}
}

// TestReadUntrailedCommits_EmptyRange: when HEAD == @{u} (already
// pushed), the unpushed range produces no commits. The helper
// should return nil without error so step 7b stays silent.
func TestReadUntrailedCommits_EmptyRange(t *testing.T) {
	t.Parallel()
	upstream := t.TempDir()
	if out, err := testutil.RunGit(upstream, "init", "--bare", "-q"); err != nil {
		t.Fatalf("git init bare: %v\n%s", err, out)
	}
	root := t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q", "-b", "main"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
		{"remote", "add", "origin", upstream},
	} {
		if out, err := testutil.RunGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := testutil.RunGit(root, "commit", "--allow-empty", "-m", "seed"); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	// Push and set upstream tracking — now HEAD == @{u}.
	if out, err := testutil.RunGit(root, "push", "-u", "origin", "HEAD:main"); err != nil {
		t.Fatalf("git push: %v\n%s", err, out)
	}
	got, err := clicheck.ReadUntrailedCommits(context.Background(), root, "@{u}..HEAD")
	if err != nil {
		t.Fatalf("readUntrailedCommits: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d commits, want 0 (HEAD == @{u})", len(got))
	}
}

// TestReadUntrailedCommits_MergeCommitSurface covers G32: a merge
// commit on the integration branch that brings in entity-file
// changes from a feature branch must surface those paths to the
// audit pass. Pre-fix the default `git log --name-only` showed no
// file list for merge commits and the audit silently skipped them.
// Post-fix `-m --first-parent` makes the merge commit's introduced
// changes (against its first parent) visible, so per-(commit,
// entity) findings fire on the integration branch boundary.
func TestReadUntrailedCommits_MergeCommitSurface(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q", "-b", "main"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
		{"config", "commit.gpgsign", "false"},
	} {
		if out, err := testutil.RunGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	// Initial empty commit on main so we have a base.
	if out, err := testutil.RunGit(root, "commit", "--allow-empty", "-m", "seed"); err != nil {
		t.Fatalf("seed: %v\n%s", err, out)
	}
	baseSHA, gErr := testutil.RunGit(root, "rev-parse", "HEAD")
	if gErr != nil {
		t.Fatalf("rev-parse HEAD: %v", gErr)
	}
	base := strings.TrimSpace(baseSHA)

	// Feature branch with one untrailered commit touching G-001.
	if out, gErr := testutil.RunGit(root, "checkout", "-q", "-b", "feature"); gErr != nil {
		t.Fatalf("checkout: %v\n%s", gErr, out)
	}
	gapDir := filepath.Join(root, "work", "gaps")
	if mkErr := os.MkdirAll(gapDir, 0o755); mkErr != nil {
		t.Fatal(mkErr)
	}
	gapPath := filepath.Join(gapDir, "G-0001-leak.md")
	if wErr := os.WriteFile(gapPath, []byte("---\nid: G-001\nstatus: wontfix\n---\n"), 0o644); wErr != nil {
		t.Fatal(wErr)
	}
	if out, gErr := testutil.RunGit(root, "add", "work/gaps/G-0001-leak.md"); gErr != nil {
		t.Fatalf("git add: %v\n%s", gErr, out)
	}
	if out, gErr := testutil.RunGit(root, "commit", "-m", "manual: flip G-001 wontfix"); gErr != nil {
		t.Fatalf("manual commit: %v\n%s", gErr, out)
	}
	// Back to main; merge feature with --no-ff so a merge commit lands.
	if out, gErr := testutil.RunGit(root, "checkout", "-q", "main"); gErr != nil {
		t.Fatalf("checkout main: %v\n%s", gErr, out)
	}
	if out, gErr := testutil.RunGit(root, "merge", "--no-ff", "-m", "merge feature into main", "feature"); gErr != nil {
		t.Fatalf("merge: %v\n%s", gErr, out)
	}

	// Range scoped to base..HEAD. Pre-fix: zero records (merge has
	// no name-only output by default; the upstream feature commit
	// is excluded by --first-parent which means the only candidate
	// is the merge itself, and without -m it has no paths). Post-
	// fix: the merge commit appears with G-001's path attached.
	commits, err := clicheck.ReadUntrailedCommits(context.Background(), root, base+"..HEAD")
	if err != nil {
		t.Fatalf("readUntrailedCommits: %v", err)
	}
	var sawMerge bool
	for _, c := range commits {
		if len(c.Paths) == 0 {
			continue
		}
		for _, p := range c.Paths {
			if strings.HasSuffix(p, "G-0001-leak.md") {
				sawMerge = true
			}
		}
	}
	if !sawMerge {
		t.Errorf("expected merge commit to surface G-001's path; got commits=%+v", commits)
	}
}

// TestParseUntrailedCommits_Malformed: records that don't have the
// expected five-field shape (SHA, parents, subject, trailers, paths)
// are silently skipped. Drives the defensive parsing branch.
// Field shape extended for G-0231 item 3 — the merge-commit
// carveout consumes %P parents from the git log stream.
func TestParseUntrailedCommits_Malformed(t *testing.T) {
	t.Parallel()
	const sep = "\x1f"
	const rec = "\x1e"
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty input", "", 0},
		{"only whitespace", "   \n\n  ", 0},
		{
			"one well-formed",
			rec + "abc1234" + sep + "ppp1111" + sep + "feat: thing" + sep + "" + sep + "work/gaps/G-001-x.md",
			1,
		},
		{
			"one record missing field separator (truncated to legacy 4-field shape)",
			rec + "abc1234" + sep + "feat: thing" + sep + "" + sep + "work/gaps/G-001-x.md",
			0,
		},
		{
			"two records, second malformed",
			rec + "aaa1111" + sep + "ppp1" + sep + "feat: a" + sep + "" + sep + "work/gaps/G-001.md" +
				rec + "bbb2222-no-seps",
			1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clicheck.ParseUntrailedCommits(tt.input)
			if len(got) != tt.want {
				t.Errorf("len = %d, want %d (got %+v)", len(got), tt.want, got)
			}
		})
	}
}

// TestLoadEntityScopeViews_NoCommits: pre-init repo; helper returns
// (nil, nil) without shelling out further.
func TestLoadEntityScopeViews_NoCommits(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q", "-b", "main"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	got, err := show.LoadEntityScopeViews(context.Background(), root, "E-0001")
	if err != nil {
		t.Fatalf("show.LoadEntityScopeViews: %v", err)
	}
	if got != nil {
		t.Errorf("got %v, want nil (no commits)", got)
	}
}

// TestLoadEntityScopeViews_NotInGitRepo: working dir is not a git
// repo. cliutil.HasCommits returns false; helper returns (nil, nil)
// gracefully without erroring.
func TestLoadEntityScopeViews_NotInGitRepo(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	got, err := show.LoadEntityScopeViews(context.Background(), tmp, "E-0001")
	if err != nil {
		t.Fatalf("show.LoadEntityScopeViews: %v", err)
	}
	if got != nil {
		t.Errorf("got %v, want nil (no repo)", got)
	}
}

// TestLoadEntityScopeViews_NoScopesTouchEntity: a repo with commits
// but none referencing the queried entity returns nil.
func TestLoadEntityScopeViews_NoScopesTouchEntity(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q", "-b", "main"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := testutil.RunGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	// One commit with an entity trailer for X-99 (not the queried
	// id) and no scope at all.
	msg := "chore: seed\n\naiwf-verb: add\naiwf-entity: X-99\naiwf-actor: human/peter\n"
	if out, err := testutil.RunGit(root, "commit", "--allow-empty", "-m", msg); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	got, err := show.LoadEntityScopeViews(context.Background(), root, "E-0001")
	if err != nil {
		t.Fatalf("show.LoadEntityScopeViews: %v", err)
	}
	if got != nil {
		t.Errorf("got %v, want nil (no scopes touch E-01)", got)
	}
}
