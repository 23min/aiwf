package worktree_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/doctor"
	"github.com/23min/aiwf/internal/cli/worktree"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/gitops"
)

// minimalRepo creates a git-inited, minimally aiwf-adopted repo: one
// commit carrying aiwf.yaml (extraYAML appended verbatim for tests
// that need a specific config knob). Lighter than a full `aiwf init`
// fixture since these tests exercise worktree.Run's own path
// resolution and error surfacing — RefreshArtifacts' own materialization
// behavior is already covered under internal/initrepo.
func minimalRepo(t *testing.T, extraYAML string) string {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	content := "hosts: [claude-code]\n" + extraYAML
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(ctx, root, "aiwf.yaml"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := gitops.Commit(ctx, root, "seed", "", nil); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	return root
}

func extractLine(haystack, prefix string) string {
	for _, line := range strings.Split(haystack, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			return line
		}
	}
	return ""
}

// TestRun_CreatesWorktreeAndMaterializesRituals is M-0233/AC-1: one
// `aiwf worktree add` call creates the worktree AND leaves `aiwf
// doctor` reporting `rituals: ok` immediately after, with no
// intervening `aiwf update`.
func TestRun_CreatesWorktreeAndMaterializesRituals(t *testing.T) {
	t.Parallel()
	root := minimalRepo(t, "")

	rc := worktree.Run("feature/x", "", "", root, false, cliutil.OutputFormat{})
	if rc != cliutil.ExitOK {
		t.Fatalf("Run rc = %d, want ExitOK", rc)
	}

	wtPath := filepath.Join(root, config.DefaultWorktreeDir, "feature", "x")
	if _, err := os.Stat(wtPath); err != nil {
		t.Fatalf("worktree not created at %s: %v", wtPath, err)
	}

	lines, _ := doctor.DoctorReport(wtPath, doctor.DoctorOptions{})
	ritualsLine := extractLine(strings.Join(lines, "\n"), "rituals:")
	if !strings.Contains(ritualsLine, "ok") {
		t.Errorf("doctor rituals line = %q, want it to report ok\nfull report:\n%s", ritualsLine, strings.Join(lines, "\n"))
	}
}

// TestRun_DefaultPathResolvesViaWorktreeDir is M-0233/AC-2's default
// half: omitting path resolves to <worktree.dir>/<branch> via
// config.WorktreeDir(), honoring a consumer's configured knob.
func TestRun_DefaultPathResolvesViaWorktreeDir(t *testing.T) {
	t.Parallel()
	root := minimalRepo(t, "worktree:\n  dir: .wt\n")

	rc := worktree.Run("feature/y", "", "", root, false, cliutil.OutputFormat{})
	if rc != cliutil.ExitOK {
		t.Fatalf("Run rc = %d, want ExitOK", rc)
	}
	want := filepath.Join(root, ".wt", "feature", "y")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("worktree not created at configured worktree.dir path %s: %v", want, err)
	}
}

// TestRun_ExplicitPathHonoredVerbatim is M-0233/AC-2's explicit half:
// an explicit path argument is used as-is, bypassing worktree.dir
// entirely — even when worktree.dir is configured.
func TestRun_ExplicitPathHonoredVerbatim(t *testing.T) {
	t.Parallel()
	root := minimalRepo(t, "worktree:\n  dir: .wt\n")
	sibling := filepath.Join(t.TempDir(), "sibling-wt")

	rc := worktree.Run("feature/z", sibling, "", root, false, cliutil.OutputFormat{})
	if rc != cliutil.ExitOK {
		t.Fatalf("Run rc = %d, want ExitOK", rc)
	}
	if _, err := os.Stat(sibling); err != nil {
		t.Errorf("worktree not created at explicit sibling path: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".wt")); err == nil {
		t.Error("explicit path must bypass the configured worktree.dir entirely; .wt should not exist")
	}
}

// TestRun_ExplicitPathBypassesRepoEscapeRejection is M-0233/AC-3:
// worktree.dir's repo-escape rejection (M-0190/AC-4) lives inside
// config.WorktreeDir() and only fires when resolving the DEFAULT
// path. An explicit path that itself escapes the repo must be
// honored verbatim, never silently redirected back in-repo.
func TestRun_ExplicitPathBypassesRepoEscapeRejection(t *testing.T) {
	t.Parallel()
	root := minimalRepo(t, "")

	rc := worktree.Run("feature/w", "../outside-wt", "", root, false, cliutil.OutputFormat{})
	if rc != cliutil.ExitOK {
		t.Fatalf("Run rc = %d, want ExitOK", rc)
	}
	want := filepath.Join(filepath.Dir(root), "outside-wt")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("explicit repo-escaping path should be honored verbatim, not rejected; got: %v", err)
	}
}

// TestRun_GitFailureSurfacesDirectly is M-0233/AC-5: a `git worktree
// add` failure (here, a branch already checked out elsewhere) must
// surface the underlying git error, not report success or swallow it
// into a generic message.
func TestRun_GitFailureSurfacesDirectly(t *testing.T) {
	root := minimalRepo(t, "")

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return worktree.Run("main", filepath.Join(t.TempDir(), "wt"), "", root, false, cliutil.OutputFormat{})
	})
	if rc == cliutil.ExitOK {
		t.Fatal("Run should fail when branch is already checked out elsewhere")
	}
	if !strings.Contains(stderr, "already") {
		t.Errorf("stderr should surface git's own explanation of the failure; got:\n%s", stderr)
	}
}

// TestRun_BaseRejectedForExistingBranch: --base only makes sense when
// creating a NEW branch; passing it alongside an already-existing
// branch is a usage error, not a silently-ignored flag.
func TestRun_BaseRejectedForExistingBranch(t *testing.T) {
	root := minimalRepo(t, "")

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return worktree.Run("main", filepath.Join(t.TempDir(), "wt"), "some-base", root, false, cliutil.OutputFormat{})
	})
	if rc != cliutil.ExitUsage {
		t.Fatalf("rc = %d, want ExitUsage", rc)
	}
	if !strings.Contains(stderr, "--base") {
		t.Errorf("stderr should explain the --base conflict; got:\n%s", stderr)
	}
}

// TestNewCmd_DispatchesToRun exercises the actual Cobra wiring (NewCmd,
// newAddCmd, and the RunE closure) end to end — the unit tests above all
// call worktree.Run directly, which never builds or executes the Cobra
// command tree itself.
func TestNewCmd_DispatchesToRun(t *testing.T) {
	t.Parallel()
	root := minimalRepo(t, "")

	cmd := worktree.NewCmd()
	cmd.SetArgs([]string{"add", "feature/cobra-dispatch", "--root", root})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	want := filepath.Join(root, config.DefaultWorktreeDir, "feature", "cobra-dispatch")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("worktree not created via Cobra dispatch: %v", err)
	}
}

// TestNewCmd_DispatchesToRun_WithPath covers the RunE closure's other
// branch — a second positional arg supplies an explicit path.
func TestNewCmd_DispatchesToRun_WithPath(t *testing.T) {
	t.Parallel()
	root := minimalRepo(t, "")
	explicit := filepath.Join(t.TempDir(), "cobra-explicit")

	cmd := worktree.NewCmd()
	cmd.SetArgs([]string{"add", "feature/cobra-path", explicit, "--root", root})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if _, err := os.Stat(explicit); err != nil {
		t.Errorf("worktree not created at explicit path via Cobra dispatch: %v", err)
	}
}

// TestRun_MissingAiwfYamlInNewWorktree covers the case where the
// branch being worktree-added never carried aiwf.yaml (a repo not yet
// aiwf-adopted, or a branch predating adoption) — Run must fail
// clearly rather than panic on a nil-unsafe config getter.
func TestRun_MissingAiwfYamlInNewWorktree(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("no aiwf.yaml here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(ctx, root, "README.md"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := gitops.Commit(ctx, root, "seed", "", nil); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return worktree.Run("feature/no-config", filepath.Join(t.TempDir(), "wt"), "", root, false, cliutil.OutputFormat{})
	})
	if rc != cliutil.ExitInternal {
		t.Fatalf("rc = %d, want ExitInternal", rc)
	}
	if !strings.Contains(stderr, "aiwf.yaml") {
		t.Errorf("stderr should explain the missing aiwf.yaml; got:\n%s", stderr)
	}
}

// TestRun_HookConflictReturnsExitFindings covers the hook-collision
// path RefreshArtifacts can report: a non-aiwf pre-push hook already
// installed (with a .local sibling already taken) in the shared hooks
// dir the new worktree resolves to.
func TestRun_HookConflictReturnsExitFindings(t *testing.T) {
	root := minimalRepo(t, "")
	hookDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hookDir, 0o755); err != nil {
		t.Fatal(err)
	}
	alien := []byte("#!/bin/sh\n# alien\nexit 0\n")
	prior := []byte("#!/bin/sh\n# prior local\nexit 0\n")
	if err := os.WriteFile(filepath.Join(hookDir, "pre-push"), alien, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hookDir, "pre-push.local"), prior, 0o755); err != nil {
		t.Fatal(err)
	}

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return worktree.Run("feature/hook-conflict", filepath.Join(t.TempDir(), "wt"), "", root, false, cliutil.OutputFormat{})
	})
	if rc != cliutil.ExitFindings {
		t.Fatalf("rc = %d, want ExitFindings", rc)
	}
	if !strings.Contains(stderr, "collision") {
		t.Errorf("stderr should explain the hook collision; got:\n%s", stderr)
	}
}

// TestRun_PrintPath_UnitLevel is the Go-level complement to the
// binary-level subprocess test in internal/cli/integration: it drives
// --print-path directly against Run so the branch registers in unit
// coverage too, not only via a subprocess the coverage instrumentation
// can't see.
func TestRun_PrintPath_UnitLevel(t *testing.T) {
	root := minimalRepo(t, "")

	var rc int
	stdout := testutil.CaptureStdout(t, func() {
		rc = worktree.Run("feature/print-unit", "", "", root, true, cliutil.OutputFormat{})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("rc = %d, want ExitOK", rc)
	}
	want := filepath.Join(root, config.DefaultWorktreeDir, "feature", "print-unit") + "\n"
	if string(stdout) != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
}

// TestRun_JSONSuccessEnvelope covers --format=json's success path:
// result.path carries the resulting absolute worktree path.
func TestRun_JSONSuccessEnvelope(t *testing.T) {
	root := minimalRepo(t, "")

	rc, stdoutStr, stderrStr := testutil.CaptureRun(t, func() int {
		return worktree.Run("feature/json", "", "", root, false, cliutil.OutputFormat{Format: "json"})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("rc = %d, want ExitOK", rc)
	}
	if stderrStr != "" {
		t.Errorf("stderr should be empty in JSON mode; got:\n%s", stderrStr)
	}
	stdout := []byte(stdoutStr)
	var env struct {
		Status string `json:"status"`
		Result struct {
			Path string `json:"path"`
		} `json:"result"`
	}
	if err := json.Unmarshal(stdout, &env); err != nil {
		t.Fatalf("stdout is not a JSON envelope: %v\nstdout: %s", err, stdout)
	}
	if env.Status != "ok" {
		t.Errorf("status = %q, want ok", env.Status)
	}
	want := filepath.Join(root, config.DefaultWorktreeDir, "feature", "json")
	if env.Result.Path != want {
		t.Errorf("result.path = %q, want %q", env.Result.Path, want)
	}
}
