package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/gitops"
)

// TestEditBody_BinaryEndToEnd is the M-058 dispatcher-seam closure:
// `aiwf edit-body <id> --body-file <path>` against a real binary
// and a real consumer repo replaces the entity body and emits a
// trailered commit. Without this test, a regression that drops the
// runEditBody case from main.go would still pass internal/verb tests.
func TestEditBody_BinaryEndToEnd(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Foundations"); err != nil {
		t.Fatalf("aiwf add epic: %v\n%s", err, out)
	}

	bodyText := "## Goal\n\nFleshed-out goal prose written by the operator.\n\n## Scope\n\nReal scope.\n"
	bodyPath := filepath.Join(root, "epic-body.md")
	if err := os.WriteFile(bodyPath, []byte(bodyText), 0o644); err != nil {
		t.Fatalf("write body file: %v", err)
	}

	out, err := runBin(t, root, binDir, nil, "edit-body", "E-01", "--body-file", bodyPath)
	if err != nil {
		t.Fatalf("aiwf edit-body: %v\n%s", err, out)
	}

	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	hasTrailer(t, tr, "aiwf-verb", "edit-body")
	hasTrailer(t, tr, "aiwf-entity", "E-01")

	matches, err := filepath.Glob(filepath.Join(root, "work", "epics", "E-*", "epic.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("glob epic.md: matches=%v err=%v", matches, err)
	}
	got, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read epic: %v", err)
	}
	if !strings.Contains(string(got), "Fleshed-out goal prose") {
		t.Errorf("epic.md missing edited body content:\n%s", got)
	}
	if !strings.Contains(string(got), "id: E-01") {
		t.Errorf("epic.md frontmatter id missing after edit:\n%s", got)
	}

	// Tree validates clean — the edit didn't introduce findings,
	// and the new commit's aiwf-verb trailer means the standing
	// untrailered-entity-commit audit doesn't fire on it.
	checkOut, err := runBin(t, root, binDir, nil, "check")
	if err != nil {
		t.Fatalf("aiwf check after edit-body: %v\n%s", err, checkOut)
	}
	if strings.Contains(checkOut, "provenance-untrailered-entity-commit") {
		t.Errorf("post-edit-body check surfaces untrailered-entity warning:\n%s", checkOut)
	}
}

// TestEditBody_StdinEndToEnd: --body-file - reads body content
// from stdin, so callers can pipe text without a temp file —
// matches the aiwf add --body-file - shape.
func TestEditBody_StdinEndToEnd(t *testing.T) {
	bin := aiwfBinary(t)

	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, filepath.Dir(bin), nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, filepath.Dir(bin), nil, "add", "gap", "--title", "Stdin gap"); err != nil {
		t.Fatalf("add gap: %v\n%s", err, out)
	}

	stdin := "## Body via stdin\n\nThis content arrived through a pipe.\n"
	cmd := exec.Command(bin, "edit-body", "G-001", "--body-file", "-")
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=aiwf-test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=aiwf-test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("edit-body stdin: %v\n%s", err, out)
	}

	matches, err := filepath.Glob(filepath.Join(root, "work", "gaps", "G-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("glob G-*.md: matches=%v err=%v", matches, err)
	}
	got, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read gap: %v", err)
	}
	if !strings.Contains(string(got), "This content arrived through a pipe.") {
		t.Errorf("gap.md missing stdin body content:\n%s", got)
	}
}

// TestEditBody_RejectsFrontmatter_BinaryEndToEnd: the dispatcher
// passes content through to the verb, which refuses leading-`---`
// content. Exit non-zero, no commit produced.
func TestEditBody_RejectsFrontmatter_BinaryEndToEnd(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Edit target"); err != nil {
		t.Fatalf("add epic: %v\n%s", err, out)
	}

	bad := "---\nid: PRETEND\n---\n\n## body\n"
	bodyPath := filepath.Join(root, "bad-body.md")
	if err := os.WriteFile(bodyPath, []byte(bad), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	out, err := runBin(t, root, binDir, nil, "edit-body", "E-01", "--body-file", bodyPath)
	if err == nil {
		t.Fatalf("expected refusal; got:\n%s", out)
	}
	if !strings.Contains(out, "frontmatter delimiter") {
		t.Errorf("expected frontmatter-delimiter message; got:\n%s", out)
	}
}

// TestEditBody_Bless_BinaryEndToEnd is the M-060 dispatcher-seam
// closure: a real subprocess invocation of `aiwf edit-body <id>`
// (no --body-file) reads the working-copy edit, validates, and
// commits with edit-body trailers. Without this test, a regression
// that drops the "body == nil → bless mode" branch from the
// dispatcher would still pass internal/verb tests.
func TestEditBody_Bless_BinaryEndToEnd(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Bless target"); err != nil {
		t.Fatalf("add epic: %v\n%s", err, out)
	}

	// Simulate the user editing the epic body in $EDITOR.
	matches, _ := filepath.Glob(filepath.Join(root, "work", "epics", "E-*", "epic.md"))
	if len(matches) != 1 {
		t.Fatalf("expected one epic.md; got %v", matches)
	}
	epicPath := matches[0]
	original, err := os.ReadFile(epicPath)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(original), "## Goal", "## Goal\n\nUser-edited goal prose, written in place.", 1)
	if writeErr := os.WriteFile(epicPath, []byte(edited), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}

	// Bless: no --body-file flag.
	out, err := runBin(t, root, binDir, nil, "edit-body", "E-01")
	if err != nil {
		t.Fatalf("aiwf edit-body (bless): %v\n%s", err, out)
	}

	// Edit landed in the committed file.
	got, err := os.ReadFile(epicPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "User-edited goal prose, written in place.") {
		t.Errorf("epic missing edited body content:\n%s", got)
	}

	// Trailer set is the standard edit-body triple — bless mode is
	// not distinguishable from explicit mode in `aiwf history`,
	// which is the right outcome.
	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	hasTrailer(t, tr, "aiwf-verb", "edit-body")
	hasTrailer(t, tr, "aiwf-entity", "E-01")

	// `aiwf check` doesn't surface provenance-untrailered-entity-commit
	// against this commit — bless mode produces a proper trailered
	// commit, closing the workflow gap that G-052 documented and
	// G-054 surfaced as M-058's residual.
	checkOut, err := runBin(t, root, binDir, nil, "check")
	if err != nil {
		t.Fatalf("aiwf check after bless: %v\n%s", err, checkOut)
	}
	if strings.Contains(checkOut, "provenance-untrailered-entity-commit") {
		t.Errorf("post-bless check surfaces untrailered-entity warning:\n%s", checkOut)
	}
}

// TestEditBody_Bless_NoChanges_BinaryRefusal: bless mode against
// a clean working copy refuses with a clear "no changes" message
// rather than producing an empty commit.
func TestEditBody_Bless_NoChanges_BinaryRefusal(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "gap", "--title", "Quiet gap"); err != nil {
		t.Fatalf("add gap: %v\n%s", err, out)
	}

	// No edit between add and bless — should refuse cleanly.
	out, err := runBin(t, root, binDir, nil, "edit-body", "G-001")
	if err == nil {
		t.Fatalf("expected no-changes refusal; got:\n%s", out)
	}
	if !strings.Contains(out, "no changes to commit") {
		t.Errorf("expected 'no changes to commit' message; got:\n%s", out)
	}
}

// TestEditBody_BareCommand_BlessModeOnNonExistentID: omitting
// --body-file enters bless mode (M-060). With no entity at the
// given id, the verb refuses with "entity not found" — a different
// shape from the pre-M-060 "--body-file required" usage error, but
// equally clear.
func TestEditBody_BareCommand_BlessModeOnNonExistentID(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}

	out, err := runBin(t, root, binDir, nil, "edit-body", "E-01")
	if err == nil {
		t.Fatalf("expected refusal on non-existent id; got:\n%s", out)
	}
	if !strings.Contains(out, "not found") {
		t.Errorf("expected entity-not-found message; got:\n%s", out)
	}
}
