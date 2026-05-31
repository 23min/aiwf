package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/doctor"
)

// initForDoctor inits a repo (materializing verb + ritual artifacts) and
// returns its root. --skip-hook keeps the test from installing the test
// binary as a git hook.
func initForDoctor(t *testing.T) string {
	t.Helper()
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	t.Setenv("HOME", t.TempDir())
	return root
}

// TestDoctorReport_RitualsMaterialized_OK covers M-0152 AC-1: after
// init materializes the rituals, doctor reports a `rituals:` ok line
// and no "not materialized" warning.
func TestDoctorReport_RitualsMaterialized_OK(t *testing.T) {
	root := initForDoctor(t)
	lines, _ := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "rituals:") {
		t.Errorf("expected a rituals: line; got:\n%s", joined)
	}
	if strings.Contains(joined, "not materialized") {
		t.Errorf("freshly-inited repo should not warn about missing rituals; got:\n%s", joined)
	}
	if strings.Contains(joined, "artifacts materialized") == false {
		t.Errorf("expected rituals ok line naming materialized artifacts; got:\n%s", joined)
	}
}

// TestDoctorReport_RitualsMissing_WarnsSoft covers M-0152 AC-1: when
// ritual artifacts are absent, doctor emits a soft warning pointing at
// `aiwf update` and does NOT increment the problem count for it.
func TestDoctorReport_RitualsMissing_WarnsSoft(t *testing.T) {
	root := initForDoctor(t)
	// Capture the baseline problem count, then remove materialized
	// templates so the rituals check has something missing.
	_, before := doctor.DoctorReport(root, doctor.DoctorOptions{})
	if err := os.RemoveAll(filepath.Join(root, ".claude", "templates")); err != nil {
		t.Fatal(err)
	}
	lines, after := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "not materialized") || !strings.Contains(joined, "aiwf update") {
		t.Errorf("expected 'not materialized — run aiwf update' warning; got:\n%s", joined)
	}
	if after != before {
		t.Errorf("missing rituals must be a soft warning (problems unchanged): before=%d after=%d", before, after)
	}
}

// TestDoctorReport_RitualsLine_SurfacesProvenance covers the
// discoverability enhancement: the rituals: ok line is followed by a
// note naming aiwf as the manager, the refresh verb, and the
// do-not-hand-edit contract — so an operator reading `aiwf doctor`
// learns these skills came from aiwf.
func TestDoctorReport_RitualsLine_SurfacesProvenance(t *testing.T) {
	root := initForDoctor(t)
	lines, _ := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"managed by aiwf", "aiwf update", "do not hand-edit"} {
		if !strings.Contains(joined, want) {
			t.Errorf("doctor rituals provenance note missing %q; got:\n%s", want, joined)
		}
	}
}
