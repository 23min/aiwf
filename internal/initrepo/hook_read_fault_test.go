package initrepo

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnsureHooks_UnreadableExistingHookIsRefused pins the one
// contract every hook installer shares: when a hook file exists but
// cannot be read, the installer refuses and leaves the file alone.
//
// A read fault is not "no hook here". The G45 auto-migration that
// moves a pre-existing user hook to its `.local` sibling can only run
// when the file's content is known, so an installer that treats an
// unreadable file as absent falls through to the atomic write —
// os.Rename replaces an unreadable file whenever the containing
// directory is writable, destroying a hook the operator never handed
// to aiwf and reporting a created action at exit 0 (G-0557).
//
// The table covers all four installers plus post-commit's opt-out
// arm, which reaches the same refusal by its own route. The four stay
// deliberately separate — G-0472 weighs collapsing them and rules it
// out on the shape of the shared unit — so one table holding all four
// to the same contract is what keeps near-identical bodies from
// drifting apart on the semantic none of them states locally.
func TestEnsureHooks_UnreadableExistingHookIsRefused(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}

	cases := []struct {
		name     string
		hookFile string
		wantErr  string
		install  func(ctx context.Context, root string, dryRun bool) (StepResult, bool, error)
	}{
		{"pre-push", "pre-push", "reading pre-push hook", ensurePreHook},
		{"pre-commit", "pre-commit", "reading pre-commit hook", ensurePreCommitHook},
		{"commit-msg", "commit-msg", "reading commit-msg hook", ensureCommitMsgHook},
		{"post-commit/regen-on", "post-commit", "reading post-commit hook", func(ctx context.Context, root string, dryRun bool) (StepResult, bool, error) {
			return ensurePostCommitHook(ctx, root, true, dryRun)
		}},
		{"post-commit/regen-off", "post-commit", "reading post-commit hook", func(ctx context.Context, root string, dryRun bool) (StepResult, bool, error) {
			return ensurePostCommitHook(ctx, root, false, dryRun)
		}},
	}

	for _, tc := range cases {
		for _, dryRun := range []bool{false, true} {
			name := tc.name
			if dryRun {
				name += "/dry-run"
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				root := freshGitRepo(t)
				hooksDir := filepath.Join(root, ".git", "hooks")
				if err := os.MkdirAll(hooksDir, 0o755); err != nil {
					t.Fatal(err)
				}
				hookPath := filepath.Join(hooksDir, tc.hookFile)
				userHook := []byte("#!/bin/sh\n# the operator's own hook\nexec ./scripts/lint.sh\n")
				if err := os.WriteFile(hookPath, userHook, 0o000); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(hookPath, 0o644) })

				_, conflict, err := tc.install(context.Background(), root, dryRun)
				if err == nil {
					t.Fatalf("want an error for an unreadable %s hook, got nil", tc.hookFile)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("error %q should mention %q", err.Error(), tc.wantErr)
				}
				if !errors.Is(err, fs.ErrPermission) {
					t.Errorf("error %v must wrap the read fault's cause so callers can classify it", err)
				}
				if conflict {
					t.Errorf("conflict = true, want false (a read fault is an error, not a .local collision)")
				}

				// The operator's hook must be exactly as they left it —
				// same bytes and same mode. Stat before the chmod that
				// makes the content readable again, since that chmod is
				// itself what would destroy the mode evidence.
				info, statErr := os.Lstat(hookPath)
				if statErr != nil {
					t.Fatalf("the operator's hook is gone: %v", statErr)
				}
				if perm := info.Mode().Perm(); perm != 0o000 {
					t.Errorf("hook mode = %04o, want 0000; a refused install must not widen permissions", perm)
				}
				if chmodErr := os.Chmod(hookPath, 0o644); chmodErr != nil {
					t.Fatal(chmodErr)
				}
				got, readErr := os.ReadFile(hookPath)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if !bytes.Equal(got, userHook) {
					t.Errorf("the operator's hook was rewritten:\ngot:  %q\nwant: %q", got, userHook)
				}
				if _, statErr := os.Stat(hookPath + ".local"); statErr == nil {
					t.Errorf("%s.local was written; a hook that could not be read must not be migrated", tc.hookFile)
				}
			})
		}
	}
}

// TestInit_UnreadableExistingHookAbortsTheInstall covers the seam the
// installers sit behind: their refusal propagates out of Init, so
// `aiwf init` reports the failure to the operator instead of a ledger
// of successful creates over a hook it replaced.
//
// pre-commit is the subject because it is the realistic shape — a repo
// carrying a lint hook and no pre-push hook, which is what lets the
// install reach a permissive installer at all.
func TestInit_UnreadableExistingHookAbortsTheInstall(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	root := freshGitRepo(t)
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(hooksDir, "pre-commit")
	userHook := []byte("#!/bin/sh\n# the operator's own hook\nexec ./scripts/lint.sh\n")
	if err := os.WriteFile(hookPath, userHook, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(hookPath, 0o644) })

	_, err := Init(context.Background(), root, Options{})
	if err == nil {
		t.Fatal("want Init to fail on an unreadable existing hook, got nil")
	}
	if !strings.Contains(err.Error(), "reading pre-commit hook") {
		t.Errorf("error %q should name the hook that could not be read", err.Error())
	}

	if _, statErr := os.Lstat(hookPath); statErr != nil {
		t.Fatalf("the operator's hook is gone: %v", statErr)
	}
	if chmodErr := os.Chmod(hookPath, 0o644); chmodErr != nil {
		t.Fatal(chmodErr)
	}
	got, readErr := os.ReadFile(hookPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, userHook) {
		t.Errorf("init rewrote the operator's hook:\ngot:  %q\nwant: %q", got, userHook)
	}
}
