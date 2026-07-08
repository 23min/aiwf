package renamearea_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/renamearea"
)

// TestRun_ErrorExits covers rename-area's guard exits — inherited coverage debt
// (the package's Run had no unit test; only a happy-path integration dispatcher
// test). Surfaced by M-0181's epic-relative coverage gate and fixed here:
//   - a malformed actor (no role/identifier slash) → ResolveActor rejects it;
//   - a non-existent root → repo-lock acquisition fails (before tree load);
//   - a malformed contracts block → LoadContractsDoc rejects it (reached past a
//     clean tree load — the contracts doc is parsed separately from the tree).
//
// The ResolveRoot (broken-cwd) and LoadTreeWithTrunk (IO) arms are
// //coverage:ignore'd in renamearea.go as not deterministically reproducible.
func TestRun_ErrorExits(t *testing.T) {
	t.Parallel()
	var out cliutil.OutputFormat

	if rc := renamearea.Run("foo", "bar", "notanactor", "", t.TempDir(), out); rc != cliutil.ExitUsage {
		t.Errorf("malformed actor: rc = %d, want ExitUsage", rc)
	}
	bad := filepath.Join(t.TempDir(), "does-not-exist")
	if rc := renamearea.Run("foo", "bar", "human/test", "", bad, out); rc == cliutil.ExitOK {
		t.Errorf("non-existent root (lock should fail): rc = %d, want non-OK", rc)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"),
		[]byte("contracts:\n  bindings:\n    - not a valid binding\n"), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
	if rc := renamearea.Run("foo", "bar", "human/test", "", root, out); rc != cliutil.ExitUsage {
		t.Errorf("malformed contracts (LoadContractsDoc): rc = %d, want ExitUsage", rc)
	}
}

// TestNewCmd_SmokeShape pins the rename-area subpackage exports NewCmd
// with the expected metadata: the two-positional Use, the standard
// flags, a wired ValidArgsFunction, and the orphan-trap warning in the
// Long help text (AC-5).
func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := renamearea.NewCmd("")
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "rename-area <old> <new>" {
		t.Errorf("Use = %q; want %q", cmd.Use, "rename-area <old> <new>")
	}
	for _, flag := range []string{"actor", "principal", "root", "format"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
	if cmd.ValidArgsFunction == nil {
		t.Error("ValidArgsFunction not wired")
	}
	// The orphan-trap warning is the load-bearing discoverability text
	// per AC-5 (skill-coverage allowlists this verb to --help).
	for _, want := range []string{"orphan", "area-unknown"} {
		if !strings.Contains(cmd.Long, want) {
			t.Errorf("Long help missing %q:\n%s", want, cmd.Long)
		}
	}
}
