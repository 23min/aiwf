package initrepo

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestEnsurePreCommitHook_InstallFresh: a fresh install lands the
// pre-commit hook with the marker and the tree-discipline gate. Per
// G-0112 the script body must not include the STATUS.md regen step
// (that lives in the post-commit hook now).
func TestEnsurePreCommitHook_InstallFresh(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	step, conflict, err := ensurePreCommitHook(context.Background(), root, false)
	if err != nil {
		t.Fatalf("ensurePreCommitHook: %v", err)
	}
	if conflict {
		t.Errorf("conflict = true, want false (no prior hook)")
	}
	if step.Action != ActionCreated {
		t.Errorf("Action = %q, want %q", step.Action, ActionCreated)
	}
	body, err := os.ReadFile(filepath.Join(root, ".git", "hooks", "pre-commit"))
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	if !strings.Contains(string(body), PreCommitHookMarker()) {
		t.Errorf("hook body missing marker:\n%s", body)
	}
	if !strings.Contains(string(body), "check --shape-only") {
		t.Errorf("hook body missing tree-discipline gate:\n%s", body)
	}
	if strings.Contains(string(body), "status --root") {
		t.Errorf("hook body still includes STATUS.md regen (G-0112: regen moved to post-commit):\n%s", body)
	}
}

// TestEnsurePreCommitHook_RefreshOurOwn: install=true when our own
// marker-managed hook is already there → ActionUpdated, body
// rewritten from the embedded template.
func TestEnsurePreCommitHook_RefreshOurOwn(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := []byte("#!/bin/sh\n" + PreCommitHookMarker() + "\n# stale body\nexit 1\n")
	if err := os.WriteFile(filepath.Join(hooksDir, "pre-commit"), stale, 0o755); err != nil {
		t.Fatal(err)
	}

	step, conflict, err := ensurePreCommitHook(context.Background(), root, false)
	if err != nil {
		t.Fatalf("ensurePreCommitHook: %v", err)
	}
	if conflict {
		t.Errorf("conflict = true, want false (own hook)")
	}
	if step.Action != ActionUpdated {
		t.Errorf("Action = %q, want %q", step.Action, ActionUpdated)
	}
	got, err := os.ReadFile(filepath.Join(hooksDir, "pre-commit"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "stale body") {
		t.Errorf("stale content survived refresh:\n%s", got)
	}
}

// TestEnsurePreCommitHook_MigratesAlien (G45): a non-marker hook in
// place → auto-migrates to pre-commit.local, installs aiwf's
// chain-aware hook.
func TestEnsurePreCommitHook_MigratesAlien(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	alien := []byte("#!/bin/sh\n# user's own hook\nexit 0\n")
	hookPath := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(hookPath, alien, 0o755); err != nil {
		t.Fatal(err)
	}

	step, conflict, err := ensurePreCommitHook(context.Background(), root, false)
	if err != nil {
		t.Fatalf("ensurePreCommitHook: %v", err)
	}
	if conflict {
		t.Error("conflict = true, want false (G45 auto-migrates)")
	}
	if step.Action != ActionMigrated {
		t.Errorf("Action = %q, want %q", step.Action, ActionMigrated)
	}
	migrated, err := os.ReadFile(filepath.Join(hooksDir, "pre-commit.local"))
	if err != nil {
		t.Fatalf("reading pre-commit.local: %v", err)
	}
	if !bytesEqual(migrated, alien) {
		t.Errorf("migrated content drifted:\nwant %q\ngot  %q", alien, migrated)
	}
	installed, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(installed), PreCommitHookMarker()) {
		t.Errorf("post-migration pre-commit lacks aiwf marker")
	}
	if !strings.Contains(string(installed), "pre-commit.local") {
		t.Errorf("post-migration pre-commit lacks chain reference to .local sibling")
	}
}

// TestEnsurePreCommitHook_RefusesMigrationOnLocalCollision (G45):
// when a non-marker hook AND an existing pre-commit.local both
// exist, ensurePreCommitHook refuses to migrate (would clobber the
// .local) and returns ActionSkipped + conflict=true.
func TestEnsurePreCommitHook_RefusesMigrationOnLocalCollision(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	alien := []byte("#!/bin/sh\n# alien\nexit 0\n")
	prior := []byte("#!/bin/sh\n# prior local\nexit 0\n")
	hookPath := filepath.Join(hooksDir, "pre-commit")
	localPath := filepath.Join(hooksDir, "pre-commit.local")
	if err := os.WriteFile(hookPath, alien, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(localPath, prior, 0o755); err != nil {
		t.Fatal(err)
	}

	step, conflict, err := ensurePreCommitHook(context.Background(), root, false)
	if err != nil {
		t.Fatalf("ensurePreCommitHook: %v", err)
	}
	if !conflict {
		t.Error("conflict = false, want true on .local collision")
	}
	if step.Action != ActionSkipped {
		t.Errorf("Action = %q, want %q", step.Action, ActionSkipped)
	}
	if got, _ := os.ReadFile(hookPath); !bytesEqual(got, alien) {
		t.Errorf("alien hook clobbered:\n got  %q\n want %q", got, alien)
	}
	if got, _ := os.ReadFile(localPath); !bytesEqual(got, prior) {
		t.Errorf("pre-commit.local clobbered:\n got  %q\n want %q", got, prior)
	}
}

// TestEnsurePreCommitHook_DryRunInstall: dryRun=true must not write
// the hook even when no prior hook exists. The reported StepResult
// still says ActionCreated so a preview ledger reads as "this would
// be created".
func TestEnsurePreCommitHook_DryRunInstall(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	step, conflict, err := ensurePreCommitHook(context.Background(), root, true)
	if err != nil {
		t.Fatalf("ensurePreCommitHook: %v", err)
	}
	if conflict {
		t.Errorf("conflict = true, want false")
	}
	if step.Action != ActionCreated {
		t.Errorf("Action = %q, want %q", step.Action, ActionCreated)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "pre-commit")); !os.IsNotExist(err) {
		t.Errorf("dry-run wrote the hook (stat err=%v)", err)
	}
}

// TestEnsurePreCommitHook_DryRunRefresh: dryRun=true with an existing
// marker-managed hook must not rewrite the file. StepResult reports
// ActionUpdated.
func TestEnsurePreCommitHook_DryRunRefresh(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	prior := []byte("#!/bin/sh\n" + PreCommitHookMarker() + "\n# untouched\nexit 0\n")
	hookPath := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(hookPath, prior, 0o755); err != nil {
		t.Fatal(err)
	}

	step, _, err := ensurePreCommitHook(context.Background(), root, true)
	if err != nil {
		t.Fatalf("ensurePreCommitHook: %v", err)
	}
	if step.Action != ActionUpdated {
		t.Errorf("Action = %q, want %q", step.Action, ActionUpdated)
	}
	got, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytesEqual(got, prior) {
		t.Errorf("dry-run rewrote the hook:\nwant %q\ngot  %q", prior, got)
	}
}

// TestInit_InstallsPreCommitByDefault: a fresh `aiwf init` against a
// new repo lands the pre-commit hook with the marker, and the
// ledger reports it Created. Default-on is the framework's contract
// for STATUS.md auto-update.
func TestInit_InstallsPreCommitByDefault(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	res, err := Init(context.Background(), root, Options{})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	step := findStep(t, res.Steps, ".git/hooks/pre-commit")
	if step.Action != ActionCreated {
		t.Errorf("pre-commit step.Action = %q, want %q", step.Action, ActionCreated)
	}
	body, err := os.ReadFile(filepath.Join(root, ".git", "hooks", "pre-commit"))
	if err != nil {
		t.Fatalf("read pre-commit hook: %v", err)
	}
	if !strings.Contains(string(body), PreCommitHookMarker()) {
		t.Errorf("pre-commit hook missing marker:\n%s", body)
	}
}

// TestInit_StatusMdAutoUpdateFalse_StillInstallsGate (G42): a repo
// whose aiwf.yaml opts out of STATUS.md auto-update on fresh init
// still gets the pre-commit hook installed — the tree-discipline
// gate is enforcement and decoupled from the regen convenience.
// Per G-0112 the pre-commit body no longer carries a regen step at
// all; the post-commit hook is what toggles with opt-out.
func TestInit_StatusMdAutoUpdateFalse_StillInstallsGate(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	yaml := []byte(`aiwf_version: 0.1.0
actor: human/peter
status_md:
  auto_update: false
`)
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Init(context.Background(), root, Options{})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	step := findStep(t, res.Steps, ".git/hooks/pre-commit")
	if step.Action != ActionCreated {
		t.Errorf("pre-commit step.Action = %q, want %q (G42: gate always installs)", step.Action, ActionCreated)
	}
	body, err := os.ReadFile(filepath.Join(root, ".git", "hooks", "pre-commit"))
	if err != nil {
		t.Fatalf("pre-commit hook not installed despite G42 contract: %v", err)
	}
	if !strings.Contains(string(body), "check --shape-only") {
		t.Errorf("hook missing tree-discipline gate:\n%s", body)
	}
	if strings.Contains(string(body), "status --root") {
		t.Errorf("pre-commit hook still includes STATUS.md regen step (G-0112: regen lives in post-commit):\n%s", body)
	}
}

// TestRefreshArtifacts_FlipFlagPreCommitUntouched: per G-0112 the
// pre-commit hook body is constant — it's the gate, nothing else.
// Flipping status_md.auto_update does NOT rewrite the pre-commit
// body. (The post-commit hook is what gets installed/uninstalled;
// that case is exercised by the post-commit tests.)
func TestRefreshArtifacts_FlipFlagPreCommitUntouched(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	if _, err := Init(context.Background(), root, Options{}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	hookPath := filepath.Join(root, ".git", "hooks", "pre-commit")
	before, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("pre-commit hook not installed by default Init: %v", err)
	}

	yaml := []byte("status_md:\n  auto_update: false\n")
	if writeErr := os.WriteFile(filepath.Join(root, "aiwf.yaml"), yaml, 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}

	steps, conflict, err := RefreshArtifacts(context.Background(), root, RefreshOptions{
		StatusMdAutoUpdate: false,
	})
	if err != nil {
		t.Fatalf("RefreshArtifacts: %v", err)
	}
	if conflict {
		t.Errorf("conflict = true on opt-out, want false")
	}
	// The pre-commit body must be byte-equal across the flip — its
	// content is invariant under status_md.auto_update.
	after, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("pre-commit hook missing after refresh (G42 violation): %v", err)
	}
	if !bytesEqual(before, after) {
		t.Errorf("pre-commit body changed across status_md.auto_update flip (G-0112: it should not):\nbefore=%q\nafter=%q", before, after)
	}
	if !strings.Contains(string(after), "check --shape-only") {
		t.Errorf("refreshed hook missing tree-discipline gate:\n%s", after)
	}
	// Step is reported in the ledger.
	_ = findStep(t, steps, ".git/hooks/pre-commit")
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestPreCommitHook_ChainsToLocalAtRuntime (G45): the installed
// pre-commit hook script, when actually executed, invokes
// pre-commit.local first and only proceeds to aiwf's check if the
// .local hook returns 0. A non-zero .local exit aborts before
// aiwf is invoked. Drives the script as `sh <hook> ...` so the test
// is portable; the chain prelude is plain POSIX sh.
func TestPreCommitHook_ChainsToLocalAtRuntime(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("aiwf hooks are unix-only")
	}
	root := freshGitRepo(t)
	hooksDir := filepath.Join(root, ".git", "hooks")

	// Install aiwf's chain-aware hook with a stub for the aiwf binary
	// path (the test cares only about the chain prelude, not the
	// aiwf step that follows). Use a shell-script "binary" that
	// records being called.
	stubBin := filepath.Join(t.TempDir(), "aiwf-stub.sh")
	stubMarker := filepath.Join(t.TempDir(), "aiwf-stub.called")
	stubScript := "#!/bin/sh\ntouch '" + stubMarker + "'\nexit 0\n"
	if err := os.WriteFile(stubBin, []byte(stubScript), 0o755); err != nil {
		t.Fatal(err)
	}

	hookBody := preCommitHookScript(stubBin)
	hookPath := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(hookPath, []byte(hookBody), 0o755); err != nil {
		t.Fatal(err)
	}
	// Need an aiwf.yaml at the repo root or the brownfield guard
	// short-circuits the hook before the chain prelude runs.
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("aiwf_version: test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("no .local sibling: aiwf step runs", func(t *testing.T) {
		_ = os.Remove(stubMarker)
		cmd := exec.Command("sh", hookPath)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("hook exited non-zero: %v\n%s", err, out)
		}
		if _, err := os.Stat(stubMarker); err != nil {
			t.Errorf("aiwf step did not run (stub marker absent): %v", err)
		}
	})

	t.Run(".local exits 0: chain falls through, aiwf step runs", func(t *testing.T) {
		_ = os.Remove(stubMarker)
		localMarker := filepath.Join(t.TempDir(), "local.called")
		localBody := "#!/bin/sh\ntouch '" + localMarker + "'\nexit 0\n"
		localPath := filepath.Join(hooksDir, "pre-commit.local")
		if err := os.WriteFile(localPath, []byte(localBody), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Remove(localPath) })

		cmd := exec.Command("sh", hookPath)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("hook exited non-zero: %v\n%s", err, out)
		}
		if _, err := os.Stat(localMarker); err != nil {
			t.Errorf("local hook did not run: %v", err)
		}
		if _, err := os.Stat(stubMarker); err != nil {
			t.Errorf("aiwf step did not run after local exit 0: %v", err)
		}
	})

	t.Run(".local exits non-zero: hook aborts before aiwf step", func(t *testing.T) {
		_ = os.Remove(stubMarker)
		localBody := "#!/bin/sh\nexit 7\n"
		localPath := filepath.Join(hooksDir, "pre-commit.local")
		if err := os.WriteFile(localPath, []byte(localBody), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Remove(localPath) })

		cmd := exec.Command("sh", hookPath)
		cmd.Dir = root
		err := cmd.Run()
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("expected exec.ExitError, got %v", err)
		}
		if exitErr.ExitCode() != 7 {
			t.Errorf("exit code = %d, want 7 (propagated from .local)", exitErr.ExitCode())
		}
		if _, err := os.Stat(stubMarker); err == nil {
			t.Errorf("aiwf step ran despite .local non-zero exit")
		}
	})

	t.Run(".local present but not executable: chain fails loud", func(t *testing.T) {
		_ = os.Remove(stubMarker)
		localPath := filepath.Join(hooksDir, "pre-commit.local")
		if err := os.WriteFile(localPath, []byte("#!/bin/sh\nexit 0\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Remove(localPath) })

		cmd := exec.Command("sh", hookPath)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("expected non-zero exit; output: %s", out)
		}
		if !strings.Contains(string(out), "not executable") {
			t.Errorf("error message missing 'not executable':\n%s", out)
		}
		if _, statErr := os.Stat(stubMarker); statErr == nil {
			t.Errorf("aiwf step ran despite non-executable .local")
		}
	})
}
