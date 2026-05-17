package history_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/history"
)

func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := history.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "history <id>" {
		t.Errorf("Use = %q", cmd.Use)
	}
	for _, flag := range []string{"root", "format", "pretty"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
}
