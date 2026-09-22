package update_test

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/update"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/gitops"
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
		if rc := update.Run(root, false, "", false, false, false, false, nil, nil); rc != cliutil.ExitOK {
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
	for _, kind := range []string{"unavailable source", "invalid catalogue", "missing pack", "missing document", "edited document", "edited retired document", "first installation"} {
		t.Run(kind, func(t *testing.T) {
			root, source := freshInitializedRepo(t), testsupport.GuidanceSource(t)
			guidanceConfig(t, root, source, "[sample/base]", true)
			if kind != "first installation" {
				guidanceInstallBaseline(t, root, source)
			}
			switch kind {
			case "unavailable source", "first installation":
				guidanceConfig(t, root, filepath.Join(root, "absent-source"), "[sample/base]", true)
			case "invalid catalogue":
				guidanceWrite(t, source, "catalogue.json", "not JSON\n")
				testsupport.CommitGuidanceSource(t, source)
			case "missing pack":
				guidanceConfig(t, root, source, "[missing/base]", true)
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
