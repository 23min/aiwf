package add_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/add"
)

// TestNewCmd_SmokeShape pins M-0115/AC-1: the add subpackage exports
// NewCmd with the expected metadata, including the `ac` subcommand.
func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := add.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "add <kind> [...]" {
		t.Errorf("Use = %q; want %q", cmd.Use, "add <kind> [...]")
	}
	for _, flag := range []string{"title", "actor", "principal", "root", "epic", "tdd", "depends-on", "discovered-in", "relates-to", "linked-adr", "validator", "schema", "fixtures", "body-file"} {
		if cmd.PersistentFlags().Lookup(flag) == nil && cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag (persistent or local)", flag)
		}
	}
	if cmd.ValidArgsFunction == nil {
		t.Error("ValidArgsFunction not wired")
	}
	// add carries `ac` as a Cobra subcommand.
	var acSubcmd bool
	for _, sub := range cmd.Commands() {
		if sub.Use == "ac <milestone-id>" {
			acSubcmd = true
		}
	}
	if !acSubcmd {
		t.Error("add subcommand `ac` not registered")
	}
}
