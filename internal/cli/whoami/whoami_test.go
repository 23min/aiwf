package whoami_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/whoami"
)

func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := whoami.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "whoami" {
		t.Errorf("Use = %q; want %q", cmd.Use, "whoami")
	}
	for _, flag := range []string{"root", "actor"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
}
