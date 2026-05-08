package initrepo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// M-069 AC-4 — Pre-push hook byte-golden plus template-equals-installed
// cross-check.
//
// The pre-push hook is the chokepoint that makes `aiwf check` mandatory
// before push. CLAUDE.md design decision §3: "aiwf check runs as a
// pre-push git hook. Validation is the chokepoint. The hook is what
// makes the framework's guarantees real; without it, skills are just
// suggestions." Hook content drift — between what `preHookScript`
// returns and what `ensurePreHook` writes — silently weakens that
// chokepoint. A regression where, say, the install path quietly
// dropped the chain prelude (G45 chaining), or the brownfield guard,
// or the exec line, would not be caught by any current test:
//
//   - TestPreHookScript_HasBrownfieldGuard checks substrings, not
//     byte content;
//   - TestInit_MigratesAlienPreHook asserts the marker is present
//     after migration, not the body shape;
//   - the existing initrepo_test.go assertions are similar
//     "contains" checks that pass even with significant drift.
//
// Substring assertions are not structural assertions (CLAUDE.md
// `Substring assertions are not structural assertions`). The hook is
// a load-bearing artifact whose every line carries semantic weight
// (chain prelude, brownfield guard, exec) — pinning it byte-for-byte
// is the right granularity.
//
// This file holds two tests:
//
//  1. TestPreHookScript_ByteGolden — renders `preHookScript` with a
//     sentinel binary path (/AIWF_BIN) and diffs the output against
//     `testdata/pre-push.golden`. A change to the template body, the
//     marker, the chain prelude, or the brownfield guard requires an
//     intentional golden update — drift surfaces as a failing diff.
//
//  2. TestPreHookScript_TemplateEqualsInstalled — runs `Init` in a
//     fresh tempdir, reads the installed `.git/hooks/pre-push` bytes,
//     re-renders `preHookScript(exePath)` with the same path init
//     used, and asserts byte-equality. This catches a regression
//     where the install path took a different code branch than the
//     template function (a parallel source of truth — the failure
//     mode CLAUDE.md `Test the seam, not just the layer` warns
//     about).

const sentinelBinaryPath = "/AIWF_BIN"

// TestPreHookScript_ByteGolden pins the rendered template against
// the golden file. A failure means the template body changed; either
// the change is intentional (regenerate the golden by inspecting the
// new template and updating testdata/pre-push.golden) or accidental
// (revert the change).
func TestPreHookScript_ByteGolden(t *testing.T) {
	got := preHookScript(sentinelBinaryPath)

	want, err := os.ReadFile("testdata/pre-push.golden")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	if diff := cmp.Diff(string(want), got); diff != "" {
		t.Errorf("preHookScript output differs from testdata/pre-push.golden (-want +got):\n%s", diff)
	}
}

// TestPreHookScript_TemplateEqualsInstalled runs aiwf init in a fresh
// tempdir, reads the installed pre-push hook, and asserts byte-equality
// against `preHookScript(exePath)` rendered with the same path init
// resolved. Cross-checks that ensurePreHook writes whatever the
// template function returns and nothing else — no parallel source of
// truth.
func TestPreHookScript_TemplateEqualsInstalled(t *testing.T) {
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

	exePath, err := resolveExecutable()
	if err != nil {
		t.Fatalf("resolveExecutable: %v", err)
	}
	rendered := preHookScript(exePath)

	if diff := cmp.Diff(rendered, string(installed)); diff != "" {
		t.Errorf("installed pre-push hook differs from preHookScript(exePath) — parallel source of truth (-template +installed):\n%s", diff)
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
