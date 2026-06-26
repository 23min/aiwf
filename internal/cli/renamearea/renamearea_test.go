package renamearea_test

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/renamearea"
)

// TestNewCmd_SmokeShape pins the rename-area subpackage exports NewCmd
// with the expected metadata: the two-positional Use, the standard
// flags, a wired ValidArgsFunction, and the orphan-trap warning in the
// Long help text (AC-5).
func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := renamearea.NewCmd()
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
