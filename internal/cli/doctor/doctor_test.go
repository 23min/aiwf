package doctor_test

import (
	"context"
	"testing"

	"github.com/23min/aiwf/internal/cli/doctor"
	"github.com/23min/aiwf/internal/initrepo"
)

// freshInitializedRootForDoctorTest builds a real, fully-materialized
// aiwf repo (via initrepo.Init) for Cobra-seam tests that need
// `aiwf doctor --check-rituals` to find nothing missing.
func freshInitializedRootForDoctorTest(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{SkipHook: true}); err != nil {
		t.Fatalf("initrepo.Init: %v", err)
	}
	return root
}

// TestNewCmd_HasFlags pins the doctor verb's flag surface. M-0117/AC-3.
func TestNewCmd_HasFlags(t *testing.T) {
	t.Parallel()
	cmd := doctor.NewCmd()
	if cmd.Use != "doctor" {
		t.Errorf("Use = %q, want %q", cmd.Use, "doctor")
	}
	for _, flag := range []string{"root", "self-check", "check-latest", "check-rituals"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("doctor missing --%s flag", flag)
		}
	}
}

// TestNewCmd_CheckRitualsFlagReachesRunCheckRituals exercises the
// actual Cobra wiring (flag registration through the RunE closure),
// not just a direct RunCheckRituals call — the seam `--check-rituals`
// depends on. A fully-materialized fixture keeps this a pure wiring
// check: RunCheckRituals's own behavior is covered directly by
// check_rituals_test.go.
func TestNewCmd_CheckRitualsFlagReachesRunCheckRituals(t *testing.T) {
	t.Parallel()
	root := freshInitializedRootForDoctorTest(t)
	cmd := doctor.NewCmd()
	cmd.SetArgs([]string{"--root", root, "--check-rituals"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute(): %v", err)
	}
}
