package move_test

import (
	"errors"
	"os"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/move"
)

// TestNewCmd_SmokeShape pins M-0115/AC-6: the move subpackage exports
// NewCmd with the expected metadata.
func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := move.NewCmd("")
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "move <milestone-id> --epic <epic-id>" {
		t.Errorf("Use = %q; want %q", cmd.Use, "move <milestone-id> --epic <epic-id>")
	}
	for _, flag := range []string{"actor", "principal", "root", "epic"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
	if cmd.ValidArgsFunction == nil {
		t.Error("ValidArgsFunction not wired")
	}
	if _, ok := cmd.GetFlagCompletionFunc("epic"); !ok {
		t.Error("--epic completion not bound")
	}
}

// TestNewCmd_RefusesMissingEpic: a move names no target without --epic,
// so the verb refuses as a usage error before touching the tree — the
// root it was pointed at is left exactly as it was, with no lock taken
// and no diagnostic written there.
func TestNewCmd_RefusesMissingEpic(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	cmd := move.NewCmd("")
	cmd.SetArgs([]string{"M-NNNN", "--root", root})
	err := cmd.Execute()
	var ee *cliutil.ExitError
	if !errors.As(err, &ee) || ee.Code != cliutil.ExitUsage {
		t.Errorf("Execute() error = %v, want an ExitUsage *cliutil.ExitError", err)
	}
	entries, rerr := os.ReadDir(root)
	if rerr != nil {
		t.Fatalf("reading root: %v", rerr)
	}
	if len(entries) != 0 {
		t.Errorf("refusal touched the root; it now holds %v", entries)
	}
}
