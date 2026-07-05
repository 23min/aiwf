package initcmd_test

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/initcmd"
)

func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := initcmd.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "init" {
		t.Errorf("Use = %q", cmd.Use)
	}
}

// TestNewCmd_HelpDocumentsIdempotentReRun: `aiwf init --help` (the
// command's Long description) must state the re-run is idempotent and
// name every artifact init never overwrites (M-0232/AC-5). Scoped to
// the Long field specifically — the one Cobra surface --help actually
// renders this prose from — not a blind grep over the file.
func TestNewCmd_HelpDocumentsIdempotentReRun(t *testing.T) {
	t.Parallel()
	cmd := initcmd.NewCmd()
	help := cmd.Long
	if !strings.Contains(help, "idempotent") {
		t.Errorf("Long missing an idempotent re-run statement: %q", help)
	}
	for _, never := range []string{"aiwf.yaml", ".claude/settings.json", "git hooks"} {
		if !strings.Contains(help, never) {
			t.Errorf("Long missing %q from the never-overwritten list: %q", never, help)
		}
	}
}
