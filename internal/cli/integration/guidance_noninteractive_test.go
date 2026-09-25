package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/projectguidance"
	"github.com/23min/aiwf/internal/testsupport"
)

func TestBinary_GuidanceNoninteractiveAdoptsOnlyUnsetSelection(t *testing.T) {
	t.Parallel()
	bin := testutil.AiwfBinary(t)
	for _, verb := range []string{"init", "update"} {
		for _, policy := range []string{"unselected", "ignored", "selected", "disabled"} {
			t.Run(verb+"/"+policy, func(t *testing.T) {
				t.Parallel()
				root := t.TempDir()
				if err := gitops.Init(t.Context(), root); err != nil {
					t.Fatal(err)
				}
				source := testsupport.GuidanceSource(t, "*.xyz")
				configuration := "hosts: []\nguidance:\n  source: " + source + "\n"
				switch policy {
				case "selected":
					configuration += "  packs: [sample/base]\n"
				case "ignored":
					configuration += "  ignored: [sample/base]\n"
				case "disabled":
					configuration = "hosts: []\nguidance:\n  enabled: false\n  source: " + filepath.Join(root, "unavailable-source") + "\n"
				}
				write := func(root, name, content string) {
					t.Helper()
					if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				write(root, config.FileName, configuration)
				// Explicit selection must install even before any project source exists.
				if policy != "selected" {
					write(root, "source.xyz", "project source")
				}
				environment := append(os.Environ(), testsupport.GuidanceEnvironment(t)...)
				run := func(verb string) string {
					t.Helper()
					args := []string{verb, "--root", root}
					if verb == "init" {
						args = append(args, "--skip-hook")
					}
					cmd := exec.CommandContext(t.Context(), bin, args...)
					cmd.Stdin = strings.NewReader("i\n") // A pipe is never read as a guidance answer.
					cmd.Env = environment
					output, err := cmd.CombinedOutput()
					if err != nil {
						t.Fatalf("%s: %v\n%s", verb, err, output)
					}
					return string(output)
				}
				out := run(verb)
				if strings.Contains(out, "guidance incomplete") {
					t.Fatalf("guidance failed: %s", out)
				}
				installed := filepath.Join(root, ".guidance", "packs", "sample", "base", "guide.md")
				if policy == "unselected" {
					cfg, err := config.Load(root)
					if err != nil {
						t.Fatal(err)
					}
					if cfg.Guidance.Packs == nil || strings.Join(*cfg.Guidance.Packs, ",") != "sample/base" {
						t.Fatalf("unset selection not adopted: %v", cfg.Guidance.Packs)
					}
					if _, err := os.Stat(installed); err != nil {
						t.Fatalf("adopted pack not installed: %v", err)
					}
					return
				}
				raw, err := os.ReadFile(filepath.Join(root, config.FileName))
				if err != nil || !bytes.Equal(raw, []byte(configuration)) {
					t.Fatalf("configuration changed: %s (%v)", raw, err)
				}
				if policy != "selected" {
					if _, err := os.Stat(installed); !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("adopted policy: %v", err)
					}
				} else {
					for _, content := range []string{"# Sample guidance\nInitial upstream content.\n", "# Revised guidance\n"} {
						if content == "# Revised guidance\n" {
							write(source, "packs/sample/base/guide.md", content)
							raw, err := os.ReadFile(filepath.Join(source, "catalogue.json"))
							if err != nil {
								t.Fatal(err)
							}
							var catalogue projectguidance.Catalogue
							if err = json.Unmarshal(raw, &catalogue); err != nil {
								t.Fatal(err)
							}
							catalogue.Packs = append(catalogue.Packs, projectguidance.Pack{ID: "sample/extra", Description: "Optional extra guidance", Files: []string{"packs/sample/extra/guide.md"}, Detect: []string{"*.abc"}})
							raw, err = json.Marshal(catalogue)
							if err != nil {
								t.Fatal(err)
							}
							write(source, "catalogue.json", string(raw))
							write(source, "packs/sample/extra/guide.md", "# Extra guidance\n")
							write(root, "new.abc", "new project source")
							testsupport.CommitGuidanceSource(t, source)
							out = run("update")
							if _, err = os.Stat(filepath.Join(root, ".guidance", "packs", "sample", "extra", "guide.md")); !errors.Is(err, os.ErrNotExist) {
								t.Fatalf("refresh adopted suggestion: %v", err)
							}
							gotConfig, err := os.ReadFile(filepath.Join(root, config.FileName))
							if err != nil || !bytes.Equal(gotConfig, []byte(configuration)) {
								t.Fatalf("refresh changed policy: %s (%v)", gotConfig, err)
							}
							if strings.Contains(out, "guidance incomplete") {
								t.Fatal(out)
							}
						}
						got, err := os.ReadFile(installed)
						if err != nil || string(got) != content {
							t.Fatalf("selection not refreshed: %s (%v)", got, err)
						}
					}
				}
			})
		}
	}
}
