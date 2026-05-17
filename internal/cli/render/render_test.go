package render_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/render"
)

func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := render.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "render" {
		t.Errorf("Use = %q", cmd.Use)
	}
}
