package update_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/update"
)

func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := update.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "update" {
		t.Errorf("Use = %q", cmd.Use)
	}
	if cmd.Flags().Lookup("root") == nil {
		t.Errorf("missing --root flag")
	}
}
