package update_test

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/update"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/skills"
	"github.com/23min/aiwf/internal/testsupport"
)

func guidanceWrite(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func guidanceConfig(t *testing.T, root, source, selection string, enabled bool) {
	t.Helper()
	maintenance := "true"
	if !enabled {
		maintenance = "false"
	}
	guidanceWrite(t, root, config.FileName, "hosts: [claude-code, codex]\nguidance:\n  source: "+source+"\n  packs: "+selection+"\n  enabled: "+maintenance+"\n")
}

func guidanceFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	files := make(map[string]string)
	err := filepath.WalkDir(filepath.Join(root, ".guidance"), func(path string, entry fs.DirEntry, err error) error {
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		name, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(name)] = string(content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"CLAUDE.md", "AGENTS.md"} {
		content, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		files[name] = string(content)
	}
	return files
}

// Serial: captures process stdout; see setup_test.go.
func guidanceUpdate(t *testing.T, root string) string {
	t.Helper()
	return string(testutil.CaptureStdout(t, func() {
		if rc := update.Run(root, false, "", false, false, false, false, false, nil, nil); rc != cliutil.ExitOK {
			t.Errorf("update exit %d", rc)
		}
	}))
}

func guidanceInstallBaseline(t *testing.T, root, source string) string {
	t.Helper()
	output := guidanceUpdate(t, root)
	if strings.Contains(output, "guidance incomplete") {
		t.Fatalf("baseline installation failed: %s", output)
	}
	files := guidanceFiles(t, root)
	expected, err := os.ReadFile(filepath.Join(source, "packs", "sample", "base", "guide.md"))
	if err != nil {
		t.Fatal(err)
	}
	if files[".guidance/packs/sample/base/guide.md"] != string(expected) {
		t.Fatal("baseline document does not match upstream")
	}
	revision, err := gitops.ResolveCommitSHA(t.Context(), source, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(files[".guidance/index.md"], "Installed commit: `"+revision+"`") {
		t.Fatal("baseline index does not record fetched revision")
	}
	for _, name := range []string{"CLAUDE.md", "AGENTS.md"} {
		if !strings.Contains(files[name], "[.guidance/index.md](.guidance/index.md)") {
			t.Fatalf("baseline %s does not route to installed guidance", name)
		}
	}
	return revision
}

func TestRun_ProjectGuidanceFailuresPreserveInstallationAndContinue(t *testing.T) {
	testsupport.IsolateGuidanceEnvironment(t)
	for _, kind := range []string{"unavailable source", "invalid catalogue", "missing document", "edited document", "edited retired document", "first installation"} {
		t.Run(kind, func(t *testing.T) {
			root, source := freshInitializedRepo(t), testsupport.GuidanceSource(t)
			guidanceConfig(t, root, source, "[sample/base]", true)
			if kind != "first installation" {
				guidanceInstallBaseline(t, root, source)
			} else {
				guidanceConfig(t, root, source, "[sample/base]", false)
				guidanceUpdate(t, root)
			}
			switch kind {
			case "unavailable source", "first installation":
				guidanceConfig(t, root, filepath.Join(root, "absent-source"), "[sample/base]", true)
			case "invalid catalogue":
				guidanceWrite(t, source, "catalogue.json", "not JSON\n")
				testsupport.CommitGuidanceSource(t, source)
			case "missing document":
				if err := os.Remove(filepath.Join(source, "packs", "sample", "base", "guide.md")); err != nil {
					t.Fatal(err)
				}
				testsupport.CommitGuidanceSource(t, source)
			case "edited document", "edited retired document":
				guidanceWrite(t, root, ".guidance/packs/sample/base/guide.md", "# Local edits\n")
				if kind == "edited retired document" {
					guidanceConfig(t, root, source, "[]", true)
				}
			}
			before := guidanceFiles(t, root)
			example := filepath.Join(root, config.ExampleFileName)
			if err := os.Remove(example); err != nil {
				t.Fatal(err)
			}
			output := guidanceUpdate(t, root)
			if !strings.Contains(output, ".guidance (project engineering guidance)") || !strings.Contains(output, "guidance incomplete:") {
				t.Fatalf("missing guidance failure diagnostic:\n%s", output)
			}
			if diff := cmp.Diff(before, guidanceFiles(t, root)); diff != "" {
				t.Fatalf("guidance changed on failure (-before +after):\n%s", diff)
			}
			if data, err := os.ReadFile(example); err != nil || len(data) == 0 {
				t.Fatalf("unrelated example refresh did not proceed: %v", err)
			}
		})
	}
}

func TestRun_ProjectGuidanceDisabledPreservesInstallationWithoutRetrieval(t *testing.T) {
	testsupport.IsolateGuidanceEnvironment(t)
	root, source := freshInitializedRepo(t), testsupport.GuidanceSource(t)
	trace := filepath.Join(t.TempDir(), "git-trace")
	t.Setenv("GIT_TRACE", trace)
	guidanceConfig(t, root, source, "[sample/base]", true)
	guidanceInstallBaseline(t, root, source)
	events, err := os.ReadFile(trace)
	if err != nil || !bytes.Contains(events, []byte("git clone")) {
		t.Fatalf("trace did not observe enabled retrieval: %s, %v", events, err)
	}
	before := guidanceFiles(t, root)
	guidanceConfig(t, root, filepath.Join(root, "unavailable"), "[]", false)
	guidanceWrite(t, filepath.Dir(trace), filepath.Base(trace), "")
	output := guidanceUpdate(t, root)
	events, err = os.ReadFile(trace)
	if err != nil || bytes.Contains(events, []byte("git clone")) {
		t.Fatalf("disabled maintenance attempted retrieval: %s, %v", events, err)
	}
	if strings.Contains(output, "guidance incomplete") {
		t.Fatalf("disabled maintenance reported a retrieval failure: %s", output)
	}
	if diff := cmp.Diff(before, guidanceFiles(t, root)); diff != "" {
		t.Fatalf("disabled maintenance changed guidance:\n%s", diff)
	}
}

func TestRun_ProjectGuidanceRemovalAndInterruptedRetry(t *testing.T) {
	testsupport.IsolateGuidanceEnvironment(t)
	for _, interrupted := range []bool{false, true} {
		name := "normal removal"
		if interrupted {
			name = "interrupted removal"
		}
		t.Run(name, func(t *testing.T) {
			if interrupted && os.Geteuid() == 0 {
				t.Skip("permission denial requires non-root")
			}
			root, source := freshInitializedRepo(t), testsupport.GuidanceSource(t)
			guidanceConfig(t, root, source, "[sample/base]", true)
			revision := guidanceInstallBaseline(t, root, source)
			guidanceWrite(t, root, ".guidance/project.md", "# Project overrides\n")
			guidanceWrite(t, root, ".guidance/packs/handwritten.md", "# Handwritten\n")
			before := guidanceFiles(t, root)
			guidanceConfig(t, root, source, "[]", true)
			parent := filepath.Join(root, ".guidance", "packs", "sample", "base")
			if interrupted {
				if err := os.Chmod(parent, 0o555); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })
				output := guidanceUpdate(t, root)
				if !strings.Contains(output, "rerun aiwf update to finish") {
					t.Fatalf("missing recovery instruction: %s", output)
				}
				partial := guidanceFiles(t, root)
				if _, ok := partial[".guidance/.aiwf-pending"]; !ok {
					t.Fatal("interrupted installation has no recovery record")
				}
				delete(partial, ".guidance/.aiwf-pending")
				if diff := cmp.Diff(before, partial); diff != "" {
					t.Fatalf("failed removal changed installed selection:\n%s", diff)
				}
				if err := os.Chmod(parent, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			output := guidanceUpdate(t, root)
			after := guidanceFiles(t, root)
			if strings.Contains(output, "guidance incomplete") {
				t.Fatalf("removal failed: %s", output)
			}
			for _, path := range []string{".guidance/packs/sample/base/guide.md", ".guidance/.aiwf-pending"} {
				if _, exists := after[path]; exists {
					t.Errorf("retired file remains: %s", path)
				}
			}
			for _, path := range []string{".guidance/project.md", ".guidance/packs/handwritten.md", "CLAUDE.md", "AGENTS.md"} {
				if after[path] != before[path] {
					t.Errorf("removal changed %s", path)
				}
			}
			if !strings.Contains(after[".guidance/index.md"], "Installed commit: `"+revision+"`") || strings.Contains(after[".guidance/index.md"], "sample/base") {
				t.Fatal("index does not reflect empty selection")
			}
			guidanceUpdate(t, root)
			if diff := cmp.Diff(after, guidanceFiles(t, root)); diff != "" {
				t.Fatalf("repeat removal changed guidance:\n%s", diff)
			}
		})
	}
}

// TestRun_GuidanceReportNamesBlockerAndFix pins that a blocked installation
// ends update with the blocker's cause and the action that clears it, keeps the
// installed set as it was, and leaves the exit code and other steps unchanged.
func TestRun_GuidanceReportNamesBlockerAndFix(t *testing.T) {
	for _, kind := range []string{"retrieval failure", "incompatible delivery", "handwritten legacy import", "locally edited file"} {
		for _, refresh := range []bool{false, true} {
			name := kind + "/first adoption"
			if refresh {
				name = kind + "/refresh"
			}
			t.Run(name, func(t *testing.T) {
				bin := testsupport.IsolateGuidanceEnvironment(t)
				root, source := freshInitializedRepo(t), testsupport.GuidanceSource(t, "*.xyz")
				guidanceWrite(t, root, "source.xyz", "project source")
				guidanceWrite(t, root, config.FileName, "hosts: [claude-code, codex]\nguidance:\n  source: "+source+"\n")
				installed := "  installed: none"
				if refresh {
					revision := guidanceInstallBaseline(t, root, source)
					installed = "  installed: sample/base at " + revision
				}
				var want string
				switch kind {
				case "retrieval failure":
					absent := filepath.Join(root, "absent-source")
					configuration := "hosts: [claude-code, codex]\nguidance:\n  source: " + absent + "\n"
					if refresh {
						configuration += "  packs: [sample/base]\n"
					}
					guidanceWrite(t, root, config.FileName, configuration)
					want = "  blocked:   fetching guidance from \"" + absent + "\"; check the source and Git access: "
				case "incompatible delivery":
					if err := testsupport.WriteExecutable(filepath.Join(bin, "dotfiles-guidance-check"), []byte("#!/bin/sh\nprintf 'reconcile old Claude routing\\n' >&2\nexit 1\n")); err != nil {
						t.Fatal(err)
					}
					want = "  blocked:   incompatible ai-dotfiles guidance delivery: dotfiles-guidance-check failed: exit status 1; reconcile old Claude routing; update/reconcile ai-dotfiles in this environment, then retry"
				case "handwritten legacy import":
					original, err := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
					if err != nil {
						t.Fatal(err)
					}
					guidanceWrite(t, root, "CLAUDE.md", string(original)+"\n@~/.agents/guidance/200-go.md\n")
					want = "  blocked:   project guidance installation conflict: CLAUDE.md contains handwritten legacy guidance references; reconcile them before retrying"
				case "locally edited file":
					guidanceWrite(t, root, ".guidance/packs/sample/base/guide.md", "# Local edits\n")
					want = "  blocked:   project guidance installation conflict: .guidance/packs/sample/base/guide.md is foreign or locally edited; preserve your changes in project.md or reconcile the generated file before retrying"
				}
				example := filepath.Join(root, config.ExampleFileName)
				if err := os.Remove(example); err != nil {
					t.Fatal(err)
				}
				section := guidanceSection(t, guidanceUpdate(t, root))
				for _, line := range section {
					if !strings.HasPrefix(line, "  ") {
						t.Fatalf("section line %q is not one of the report's labelled lines:\n%s", line, strings.Join(section, "\n"))
					}
				}
				if !slices.Contains(section, installed) {
					t.Fatalf("installed set not reported as %q:\n%s", installed, strings.Join(section, "\n"))
				}
				blocked := slices.IndexFunc(section, func(line string) bool { return strings.HasPrefix(line, "  blocked:") })
				if blocked < 0 || !strings.HasPrefix(section[blocked], want) || (kind != "retrieval failure" && section[blocked] != want) {
					t.Fatalf("blocked line, want %q:\n%s", want, strings.Join(section, "\n"))
				}
				if slices.Contains(section, "  adopted:   sample/base") {
					t.Fatalf("blocked run reported an adoption:\n%s", strings.Join(section, "\n"))
				}
				if _, err := os.Stat(example); err != nil {
					t.Fatalf("unrelated update step did not run: %v", err)
				}
			})
		}
	}
}

// TestRun_LaterFailureStillReportsGuidance makes the hook step fail after
// guidance ran: update exits with an internal error and still prints the
// guidance section. Disabled maintenance keeps the run hermetic.
func TestRun_LaterFailureStillReportsGuidance(t *testing.T) {
	root := freshInitializedRepo(t)
	guidanceWrite(t, root, config.FileName, "hosts: [codex]\nguidance:\n  enabled: false\n")
	hooks := filepath.Join(root, ".git", "hooks")
	if err := os.RemoveAll(hooks); err != nil {
		t.Fatal(err)
	}
	guidanceWrite(t, root, filepath.Join(".git", "hooks"), "not a directory")
	rc := cliutil.ExitOK
	output := string(testutil.CaptureStdout(t, func() {
		rc = update.Run(root, false, "", false, false, false, false, true, nil, nil)
	}))
	if rc != cliutil.ExitInternal || !strings.Contains(output, "\nGuidance:\n") {
		t.Fatalf("rc = %d, want ExitInternal with the guidance section:\n%s", rc, output)
	}
}

// TestRun_HookSyncFailureStillReportsGuidance enables a Claude hook while
// .claude/settings.json is malformed, so wiring the hook fails after guidance
// ran: update exits with an internal error and still prints the guidance
// section. Disabled maintenance keeps the run hermetic.
func TestRun_HookSyncFailureStillReportsGuidance(t *testing.T) {
	root := freshInitializedRepo(t)
	hook := skills.ShippedHooks[0].Name
	guidanceWrite(t, root, config.FileName, "hosts: [claude-code]\nguidance:\n  enabled: false\nhooks:\n  "+hook+":\n    enabled: true\n")
	guidanceWrite(t, root, filepath.Join(".claude", "settings.json"), "{not json")
	rc := cliutil.ExitOK
	output := string(testutil.CaptureStdout(t, func() {
		rc = update.Run(root, false, "", false, false, false, false, true, nil, skills.ShippedHooks)
	}))
	if rc != cliutil.ExitInternal || !strings.Contains(output, "\nGuidance:\n") {
		t.Fatalf("rc = %d, want ExitInternal with the guidance section:\n%s", rc, output)
	}
}

// TestRun_HookConsentFailureStillReportsGuidance gives aiwf.yaml a hooks block
// the consent gate cannot decode: update exits with an internal error rather
// than success and still prints the guidance section.
func TestRun_HookConsentFailureStillReportsGuidance(t *testing.T) {
	root := freshInitializedRepo(t)
	guidanceWrite(t, root, config.FileName, "hosts: [claude-code]\nguidance:\n  enabled: false\nhooks:\n  bad-hook:\n    unknown_field: true\n")
	rc := cliutil.ExitOK
	output := string(testutil.CaptureStdout(t, func() {
		rc = update.Run(root, false, "", false, false, false, false, true, nil, skills.ShippedHooks)
	}))
	if rc != cliutil.ExitInternal || !strings.Contains(output, "\nGuidance:\n") {
		t.Fatalf("rc = %d, want ExitInternal with the guidance section:\n%s", rc, output)
	}
}
