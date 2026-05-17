package editbody_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/editbody"
)

// TestNewCmd_SmokeShape pins M-0115/AC-3: the editbody subpackage
// exports NewCmd with the expected metadata.
func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := editbody.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "edit-body <id>" {
		t.Errorf("Use = %q; want %q", cmd.Use, "edit-body <id>")
	}
	for _, flag := range []string{"actor", "principal", "root", "reason", "body-file"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
	if cmd.ValidArgsFunction == nil {
		t.Error("ValidArgsFunction not wired")
	}
}
