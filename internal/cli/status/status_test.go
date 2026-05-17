package status_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/status"
)

func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := status.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "status" {
		t.Errorf("Use = %q", cmd.Use)
	}
	for _, flag := range []string{"root", "format", "pretty", "no-trunc"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
}
