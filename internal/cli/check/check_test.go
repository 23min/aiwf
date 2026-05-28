package check

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
)

// TestNewCmd_FlagShape pins the check verb's flag surface so a
// future migration can't silently drop or rename a flag without
// the test failing. The completion drift test in cmd/aiwf/
// catches the same regression at the binary level.
func TestNewCmd_FlagShape(t *testing.T) {
	t.Parallel()
	cmd := NewCmd()
	if cmd.Use != "check" {
		t.Errorf("Use = %q, want check", cmd.Use)
	}
	expected := []string{"root", "format", "pretty", "since", "shape-only", "verbose"}
	for _, name := range expected {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("flag %q missing", name)
		}
	}
}

// TestRun_BadFormat pins the format-validation guard at the top of
// Run. A non-{text,json} value returns ExitUsage immediately
// without loading the tree.
func TestRun_BadFormat(t *testing.T) {
	t.Parallel()
	code := Run("", "yaml", false, "", false, false, nil)
	if code != cliutil.ExitUsage {
		t.Errorf("Run with --format=yaml: got %d, want %d", code, cliutil.ExitUsage)
	}
}
