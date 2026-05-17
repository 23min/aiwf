package move_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/move"
)

// TestNewCmd_SmokeShape pins M-0115/AC-6: the move subpackage exports
// NewCmd with the expected metadata.
func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := move.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "move <M-id> --epic <E-id>" {
		t.Errorf("Use = %q; want %q", cmd.Use, "move <M-id> --epic <E-id>")
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
