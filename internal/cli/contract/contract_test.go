package contract_test

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/contract"
)

// findChild returns the cobra subcommand of parent whose Use field
// begins with prefix, or nil. The first-token match handles "verify"
// vs "bind <C-id>" without sensitivity to the argspec.
func findChild(parent *cobra.Command, prefix string) *cobra.Command {
	for _, c := range parent.Commands() {
		first := strings.SplitN(c.Use, " ", 2)[0]
		if first == prefix {
			return c
		}
	}
	return nil
}

// TestNewCmd_HasParentAndChildren pins the contract parent verb's
// shape: name, non-Runnable behavior, and the five direct subcommand
// children (verify, bind, unbind, recipes, recipe). M-0117/AC-1.
func TestNewCmd_HasParentAndChildren(t *testing.T) {
	t.Parallel()
	cmd := contract.NewCmd()
	if cmd.Use != "contract" {
		t.Errorf("Use = %q, want %q", cmd.Use, "contract")
	}
	for _, name := range []string{"verify", "bind", "unbind", "recipes", "recipe"} {
		if findChild(cmd, name) == nil {
			t.Errorf("contract.NewCmd missing %q subcommand", name)
		}
	}
}

// TestVerifyCmd_FlagShape pins the verify subcommand's expected
// flags. Drift here would silently break the JSON envelope contract
// (--format) or the CI integration (--root).
func TestVerifyCmd_FlagShape(t *testing.T) {
	t.Parallel()
	cmd := findChild(contract.NewCmd(), "verify")
	if cmd == nil {
		t.Fatal("verify subcommand not found")
	}
	for _, flag := range []string{"root", "format", "pretty"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("verify missing --%s flag", flag)
		}
	}
}

// TestBindCmd_FlagShape pins bind's flags. M-0117/AC-2: drift here
// would silently break the canonical bind invocation shape.
func TestBindCmd_FlagShape(t *testing.T) {
	t.Parallel()
	cmd := findChild(contract.NewCmd(), "bind")
	if cmd == nil {
		t.Fatal("bind subcommand not found")
	}
	for _, flag := range []string{"root", "actor", "validator", "schema", "fixtures", "force"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("bind missing --%s flag", flag)
		}
	}
}

// TestUnbindCmd_FlagShape pins unbind's flags. M-0117/AC-2.
func TestUnbindCmd_FlagShape(t *testing.T) {
	t.Parallel()
	cmd := findChild(contract.NewCmd(), "unbind")
	if cmd == nil {
		t.Fatal("unbind subcommand not found")
	}
	for _, flag := range []string{"root", "actor"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("unbind missing --%s flag", flag)
		}
	}
}

// TestRecipesCmd_FlagShape pins recipes's flags. M-0117/AC-2.
func TestRecipesCmd_FlagShape(t *testing.T) {
	t.Parallel()
	cmd := findChild(contract.NewCmd(), "recipes")
	if cmd == nil {
		t.Fatal("recipes subcommand not found")
	}
	if cmd.Flags().Lookup("root") == nil {
		t.Error("recipes missing --root flag")
	}
}

// TestRecipeCmd_HasChildren pins the recipe sub-parent's three
// children (show, install, remove). The recipe parent itself is
// non-Runnable. M-0117/AC-2.
func TestRecipeCmd_HasChildren(t *testing.T) {
	t.Parallel()
	cmd := findChild(contract.NewCmd(), "recipe")
	if cmd == nil {
		t.Fatal("recipe sub-parent not found")
	}
	for _, name := range []string{"show", "install", "remove"} {
		if findChild(cmd, name) == nil {
			t.Errorf("recipe missing %q subcommand", name)
		}
	}
}

// TestRecipeInstallCmd_FlagShape pins install's flags. The dual
// shape (positional name or --from) is enforced at runtime via
// runRecipeInstall; the flag set itself just declares the surface.
// M-0117/AC-2.
func TestRecipeInstallCmd_FlagShape(t *testing.T) {
	t.Parallel()
	cmd := findChild(findChild(contract.NewCmd(), "recipe"), "install")
	if cmd == nil {
		t.Fatal("recipe install subcommand not found")
	}
	for _, flag := range []string{"root", "actor", "from", "force"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("recipe install missing --%s flag", flag)
		}
	}
}

// TestRecipeRemoveCmd_FlagShape pins remove's flags. M-0117/AC-2.
func TestRecipeRemoveCmd_FlagShape(t *testing.T) {
	t.Parallel()
	cmd := findChild(findChild(contract.NewCmd(), "recipe"), "remove")
	if cmd == nil {
		t.Fatal("recipe remove subcommand not found")
	}
	for _, flag := range []string{"root", "actor"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("recipe remove missing --%s flag", flag)
		}
	}
}

// TestRun_BadFormat exercises Run's --format validation branch.
// Invalid format returns ExitUsage without touching disk. M-0117/AC-2.
func TestRun_BadFormat(t *testing.T) {
	t.Parallel()
	rc := contract.Run("", "yaml", false)
	if rc != cliutil.ExitUsage {
		t.Errorf("Run(format=yaml) = %d, want ExitUsage=%d", rc, cliutil.ExitUsage)
	}
}

// TestBindingCount covers BindingCount's two branches: nil and
// populated. The function is small but consumed by the JSON envelope
// metadata, so the contract is worth pinning. M-0117/AC-2.
func TestBindingCount(t *testing.T) {
	t.Parallel()
	if got := contract.BindingCount(nil); got != 0 {
		t.Errorf("BindingCount(nil) = %d, want 0", got)
	}
}
