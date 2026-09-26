package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/testsupport"
)

// trailingGuidanceSection returns the lines from the last "Guidance:" heading
// to the end of output, requiring that heading to exist.
func trailingGuidanceSection(t *testing.T, output string) []string {
	t.Helper()
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i] == "Guidance:" {
			return lines[i:]
		}
	}
	t.Fatalf("output does not end with a guidance section:\n%s", output)
	return nil
}

// TestBinary_InitEndsWithTheGuidanceSectionUpdatePrints runs init and update
// on two identical repositories whose aiwf.yaml already ignores a pack, so the
// same outcome must end both runs with the same section.
func TestBinary_InitEndsWithTheGuidanceSectionUpdatePrints(t *testing.T) {
	t.Parallel()
	source := testsupport.GuidanceSource(t, "*.xyz")
	sections := map[string][]string{}
	for _, verb := range []string{"init", "update"} {
		root := t.TempDir()
		if err := gitops.Init(t.Context(), root); err != nil {
			t.Fatal(err)
		}
		configuration := "hosts: [codex]\nguidance:\n  source: " + source + "\n  ignored: [other/pack]\n"
		if err := os.WriteFile(filepath.Join(root, config.FileName), []byte(configuration), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "source.xyz"), []byte("project source"), 0o644); err != nil {
			t.Fatal(err)
		}
		args := []string{verb, "--root", root}
		if verb == "init" {
			args = append(args, "--skip-hook", "--actor", "human/test")
		}
		cmd := exec.CommandContext(t.Context(), testutil.AiwfBinary(t), args...)
		cmd.Env = append(os.Environ(), testsupport.GuidanceEnvironment(t)...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v\n%s", verb, err, output)
		}
		sections[verb] = trailingGuidanceSection(t, string(output))
	}
	if diff := cmp.Diff(sections["update"], sections["init"]); diff != "" {
		t.Fatalf("init guidance section differs from update's (-update +init):\n%s", diff)
	}
	if !cmp.Equal(sections["init"][1:3], []string{"  selected:  sample/base", "  adopted:   sample/base"}) {
		t.Fatalf("init did not report the adoption:\n%s", strings.Join(sections["init"], "\n"))
	}
}
