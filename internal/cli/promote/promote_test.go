package promote_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/promote"
)

// TestNewCmd_SmokeShape pins M-0115/AC-2: the promote subpackage
// exports NewCmd with the expected metadata.
func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := promote.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "promote <id> [new-status]" {
		t.Errorf("Use = %q; want %q", cmd.Use, "promote <id> [new-status]")
	}
	for _, flag := range []string{"actor", "principal", "root", "reason", "phase", "tests", "by", "by-commit", "superseded-by", "force", "audit-only"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
	if cmd.ValidArgsFunction == nil {
		t.Error("ValidArgsFunction not wired")
	}
	if _, ok := cmd.GetFlagCompletionFunc("phase"); !ok {
		t.Error("--phase completion not bound")
	}
}
