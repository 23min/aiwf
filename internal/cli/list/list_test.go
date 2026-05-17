package list_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/list"
)

func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := list.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "list" {
		t.Errorf("Use = %q", cmd.Use)
	}
	for _, flag := range []string{"root", "kind", "status", "parent", "archived", "format", "pretty", "no-trunc"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
}
