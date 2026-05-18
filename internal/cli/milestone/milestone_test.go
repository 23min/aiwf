package milestone_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/milestone"
)

// TestNewCmd_HasDependsOnChild pins the milestone parent verb's
// shape and its single subcommand. M-0117/AC-5+AC-6.
func TestNewCmd_HasDependsOnChild(t *testing.T) {
	t.Parallel()
	cmd := milestone.NewCmd()
	if cmd.Use != "milestone" {
		t.Errorf("Use = %q, want %q", cmd.Use, "milestone")
	}
	var dependsOn bool
	for _, c := range cmd.Commands() {
		if c.Use == "depends-on <milestone-id>" {
			dependsOn = true
		}
	}
	if !dependsOn {
		t.Error("milestone.NewCmd missing depends-on subcommand")
	}
}

// TestDependsOnCmd_FlagShape pins the depends-on subcommand's
// flag surface. Drift here would silently break the canonical
// invocation. M-0117/AC-6.
func TestDependsOnCmd_FlagShape(t *testing.T) {
	t.Parallel()
	cmd := milestone.NewCmd()
	var dependsOn *struct{}
	_ = dependsOn
	for _, c := range cmd.Commands() {
		if c.Use == "depends-on <milestone-id>" {
			for _, flag := range []string{"actor", "principal", "root", "reason", "on", "clear"} {
				if c.Flags().Lookup(flag) == nil {
					t.Errorf("depends-on missing --%s flag", flag)
				}
			}
			return
		}
	}
	t.Error("depends-on subcommand not found")
}
