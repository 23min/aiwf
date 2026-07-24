package milestone_test

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/milestone"
	"github.com/23min/aiwf/internal/entity"
)

// TestNewCmd_HasDependsOnChild pins the milestone parent verb's
// shape and its single subcommand. M-0117/AC-5+AC-6.
func TestNewCmd_HasDependsOnChild(t *testing.T) {
	t.Parallel()
	cmd := milestone.NewCmd("")
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
	cmd := milestone.NewCmd("")
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

// TestNewCmd_HasTDDChild pins AC-5 (E-0071): the milestone parent
// carries the tdd subcommand.
func TestNewCmd_HasTDDChild(t *testing.T) {
	t.Parallel()
	cmd := milestone.NewCmd("")
	var found bool
	for _, c := range cmd.Commands() {
		if c.Use == "tdd <milestone-id>" {
			found = true
		}
	}
	if !found {
		t.Error("milestone.NewCmd missing tdd subcommand")
	}
}

// tddCmd returns the milestone tdd subcommand, failing the test if it
// is not wired.
func tddCmd(t *testing.T) *cobra.Command {
	t.Helper()
	for _, c := range milestone.NewCmd("").Commands() {
		if c.Use == "tdd <milestone-id>" {
			return c
		}
	}
	t.Fatal("tdd subcommand not found")
	return nil
}

// TestTDDCmd_FlagShape pins AC-5: the tdd subcommand's flag surface and
// its positional-id completion.
func TestTDDCmd_FlagShape(t *testing.T) {
	t.Parallel()
	cmd := tddCmd(t)
	for _, flag := range []string{"actor", "principal", "root", "reason", "policy"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("tdd missing --%s flag", flag)
		}
	}
	if cmd.ValidArgsFunction == nil {
		t.Error("tdd ValidArgsFunction (milestone-id completion) not wired")
	}
}

// TestTDDCmd_PolicyCompletion pins AC-5: --policy completes the closed
// TDD-policy set, so a shell offers exactly {required, advisory, none}.
func TestTDDCmd_PolicyCompletion(t *testing.T) {
	t.Parallel()
	cmd := tddCmd(t)
	fn, ok := cmd.GetFlagCompletionFunc("policy")
	if !ok {
		t.Fatal("--policy completion not bound")
	}
	got, directive := fn(cmd, nil, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("--policy completion directive = %v, want NoFileComp", directive)
	}
	want := entity.AllowedTDDPolicies()
	if len(got) != len(want) {
		t.Fatalf("--policy completions = %v, want %v", got, want)
	}
	set := map[string]bool{}
	for _, v := range got {
		set[v] = true
	}
	for _, w := range want {
		if !set[w] {
			t.Errorf("--policy completion missing %q (got %v)", w, got)
		}
	}
}
