package check

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// TestRunProvenanceCheck_TrailerVerbUnknown_FiresOnUnpushedFabrication
// is the seam test for the new G-0150 wiring: a fabricated `aiwf-verb`
// trailer on an unpushed commit must surface a trailer-verb-unknown
// finding through the full RunProvenanceCheck chain (RunE-style
// invocation skipped — we exercise the orchestrator directly, the
// caller-side enumeration is tested in verbs_test.go).
//
// We use --since rather than @{u}..HEAD so the test doesn't need to
// configure upstream tracking — the rangeArg ResolveUntrailedRange
// returns is the same `<sha>..HEAD` shape either way.
//
// Closes G-0150.
func TestRunProvenanceCheck_TrailerVerbUnknown_FiresOnUnpushedFabrication(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ctx := context.Background()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	// C0: baseline commit, no aiwf-* trailers. Used as the --since
	// anchor so C1 is the only commit in range.
	seed := filepath.Join(root, "seed.md")
	if err := os.WriteFile(seed, []byte("seed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(ctx, root, "seed.md"); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(ctx, root, "baseline", "", nil); err != nil {
		t.Fatal(err)
	}
	c0 := headSHA(t, root)

	// C1: a hand-rolled `feat(...)` style commit carrying the gap's
	// worked LLM-fabrication example as an `aiwf-verb:` trailer.
	more := filepath.Join(root, "more.md")
	if err := os.WriteFile(more, []byte("more\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(ctx, root, "more.md"); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(ctx, root, "feat(check): implement something",
		"", []gitops.Trailer{{Key: gitops.TrailerVerb, Value: "implement"}}); err != nil {
		t.Fatal(err)
	}

	registered := map[string]struct{}{
		"add":     {},
		"promote": {},
		// "implement" deliberately absent
	}
	findings, err := RunProvenanceCheck(ctx, root, &tree.Tree{}, c0, registered)
	if err != nil {
		t.Fatalf("RunProvenanceCheck: %v", err)
	}
	var found *check.Finding
	for i := range findings {
		if findings[i].Code == check.CodeTrailerVerbUnknown {
			found = &findings[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("trailer-verb-unknown finding did not fire; got %d findings", len(findings))
	}
	if !strings.Contains(found.Message, "implement") {
		t.Errorf("finding message must name the offending value; got %q", found.Message)
	}
}

// TestRunProvenanceCheck_TrailerVerbUnknown_SilentOnRegisteredVerb is
// the symmetric seam test for G-0150: a commit whose `aiwf-verb:`
// value IS in the registered set must produce NO trailer-verb-unknown
// finding through the full RunProvenanceCheck chain. The firing
// direction is pinned by
// TestRunProvenanceCheck_TrailerVerbUnknown_FiresOnUnpushedFabrication;
// this case catches the failure mode where the verb set arrives at
// the rule empty or misshapen (which would let every commit fire).
//
// Closes G-0150.
func TestRunProvenanceCheck_TrailerVerbUnknown_SilentOnRegisteredVerb(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ctx := context.Background()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	seed := filepath.Join(root, "seed.md")
	if err := os.WriteFile(seed, []byte("seed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(ctx, root, "seed.md"); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(ctx, root, "baseline", "", nil); err != nil {
		t.Fatal(err)
	}
	c0 := headSHA(t, root)

	// C1 carries a registered verb — promote.
	more := filepath.Join(root, "more.md")
	if err := os.WriteFile(more, []byte("more\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(ctx, root, "more.md"); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(ctx, root, "aiwf promote E-0001 draft -> active",
		"", []gitops.Trailer{{Key: gitops.TrailerVerb, Value: "promote"}}); err != nil {
		t.Fatal(err)
	}

	registered := map[string]struct{}{
		"add":     {},
		"promote": {},
	}
	findings, err := RunProvenanceCheck(ctx, root, &tree.Tree{}, c0, registered)
	if err != nil {
		t.Fatalf("RunProvenanceCheck: %v", err)
	}
	for i := range findings {
		if findings[i].Code == check.CodeTrailerVerbUnknown {
			t.Errorf("trailer-verb-unknown must NOT fire on registered verb; got: %s", findings[i].Message)
		}
	}
}

// headSHA returns the abbreviated HEAD SHA of root for use as a
// --since anchor.
func headSHA(t *testing.T, root string) string {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", "rev-parse", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// TestAsScopeCommits_EmptyInputReturnsNil guards the fast path: no
// untrailered commits in range → no allocation, no findings.
//
// Closes G-0150.
func TestAsScopeCommits_EmptyInputReturnsNil(t *testing.T) {
	t.Parallel()
	if got := asScopeCommits(nil); got != nil {
		t.Errorf("got %v, want nil for nil input", got)
	}
	if got := asScopeCommits([]check.UntrailedCommit{}); got != nil {
		t.Errorf("got %v, want nil for empty input", got)
	}
}

// TestAsScopeCommits_CopiesSHAAndTrailers pins the adapter's
// contract: SHA + Trailers flow through; other UntrailedCommit
// fields (Subject, Paths) are intentionally dropped — the
// trailer-verb rule needs only SHA and Trailers.
//
// Closes G-0150.
func TestAsScopeCommits_CopiesSHAAndTrailers(t *testing.T) {
	t.Parallel()
	in := []check.UntrailedCommit{
		{
			SHA:     "aaa1111",
			Subject: "ignored by the adapter",
			Trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "promote"},
				{Key: gitops.TrailerActor, Value: "human/peter"},
			},
			Paths: []string{"work/gaps/G-0001-foo.md"},
		},
		{SHA: "bbb2222"}, // no trailers, no paths
	}
	got := asScopeCommits(in)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].SHA != "aaa1111" {
		t.Errorf("got[0].SHA = %q, want aaa1111", got[0].SHA)
	}
	if len(got[0].Trailers) != 2 {
		t.Errorf("got[0].Trailers len = %d, want 2", len(got[0].Trailers))
	}
	if got[1].SHA != "bbb2222" {
		t.Errorf("got[1].SHA = %q, want bbb2222", got[1].SHA)
	}
	if len(got[1].Trailers) != 0 {
		t.Errorf("got[1].Trailers len = %d, want 0", len(got[1].Trailers))
	}
}

// TestRunProvenanceCheck_EmptyRepoIsNoop pins the fast-path: when the
// root isn't a git repo (no HEAD), RunProvenanceCheck returns nil
// without erroring on the absent git log.
func TestRunProvenanceCheck_EmptyRepoIsNoop(t *testing.T) {
	t.Parallel()
	findings, err := RunProvenanceCheck(context.Background(), t.TempDir(), &tree.Tree{}, "", nil)
	if err != nil {
		t.Fatalf("RunProvenanceCheck: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected nil findings on non-git tempdir; got %+v", findings)
	}
}

// TestParseUntrailedCommits_EmptyInput pins the parser's empty-input
// branch. Other parser shapes are exercised via the cmd/aiwf-side
// integration tests (TestParseUntrailedCommits_Malformed) that
// migrate with the rest of the integration test set in AC-6.
func TestParseUntrailedCommits_EmptyInput(t *testing.T) {
	t.Parallel()
	got := ParseUntrailedCommits("")
	if len(got) != 0 {
		t.Errorf("ParseUntrailedCommits(\"\") = %+v, want empty", got)
	}
}
