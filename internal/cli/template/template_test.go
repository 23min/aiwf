package template_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/template"
)

func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := template.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "template [kind]" {
		t.Errorf("Use = %q", cmd.Use)
	}
	for _, flag := range []string{"format", "pretty"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
}
