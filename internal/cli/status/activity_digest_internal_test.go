package status

import (
	"context"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

// TestReadDigestCommits_NoCommits: a repo with git init but zero
// commits returns (nil, nil) rather than erroring — the same "fresh
// repo is a legitimate state" contract ReadRecentActivity follows.
func TestReadDigestCommits_NoCommits(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	gitDo(t, root, "init", "-q")

	commits, err := readDigestCommits(context.Background(), root)
	if err != nil {
		t.Fatalf("readDigestCommits: %v", err)
	}
	if commits != nil {
		t.Errorf("commits = %+v, want nil", commits)
	}
}

// TestReadDigestCommits_GitLogFailure: a repo with commits (so the
// HasCommits guard passes) but a syntactically unresolvable revision
// range makes the underlying `git log` fail for real — readDigestCommits
// surfaces the *exec.ExitError, naming "git log:" and including git's
// own stderr.
func TestReadDigestCommits_GitLogFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	gitDo(t, root, "init", "-q")
	gitDo(t, root, "commit", "--allow-empty", "-m", "seed")

	_, err := readDigestCommits(context.Background(), root, "no-such-tag-anywhere..HEAD")
	if err == nil {
		t.Fatal("readDigestCommits: want an error for an unresolvable revision range, got nil")
	}
	if !strings.Contains(err.Error(), "git log:") {
		t.Errorf("err = %v, want it to name %q", err, "git log:")
	}
}

// TestDigestFromCommits_MalformedDateSkipped exercises the parse-error
// guard directly: %aI always emits a valid ISO-8601 timestamp for a
// real commit, so this can't be reached through git itself, but the
// defensive skip (rather than a panic on a future format regression)
// still gets a test.
func TestDigestFromCommits_MalformedDateSkipped(t *testing.T) {
	t.Parallel()
	commits := []digestCommit{
		{AuthorDate: "not-a-date", Verb: "add", EntityID: "G-0001"},
	}
	d := digestFromCommits(commits, &tree.Tree{}, "2026-01-01", "label")
	if len(d.GapsOpened) != 0 || len(d.GapsClosed) != 0 || len(d.ADRsCreated) != 0 {
		t.Errorf("expected an empty digest (malformed date skipped), got %+v", d)
	}
}

// TestBucketDigestCommits_UnresolvableKindSkipped: an aiwf-entity value
// that matches no kind's id format (entity.KindFromID reports ok=false)
// lands in no bucket rather than panicking on the kind switch below.
func TestBucketDigestCommits_UnresolvableKindSkipped(t *testing.T) {
	t.Parallel()
	commits := []digestCommit{
		{Verb: "add", EntityID: "not-a-real-id"},
	}
	d := bucketDigestCommits(commits, &tree.Tree{})
	if len(d.GapsOpened) != 0 || len(d.GapsClosed) != 0 || len(d.ADRsCreated) != 0 {
		t.Errorf("expected no bucket populated for an unresolvable entity id, got %+v", d)
	}
}
