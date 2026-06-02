package check

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	clicontract "github.com/23min/aiwf/internal/cli/contract"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// TestRunProvenanceCheck_AC13_IsolationEscapeWired pins M-0106/AC-13's
// CLI-integration half: the RunProvenanceCheck function in this
// package contains a literal call to `check.RunIsolationEscape`.
// The function call is the wire-up that hooks the new kernel rule
// into the pre-push pipeline; without it the rule is dead code
// regardless of how complete its algorithm is.
//
// The assertion is AST-level (not a substring match on the source)
// per CLAUDE.md §"Substring assertions are not structural
// assertions". A regression that comments out the call,
// accidentally reorders the function so the call lives in dead
// code, or renames it fires this test.
//
// The test is deliberately strict on identifier shape: it matches
// any call expression whose `Fun` is a `*ast.SelectorExpr` with
// X.Name == "check" AND Sel.Name == "RunIsolationEscape". A regression
// that swaps the package alias breaks the package alias test, not
// this test; if a downstream rename is intended, the test fails
// loudly and the author updates it deliberately.
func TestRunProvenanceCheck_AC13_IsolationEscapeWired(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	path, err := filepath.Abs("provenance.go")
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parser.ParseFile(%s): %v", path, err)
	}

	var runProvenanceCheck *ast.FuncDecl
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fd.Name.Name == "RunProvenanceCheck" {
			runProvenanceCheck = fd
			break
		}
	}
	if runProvenanceCheck == nil {
		t.Fatal("RunProvenanceCheck function declaration not found in provenance.go")
	}

	var found bool
	ast.Inspect(runProvenanceCheck, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		x, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if x.Name == "check" && sel.Sel.Name == "RunIsolationEscape" {
			found = true
			return false
		}
		return true
	})

	if !found {
		t.Error("RunProvenanceCheck must contain a call to check.RunIsolationEscape — the wire-up that hooks M-0106's isolation-escape rule into the pre-push pipeline (AC-13)")
	}
}

// TestRunProvenanceCheck_IsolationEscape_FiresOnViolatingCommit
// pins M-0106/F-1 + F-8: the end-to-end seam from production
// `RunProvenanceCheck` through the git-backed BranchOracle into
// the rule's fire path. Without this test the AST-level wire-up
// assertion at TestRunProvenanceCheck_AC13_IsolationEscapeWired is
// the only seam pin — and it would pass even if the call were
// invoked with nil/garbage inputs that produced zero findings (the
// shipped-disabled failure mode F-1 caught the milestone in).
//
// Setup: a fresh git repo with `main` + an `epic/E-0001-engine`
// branch. Two commits land:
//
//  1. An authorize-opens-scope commit on main carrying
//     aiwf-branch: epic/E-0001-engine — the binding the rule will
//     compare against.
//  2. A SECOND commit on main carrying aiwf-actor: ai/claude +
//     aiwf-entity: E-0001 — an AI work commit landing on main
//     instead of the bound epic branch. Per AC-1, the rule must fire.
//
// The fixture would silently pass an AC-1 assertion if the oracle
// were nil — F-1 — so the test asserts the finding fires through
// the full chain.
func TestRunProvenanceCheck_IsolationEscape_FiresOnViolatingCommit(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ctx := context.Background()

	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	// git init defaults to master in some envs; force main so the
	// rule's bound-vs-actual comparison is deterministic.
	if out, err := exec.CommandContext(ctx, "git", "-C", root, "branch", "-M", "main").CombinedOutput(); err != nil {
		t.Fatalf("git branch -M main: %v\n%s", err, out)
	}

	// C0: baseline (no aiwf trailers). Anchors --since so the
	// untrailered audit window is empty; we want only the M-0106
	// pass to run findings.
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

	// C1: authorize-opens-scope on main, bound to epic/E-0001-engine.
	if err := os.WriteFile(filepath.Join(root, "auth.md"), []byte("auth\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(ctx, root, "auth.md"); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(ctx, root, "aiwf authorize E-0001 --to ai/claude --branch epic/E-0001-engine", "",
		[]gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "authorize"},
			{Key: gitops.TrailerEntity, Value: "E-0001"},
			{Key: gitops.TrailerActor, Value: "human/peter"},
			{Key: gitops.TrailerTo, Value: "ai/claude"},
			{Key: gitops.TrailerScope, Value: "opened"},
			{Key: gitops.TrailerBranch, Value: "epic/E-0001-engine"},
		}); err != nil {
		t.Fatalf("authorize commit: %v", err)
	}

	// Create the epic branch from this point so the oracle can
	// distinguish "the bound branch exists" from "the AI commit
	// landed somewhere else". The epic branch will share C0 + C1
	// initially; the AI commit at C2 lands ONLY on main.
	if out, err := exec.CommandContext(ctx, "git", "-C", root, "branch", "epic/E-0001-engine").CombinedOutput(); err != nil {
		t.Fatalf("git branch epic/E-0001-engine: %v\n%s", err, out)
	}

	// C2: AI-actor work commit on main — the escape.
	if err := os.WriteFile(filepath.Join(root, "work.md"), []byte("work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(ctx, root, "work.md"); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(ctx, root, "aiwf edit-body M-0001 (escaped)", "",
		[]gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "edit-body"},
			{Key: gitops.TrailerEntity, Value: "E-0001"},
			{Key: gitops.TrailerActor, Value: "ai/claude"},
		}); err != nil {
		t.Fatalf("ai work commit: %v", err)
	}

	registered := map[string]struct{}{
		"authorize": {},
		"edit-body": {},
	}
	findings, err := RunProvenanceCheck(ctx, root, &tree.Tree{}, c0, registered, nil)
	if err != nil {
		t.Fatalf("RunProvenanceCheck: %v", err)
	}

	// T-7 (third-pass review): assert EXACTLY ONE isolation-escape
	// finding, not "at least one." A regression that caused per-
	// commit firing to double-count would silently pass the
	// "first match wins" loop the original test used. Filter to
	// isolation-escape findings (other rules also fire on this
	// fixture — provenance-no-active-scope etc.) and pin
	// cardinality. Mirrors the WarningDoesNotMarkErrors filter
	// idiom for consistency across seam tests.
	var iso []check.Finding
	for _, f := range findings {
		if f.Code == check.CodeIsolationEscape.ID {
			iso = append(iso, f)
		}
	}
	if len(iso) != 1 {
		t.Fatalf("isolation-escape finding count = %d; want exactly 1 (F-1: oracle wire-up + AC-10: per-commit firing); all findings: %+v", len(iso), findings)
	}
	found := &iso[0]
	if found.Severity != check.SeverityWarning {
		t.Errorf("isolation-escape Severity = %q; want %q (F-4: AC-11 — warning, not error)", found.Severity, check.SeverityWarning)
	}
	if found.EntityID != "E-0001" {
		t.Errorf("isolation-escape EntityID = %q; want %q", found.EntityID, "E-0001")
	}
	if !strings.Contains(found.Message, "main") {
		t.Errorf("isolation-escape Message %q does not name the actual branch (main)", found.Message)
	}
	if !strings.Contains(found.Message, "epic/E-0001-engine") {
		t.Errorf("isolation-escape Message %q does not name the bound branch", found.Message)
	}
}

// TestRunProvenanceCheck_IsolationEscape_SilentOnBoundBranchCommit
// is the symmetric seam for F-1 + AC-4: when the AI commit rides
// the bound branch, the rule must NOT fire through the full chain.
// Without this test, a regression that always fires would surface
// only at the firing-test level — false positives would never get
// caught by the seam.
func TestRunProvenanceCheck_IsolationEscape_SilentOnBoundBranchCommit(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ctx := context.Background()

	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if out, err := exec.CommandContext(ctx, "git", "-C", root, "branch", "-M", "main").CombinedOutput(); err != nil {
		t.Fatalf("git branch -M main: %v\n%s", err, out)
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

	// C1: authorize-opens on main with bound branch.
	if err := os.WriteFile(filepath.Join(root, "auth.md"), []byte("auth\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(ctx, root, "auth.md"); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(ctx, root, "aiwf authorize E-0001", "",
		[]gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "authorize"},
			{Key: gitops.TrailerEntity, Value: "E-0001"},
			{Key: gitops.TrailerActor, Value: "human/peter"},
			{Key: gitops.TrailerTo, Value: "ai/claude"},
			{Key: gitops.TrailerScope, Value: "opened"},
			{Key: gitops.TrailerBranch, Value: "epic/E-0001-engine"},
		}); err != nil {
		t.Fatalf("authorize commit: %v", err)
	}

	// Cut + switch to the bound branch; the AI work commit lands here.
	if out, err := exec.CommandContext(ctx, "git", "-C", root, "checkout", "-b", "epic/E-0001-engine").CombinedOutput(); err != nil {
		t.Fatalf("git checkout -b epic/E-0001-engine: %v\n%s", err, out)
	}

	if err := os.WriteFile(filepath.Join(root, "work.md"), []byte("work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(ctx, root, "work.md"); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(ctx, root, "aiwf edit-body M-0001 (on bound)", "",
		[]gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "edit-body"},
			{Key: gitops.TrailerEntity, Value: "E-0001"},
			{Key: gitops.TrailerActor, Value: "ai/claude"},
		}); err != nil {
		t.Fatalf("ai work commit: %v", err)
	}

	registered := map[string]struct{}{"authorize": {}, "edit-body": {}}
	findings, err := RunProvenanceCheck(ctx, root, &tree.Tree{}, c0, registered, nil)
	if err != nil {
		t.Fatalf("RunProvenanceCheck: %v", err)
	}

	for _, f := range findings {
		if f.Code == check.CodeIsolationEscape.ID {
			t.Errorf("isolation-escape fired on bound-branch commit (false positive at the seam): %+v", f)
		}
	}
}

// TestRunProvenanceCheck_IsolationEscape_FindingCarriesHint pins
// M-0106/T-1 from the third-pass retrospective: a future regression
// that broke the hint-application chain (e.g., a refactor that
// moved `contract.ApplyHintsLikeRun` to a position where the
// provenance findings slice didn't flow through it, or a code
// re-organization that detached the isolation-escape rule from the
// post-processing pass) would silently ship findings without
// hints — readable when the operator hits one in CI but missing
// the override-path suggestion the AC-12 hint provides.
//
// The unit-level TestIsolationEscape_AC12_HintTextNamesBothOverridePaths
// pins the hintTable content; this test pins that the same hint
// reaches a real finding through the production composition.
// Together they cover the full hint surface.
//
// Note on test layering: `RunProvenanceCheck` itself does NOT call
// ApplyHintsLikeRun; the outer `Run` (which composes provenance +
// other rules) does. To test the seam through `Run`'s chain we
// invoke `contract.ApplyHintsLikeRun` manually on the
// RunProvenanceCheck output — mirroring exactly what the `Run`
// orchestrator does (see internal/cli/check/check.go:158, 224).
// The cleaner shape would be a full `Run`-driven integration test
// fixture, but those exist elsewhere and inflating that surface
// here is YAGNI.
func TestRunProvenanceCheck_IsolationEscape_FindingCarriesHint(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ctx := context.Background()

	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if out, err := exec.CommandContext(ctx, "git", "-C", root, "branch", "-M", "main").CombinedOutput(); err != nil {
		t.Fatalf("git branch -M main: %v\n%s", err, out)
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

	if err := os.WriteFile(filepath.Join(root, "auth.md"), []byte("auth\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(ctx, root, "auth.md"); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(ctx, root, "aiwf authorize E-0001 --to ai/claude --branch epic/E-0001-engine", "",
		[]gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "authorize"},
			{Key: gitops.TrailerEntity, Value: "E-0001"},
			{Key: gitops.TrailerActor, Value: "human/peter"},
			{Key: gitops.TrailerTo, Value: "ai/claude"},
			{Key: gitops.TrailerScope, Value: "opened"},
			{Key: gitops.TrailerBranch, Value: "epic/E-0001-engine"},
		}); err != nil {
		t.Fatalf("authorize commit: %v", err)
	}
	if out, err := exec.CommandContext(ctx, "git", "-C", root, "branch", "epic/E-0001-engine").CombinedOutput(); err != nil {
		t.Fatalf("git branch epic/E-0001-engine: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(root, "work.md"), []byte("work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(ctx, root, "work.md"); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(ctx, root, "aiwf edit-body M-0001 (escaped)", "",
		[]gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "edit-body"},
			{Key: gitops.TrailerEntity, Value: "E-0001"},
			{Key: gitops.TrailerActor, Value: "ai/claude"},
		}); err != nil {
		t.Fatalf("ai work commit: %v", err)
	}

	registered := map[string]struct{}{"authorize": {}, "edit-body": {}}
	findings, err := RunProvenanceCheck(ctx, root, &tree.Tree{}, c0, registered, nil)
	if err != nil {
		t.Fatalf("RunProvenanceCheck: %v", err)
	}

	// Apply hints exactly as the outer `Run` does.
	clicontract.ApplyHintsLikeRun(findings)

	var iso *check.Finding
	for i := range findings {
		if findings[i].Code == check.CodeIsolationEscape.ID {
			iso = &findings[i]
			break
		}
	}
	if iso == nil {
		t.Fatalf("isolation-escape finding not present; cannot pin hint-flow")
	}
	if iso.Hint == "" {
		t.Fatal("isolation-escape finding has empty Hint after ApplyHintsLikeRun — hint-table-to-finding flow broken")
	}
	if !strings.Contains(iso.Hint, "cherry-pick -x") {
		t.Errorf("isolation-escape Hint %q does not include cherry-pick override (AC-12 anchor)", iso.Hint)
	}
	if !strings.Contains(iso.Hint, "aiwf-force") {
		t.Errorf("isolation-escape Hint %q does not include aiwf-force override (AC-12 anchor)", iso.Hint)
	}
}

// TestRunProvenanceCheck_IsolationEscape_WarningDoesNotMarkErrors
// closes M-0106/N-2 from the second-pass retrospective. The
// original AC-11 wrap claimed "the exit-code half is enforced by
// the existing CLI check machinery"; the second-pass reviewer
// flagged that no test actually asserted exit-code 0 with an
// isolation-escape warning present. The exit code in
// internal/cli/check/check.go:195-198 maps to ExitFindings only
// when check.HasErrors returns true; a warning-only set must
// leave HasErrors false.
//
// This test drives the same fixture as
// TestRunProvenanceCheck_IsolationEscape_FiresOnViolatingCommit
// and then asserts the rule's effect at the exit-code seam:
// findings contain isolation-escape AND HasErrors is false. A
// regression that flipped the severity to SeverityError would
// fire this test on the HasErrors assertion; a regression that
// dropped the finding would fire the firing-test counterpart.
//
// The two tests jointly pin the AC-11 "warning, check exits 0"
// claim end-to-end through the production composition.
func TestRunProvenanceCheck_IsolationEscape_WarningDoesNotMarkErrors(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ctx := context.Background()

	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if out, err := exec.CommandContext(ctx, "git", "-C", root, "branch", "-M", "main").CombinedOutput(); err != nil {
		t.Fatalf("git branch -M main: %v\n%s", err, out)
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

	if err := os.WriteFile(filepath.Join(root, "auth.md"), []byte("auth\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(ctx, root, "auth.md"); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(ctx, root, "aiwf authorize E-0001 --to ai/claude --branch epic/E-0001-engine", "",
		[]gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "authorize"},
			{Key: gitops.TrailerEntity, Value: "E-0001"},
			{Key: gitops.TrailerActor, Value: "human/peter"},
			{Key: gitops.TrailerTo, Value: "ai/claude"},
			{Key: gitops.TrailerScope, Value: "opened"},
			{Key: gitops.TrailerBranch, Value: "epic/E-0001-engine"},
		}); err != nil {
		t.Fatalf("authorize commit: %v", err)
	}
	if out, err := exec.CommandContext(ctx, "git", "-C", root, "branch", "epic/E-0001-engine").CombinedOutput(); err != nil {
		t.Fatalf("git branch epic/E-0001-engine: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(root, "work.md"), []byte("work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(ctx, root, "work.md"); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(ctx, root, "aiwf edit-body M-0001 (escaped)", "",
		[]gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "edit-body"},
			{Key: gitops.TrailerEntity, Value: "E-0001"},
			{Key: gitops.TrailerActor, Value: "ai/claude"},
		}); err != nil {
		t.Fatalf("ai work commit: %v", err)
	}

	registered := map[string]struct{}{"authorize": {}, "edit-body": {}}
	findings, err := RunProvenanceCheck(ctx, root, &tree.Tree{}, c0, registered, nil)
	if err != nil {
		t.Fatalf("RunProvenanceCheck: %v", err)
	}

	// Filter to JUST isolation-escape findings — provenance/other
	// rules may also fire on the fixture (and they do — the AI
	// commit has no aiwf-on-behalf-of: trailer so
	// provenance-no-active-scope fires as error severity).
	// The N-2 assertion is specifically about the isolation-escape
	// rule's severity not pushing HasErrors to true on its own;
	// we isolate the rule's findings and assert HasErrors over
	// THAT subset.
	var iso []check.Finding
	for _, f := range findings {
		if f.Code == check.CodeIsolationEscape.ID {
			iso = append(iso, f)
		}
	}
	if len(iso) == 0 {
		t.Fatalf("isolation-escape finding not present; cannot pin warning-vs-error effect")
	}
	if check.HasErrors(iso) {
		t.Errorf("isolation-escape findings include an error-severity finding (would push exit code to ExitFindings); got: %+v", iso)
	}
}
