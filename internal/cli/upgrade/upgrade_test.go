package upgrade_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/upgrade"
)

func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := upgrade.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "upgrade" {
		t.Errorf("Use = %q", cmd.Use)
	}
}
