package verb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/verb"
)

// M-0186/AC-6 pins two independent claims, both provable without a
// git hook ever running: `git commit-tree` (the primitive `Apply`
// commits through as of AC-3/AC-5) fires no git hooks at all, so the
// pre-commit hook's `aiwf check --shape-only` never runs against a
// verb's own commit. That hook only ever ran `check.TreeDiscipline`
// (stray-file tree layout) — it never validated frontmatter shape
// (ADR-0029) — so its absence from the verb-commit path changes
// nothing for frontmatter-shape enforcement, which has always lived
// in each verb's own pre-write projection check instead. What it DOES
// remove is TreeDiscipline running at commit time for a verb commit.
// Both tests below drive `verb.Apply` directly with a hand-built
// *Plan — bypassing the projection check a real verb (add/promote/…)
// would normally run first — then confirm full `aiwf check` (the
// pre-push boundary; not `--shape-only`) still catches the resulting
// violation. Neither test's plan goes through a real git hook: this
// package never installs one, matching how a `commit-tree`-built
// commit never triggers one in production either.

// TestApply_MalformedFrontmatterStillCaughtByFullCheck bypasses a
// verb's own projectionFindings/check.Run guard (ADR-0029) by driving
// verb.Apply directly with a Plan that writes an entity missing its
// required `status` field. Apply performs no content validation
// (that's the point — see ADR-0029), so the commit lands; full `aiwf
// check` must still flag it via `frontmatterShape` at the pre-push
// boundary — proving that guarantee never depended on the retired
// pre-commit hook.
// Not t.Parallel(): testutil.CaptureStdout swaps the process-wide
// os.Stdout for the duration of the callback, so two CaptureStdout
// calls running concurrently would interleave each other's captured
// output (see internal/policies/capture_stdout_singleton.go and the
// existing check_shape_only_test.go tests, which are serial for the
// same reason).
func TestApply_MalformedFrontmatterStillCaughtByFullCheck(t *testing.T) {
	r := newApplyTestRepo(t)

	plan := &verb.Plan{
		Subject:  "bypass: write a shape-malformed entity",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			{Type: verb.OpWrite, Path: "work/gaps/G-0001-broken.md", Content: []byte("---\nid: G-0001\n---\nmissing its status field\n")},
		},
	}
	if _, err := verb.Apply(r.ctx, r.root, plan); err != nil {
		t.Fatalf("apply: %v (Apply must not itself reject malformed content — that's not its job)", err)
	}

	captured := testutil.CaptureStdout(t, func() {
		if rc := cli.Execute([]string{"check", "--root", r.root}); rc != cliutil.ExitFindings {
			t.Errorf("got rc=%d, want %d (frontmatter-shape is error severity)", rc, cliutil.ExitFindings)
		}
	})
	out := string(captured)
	// Tied to the offending path on the same line — not just "the code
	// appears somewhere" — so this can't pass on the shared fixture's
	// own pre-existing frontmatter-shape violation (newApplyTestRepo's
	// seed entity is also missing `status`, at a different path). A
	// milestone-wrap review found the untied checks below were
	// independently satisfiable by that unrelated seed defect.
	want := "work/gaps/G-0001-broken.md:1: error " + check.CodeFrontmatterShape
	if !strings.Contains(out, want) {
		t.Errorf("expected %q in output:\n%s", want, out)
	}
}

// TestApply_StrayFileStillCaughtByFullCheck bypasses the same guard
// with a Plan that writes a file under work/gaps/ whose name doesn't
// match the entity naming convention (no `G-NNNN-` id prefix) —
// tree.Load classifies a file as a "stray" by path shape alone
// (entity.PathKind), before it ever attempts to parse content, so this
// fires regardless of what the file contains (confirmed by mutation:
// giving "scratch.md" valid frontmatter does not stop it from being a
// stray; renaming it to a `G-NNNN-`-prefixed path does). This is the
// check class the pre-commit hook actually ran (TreeDiscipline) before
// M-0186 retired hook-firing from the verb-commit path. With
// `tree.strict: true`, full `aiwf check` must still block on it at
// the pre-push boundary — proving no silent gap was left behind.
// Not t.Parallel() — see the comment on the sibling test above.
func TestApply_StrayFileStillCaughtByFullCheck(t *testing.T) {
	r := newApplyTestRepo(t)

	if err := os.WriteFile(filepath.Join(r.root, "aiwf.yaml"), []byte("tree:\n  strict: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan := &verb.Plan{
		Subject:  "bypass: write a stray file under work/gaps",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			{Type: verb.OpWrite, Path: "work/gaps/scratch.md", Content: []byte("not an entity\n")},
		},
	}
	if _, err := verb.Apply(r.ctx, r.root, plan); err != nil {
		t.Fatalf("apply: %v (Apply must not itself reject a stray path — that's not its job)", err)
	}

	captured := testutil.CaptureStdout(t, func() {
		if rc := cli.Execute([]string{"check", "--root", r.root}); rc != cliutil.ExitFindings {
			t.Errorf("got rc=%d, want %d (tree.strict promotes the stray to error severity)", rc, cliutil.ExitFindings)
		}
	})
	out := string(captured)
	// Asserting "error" (not just the code's presence) is what actually
	// makes tree.strict load-bearing: without it, TreeDiscipline's
	// default severity is warning, and this exact string would not
	// appear even though the code and path still would. A milestone-wrap
	// review found the prior version's untied checks passed identically
	// whether or not tree.strict was set, since the shared fixture's own
	// pre-existing frontmatter-shape violation drove ExitFindings on its
	// own regardless.
	want := "work/gaps/scratch.md: error " + check.CodeUnexpectedTreeFile
	if !strings.Contains(out, want) {
		t.Errorf("expected %q in output:\n%s", want, out)
	}
}
