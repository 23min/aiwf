package reallocate_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/reallocate"
)

// TestNewCmd_SmokeShape pins M-0115/AC-7: the reallocate subpackage
// exports NewCmd with the expected metadata.
func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := reallocate.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "reallocate <id-or-path>" {
		t.Errorf("Use = %q; want %q", cmd.Use, "reallocate <id-or-path>")
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
