package archive_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/archive"
)

func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := archive.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "archive [--apply | --dry-run] [--kind <kind>]" {
		t.Errorf("Use = %q", cmd.Use)
	}
	for _, flag := range []string{"actor", "principal", "root", "apply", "dry-run", "kind"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
}
