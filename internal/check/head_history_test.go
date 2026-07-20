package check

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/23min/aiwf/internal/gitops"
)

// TestWalkHeadCommits_CapturesAuthorDateAndSubject pins M-0221's
// additive extension: the shared HEAD walk now also captures the author
// date (%aI) and subject (%s). render's single pass reproduces
// entityview.HistoryEvent's Date + Detail and derives scope open/end dates
// from these fields — eliminating the per-SHA `git show` the scope views
// used. The existing check consumers read neither field, so this is
// purely additive; TestParseHeadCommits pins the widened record layout.
func TestWalkHeadCommits_CapturesAuthorDateAndSubject(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	sha := r.gitCommit("feat(x): a subject line")
	head := mustHead(t, r.root)
	if len(head) != 1 {
		t.Fatalf("WalkHeadCommits returned %d commits, want 1", len(head))
	}
	c := head[0]
	if c.SHA != sha {
		t.Errorf("SHA = %q, want %q", c.SHA, sha)
	}
	if c.Subject != "feat(x): a subject line" {
		t.Errorf("Subject = %q, want %q", c.Subject, "feat(x): a subject line")
	}
	// %aI is an ISO-8601 author date; assert it round-trips through
	// RFC3339 so the render date derivation has a well-formed input.
	if _, err := time.Parse(time.RFC3339, c.AuthorDate); err != nil {
		t.Errorf("AuthorDate = %q, not RFC3339-parseable: %v", c.AuthorDate, err)
	}
}

// mustHead runs WalkHeadCommits over a healthy fixture repo, asserting
// the Finding-1 fail-loud error is nil. Every fixture these tests build
// is a readable repo, so a non-nil error here is a fixture/regression
// bug, not an expected branch; the error path itself is covered by
// TestWalkHeadCommits_FailsLoudOnUnreadableHistory.
func mustHead(t *testing.T, root string) []HeadCommit {
	t.Helper()
	h, err := WalkHeadCommits(context.Background(), root)
	if err != nil {
		t.Fatalf("WalkHeadCommits over fixture %s: %v", root, err)
	}
	return h
}

// TestParseHeadCommits covers WalkHeadCommits' pure parser over the
// M-0221 seven-field layout: a valid two-record stream (asserting the
// added author-date and subject fields), a record with too few
// unit-separated fields (dropped), and a record with an empty SHA
// (dropped). Field order: SHA, author-email, committer-email,
// author-date, subject, trailers, body.
func TestParseHeadCommits(t *testing.T) {
	t.Parallel()
	const m = headRecMarker
	out := m + "\n" +
		"sha1\x1fauthor@a\x1fcommitter@a\x1f2026-01-01T00:00:00Z\x1fsubject one\x1faiwf-verb: add\x1fbody one\n" +
		m + "\n" +
		"\x1fauthor@b\x1fcommitter@b\x1f2026-01-02T00:00:00Z\x1fsubj\x1faiwf-verb: edit\x1fbody\n" + // empty SHA → dropped
		m + "\n" +
		"sha3-too-few-fields\n" + // no \x1f separators → len(fields)<7 → dropped
		m + "\n" +
		"sha4\x1fa@x\x1fb@y\x1f2026-01-03T00:00:00Z\x1fsubject four\x1faiwf-verb: promote\x1fbody four\n"

	got := parseHeadCommits(out)
	if len(got) != 2 {
		t.Fatalf("parseHeadCommits returned %d records, want 2 (sha1, sha4); got %+v", len(got), got)
	}
	if got[0].SHA != "sha1" || got[0].AuthorEmail != "author@a" || got[0].CommitterEmail != "committer@a" {
		t.Errorf("record 0 identity = %+v, want sha1/author@a/committer@a", got[0])
	}
	if got[0].AuthorDate != "2026-01-01T00:00:00Z" || got[0].Subject != "subject one" {
		t.Errorf("record 0 date/subject = (%q, %q), want (2026-01-01T00:00:00Z, subject one)", got[0].AuthorDate, got[0].Subject)
	}
	if got[0].Body != "body one\n" {
		t.Errorf("record 0 Body = %q, want %q", got[0].Body, "body one\n")
	}
	if got[1].SHA != "sha4" || got[1].AuthorDate != "2026-01-03T00:00:00Z" || got[1].Subject != "subject four" {
		t.Errorf("record 1 = %+v, want sha4 / 2026-01-03 / subject four", got[1])
	}
}

// TestWalkHeadCommits_EmptyAndNonGitReturnNoError pins the (nil, nil)
// contract for the three "no commits" inputs: an empty root string, a
// non-git directory, and a freshly-init'd repo with no commits yet.
// None of these is a failure — a nil slice means "no commits / no
// exemptions", and the error must stay nil so the caller does not fail
// loud (M-0216 Finding 1).
func TestWalkHeadCommits_EmptyAndNonGitReturnNoError(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"empty root":  "",
		"non-git dir": t.TempDir(),
	}
	for name, root := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := WalkHeadCommits(context.Background(), root)
			if got != nil || err != nil {
				t.Errorf("WalkHeadCommits(%q) = (%v, %v), want (nil, nil)", root, got, err)
			}
		})
	}
	t.Run("git repo with no commits", func(t *testing.T) {
		t.Parallel()
		r := newRepoFixture(t) // git init, no commits yet
		got, err := WalkHeadCommits(context.Background(), r.root)
		if got != nil || err != nil {
			t.Errorf("WalkHeadCommits(no-commits) = (%v, %v), want (nil, nil)", got, err)
		}
	})
}

// TestWalkHeadCommits_FailsLoudOnUnreadableHistory is the Finding-1
// regression test: a repo whose HEAD resolves (so hasGitCommits is
// true) but whose reachable history cannot be fully read must return a
// non-nil error, NOT a silent nil that would disable the provenance /
// isolation-escape gathers deriving from the walk.
//
// The repro removes a non-HEAD parent commit object: `git rev-parse
// --verify --quiet HEAD` still resolves the tip (hasGitCommits true),
// but `git log --reverse HEAD` — which collects the full history before
// emitting — fails when it reaches the missing parent.
func TestWalkHeadCommits_FailsLoudOnUnreadableHistory(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	parent := r.gitCommit("parent")
	r.gitCommit("child") // HEAD = child; parent is a reachable ancestor

	// Delete the parent's loose commit object. Auto-gc is disabled in the
	// test env (HardenGitTestEnv), so the object is loose and this is the
	// whole object — HEAD (child) still resolves, but the ancestry walk
	// can no longer be completed.
	obj := filepath.Join(r.root, ".git", "objects", parent[:2], parent[2:])
	if err := os.Remove(obj); err != nil {
		t.Fatalf("removing parent commit object %s: %v", obj, err)
	}

	if !hasGitCommits(context.Background(), r.root) {
		t.Fatal("precondition: HEAD must still resolve after removing the parent object")
	}

	got, err := WalkHeadCommits(context.Background(), r.root)
	if err == nil {
		t.Fatalf("WalkHeadCommits: want fail-loud error on unreadable history, got nil (commits=%d)", len(got))
	}
	if got != nil {
		t.Errorf("WalkHeadCommits: want nil slice alongside error, got %d commits", len(got))
	}
}

// TestWalkCherryPicks_FromHead exercises every WalkCherryPicks branch:
// the empty-head short-circuit, the both-signals predicate (identity gap
// AND marker), and the skips when either signal is absent.
func TestWalkCherryPicks_FromHead(t *testing.T) {
	t.Parallel()
	if got := WalkCherryPicks(nil); got != nil {
		t.Errorf("WalkCherryPicks(nil) = %v, want nil", got)
	}
	head := []HeadCommit{
		// qualifies: identity gap + marker.
		{SHA: "cp", AuthorEmail: "a@x", CommitterEmail: "b@y", Body: "pick\n\n(cherry picked from commit deadbeef)"},
		// gap but no marker → skip.
		{SHA: "gap-no-marker", AuthorEmail: "a@x", CommitterEmail: "b@y", Body: "no marker here"},
		// marker but no gap (same email) → skip.
		{SHA: "marker-no-gap", AuthorEmail: "a@x", CommitterEmail: "a@x", Body: "(cherry picked from commit abcd1234)"},
		// missing committer email → skip.
		{SHA: "no-committer", AuthorEmail: "a@x", CommitterEmail: "", Body: "(cherry picked from commit abcd1234)"},
	}
	got := WalkCherryPicks(head)
	if len(got) != 1 || !got["cp"] {
		t.Errorf("WalkCherryPicks = %v, want only {cp:true}", got)
	}
}

// TestWalkAuditOnlyAcksByEntity_FromHead covers the audit-only gather:
// empty head, the empty-SHA skip, the both-trailers requirement, and the
// composite-root canonicalization of the entity key.
func TestWalkAuditOnlyAcksByEntity_FromHead(t *testing.T) {
	t.Parallel()
	if got := walkAuditOnlyAcksByEntity(nil); got != nil {
		t.Errorf("walkAuditOnlyAcksByEntity(nil) = %v, want nil", got)
	}
	head := []HeadCommit{
		{SHA: "", Trailers: []gitops.Trailer{{Key: gitops.TrailerAuditOnly, Value: ""}, {Key: gitops.TrailerEntity, Value: "M-0001"}}}, // empty SHA → skip
		{SHA: "ack1", Trailers: []gitops.Trailer{{Key: gitops.TrailerAuditOnly, Value: ""}, {Key: gitops.TrailerEntity, Value: "M-001/AC-1"}}},
		{SHA: "not-audit", Trailers: []gitops.Trailer{{Key: gitops.TrailerEntity, Value: "M-0002"}}}, // no audit-only → skip
	}
	got := walkAuditOnlyAcksByEntity(head)
	// M-001/AC-1 rolls up to composite root M-0001 (canonical width).
	if shas := got["M-0001"]; len(shas) != 1 || shas[0] != "ack1" {
		t.Errorf("walkAuditOnlyAcksByEntity[M-0001] = %v, want [ack1]", got["M-0001"])
	}
	if len(got) != 1 {
		t.Errorf("walkAuditOnlyAcksByEntity returned %d entities, want 1; got %v", len(got), got)
	}
}
