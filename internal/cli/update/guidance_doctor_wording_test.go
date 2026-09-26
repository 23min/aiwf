package update_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/testsupport"
)

// doctorPolicyLines returns doctor's project-policy lines; doctor exits
// nonzero when it reports a problem, so only a failure to run is fatal.
func doctorPolicyLines(t *testing.T, root string) []string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), testutil.AiwfBinary(t), "doctor", "--root", root)
	cmd.Env = append(os.Environ(), testsupport.GuidanceEnvironment(t)...)
	output, err := cmd.CombinedOutput()
	var exited *exec.ExitError
	if err != nil && !errors.As(err, &exited) {
		t.Fatalf("doctor: %v\n%s", err, output)
	}
	var lines []string
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "project policy:") {
			lines = append(lines, line)
		}
	}
	return lines
}

// TestBinary_GuidanceStateWordingMatchesDoctor runs update and then doctor
// against the same repository and requires every state line of the report to
// appear, after its label, as a project-policy line of doctor.
func TestBinary_GuidanceStateWordingMatchesDoctor(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		// configuration is aiwf.yaml after the baseline run; %s is the source.
		configuration string
		pending       bool
		// states lists the report labels whose lines doctor must match.
		states []string
	}{
		{name: "selection unset", configuration: "guidance:\n  source: %s\n", states: []string{"selected:"}},
		{name: "explicitly empty", configuration: "guidance:\n  source: %s\n  packs: []\n", states: []string{"selected:"}},
		{name: "populated", configuration: "guidance:\n  source: %s\n  packs: [sample/base]\n", states: []string{"selected:"}},
		{name: "maintenance disabled", configuration: "guidance:\n  enabled: false\n  source: %s\n  packs: [sample/base]\n", states: []string{"selected:", "maintenance disabled"}},
		{name: "installation pending", configuration: "guidance:\n  source: %s/absent\n  packs: [sample/base]\n", pending: true, states: []string{"selected:", "pending:"}},
		{name: "selected pack not installed", configuration: "guidance:\n  source: %s\n  packs: [sample/base, gone/pack]\n", states: []string{"selected:", "missing:"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root, source := t.TempDir(), adoptionSource(t, "*.xyz", "*.abc")
			if err := gitops.Init(t.Context(), root); err != nil {
				t.Fatal(err)
			}
			guidanceWrite(t, root, config.FileName, "hosts: [codex]\nguidance:\n  source: "+source+"\n  packs: [sample/base]\n")
			runUpdateBinary(t, root)
			guidanceWrite(t, root, config.FileName, "hosts: [codex]\n"+strings.ReplaceAll(tc.configuration, "%s", source))
			if tc.pending {
				// An interrupted installation leaves its recovery record behind.
				guidanceWrite(t, root, filepath.Join(".guidance", ".aiwf-pending"), "{}")
			}
			section := guidanceSection(t, runUpdateBinary(t, root))
			doctor := doctorPolicyLines(t, root)
			for _, label := range tc.states {
				found := false
				for _, line := range section {
					value, ok := strings.CutPrefix(strings.TrimSpace(line), label)
					if !ok {
						continue
					}
					found = true
					if label == "maintenance disabled" {
						value = label + value
					}
					value = strings.TrimSpace(value)
					matched := false
					for _, doctorLine := range doctor {
						matched = matched || strings.HasSuffix(doctorLine, " "+value)
					}
					if !matched {
						t.Errorf("report state %q has no doctor line ending %q:\n%s", label, value, strings.Join(doctor, "\n"))
					}
				}
				if !found {
					t.Errorf("report has no %q line:\n%s", label, strings.Join(section, "\n"))
				}
			}
		})
	}
}
