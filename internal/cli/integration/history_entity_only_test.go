package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/entityview"
)

// history_entity_only_test.go pins the seam, not the layer. The constructor
// test in internal/entityview covers the admit rule for one commit; this drives
// ReadHistory over a real repository, so the `git log` query's own copy of that
// rule is covered too. Without it, changing only the constructor leaves the
// query dropping the same commits and the suite stays green.

// entityOnlyRepo builds a repo whose commits cover the three shapes the query
// must tell apart, and returns the root with each commit's SHA by name.
func entityOnlyRepo(t *testing.T) (root string, shas map[string]string) {
	t.Helper()
	root = t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "--initial-branch=main"},
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
		{"config", "commit.gpgsign", "false"},
	} {
		if out, err := testutil.RunGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	shas = map[string]string{}
	n := 0
	commit := func(name, msg string) {
		n++
		date := fmt.Sprintf("2026-01-01T00:%02d:00Z", n)
		env := []string{"GIT_AUTHOR_DATE=" + date, "GIT_COMMITTER_DATE=" + date}
		if out, err := testutil.RunGitWithExtraEnv(root, env, "commit", "--allow-empty", "-m", msg); err != nil {
			t.Fatalf("git commit %s: %v\n%s", name, err, out)
		}
		out, err := testutil.RunGit(root, "rev-parse", "HEAD")
		if err != nil {
			t.Fatalf("git rev-parse: %v\n%s", err, out)
		}
		shas[name] = strings.TrimSpace(out)
	}

	commit("verbEvent", "add E-0001\n\naiwf-verb: add\naiwf-entity: E-0001\naiwf-actor: human/peter\n")
	// The shape D-0071 creates: a shipped-surface edit whose whole provenance
	// is the entity trailer, with no verb to name what it did.
	commit("entityOnly", "fix(x): correct a shipped surface\n\naiwf-entity: E-0001\n")
	// A reallocate's lineage key, likewise alone.
	commit("priorOnly", "chore: renumbered\n\naiwf-prior-entity: E-0001\n")
	// The false positive the drop exists for: the id-shaped line sits in body
	// prose with a paragraph after it, so `--grep` matches and git's trailer
	// parser returns nothing.
	commit("proseMention", "docs: discuss the trailer\n\naiwf-entity: E-0001\n\nThe line above is prose, not a trailer block.\n")
	// A grep match on a prose line whose real trailer block carries the key
	// with no value. Git renders that cell as whitespace rather than empty,
	// so the drop test must trim before comparing or the prose match is
	// admitted as an event.
	commit("blankValue", "docs: discuss the trailer\n\naiwf-entity: E-0001\n\naiwf-entity:\n")
	return root, shas
}

func TestReadHistory_ListsACommitCarryingTheEntityTrailerAlone(t *testing.T) {
	t.Parallel()
	root, shas := entityOnlyRepo(t)

	events, err := entityview.ReadHistory(context.Background(), root, "E-0001")
	if err != nil {
		t.Fatalf("ReadHistory: %v", err)
	}
	got := map[string]bool{}
	for i := range events {
		got[events[i].Commit] = true
	}

	for _, want := range []string{"verbEvent", "entityOnly", "priorOnly"} {
		if !got[entityview.ShortHash(shas[want])] {
			t.Errorf("%s commit is absent from ReadHistory(E-0001); events = %+v", want, events)
		}
	}
	// The prose mention must stay excluded, or the drop rule has been
	// removed rather than corrected.
	if got[entityview.ShortHash(shas["proseMention"])] {
		t.Errorf("the prose-mention commit is present in ReadHistory(E-0001); the false-positive guard is gone")
	}
	if got[entityview.ShortHash(shas["blankValue"])] {
		t.Errorf("a commit whose parsed entity trailer carries only whitespace is present; the value is not being trimmed before the drop test")
	}
}

// TestReadHistory_EntityOnlyRowCarriesNoVerbOrActor pins what such an event
// holds. The row renders from these fields, and a synthesized verb value would
// put a token in the JSON `verb` field that names no aiwf verb.
func TestReadHistory_EntityOnlyRowCarriesNoVerbOrActor(t *testing.T) {
	t.Parallel()
	root, shas := entityOnlyRepo(t)

	events, err := entityview.ReadHistory(context.Background(), root, "E-0001")
	if err != nil {
		t.Fatalf("ReadHistory: %v", err)
	}
	want := entityview.ShortHash(shas["entityOnly"])
	for i := range events {
		if events[i].Commit != want {
			continue
		}
		if events[i].Verb != "" {
			t.Errorf("Verb = %q, want empty — no aiwf verb committed this", events[i].Verb)
		}
		if events[i].Actor != "" {
			t.Errorf("Actor = %q, want empty", events[i].Actor)
		}
		if events[i].Detail != "fix(x): correct a shipped surface" {
			t.Errorf("Detail = %q, want the subject", events[i].Detail)
		}
		return
	}
	t.Fatalf("entity-only commit %s not found; events = %+v", want, events)
}
