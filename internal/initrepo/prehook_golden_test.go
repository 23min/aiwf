package initrepo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// M-0069 AC-4 — hook byte-goldens plus a template-equals-installed
// cross-check.
//
// Every hook here is installed into a consumer's repository, so each
// byte is a shipped byte and drift in any of them is invisible to a
// substring assertion. The pre-push hook additionally carries the
// chokepoint that makes `aiwf check` mandatory before push (CLAUDE.md
// design decision §3), so drift between what `preHookScript` returns
// and what `ensurePreHook` writes weakens a stated guarantee rather
// than merely changing text.
//
// Substring assertions are not structural assertions (CLAUDE.md
// `Substring assertions are not structural assertions`): a `contains`
// check passes through a dropped chain prelude, brownfield guard, or
// exec line. Byte-for-byte is the right granularity for an artifact
// whose every line carries semantic weight.
//
// This file holds two tests:
//
//  1. TestHookScripts_ByteGolden — renders each hook template and
//     diffs it against `testdata/<hook>.golden`. Any change to a
//     template body, marker, chain prelude, or guard requires an
//     intentional golden update; drift surfaces as a failing diff.
//
//  2. TestPreHookScript_TemplateEqualsInstalled — runs `Init` in a
//     fresh tempdir, reads the installed `.git/hooks/pre-push` bytes,
//     re-renders `preHookScript()`, and asserts byte-equality. This
//     catches a regression where the install path took a different
//     code branch than the template function (a parallel source of
//     truth — the failure mode CLAUDE.md `Test the seam, not just the
//     layer` warns about).

// TestHookScripts_ByteGolden pins every rendered hook template against
// its golden file. A failure means that template's body changed; either
// the change is intentional (regenerate the golden by inspecting the new
// template and updating testdata/<hook>.golden) or accidental (revert
// the change).
//
// All four hooks are covered because all four are installed into a
// consumer's repository, where an unreviewed byte is a shipped byte.
func TestHookScripts_ByteGolden(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		hook   string
		render func() string
	}{
		{"pre-push", preHookScript},
		{"pre-commit", preCommitHookScript},
		{"post-commit", postCommitHookScript},
		{"commit-msg", commitMsgHookScript},
	} {
		t.Run(tc.hook, func(t *testing.T) {
			t.Parallel()
			golden := filepath.Join("testdata", tc.hook+".golden")
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("read golden: %v", err)
			}
			if diff := cmp.Diff(string(want), tc.render()); diff != "" {
				t.Errorf("rendered %s hook differs from %s (-want +got):\n%s", tc.hook, golden, diff)
			}
		})
	}
}

// TestPreHookScript_TemplateEqualsInstalled runs aiwf init in a fresh
// tempdir, reads the installed pre-push hook, and asserts byte-equality
// against `preHookScript()`. Cross-checks that ensurePreHook writes
// whatever the template function returns and nothing else — no
// parallel source of truth.
func TestPreHookScript_TemplateEqualsInstalled(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	if err := exec.Command("git", "init", "-q", tmp).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	for _, kv := range [][]string{
		{"user.email", "test@example.com"},
		{"user.name", "aiwf-test"},
	} {
		c := exec.Command("git", "config", kv[0], kv[1])
		c.Dir = tmp
		if err := c.Run(); err != nil {
			t.Fatalf("git config %v: %v", kv, err)
		}
	}

	res, err := Init(context.Background(), tmp, Options{
		ActorOverride: "human/test",
		// Hook MUST be installed for this test (the whole point is
		// to cross-check the installed bytes against the template).
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if res.HookConflict {
		t.Fatalf("unexpected hook conflict in fresh tempdir: %+v", res)
	}

	hookPath := filepath.Join(tmp, ".git", "hooks", "pre-push")
	installed, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read installed hook %s: %v", hookPath, err)
	}

	rendered := preHookScript()

	if diff := cmp.Diff(rendered, string(installed)); diff != "" {
		t.Errorf("installed pre-push hook differs from preHookScript() — parallel source of truth (-template +installed):\n%s", diff)
	}

	// The installed hook must also be executable (mode 0o755).
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatalf("stat installed hook: %v", err)
	}
	if mode := info.Mode().Perm(); mode&0o111 == 0 {
		t.Errorf("installed hook mode = %v, want executable (0o755)", mode)
	}
}
