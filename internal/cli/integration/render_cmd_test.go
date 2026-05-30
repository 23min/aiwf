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

// TestRun_RenderRoadmap_WriteCommits: --write writes ROADMAP.md and
// produces a commit with structured trailers. A second --write is a
// no-op (HEAD doesn't advance) because content is unchanged.
func TestRun_RenderRoadmap_WriteCommits(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := cli.Execute([]string{"add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("add: %d", rc)
	}

	if rc := cli.Execute([]string{"render", "roadmap", "--write", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("render --write: %d", rc)
	}

	body, err := os.ReadFile(filepath.Join(root, "ROADMAP.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "Foundations") {
		t.Errorf("ROADMAP.md missing epic title:\n%s", body)
	}

	ctx := context.Background()
	subj, err := gitops.HeadSubject(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if subj != "aiwf render roadmap" {
		t.Errorf("HEAD subject = %q, want %q", subj, "aiwf render roadmap")
	}
	trailers, err := gitops.HeadTrailers(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	wantPairs := map[string]string{
		"aiwf-verb":  "render-roadmap",
		"aiwf-actor": "human/test",
	}
	for _, tr := range trailers {
		if want, ok := wantPairs[tr.Key]; ok {
			if tr.Value != want {
				t.Errorf("trailer %s = %q, want %q", tr.Key, tr.Value, want)
			}
			delete(wantPairs, tr.Key)
		}
	}
	for k := range wantPairs {
		t.Errorf("missing trailer %q", k)
	}

	// Second --write with no tree changes should be a no-op.
	subjectBefore, _ := gitops.HeadSubject(ctx, root)
	if rc := cli.Execute([]string{"render", "roadmap", "--write", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("re-render --write: %d", rc)
	}
	subjectAfter, _ := gitops.HeadSubject(ctx, root)
	if subjectAfter != subjectBefore {
		t.Errorf("idempotent --write should not advance HEAD: %q -> %q", subjectBefore, subjectAfter)
	}
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

	if rc := cli.Execute([]string{"render", "roadmap", "--write", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
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

	// The commit staged roadmap.md: HEAD advanced with the render
	// subject, and the working tree is clean afterward (the regenerated
	// content is committed, not left dangling).
	ctx := context.Background()
	subj, err := gitops.HeadSubject(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if subj != "aiwf render roadmap" {
		t.Errorf("HEAD subject = %q, want %q", subj, "aiwf render roadmap")
	}
	staged, err := gitops.StagedPaths(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(staged) != 0 {
		t.Errorf("working tree should be clean after the render commit; staged: %v", staged)
	}
	// The HEAD commit changed roadmap.md and not ROADMAP.md.
	changed := headChangedFiles(t, root)
	if !containsString(changed, "roadmap.md") {
		t.Errorf("HEAD commit did not change roadmap.md; changed: %v", changed)
	}
	if containsString(changed, "ROADMAP.md") {
		t.Errorf("HEAD commit unexpectedly changed ROADMAP.md; changed: %v", changed)
	}
}

// headChangedFiles returns the repo-relative paths changed by the HEAD
// commit, via `git show --name-only`. Used to assert which roadmap
// variant the render commit actually touched.
func headChangedFiles(t *testing.T, root string) []string {
	t.Helper()
	out, err := exec.Command("git", "-C", root, "show", "--name-only", "--pretty=format:", "HEAD").Output()
	if err != nil {
		t.Fatalf("git show HEAD: %v", err)
	}
	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files
}

// TestRun_RenderRoadmap_ReconcilesLowercase_Idempotent: a second --write
// against the reconciled lowercase file with unchanged content is a
// no-op (HEAD does not advance), keyed off the resolved name.
func TestRun_RenderRoadmap_ReconcilesLowercase_Idempotent(t *testing.T) {
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

	// First --write reconciles to roadmap.md and commits.
	if rc := cli.Execute([]string{"render", "roadmap", "--write", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("first render --write: %d", rc)
	}
	ctx := context.Background()
	subjectBefore, err := gitops.HeadSubject(ctx, root)
	if err != nil {
		t.Fatal(err)
	}

	// Second --write with no tree changes must be a no-op.
	if rc := cli.Execute([]string{"render", "roadmap", "--write", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("second render --write: %d", rc)
	}
	subjectAfter, err := gitops.HeadSubject(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if subjectAfter != subjectBefore {
		t.Errorf("idempotent --write advanced HEAD: %q -> %q", subjectBefore, subjectAfter)
	}
	if _, statErr := os.Stat(filepath.Join(root, "ROADMAP.md")); !os.IsNotExist(statErr) {
		t.Errorf("ROADMAP.md should not exist after idempotent reconcile (err=%v)", statErr)
	}
}

// TestRun_RenderRoadmap_StagedLowercaseGuard: the staged-edit guard
// trips (usage exit, no commit) when the resolved lowercase roadmap.md
// is already staged with the user's own edits — the case-insensitive
// guard catches the variant the old case-sensitive compare would miss.
func TestRun_RenderRoadmap_StagedLowercaseGuard(t *testing.T) {
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

	ctx := context.Background()
	subjectBefore, err := gitops.HeadSubject(ctx, root)
	if err != nil {
		t.Fatal(err)
	}

	if rc := cli.Execute([]string{"render", "roadmap", "--write", "--actor", "human/test", "--root", root}); rc != cliutil.ExitUsage {
		t.Fatalf("staged roadmap.md should trip the guard with usage exit; got %d", rc)
	}

	subjectAfter, err := gitops.HeadSubject(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if subjectAfter != subjectBefore {
		t.Errorf("guard should prevent a commit: HEAD advanced %q -> %q", subjectBefore, subjectAfter)
	}
}

// TestRun_RenderRoadmap_FreshRepoCreatesCanonical: a fresh repo with no
// roadmap file gets the canonical uppercase ROADMAP.md created — the
// unchanged default behavior when there is nothing to reconcile.
func TestRun_RenderRoadmap_FreshRepoCreatesCanonical(t *testing.T) {
	t.Parallel()
	root := initRepoWithEpic(t)

	if rc := cli.Execute([]string{"render", "roadmap", "--write", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
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
