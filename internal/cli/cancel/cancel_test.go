package cancel_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/cancel"
)

// TestNewCmd_SmokeShape pins M-0115/AC-4: the cancel subpackage exposes
// NewCmd as the package's canonical Cobra constructor. The dispatcher-
// level behavior (cancel transition, --reason gating, --audit-only path)
// is exercised by the cross-verb integration tests under cmd/aiwf;
// this test just asserts the export shape and command metadata so the
// subpackage's pattern is mechanically pinned.
func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := cancel.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "cancel <id>" {
		t.Errorf("Use = %q; want %q", cmd.Use, "cancel <id>")
	}
	for _, flag := range []string{"actor", "principal", "root", "reason", "force", "audit-only"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
	if cmd.ValidArgsFunction == nil {
		t.Error("ValidArgsFunction not wired")
	}
}
