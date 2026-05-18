package contract_test

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/contract"
)

// TestNewCmd_HasParentAndChildren is the M-0117/AC-1 smoke test for
// the contract verb's migration into internal/cli/contract/. The
// parent verb is non-Runnable; all six subcommands wire into it.
func TestNewCmd_HasParentAndChildren(t *testing.T) {
	t.Parallel()
	cmd := contract.NewCmd()
	if cmd.Use != "contract" {
		t.Errorf("Use = %q, want %q", cmd.Use, "contract")
	}
	want := map[string]bool{
		"verify":  false,
		"bind":    false,
		"unbind":  false,
		"recipes": false,
		"recipe":  false,
	}
	for _, c := range cmd.Commands() {
		name := strings.SplitN(c.Use, " ", 2)[0]
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("contract.NewCmd missing %q subcommand", name)
		}
	}
}
