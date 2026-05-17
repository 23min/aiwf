package rename_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/rename"
)

// TestNewCmd_SmokeShape pins M-0115/AC-5: the rename subpackage exports
// NewCmd with the expected metadata (Use, flags, ValidArgsFunction).
// Behavioral coverage stays with the cross-verb integration tests
// under cmd/aiwf/.
func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := rename.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "rename <id> <new-slug>" {
		t.Errorf("Use = %q; want %q", cmd.Use, "rename <id> <new-slug>")
	}
	for _, flag := range []string{"actor", "principal", "root"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
	if cmd.ValidArgsFunction == nil {
		t.Error("ValidArgsFunction not wired")
	}
}
