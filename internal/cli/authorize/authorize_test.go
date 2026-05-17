package authorize_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/authorize"
)

func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := authorize.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "authorize <id>" {
		t.Errorf("Use = %q", cmd.Use)
	}
	for _, flag := range []string{"actor", "root", "to", "pause", "resume", "reason", "force"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
}
