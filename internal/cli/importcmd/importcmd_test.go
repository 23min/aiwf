package importcmd_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/importcmd"
)

func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := importcmd.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "import <manifest>" {
		t.Errorf("Use = %q", cmd.Use)
	}
	for _, flag := range []string{"actor", "principal", "root", "on-collision", "dry-run"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
}
