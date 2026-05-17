package initcmd_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/initcmd"
)

func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := initcmd.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "init" {
		t.Errorf("Use = %q", cmd.Use)
	}
}
