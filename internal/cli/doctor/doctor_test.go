package doctor_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/doctor"
)

// TestNewCmd_HasFlags pins the doctor verb's flag surface. M-0117/AC-3.
func TestNewCmd_HasFlags(t *testing.T) {
	t.Parallel()
	cmd := doctor.NewCmd()
	if cmd.Use != "doctor" {
		t.Errorf("Use = %q, want %q", cmd.Use, "doctor")
	}
	for _, flag := range []string{"root", "self-check", "check-latest"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("doctor missing --%s flag", flag)
		}
	}
}
