package integration

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/scope"
)

// TestRunAuthorize_OpenPauseResumeRoundTrip drives `aiwf authorize`
// end-to-end through the built binary: open a scope, then read it back
// via cliutil.LoadEntityScopes; pause it; load again and assert paused; resume
// it; load again and assert active. This is the integration-level
// proof that the cmd dispatcher, the verb function, and the scope
// loader all line up on a real consumer repo.
func TestRunAuthorize_OpenPauseResumeRoundTrip(t *testing.T) {
	t.Parallel()
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q"); err != nil {
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
	if out, err := testutil.RunBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "add", "epic", "--title", "Engine"); err != nil {
		t.Fatalf("aiwf add: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "promote", "E-0001", "active"); err != nil {
		t.Fatalf("aiwf promote E-01 active: %v\n%s", err, out)
	}

	// M-0103: open requires a ritual branch context. Move HEAD to a
	// ritual-shape branch so the preflight's implicit signal passes.
	// `aiwf init` makes an initial commit, so the branch is non-empty
	// and `git checkout -b` can fork off it.
	if out, err := testutil.RunGit(root, "checkout", "-b", "epic/E-0001-engine"); err != nil {
		t.Fatalf("git checkout -b epic/E-0001-engine: %v\n%s", err, out)
	}

	// Open a scope.
	if out, err := testutil.RunBin(t, root, binDir, nil,
		"authorize", "E-0001", "--to", "ai/claude", "--reason", "implement E-01"); err != nil {
		t.Fatalf("aiwf authorize --to: %v\n%s", err, out)
	}
	scopes := mustLoadScopes(t, root, "E-0001")
	if len(scopes) != 1 {
		t.Fatalf("after open: scopes len=%d, want 1", len(scopes))
	}
	if scopes[0].State != scope.StateActive || scopes[0].Agent != "ai/claude" || scopes[0].Principal != "human/peter" {
		t.Errorf("after open: scope = %+v", scopes[0])
	}

	// Pause it.
	if out, err := testutil.RunBin(t, root, binDir, nil,
		"authorize", "E-0001", "--pause", "blocked by E-09"); err != nil {
		t.Fatalf("aiwf authorize --pause: %v\n%s", err, out)
	}
	scopes = mustLoadScopes(t, root, "E-0001")
	if scopes[0].State != scope.StatePaused {
		t.Errorf("after pause: state = %s, want paused", scopes[0].State)
	}

	// Resume it.
	if out, err := testutil.RunBin(t, root, binDir, nil,
		"authorize", "E-0001", "--resume", "back to it"); err != nil {
		t.Fatalf("aiwf authorize --resume: %v\n%s", err, out)
	}
	scopes = mustLoadScopes(t, root, "E-0001")
	if scopes[0].State != scope.StateActive {
		t.Errorf("after resume: state = %s, want active", scopes[0].State)
	}

	// HEAD trailer set carries the resume.
	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	hasTrailer(t, tr, "aiwf-verb", "authorize")
	hasTrailer(t, tr, "aiwf-entity", "E-0001")
	hasTrailer(t, tr, "aiwf-scope", "resumed")
	hasTrailer(t, tr, "aiwf-reason", "back to it")
}

// TestRunAuthorize_BranchCompletion_ReturnsRitualBranches
// (M-0102/AC-6, cobra-adapter seam): drive `aiwf __complete authorize
// <id> --branch ""` through the built binary in a test git repo
// carrying both ritual and non-ritual local branches. The hidden
// __complete invocation is cobra's standard plumbing that the shell
// scripts use to query a flag's completion func. Asserting that
// only ritual-shaped branches surface end-to-end pins the
// completeBranchFlag adapter's wiring (cwd = ".", directive =
// NoFileComp) in addition to the helper's filter, which the unit
// tests in internal/cli/authorize/ cover.
func TestRunAuthorize_BranchCompletion_ReturnsRitualBranches(t *testing.T) {
	t.Parallel()
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q"); err != nil {
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
	if out, err := testutil.RunGit(root, "commit", "--allow-empty", "-m", "init"); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	for _, b := range []string{
		"epic/E-0010-cobra",
		"milestone/M-0007-cache",
		"patch/g-0099-iso",
		"fix/some-bug",
		"chore/dep-bump",
	} {
		if out, err := testutil.RunGit(root, "branch", b); err != nil {
			t.Fatalf("git branch %s: %v\n%s", b, err, out)
		}
	}

	// __complete returns one candidate per line followed by a `:<N>`
	// directive line and a trailing comment. We don't care about exit
	// code (cobra exits with the directive value); we care about which
	// candidates surface.
	out, _ := testutil.RunBin(t, root, binDir, nil,
		"__complete", "authorize", "E-0001", "--branch", "")
	wantPresent := []string{
		"epic/E-0010-cobra",
		"milestone/M-0007-cache",
		"patch/g-0099-iso",
	}
	wantAbsent := []string{"fix/some-bug", "chore/dep-bump", "main"}
	for _, b := range wantPresent {
		if !strings.Contains(out, b) {
			t.Errorf("__complete output missing ritual branch %q\noutput:\n%s", b, out)
		}
	}
	for _, b := range wantAbsent {
		// Be specific: the candidate must not appear as a standalone
		// completion line. A bare strings.Contains check would false-fire
		// if the branch name is a substring of an unrelated cobra line.
		for _, line := range strings.Split(out, "\n") {
			if strings.TrimSpace(line) == b {
				t.Errorf("__complete output includes non-ritual branch %q\noutput:\n%s", b, out)
			}
		}
	}
	// Pin the directive code (4 = ShellCompDirectiveNoFileComp). A
	// future refactor that drops the directive or swaps it for
	// ShellCompDirectiveDefault would re-enable shell-level file
	// completion as a silent fallback — a UX regression the candidate
	// assertions above wouldn't catch.
	if !strings.Contains(out, "\n:4\n") {
		t.Errorf("__complete output missing directive :4 (NoFileComp); shell would fall back to file completion\noutput:\n%s", out)
	}
}

// TestRunAuthorize_WithBranch_EmitsTrailer (M-0102/AC-3, cli-layer seam):
// drive `aiwf authorize <id> --to <agent> --branch <name>` through the
// built binary and assert the resulting authorize commit carries an
// aiwf-branch: trailer with the passed value. This is the load-bearing
// end-to-end check on the cli's flag → opts.Branch → verb → trailer
// propagation; a typo on the cli's `opts.Branch = branch` line would
// pass the verb-layer test but fail here.
func TestRunAuthorize_WithBranch_EmitsTrailer(t *testing.T) {
	t.Parallel()
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q"); err != nil {
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
	if out, err := testutil.RunBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "add", "epic", "--title", "Engine"); err != nil {
		t.Fatalf("aiwf add: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "promote", "E-0001", "active"); err != nil {
		t.Fatalf("aiwf promote: %v\n%s", err, out)
	}

	// M-0103: --branch refers to a named local branch; the preflight
	// requires it to exist. Cut it (without checking it out — we stay
	// on master to prove the explicit signal is enough on its own).
	if out, err := testutil.RunGit(root, "branch", "epic/E-0001-engine"); err != nil {
		t.Fatalf("git branch epic/E-0001-engine: %v\n%s", err, out)
	}

	if out, err := testutil.RunBin(t, root, binDir, nil,
		"authorize", "E-0001",
		"--to", "ai/claude",
		"--branch", "epic/E-0001-engine",
		"--reason", "implement E-01",
	); err != nil {
		t.Fatalf("aiwf authorize --to --branch: %v\n%s", err, out)
	}

	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	hasTrailer(t, tr, "aiwf-verb", "authorize")
	hasTrailer(t, tr, "aiwf-to", "ai/claude")
	hasTrailer(t, tr, "aiwf-scope", "opened")
	// The load-bearing assertion: the cli's --branch flag landed as an
	// aiwf-branch: trailer on the authorize commit. Pins the cli → verb
	// → commit propagation path end-to-end.
	hasTrailer(t, tr, "aiwf-branch", "epic/E-0001-engine")
}

// TestRunAuthorize_AITarget_OnNonRitualBranch_NoBranch_Refuses
// (M-0103/AC-1, cli-layer seam): drive `aiwf authorize <id> --to
// ai/<agent>` through the built binary on a fresh repo whose initial
// branch is master/main (non-ritual). Asserts the CLI's gather of
// CurrentBranch via `git symbolic-ref` flows through to the verb's
// preflight, which refuses with branch-context-required. Pins the
// end-to-end seam from --to + current-branch-state → opts.CurrentBranch
// → preflight refusal → non-zero exit.
func TestRunAuthorize_AITarget_OnNonRitualBranch_NoBranch_Refuses(t *testing.T) {
	t.Parallel()
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q"); err != nil {
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
	if out, err := testutil.RunBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "add", "epic", "--title", "Engine"); err != nil {
		t.Fatalf("aiwf add: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "promote", "E-0001", "active"); err != nil {
		t.Fatalf("aiwf promote: %v\n%s", err, out)
	}

	out, err := testutil.RunBin(t, root, binDir, nil,
		"authorize", "E-0001", "--to", "ai/claude", "--reason", "test")
	if err == nil {
		t.Fatalf("expected non-zero exit; output:\n%s", out)
	}
	if !strings.Contains(out, "branch-context-required") {
		t.Errorf("expected branch-context-required code; got:\n%s", out)
	}
	if !strings.Contains(out, "--force --reason") {
		t.Errorf("expected --force --reason override hint; got:\n%s", out)
	}
}

// TestRunAuthorize_AITarget_BranchMissing_Refuses (M-0103/AC-2,
// cli-layer seam): drive `aiwf authorize <id> --to ai/<agent>
// --branch <typo>` through the binary against a repo that has no
// branch by that name. Asserts the CLI's `git show-ref --verify`
// gather flows through to the preflight, which refuses with
// branch-not-found. Pins the --branch + branchExists → opts.BranchExists
// → preflight propagation.
func TestRunAuthorize_AITarget_BranchMissing_Refuses(t *testing.T) {
	t.Parallel()
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q"); err != nil {
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
	if out, err := testutil.RunBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "add", "epic", "--title", "Engine"); err != nil {
		t.Fatalf("aiwf add: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "promote", "E-0001", "active"); err != nil {
		t.Fatalf("aiwf promote: %v\n%s", err, out)
	}

	out, err := testutil.RunBin(t, root, binDir, nil,
		"authorize", "E-0001",
		"--to", "ai/claude",
		"--branch", "epic/E-9999-typo",
		"--reason", "test",
	)
	if err == nil {
		t.Fatalf("expected non-zero exit; output:\n%s", out)
	}
	if !strings.Contains(out, "branch-not-found") {
		t.Errorf("expected branch-not-found code; got:\n%s", out)
	}
	if !strings.Contains(out, "epic/E-9999-typo") {
		t.Errorf("expected the typo'd branch name quoted in error; got:\n%s", out)
	}
}

// TestRunAuthorize_AITarget_ImplicitRitualBranch_AcceptsAndRecords
// (M-0103/AC-3, cli-layer seam): from a checkout on a ritual-shape
// branch, `aiwf authorize <id> --to ai/<agent>` (no --branch) accepts
// AND emits aiwf-branch: trailer with the current branch name. Pins
// the implicit-current → opts.CurrentBranch → verb-promotes-to-explicit
// → trailer end-to-end.
func TestRunAuthorize_AITarget_ImplicitRitualBranch_AcceptsAndRecords(t *testing.T) {
	t.Parallel()
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q"); err != nil {
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
	if out, err := testutil.RunBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "add", "epic", "--title", "Engine"); err != nil {
		t.Fatalf("aiwf add: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "promote", "E-0001", "active"); err != nil {
		t.Fatalf("aiwf promote: %v\n%s", err, out)
	}
	if out, err := testutil.RunGit(root, "checkout", "-b", "epic/E-0001-engine"); err != nil {
		t.Fatalf("git checkout -b epic/E-0001-engine: %v\n%s", err, out)
	}

	if out, err := testutil.RunBin(t, root, binDir, nil,
		"authorize", "E-0001", "--to", "ai/claude", "--reason", "implicit ritual"); err != nil {
		t.Fatalf("aiwf authorize: %v\n%s", err, out)
	}

	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	hasTrailer(t, tr, "aiwf-verb", "authorize")
	hasTrailer(t, tr, "aiwf-to", "ai/claude")
	// Implicit-from-current promoted to explicit aiwf-branch trailer.
	hasTrailer(t, tr, "aiwf-branch", "epic/E-0001-engine")
}

// TestRunAuthorize_RefusesNonHumanActor: --actor ai/claude is rejected
// before any state is touched — only humans authorize.
func TestRunAuthorize_RefusesNonHumanActor(t *testing.T) {
	t.Parallel()
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q"); err != nil {
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
	if out, err := testutil.RunBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "add", "epic", "--title", "Engine"); err != nil {
		t.Fatalf("aiwf add: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "promote", "E-0001", "active"); err != nil {
		t.Fatalf("aiwf promote: %v\n%s", err, out)
	}

	out, err := testutil.RunBin(t, root, binDir, nil,
		"authorize", "E-0001", "--actor", "ai/claude", "--to", "ai/cursor")
	if err == nil {
		t.Fatalf("expected non-zero exit for non-human actor; output:\n%s", out)
	}
	if !strings.Contains(out, "human/") {
		t.Errorf("expected human/ requirement in error; got:\n%s", out)
	}
}

// TestRunAuthorize_PauseRefusedWhenNoActiveScope: --pause with no
// open scope on the entity exits non-zero with a clear message.
func TestRunAuthorize_PauseRefusedWhenNoActiveScope(t *testing.T) {
	t.Parallel()
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q"); err != nil {
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
	if out, err := testutil.RunBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "add", "epic", "--title", "Engine"); err != nil {
		t.Fatalf("aiwf add: %v\n%s", err, out)
	}

	out, err := testutil.RunBin(t, root, binDir, nil,
		"authorize", "E-0001", "--pause", "trying without a scope")
	if err == nil {
		t.Fatalf("expected non-zero exit; output:\n%s", out)
	}
	if !strings.Contains(out, "no active scope") {
		t.Errorf("expected no-active-scope error; got:\n%s", out)
	}
}

// TestRunAuthorize_RejectsMixedModes: passing both --pause and
// --resume (or --to + --pause) is a usage error.
func TestRunAuthorize_RejectsMixedModes(t *testing.T) {
	t.Parallel()
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q"); err != nil {
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
	if out, err := testutil.RunBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	out, err := testutil.RunBin(t, root, binDir, nil,
		"authorize", "E-0001", "--pause", "x", "--resume", "y")
	if err == nil {
		t.Fatalf("expected mixed-mode usage error; got:\n%s", out)
	}
	if !strings.Contains(out, "exactly one") {
		t.Errorf("expected usage error mentioning exactly-one; got:\n%s", out)
	}
}

func mustLoadScopes(t *testing.T, root, id string) []*scope.Scope {
	t.Helper()
	scopes, err := cliutil.LoadEntityScopes(context.Background(), root, id)
	if err != nil {
		t.Fatalf("cliutil.LoadEntityScopes: %v", err)
	}
	return scopes
}

func hasTrailer(t *testing.T, trailers []gitops.Trailer, key, value string) {
	t.Helper()
	for _, tr := range trailers {
		if tr.Key == key && tr.Value == value {
			return
		}
	}
	t.Errorf("trailer %s=%q not found in %+v", key, value, trailers)
}
