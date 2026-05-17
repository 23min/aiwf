package rewidth_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/rewidth"
)

func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := rewidth.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "rewidth [--apply]" {
		t.Errorf("Use = %q", cmd.Use)
	}
	for _, flag := range []string{"actor", "principal", "root", "apply"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
}
