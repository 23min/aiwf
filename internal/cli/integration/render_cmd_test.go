package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/gitops"
)

// TestRun_RenderRoadmap_Stdout: a freshly-init'd repo with one epic
// and one milestone produces a markdown table on stdout. No commit
// lands without --write.
func TestRun_RenderRoadmap_Stdout(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := cli.Execute([]string{"add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("add epic: %d", rc)
	}
	if rc := cli.Execute([]string{"add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Schema", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("add milestone: %d", rc)
	}

	subjectBefore, err := gitops.HeadSubject(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}

	captured := testutil.CaptureStdout(t, func() {
		if rc := cli.Execute([]string{"render", "roadmap", "--root", root}); rc != cliutil.ExitOK {
			t.Fatalf("render roadmap: %d", rc)
		}
	})

	out := string(captured)
	for _, want := range []string{
		"# Roadmap",
		"## E-0001 — Foundations (proposed)",
		"| M-0001 | Schema | draft |",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout missing %q:\n%s", want, out)
		}
	}

	// No commit landed.
	if _, statErr := os.Stat(filepath.Join(root, "ROADMAP.md")); !os.IsNotExist(statErr) {
		t.Errorf("ROADMAP.md should not exist after stdout render: err=%v", statErr)
	}
	subjectAfter, err := gitops.HeadSubject(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if subjectAfter != subjectBefore {
		t.Errorf("HEAD advanced without --write: %q -> %q", subjectBefore, subjectAfter)
	}
}

// TestRun_RenderRoadmap_WriteNoCommit: --write writes ROADMAP.md to
// disk only — no commit, no trailers (G-0350). HEAD never advances;
// committing the file is the caller's concern. A second --write with
// unchanged content is a no-op, reported via the "already up to date"
// message rather than a re-write.
//
// Serial, not t.Parallel(): calls testutil.CaptureStdout, which
// mutates the process-level os.Stdout fd (see setup_test.go's serial
// skip-list rationale).
func TestRun_RenderRoadmap_WriteNoCommit(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := cli.Execute([]string{"add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("add: %d", rc)
	}

	ctx := context.Background()
	subjectBefore, err := gitops.HeadSubject(ctx, root)
	if err != nil {
		t.Fatal(err)
	}

	if rc := cli.Execute([]string{"render", "roadmap", "--write", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("render --write: %d", rc)
	}

	body, err := os.ReadFile(filepath.Join(root, "ROADMAP.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "Foundations") {
		t.Errorf("ROADMAP.md missing epic title:\n%s", body)
	}

	subjectAfter, err := gitops.HeadSubject(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if subjectAfter != subjectBefore {
		t.Errorf("--write must not commit: HEAD advanced %q -> %q", subjectBefore, subjectAfter)
	}
	staged, err := gitops.StagedPaths(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(staged) != 0 {
		t.Errorf("--write must not stage anything either; staged: %v", staged)
	}
	if changed := workingTreeModifiedFiles(t, root); !containsString(changed, "ROADMAP.md") {
		t.Errorf("ROADMAP.md should show as an unstaged working-tree change: %v", changed)
	}

	// Second --write with no tree changes should be a no-op: reported
	// via the "already up to date" message, not a re-write.
	out := testutil.CaptureStdout(t, func() {
		if rc := cli.Execute([]string{"render", "roadmap", "--write", "--root", root}); rc != cliutil.ExitOK {
			t.Fatalf("re-render --write: %d", rc)
		}
	})
	if !strings.Contains(string(out), "already up to date") {
		t.Errorf("idempotent --write should report up-to-date, got:\n%s", out)
	}
}

// workingTreeModifiedFiles returns the repo-relative paths `git status
// --porcelain` reports as changed — tracked-modified or untracked.
// Callers pair this with an explicit gitops.StagedPaths check to
// confirm the change is unstaged, since porcelain's two-column status
// covers both staged and unstaged state.
func workingTreeModifiedFiles(t *testing.T, root string) []string {
	t.Helper()
	out, err := exec.Command("git", "-C", root, "status", "--porcelain").Output()
	if err != nil {
		t.Fatalf("git status: %v", err)
	}
	var files []string
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if len(line) < 4 {
			continue
		}
		files = append(files, strings.TrimSpace(line[3:]))
	}
	return files
}

// TestRun_RenderRoadmap_UnknownSubcommand reports a usage error.
func TestRun_RenderRoadmap_UnknownSubcommand(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if got := cli.Execute([]string{"render", "treemap", "--root", root}); got != cliutil.ExitUsage {
		t.Errorf("got %d, want %d", got, cliutil.ExitUsage)
	}
	if got := cli.Execute([]string{"render", "--root", root}); got != cliutil.ExitUsage {
		t.Errorf("got %d, want %d (no subcommand)", got, cliutil.ExitUsage)
	}
}

// TestRun_RenderRoadmap_EmptyRepo prints the empty-tree placeholder
// without errors and without writing a commit.
func TestRun_RenderRoadmap_EmptyRepo(t *testing.T) {
	root := setupCLITestRepo(t)
	captured := testutil.CaptureStdout(t, func() {
		if rc := cli.Execute([]string{"render", "roadmap", "--root", root}); rc != cliutil.ExitOK {
			t.Fatalf("render: %d", rc)
		}
	})
	if !strings.Contains(string(captured), "_No epics yet._") {
		t.Errorf("empty-tree marker missing:\n%s", captured)
	}
}

// initRepoWithEpic inits a repo and adds one epic, returning the root.
// Shared by the case-reconciliation tests below.
func initRepoWithEpic(t *testing.T) string {
	t.Helper()
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := cli.Execute([]string{"add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("add epic: %d", rc)
	}
	return root
}

// TestRun_RenderRoadmap_ReconcilesLowercase: a repo that already tracks
// a lowercase `roadmap.md` (a legitimate consumer convention) gets that
// file updated by --write — not a second divergent `ROADMAP.md`. This is
// the G-0185 cross-filesystem fix: on a case-sensitive filesystem the
// old code created `ROADMAP.md` and left `roadmap.md` stale.
func TestRun_RenderRoadmap_ReconcilesLowercase(t *testing.T) {
	t.Parallel()
	root := initRepoWithEpic(t)

	// Seed a lowercase roadmap.md tracked in git, with a hand-curated
	// Candidates block that must survive the regenerate.
	lower := filepath.Join(root, "roadmap.md")
	seed := "# Roadmap\n\n## Candidates\n\n- hand-curated idea\n"
	if err := os.WriteFile(lower, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(context.Background(), root, "roadmap.md"); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(context.Background(), root, "seed roadmap.md", "", nil); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	subjectBefore, err := gitops.HeadSubject(ctx, root)
	if err != nil {
		t.Fatal(err)
	}

	if rc := cli.Execute([]string{"render", "roadmap", "--write", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("render --write: %d", rc)
	}

	// The lowercase file is the one that got updated...
	body, err := os.ReadFile(lower)
	if err != nil {
		t.Fatalf("read roadmap.md: %v", err)
	}
	if !strings.Contains(string(body), "Foundations") {
		t.Errorf("roadmap.md missing regenerated epic title:\n%s", body)
	}
	// ...the hand-curated Candidates block survived...
	if !strings.Contains(string(body), "hand-curated idea") {
		t.Errorf("roadmap.md lost its hand-curated Candidates block:\n%s", body)
	}
	// ...and no divergent uppercase ROADMAP.md was created.
	if _, statErr := os.Stat(filepath.Join(root, "ROADMAP.md")); !os.IsNotExist(statErr) {
		t.Errorf("ROADMAP.md should not exist; case-reconciliation must target roadmap.md (err=%v)", statErr)
	}

	// --write never commits (G-0350): HEAD stays put and roadmap.md
	// shows up as an unstaged working-tree change, not a commit.
	subjectAfter, err := gitops.HeadSubject(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if subjectAfter != subjectBefore {
		t.Errorf("--write must not commit: HEAD advanced %q -> %q", subjectBefore, subjectAfter)
	}
	staged, err := gitops.StagedPaths(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(staged) != 0 {
		t.Errorf("--write must not stage anything; staged: %v", staged)
	}
	changed := workingTreeModifiedFiles(t, root)
	if !containsString(changed, "roadmap.md") {
		t.Errorf("roadmap.md should show as an unstaged working-tree change: %v", changed)
	}
	if containsString(changed, "ROADMAP.md") {
		t.Errorf("ROADMAP.md unexpectedly touched: %v", changed)
	}
}

// TestRun_RenderRoadmap_ReconcilesLowercase_Idempotent: a second --write
// against the reconciled lowercase file with unchanged content is a
// no-op, reported via the "already up to date" message rather than a
// re-write, keyed off the resolved name.
//
// Serial, not t.Parallel(): calls testutil.CaptureStdout, which
// mutates the process-level os.Stdout fd (see setup_test.go's serial
// skip-list rationale).
func TestRun_RenderRoadmap_ReconcilesLowercase_Idempotent(t *testing.T) {
	root := initRepoWithEpic(t)

	lower := filepath.Join(root, "roadmap.md")
	if err := os.WriteFile(lower, []byte("# Roadmap\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(context.Background(), root, "roadmap.md"); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(context.Background(), root, "seed roadmap.md", "", nil); err != nil {
		t.Fatal(err)
	}

	// First --write reconciles to roadmap.md.
	if rc := cli.Execute([]string{"render", "roadmap", "--write", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("first render --write: %d", rc)
	}

	// Second --write with no tree changes must be a no-op.
	out := testutil.CaptureStdout(t, func() {
		if rc := cli.Execute([]string{"render", "roadmap", "--write", "--root", root}); rc != cliutil.ExitOK {
			t.Fatalf("second render --write: %d", rc)
		}
	})
	if !strings.Contains(string(out), "already up to date") {
		t.Errorf("idempotent --write should report up-to-date, got:\n%s", out)
	}
	if _, statErr := os.Stat(filepath.Join(root, "ROADMAP.md")); !os.IsNotExist(statErr) {
		t.Errorf("ROADMAP.md should not exist after idempotent reconcile (err=%v)", statErr)
	}
}

// TestRun_RenderRoadmap_WriteOverwritesDespiteStagedEdit: --write no
// longer touches git state at all (G-0350), so a pre-staged edit to
// the resolved roadmap file no longer blocks it — the write proceeds,
// overwrites the working-tree copy, and leaves the user's staged index
// entry untouched (committing, and reconciling any conflict with it,
// is the caller's concern now).
func TestRun_RenderRoadmap_WriteOverwritesDespiteStagedEdit(t *testing.T) {
	t.Parallel()
	root := initRepoWithEpic(t)

	lower := filepath.Join(root, "roadmap.md")
	if err := os.WriteFile(lower, []byte("# Roadmap\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(context.Background(), root, "roadmap.md"); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(context.Background(), root, "seed roadmap.md", "", nil); err != nil {
		t.Fatal(err)
	}

	// User makes a manual edit and stages it.
	if err := os.WriteFile(lower, []byte("# Roadmap\n\nmy own edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(context.Background(), root, "roadmap.md"); err != nil {
		t.Fatal(err)
	}

	if rc := cli.Execute([]string{"render", "roadmap", "--write", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("render --write should succeed despite a staged edit; got %d", rc)
	}

	body, err := os.ReadFile(lower)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "Foundations") {
		t.Errorf("write should have overwritten the working tree with regenerated content:\n%s", body)
	}

	// The index still carries the user's staged edit — write touched
	// only the working tree, not the index.
	ctx := context.Background()
	staged, err := gitops.StagedPaths(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(staged, "roadmap.md") {
		t.Errorf("the user's staged edit should remain in the index: staged=%v", staged)
	}
}

// TestRun_RenderRoadmap_FreshRepoCreatesCanonical: a fresh repo with no
// roadmap file gets the canonical uppercase ROADMAP.md created — the
// unchanged default behavior when there is nothing to reconcile.
func TestRun_RenderRoadmap_FreshRepoCreatesCanonical(t *testing.T) {
	t.Parallel()
	root := initRepoWithEpic(t)

	if rc := cli.Execute([]string{"render", "roadmap", "--write", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("render --write: %d", rc)
	}

	if _, err := os.Stat(filepath.Join(root, "ROADMAP.md")); err != nil {
		t.Errorf("canonical ROADMAP.md should be created in a fresh repo: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "roadmap.md")); !os.IsNotExist(err) {
		t.Errorf("no lowercase roadmap.md should be created in a fresh repo (err=%v)", err)
	}
}

// containsString reports whether s is in xs.
func containsString(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
