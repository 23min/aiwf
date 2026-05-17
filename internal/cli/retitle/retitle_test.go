package retitle_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/retitle"
)

func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := retitle.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "retitle <id> <new-title>" {
		t.Errorf("Use = %q; want %q", cmd.Use, "retitle <id> <new-title>")
	}
	for _, flag := range []string{"actor", "principal", "root", "reason"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
	if cmd.ValidArgsFunction == nil {
		t.Error("ValidArgsFunction not wired")
	}
}
