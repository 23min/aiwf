package initrepo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// G-0112: STATUS.md regeneration moved from the pre-commit hook to a
// new post-commit hook. The pre-commit hook is now solely the
// tree-discipline gate. The post-commit hook regenerates STATUS.md
// after the commit lands, and STATUS.md is gitignored — so it never
// enters a commit and therefore never produces a merge conflict on a
// fully-derived artifact.

// TestPostCommitHookScript_HasMarker pins the load-bearing marker
// line. Without it, re-running `aiwf init`/`aiwf update` cannot
// distinguish its own hook from a user-written one and refuses to
// overwrite, breaking idempotent updates.
func TestPostCommitHookScript_HasMarker(t *testing.T) {
	t.Parallel()
	body := postCommitHookScript("/some/aiwf")
	if !strings.Contains(body, PostCommitHookMarker()) {
		t.Errorf("post-commit hook missing marker line %q:\n%s", PostCommitHookMarker(), body)
	}
}

// TestPostCommitHookScript_HasBrownfieldGuard mirrors the pre-push and
// pre-commit guards: a clone with no aiwf.yaml at the root is a
// brownfield repo and the hook is a no-op for it.
func TestPostCommitHookScript_HasBrownfieldGuard(t *testing.T) {
	t.Parallel()
	body := postCommitHookScript("/some/aiwf")
	if !strings.Contains(body, `[ -f "$repo_root/aiwf.yaml" ] || exit 0`) {
		t.Errorf("post-commit hook missing brownfield guard:\n%s", body)
	}
}

// TestPostCommitHookScript_RegeneratesStatusMd pins the regen
// invocation. This is the hook's whole point.
func TestPostCommitHookScript_RegeneratesStatusMd(t *testing.T) {
	t.Parallel()
	body := postCommitHookScript("/some/aiwf")
	if !strings.Contains(body, "status --root") {
		t.Errorf("post-commit hook missing `aiwf status --root` invocation:\n%s", body)
	}
	if !strings.Contains(body, "--format=md") {
		t.Errorf("post-commit hook missing --format=md flag:\n%s", body)
	}
	if !strings.Contains(body, "STATUS.md") {
		t.Errorf("post-commit hook does not target STATUS.md:\n%s", body)
	}
}

// TestPostCommitHookScript_NoGitAdd asserts the hook does NOT try to
// `git add STATUS.md`. Two reasons:
//  1. STATUS.md is gitignored under the G-0112 design (the whole
//     point — keep the file out of git).
//  2. post-commit fires *after* the commit; `git add` here cannot
//     amend the just-finished commit anyway.
//
// A `git add` invocation here is a regression — likely a refactor
// dragged the line over from the old pre-commit body.
func TestPostCommitHookScript_NoGitAdd(t *testing.T) {
	t.Parallel()
	body := postCommitHookScript("/some/aiwf")
	if strings.Contains(body, "git add") {
		t.Errorf("post-commit hook must not `git add` (STATUS.md is gitignored; post-commit can't amend the commit):\n%s", body)
	}
}

// TestPreCommitHookScript_HasNoRegen pins the G-0112 contract for the
// pre-commit hook: it carries only the tree-discipline gate now,
// never the STATUS.md regen logic. A regression here would
// re-introduce the merge-conflict-on-derived-artifact failure mode.
//
// We assert on the executable shape (no `status --root` invocation,
// no `mv ... STATUS.md` write) rather than blanket-banning the
// substring "STATUS.md": a load-bearing comment line that names the
// post-commit hook as the new home of the regen is informative
// documentation, not a regression.
func TestPreCommitHookScript_HasNoRegen(t *testing.T) {
	t.Parallel()
	body := preCommitHookScript("/some/aiwf")
	if !strings.Contains(body, "check --shape-only") {
		t.Errorf("pre-commit hook missing tree-discipline gate:\n%s", body)
	}
	if strings.Contains(body, "status --root") {
		t.Errorf("pre-commit hook still includes STATUS.md regen invocation (G-0112: regen lives in post-commit now):\n%s", body)
	}
	if strings.Contains(body, "mv ") && strings.Contains(body, "STATUS.md") {
		t.Errorf("pre-commit hook still writes STATUS.md via mv (G-0112: regen lives in post-commit):\n%s", body)
	}
}

// TestEnsurePostCommitHook_InstallFresh: a fresh install lands the
// post-commit hook with the marker.
func TestEnsurePostCommitHook_InstallFresh(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	step, conflict, err := ensurePostCommitHook(context.Background(), root, true, false)
	if err != nil {
		t.Fatalf("ensurePostCommitHook: %v", err)
	}
	if conflict {
		t.Errorf("conflict = true, want false (no prior hook)")
	}
	if step.Action != ActionCreated {
		t.Errorf("Action = %q, want %q", step.Action, ActionCreated)
	}
	body, err := os.ReadFile(filepath.Join(root, ".git", "hooks", "post-commit"))
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	if !strings.Contains(string(body), PostCommitHookMarker()) {
		t.Errorf("hook body missing marker:\n%s", body)
	}
	if !strings.Contains(string(body), "status --root") {
		t.Errorf("hook body missing status invocation:\n%s", body)
	}
}

// TestEnsurePostCommitHook_AutoUpdateOff_FreshInstall: when the
// consumer opts out via status_md.auto_update: false, a fresh install
// must NOT install the post-commit hook. The hook is pure
// convenience; with regen off there is nothing for it to do.
//
// (Contrast with the pre-commit hook, which always installs because
// the tree-discipline gate is enforcement and not opt-out-able.)
func TestEnsurePostCommitHook_AutoUpdateOff_FreshInstall(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	step, conflict, err := ensurePostCommitHook(context.Background(), root, false, false)
	if err != nil {
		t.Fatalf("ensurePostCommitHook: %v", err)
	}
	if conflict {
		t.Errorf("conflict = true, want false")
	}
	if step.Action != ActionSkipped {
		t.Errorf("Action = %q, want %q (regen off → nothing to install)", step.Action, ActionSkipped)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "post-commit")); !os.IsNotExist(err) {
		t.Errorf("post-commit hook installed despite regen off (stat err=%v)", err)
	}
}

// TestEnsurePostCommitHook_AutoUpdateOff_UninstallsOwn: when the hook
// was previously installed and the consumer flips
// status_md.auto_update to false, the next refresh removes our
// marker-managed hook. Mirrors the old preCommit opt-out shape.
func TestEnsurePostCommitHook_AutoUpdateOff_UninstallsOwn(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(hooksDir, "post-commit")
	prior := []byte("#!/bin/sh\n" + PostCommitHookMarker() + "\n# our regen body\nexit 0\n")
	if err := os.WriteFile(hookPath, prior, 0o755); err != nil {
		t.Fatal(err)
	}

	step, conflict, err := ensurePostCommitHook(context.Background(), root, false, false)
	if err != nil {
		t.Fatalf("ensurePostCommitHook: %v", err)
	}
	if conflict {
		t.Errorf("conflict = true, want false (own hook removal)")
	}
	if step.Action != ActionRemoved {
		t.Errorf("Action = %q, want %q", step.Action, ActionRemoved)
	}
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Errorf("post-commit hook still present after opt-out (stat err=%v)", err)
	}
}

// TestEnsurePostCommitHook_AutoUpdateOff_LeavesAlien: when the
// consumer opts out and an *alien* (non-marker) post-commit hook is
// in place, we leave it alone — never delete user-written hooks.
func TestEnsurePostCommitHook_AutoUpdateOff_LeavesAlien(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(hooksDir, "post-commit")
	alien := []byte("#!/bin/sh\n# user's own post-commit hook\nexit 0\n")
	if err := os.WriteFile(hookPath, alien, 0o755); err != nil {
		t.Fatal(err)
	}

	step, conflict, err := ensurePostCommitHook(context.Background(), root, false, false)
	if err != nil {
		t.Fatalf("ensurePostCommitHook: %v", err)
	}
	if conflict {
		t.Errorf("conflict = true, want false")
	}
	if step.Action != ActionSkipped {
		t.Errorf("Action = %q, want %q (alien hook left alone)", step.Action, ActionSkipped)
	}
	got, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytesEqual(got, alien) {
		t.Errorf("alien post-commit hook mutated:\nwant %q\ngot  %q", alien, got)
	}
}

// TestEnsurePostCommitHook_RefreshOurOwn: when our own marker-managed
// hook is present and regen is on, a refresh rewrites the body from
// the template (idempotent against tampering).
func TestEnsurePostCommitHook_RefreshOurOwn(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := []byte("#!/bin/sh\n" + PostCommitHookMarker() + "\n# stale body\nexit 1\n")
	hookPath := filepath.Join(hooksDir, "post-commit")
	if err := os.WriteFile(hookPath, stale, 0o755); err != nil {
		t.Fatal(err)
	}

	step, conflict, err := ensurePostCommitHook(context.Background(), root, true, false)
	if err != nil {
		t.Fatalf("ensurePostCommitHook: %v", err)
	}
	if conflict {
		t.Errorf("conflict = true, want false (own hook)")
	}
	if step.Action != ActionUpdated {
		t.Errorf("Action = %q, want %q", step.Action, ActionUpdated)
	}
	got, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "stale body") {
		t.Errorf("stale content survived refresh:\n%s", got)
	}
	if !strings.Contains(string(got), "status --root") {
		t.Errorf("refreshed hook missing regen invocation:\n%s", got)
	}
}

// TestEnsurePostCommitHook_MigratesAlien (G45-shape): a non-marker
// post-commit hook in place — auto-migrate to post-commit.local, then
// install aiwf's chain-aware hook.
func TestEnsurePostCommitHook_MigratesAlien(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	alien := []byte("#!/bin/sh\n# user's own hook\nexit 0\n")
	hookPath := filepath.Join(hooksDir, "post-commit")
	if err := os.WriteFile(hookPath, alien, 0o755); err != nil {
		t.Fatal(err)
	}

	step, conflict, err := ensurePostCommitHook(context.Background(), root, true, false)
	if err != nil {
		t.Fatalf("ensurePostCommitHook: %v", err)
	}
	if conflict {
		t.Error("conflict = true, want false (G45 auto-migrates)")
	}
	if step.Action != ActionMigrated {
		t.Errorf("Action = %q, want %q", step.Action, ActionMigrated)
	}
	migrated, err := os.ReadFile(filepath.Join(hooksDir, "post-commit.local"))
	if err != nil {
		t.Fatalf("reading post-commit.local: %v", err)
	}
	if !bytesEqual(migrated, alien) {
		t.Errorf("migrated content drifted:\nwant %q\ngot  %q", alien, migrated)
	}
	installed, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(installed), PostCommitHookMarker()) {
		t.Errorf("post-migration post-commit lacks aiwf marker")
	}
	if !strings.Contains(string(installed), "post-commit.local") {
		t.Errorf("post-migration post-commit lacks chain reference to .local sibling")
	}
}

// TestEnsurePostCommitHook_RefusesMigrationOnLocalCollision (G45):
// when a non-marker hook AND an existing post-commit.local both
// exist, we refuse to migrate.
func TestEnsurePostCommitHook_RefusesMigrationOnLocalCollision(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	alien := []byte("#!/bin/sh\n# alien\nexit 0\n")
	prior := []byte("#!/bin/sh\n# prior local\nexit 0\n")
	hookPath := filepath.Join(hooksDir, "post-commit")
	localPath := filepath.Join(hooksDir, "post-commit.local")
	if err := os.WriteFile(hookPath, alien, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(localPath, prior, 0o755); err != nil {
		t.Fatal(err)
	}

	step, conflict, err := ensurePostCommitHook(context.Background(), root, true, false)
	if err != nil {
		t.Fatalf("ensurePostCommitHook: %v", err)
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
		t.Errorf("post-commit.local clobbered:\n got  %q\n want %q", got, prior)
	}
}

// TestEnsurePostCommitHook_DryRunInstall: dry-run reports
// ActionCreated but writes nothing.
func TestEnsurePostCommitHook_DryRunInstall(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	step, conflict, err := ensurePostCommitHook(context.Background(), root, true, true)
	if err != nil {
		t.Fatalf("ensurePostCommitHook: %v", err)
	}
	if conflict {
		t.Errorf("conflict = true, want false")
	}
	if step.Action != ActionCreated {
		t.Errorf("Action = %q, want %q", step.Action, ActionCreated)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "post-commit")); !os.IsNotExist(err) {
		t.Errorf("dry-run wrote the hook (stat err=%v)", err)
	}
}

// TestInit_InstallsPostCommitByDefault: a fresh `aiwf init` lands the
// post-commit hook (default-on status_md.auto_update).
func TestInit_InstallsPostCommitByDefault(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	res, err := Init(context.Background(), root, Options{})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	step := findStep(t, res.Steps, ".git/hooks/post-commit")
	if step.Action != ActionCreated {
		t.Errorf("post-commit step.Action = %q, want %q", step.Action, ActionCreated)
	}
	body, err := os.ReadFile(filepath.Join(root, ".git", "hooks", "post-commit"))
	if err != nil {
		t.Fatalf("read post-commit hook: %v", err)
	}
	if !strings.Contains(string(body), PostCommitHookMarker()) {
		t.Errorf("post-commit hook missing marker:\n%s", body)
	}
}

// TestInit_StatusMdAutoUpdateFalse_SkipsPostCommit: opt-out means no
// post-commit hook is installed. Contrast with the pre-commit hook,
// which still installs because it carries the enforcement gate.
func TestInit_StatusMdAutoUpdateFalse_SkipsPostCommit(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	yaml := []byte(`status_md:
  auto_update: false
`)
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Init(context.Background(), root, Options{})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	step := findStep(t, res.Steps, ".git/hooks/post-commit")
	if step.Action != ActionSkipped {
		t.Errorf("post-commit step.Action = %q, want %q (opt-out should skip post-commit)", step.Action, ActionSkipped)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "post-commit")); !os.IsNotExist(err) {
		t.Errorf("post-commit hook installed despite opt-out (stat err=%v)", err)
	}
}

// TestRefreshArtifacts_FlipOnInstallsPostCommit: install with opt-out,
// then flip to default and re-refresh — the post-commit hook should
// appear. Mirrors the pre-commit flip-flag test.
func TestRefreshArtifacts_FlipOnInstallsPostCommit(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	yaml := []byte(`status_md:
  auto_update: false
`)
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(context.Background(), root, Options{}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	hookPath := filepath.Join(root, ".git", "hooks", "post-commit")
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Fatalf("post-commit installed under opt-out (stat err=%v)", err)
	}

	// Now flip to default.
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	steps, conflict, err := RefreshArtifacts(context.Background(), root, RefreshOptions{
		StatusMdAutoUpdate: true,
	})
	if err != nil {
		t.Fatalf("RefreshArtifacts: %v", err)
	}
	if conflict {
		t.Errorf("conflict = true, want false")
	}
	step := findStep(t, steps, ".git/hooks/post-commit")
	if step.Action != ActionCreated {
		t.Errorf("post-commit step.Action = %q, want %q (fresh install on flip)", step.Action, ActionCreated)
	}
	if _, err := os.Stat(hookPath); err != nil {
		t.Errorf("post-commit hook missing after flip-on refresh: %v", err)
	}
}

// TestPostCommitHook_RegeneratesAndLeavesUntracked (G-0112): the
// installed post-commit hook, when actually executed, writes STATUS.md
// in the working tree but does NOT call `git add` and does NOT mutate
// the just-finished commit. The end-to-end behavior the user sees:
// `git status` after a commit shows STATUS.md as untracked-but-gitignored
// (or, equivalently, the file is regenerated and git silently ignores it
// because of .gitignore).
func TestPostCommitHook_RegeneratesStatusMd(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("aiwf hooks are unix-only")
	}
	root := freshGitRepo(t)

	// Stub binary mimics `aiwf status --format=md`.
	stubBin := filepath.Join(t.TempDir(), "aiwf-stub.sh")
	stubScript := "#!/bin/sh\n" +
		"case \"$1\" in\n" +
		"  status) printf '# regen content\\n'; exit 0 ;;\n" +
		"  *)      exit 0 ;;\n" +
		"esac\n"
	if err := os.WriteFile(stubBin, []byte(stubScript), 0o755); err != nil {
		t.Fatal(err)
	}

	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	hookBody := postCommitHookScript(stubBin)
	hookPath := filepath.Join(hooksDir, "post-commit")
	if err := os.WriteFile(hookPath, []byte(hookBody), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("# test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("sh", hookPath)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("post-commit hook exited non-zero: %v\n%s", err, out)
	}
	body, err := os.ReadFile(filepath.Join(root, "STATUS.md"))
	if err != nil {
		t.Fatalf("post-commit hook did not write STATUS.md: %v", err)
	}
	if !strings.Contains(string(body), "regen content") {
		t.Errorf("STATUS.md content unexpected: %q", body)
	}
}

// TestPostCommitHook_BrownfieldShortCircuits (G-0112 + brownfield): a
// post-commit hook in a clone without aiwf.yaml must exit 0 silently
// and never invoke the binary. Mirrors the pre-push / pre-commit
// brownfield guard.
func TestPostCommitHook_BrownfieldShortCircuits(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("aiwf hooks are unix-only")
	}
	root := freshGitRepo(t)
	hookPath := filepath.Join(t.TempDir(), "post-commit.sh")
	if err := os.WriteFile(hookPath, []byte(postCommitHookScript("/bin/false")), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("/bin/sh", hookPath)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("post-commit hook exited non-zero on brownfield repo:\n%s\nerr: %v", out, err)
	}
	if _, err := os.Stat(filepath.Join(root, "STATUS.md")); !os.IsNotExist(err) {
		t.Errorf("post-commit hook wrote STATUS.md in brownfield repo (stat err=%v)", err)
	}
}

// TestEnsureGitignore_AddsStatusMdWhenAutoUpdate (G-0112): the
// gitignore reconciler adds STATUS.md when status_md.auto_update is
// true (the default). Locally regenerated derived artifact → it must
// not be tracked.
func TestEnsureGitignore_AddsStatusMdWhenAutoUpdate(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	if _, err := Init(context.Background(), root, Options{}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	gi, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.Contains(string(gi), "\nSTATUS.md\n") && !strings.HasSuffix(string(gi), "STATUS.md\n") {
		t.Errorf(".gitignore missing STATUS.md entry under default status_md.auto_update:\n%s", gi)
	}
}

// TestEnsureGitignore_OmitsStatusMdWhenAutoUpdateFalse: when the
// consumer opts out, the gitignore reconciler does NOT add STATUS.md
// (the consumer decides whether to commit the file or not).
func TestEnsureGitignore_OmitsStatusMdWhenAutoUpdateFalse(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	yaml := []byte("status_md:\n  auto_update: false\n")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(context.Background(), root, Options{}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	gi, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	// Match exactly the standalone "STATUS.md" line; an unrelated path
	// like "internal/STATUS.md" would be a false positive.
	for _, line := range strings.Split(string(gi), "\n") {
		if strings.TrimSpace(line) == "STATUS.md" {
			t.Errorf(".gitignore added STATUS.md under opt-out (consumer's choice, not ours):\n%s", gi)
			return
		}
	}
}

// TestEnsureGitignore_StatusMdFlipFalseToTrue: a previous run that
// landed STATUS.md as an opt-out (absent) gets it added when the
// consumer flips to auto_update: true. Mirrors the html-out-dir flip
// pattern already in ensureGitignore.
func TestEnsureGitignore_StatusMdFlipFalseToTrue(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	yaml := []byte("status_md:\n  auto_update: false\n")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(context.Background(), root, Options{}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Flip on by writing an empty aiwf.yaml (default = on).
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := RefreshArtifacts(context.Background(), root, RefreshOptions{
		StatusMdAutoUpdate: true,
	}); err != nil {
		t.Fatalf("RefreshArtifacts: %v", err)
	}
	gi, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.Contains(string(gi), "\nSTATUS.md\n") && !strings.HasSuffix(string(gi), "STATUS.md\n") {
		t.Errorf(".gitignore should contain STATUS.md after flip-on:\n%s", gi)
	}
	// Idempotency: ensure only one entry.
	count := 0
	for _, line := range strings.Split(string(gi), "\n") {
		if strings.TrimSpace(line) == "STATUS.md" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("STATUS.md appears %d times in .gitignore, want 1:\n%s", count, gi)
	}
}

// TestEnsureGitignore_StatusMdFlipTrueToFalse: a previous run that
// landed STATUS.md gets it removed when the consumer flips opt-out.
// Symmetric to the html-out-dir flip-true-to-false test.
func TestEnsureGitignore_StatusMdFlipTrueToFalse(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	if _, err := Init(context.Background(), root, Options{}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	yaml := []byte("status_md:\n  auto_update: false\n")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := RefreshArtifacts(context.Background(), root, RefreshOptions{
		StatusMdAutoUpdate: false,
	}); err != nil {
		t.Fatalf("RefreshArtifacts: %v", err)
	}
	gi, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	for _, line := range strings.Split(string(gi), "\n") {
		if strings.TrimSpace(line) == "STATUS.md" {
			t.Errorf(".gitignore still carries STATUS.md after flip-off:\n%s", gi)
			return
		}
	}
}

// TestEnsureGitignore_StatusMdPreservesUserStatusMdLine: when the user
// already authored a STATUS.md entry (a different shape, e.g.
// "/STATUS.md" or "STATUS.md  # my own comment"), reconciliation
// leaves it alone — we only match the exact line we write.
func TestEnsureGitignore_PreservesUserStatusMdShape(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	yaml := []byte("status_md:\n  auto_update: false\n")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	// User-authored shape that we should never touch.
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("/STATUS.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(context.Background(), root, Options{}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	gi, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gi), "/STATUS.md") {
		t.Errorf("user-authored '/STATUS.md' entry stripped:\n%s", gi)
	}
}
