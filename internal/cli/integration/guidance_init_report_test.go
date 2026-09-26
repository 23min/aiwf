package integration

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/testsupport"
)

// trailingGuidanceSection returns the lines from the "Guidance:" heading to the
// end of output, requiring exactly one such section.
func trailingGuidanceSection(t *testing.T, output string) []string {
	t.Helper()
	if n := strings.Count(output, "\nGuidance:\n"); n != 1 {
		t.Fatalf("output has %d guidance sections, want 1:\n%s", n, output)
	}
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

// TestBinary_UpgradeOutputEndsWithTheGuidanceSection runs upgrade through the
// real re-exec into update; the re-executed update prints the section, so the
// upgrade's output ends with it.
func TestBinary_UpgradeOutputEndsWithTheGuidanceSection(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	binary := testutil.BuildBinary(t, t.TempDir())
	root, home, source := t.TempDir(), t.TempDir(), testsupport.GuidanceSource(t, "*.xyz")
	path := hostLifecyclePATH(t, binary, nil)
	hostLifecycleRun(t, root, home, path, "git", "init", "-q", "-b", "main")
	claudeBaselineWrite(t, root, "aiwf.yaml", "hosts: [codex]\nguidance:\n  source: "+source+"\n")
	claudeBaselineWrite(t, root, "source.xyz", "project source")
	output := hostUpgradeRun(t, binary, root, home, path, false, 0, "upgrade", "--version", "v0.1.0")
	section := trailingGuidanceSection(t, output)
	if !cmp.Equal(section[1:3], []string{"  selected:  sample/base", "  adopted:   sample/base"}) {
		t.Fatalf("upgrade output did not report the adoption:\n%s", strings.Join(section, "\n"))
	}
}

// TestBinary_HookCollisionStillEndsWithTheGuidanceSection runs init and update
// where the pre-push hook cannot be migrated: the run exits with findings, but
// guidance has already been adopted and recorded, so the section still ends it.
func TestBinary_HookCollisionStillEndsWithTheGuidanceSection(t *testing.T) {
	t.Parallel()
	source := testsupport.GuidanceSource(t, "*.xyz")
	for _, verb := range []string{"init", "update"} {
		t.Run(verb, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if err := gitops.Init(t.Context(), root); err != nil {
				t.Fatal(err)
			}
			files := map[string]string{
				config.FileName:             "hosts: [codex]\nguidance:\n  source: " + source + "\n",
				"source.xyz":                "project source",
				".git/hooks/pre-push":       "#!/bin/sh\nexit 0\n",
				".git/hooks/pre-push.local": "#!/bin/sh\nexit 0\n",
			}
			for name, content := range files {
				if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			args := []string{verb, "--root", root}
			if verb == "init" {
				args = append(args, "--actor", "human/test")
			}
			cmd := exec.CommandContext(t.Context(), testutil.AiwfBinary(t), args...)
			cmd.Env = append(os.Environ(), testsupport.GuidanceEnvironment(t)...)
			output, err := cmd.CombinedOutput()
			var exited *exec.ExitError
			if !errors.As(err, &exited) || exited.ExitCode() != cliutil.ExitFindings {
				t.Fatalf("%s: want exit %d for a hook collision, got %v\n%s", verb, cliutil.ExitFindings, err, output)
			}
			section := trailingGuidanceSection(t, string(output))
			if !cmp.Equal(section[1:3], []string{"  selected:  sample/base", "  adopted:   sample/base"}) {
				t.Fatalf("hook collision hid the adoption:\n%s", output)
			}
		})
	}
}

// TestBinary_GuidanceReportHelpReachesInitAndUpdate pins that the verbs which
// print the section describe it in their help.
func TestBinary_GuidanceReportHelpReachesInitAndUpdate(t *testing.T) {
	t.Parallel()
	for _, verb := range []string{"init", "update"} {
		output, err := exec.CommandContext(t.Context(), testutil.AiwfBinary(t), verb, "--help").CombinedOutput()
		if err != nil {
			t.Fatalf("%s --help: %v", verb, err)
		}
		if !strings.Contains(string(output), cliutil.GuidanceReportHelp) {
			t.Errorf("%s --help does not describe the guidance section", verb)
		}
	}
}

// TestBinary_LaterFailureStillEndsWithTheGuidanceSection makes a step after
// guidance fail (the hooks directory is a file): the run exits with an internal
// error, but guidance has already adopted and recorded a pack, so the section
// still ends its output.
func TestBinary_LaterFailureStillEndsWithTheGuidanceSection(t *testing.T) {
	t.Parallel()
	source := testsupport.GuidanceSource(t, "*.xyz")
	for _, verb := range []string{"init", "update"} {
		t.Run(verb, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if err := gitops.Init(t.Context(), root); err != nil {
				t.Fatal(err)
			}
			if err := os.RemoveAll(filepath.Join(root, ".git", "hooks")); err != nil {
				t.Fatal(err)
			}
			files := map[string]string{
				config.FileName: "hosts: [codex]\nguidance:\n  source: " + source + "\n",
				"source.xyz":    "project source",
				".git/hooks":    "not a directory",
			}
			for name, content := range files {
				if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			args := []string{verb, "--root", root}
			if verb == "init" {
				args = append(args, "--actor", "human/test")
			}
			cmd := exec.CommandContext(t.Context(), testutil.AiwfBinary(t), args...)
			cmd.Env = append(os.Environ(), testsupport.GuidanceEnvironment(t)...)
			stdout, err := cmd.Output()
			var exited *exec.ExitError
			if !errors.As(err, &exited) || exited.ExitCode() != cliutil.ExitInternal {
				t.Fatalf("%s: want exit %d from the failing hook step, got %v\n%s", verb, cliutil.ExitInternal, err, stdout)
			}
			section := trailingGuidanceSection(t, string(stdout))
			if !cmp.Equal(section[1:3], []string{"  selected:  sample/base", "  adopted:   sample/base"}) {
				t.Fatalf("later failure hid the adoption:\n%s", stdout)
			}
		})
	}
}
