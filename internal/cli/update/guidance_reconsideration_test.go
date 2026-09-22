package update_test

import (
	"bytes"
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

func TestBinary_GuidanceConfigurationRemovalAndReconsideration(t *testing.T) {
	t.Parallel()
	binary := testutil.AiwfBinary(t)
	for _, action := range []string{"ignore then reconsider", "remove selection only", "edited removal", "edited replacement"} {
		t.Run(action, func(t *testing.T) {
			t.Parallel()
			root, source := t.TempDir(), testsupport.GuidanceSource(t, "*.xyz")
			if err := gitops.Init(t.Context(), root); err != nil {
				t.Fatal(err)
			}
			environment := append(os.Environ(), testsupport.GuidanceEnvironment(t)...)
			configuration := ""
			configure := func(packs, ignored string) {
				t.Helper()
				configuration = "# Keep the project's notes.\nhosts: [claude-code, codex]\nhooks: {}\ncustom_setting: preserved\nguidance:\n  source: " + source + "\n  packs: " + packs + "\n  ignored: " + ignored + "\n"
				guidanceWrite(t, root, config.FileName, configuration)
			}
			update := func() string {
				t.Helper()
				cmd := exec.CommandContext(t.Context(), binary, "update", "--root", root)
				cmd.Env = environment
				output, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("update: %v\n%s", err, output)
				}
				raw, err := os.ReadFile(filepath.Join(root, config.FileName))
				if err != nil || !bytes.Equal(raw, []byte(configuration)) {
					t.Fatalf("update changed configuration: %s (%v)", raw, err)
				}
				return string(output)
			}
			configure("[sample/base]", "[]")
			guidanceWrite(t, root, "source.xyz", "project source")
			guidanceWrite(t, root, ".guidance/project.md", "# Project overrides\n")
			guidanceWrite(t, root, ".guidance/packs/handwritten.md", "# Unowned guidance\n")
			if out := update(); strings.Contains(out, "guidance incomplete") {
				t.Fatal(out)
			}
			const document = ".guidance/packs/sample/base/guide.md"
			baseline := guidanceFiles(t, root)
			if baseline[document] != "# Sample guidance\nInitial upstream content.\n" {
				t.Fatalf("baseline not installed: %q", baseline[document])
			}
			for _, host := range []string{"CLAUDE.md", "AGENTS.md"} {
				if !strings.Contains(baseline[host], "[.guidance/index.md](.guidance/index.md)") {
					t.Fatalf("%s missing guidance route", host)
				}
			}
			edited := strings.HasPrefix(action, "edited")
			if edited {
				guidanceWrite(t, root, document, "# Maintainer's edited guidance\n")
			}
			before := guidanceFiles(t, root)
			switch action {
			case "ignore then reconsider", "edited removal":
				configure("[]", "[sample/base]")
			case "remove selection only":
				configure("[]", "[]")
			case "edited replacement":
				guidanceWrite(t, source, "packs/sample/base/guide.md", "# Revised upstream guidance\n")
				testsupport.CommitGuidanceSource(t, source)
			}
			out := update()
			after := guidanceFiles(t, root)
			if edited {
				if !strings.Contains(out, "guidance incomplete") || !strings.Contains(out, "guide.md") {
					t.Fatalf("missing edited-file diagnostic: %s", out)
				}
				if diff := cmp.Diff(before, after); diff != "" {
					t.Fatalf("edited guidance was changed:\n%s", diff)
				}
				return
			}
			if strings.Contains(out, "guidance incomplete") {
				t.Fatal(out)
			}
			if _, exists := after[document]; exists {
				t.Fatal("removed pack still installed")
			}
			if strings.Contains(after[".guidance/index.md"], "sample/base") || strings.Contains(after[".guidance/.aiwf-owned"], document) {
				t.Fatal("retired pack still indexed or owned")
			}
			for _, name := range []string{".guidance/project.md", ".guidance/packs/handwritten.md", "CLAUDE.md", "AGENTS.md"} {
				if before[name] != after[name] {
					t.Fatalf("removal changed %s", name)
				}
			}
			suggested := strings.Contains(out, "Suggested guidance sample/base")
			if suggested != (action == "remove selection only") {
				t.Fatalf("unexpected suggestion state: %s", out)
			}
			if action == "ignore then reconsider" {
				// Removing an ignored id makes it eligible again, without reinstalling it.
				configure("[]", "[]")
				out = update()
				if !strings.Contains(out, "Suggested guidance sample/base") {
					t.Fatalf("ignored pack was not reconsidered: %s", out)
				}
			}
			if diff := cmp.Diff(after, guidanceFiles(t, root)); diff != "" {
				t.Fatalf("suggestion changed installed policy:\n%s", diff)
			}
			configure("[sample/base]", "[]")
			if out := update(); strings.Contains(out, "guidance incomplete") {
				t.Fatal(out)
			}
			if diff := cmp.Diff(baseline, guidanceFiles(t, root)); diff != "" {
				t.Fatalf("explicit re-selection did not restore policy:\n%s", diff)
			}
		})
	}
}
