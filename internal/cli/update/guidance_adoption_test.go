package update_test

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

// adoptionSource publishes a catalogue of two packs, each detected by its own
// filename patterns, so a test can tell an adopted pack from a skipped one.
func adoptionSource(t *testing.T, sampleDetect, otherDetect string) string {
	t.Helper()
	source := testsupport.GuidanceSource(t)
	guidanceWrite(t, source, "catalogue.json", `{"packs":[`+
		`{"id":"sample/base","description":"Sample project guidance","files":["packs/sample/base/guide.md"],"detect":["`+sampleDetect+`"]},`+
		`{"id":"other/pack","description":"Other project guidance","files":["packs/other/pack/guide.md"],"detect":["`+otherDetect+`"]}]}`)
	guidanceWrite(t, source, "packs/other/pack/guide.md", "# Other guidance\n")
	testsupport.CommitGuidanceSource(t, source)
	return source
}

// runUpdateBinary runs the built binary with stdin detached from any terminal.
func runUpdateBinary(t *testing.T, root string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), testutil.AiwfBinary(t), "update", "--root", root)
	cmd.Env = append(os.Environ(), testsupport.GuidanceEnvironment(t)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("update: %v\n%s", err, output)
	}
	return string(output)
}

func guidanceSelection(t *testing.T, root string) *[]string {
	t.Helper()
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return cfg.Guidance.Packs
}

func TestBinary_UnsetGuidanceSelectionAdoptsApplicablePacksWithoutTerminal(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		sampleDetect string
		files        map[string]string
		want         []string
	}{
		{
			name:         "adopts the matching pack and skips the other",
			sampleDetect: "*.xyz",
			files:        map[string]string{"source.xyz": "project source"},
			want:         []string{"sample/base"},
		},
		{
			name:         "adopts a pack matching every tracked file",
			sampleDetect: "*",
			files:        map[string]string{"notes.txt": "notes"},
			want:         []string{"sample/base"},
		},
		{
			name:         "adopts through a nested file but not a gitignored one",
			sampleDetect: "*.xyz",
			files:        map[string]string{"src/deep/source.xyz": "nested", ".gitignore": "*.ign\n", "skipped.ign": "ignored"},
			want:         []string{"sample/base"},
		},
		{
			name:         "writes no selection when nothing matches",
			sampleDetect: "*.xyz",
			files:        map[string]string{"notes.txt": "notes"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root, source := t.TempDir(), adoptionSource(t, tc.sampleDetect, "*.ign")
			if err := gitops.Init(t.Context(), root); err != nil {
				t.Fatal(err)
			}
			guidanceWrite(t, root, config.FileName, "hosts: [claude-code, codex]\nguidance:\n  source: "+source+"\n")
			for name, content := range tc.files {
				guidanceWrite(t, root, name, content)
			}
			if out := runUpdateBinary(t, root); strings.Contains(out, "guidance incomplete") {
				t.Fatal(out)
			}
			selection := guidanceSelection(t, root)
			installed := guidanceFiles(t, root)
			if tc.want == nil {
				if selection != nil {
					t.Fatalf("selection written with no applicable pack: %v", *selection)
				}
				for name := range installed {
					if strings.HasPrefix(name, ".guidance/") {
						t.Fatalf("guidance installed with no applicable pack: %s", name)
					}
				}
				return
			}
			if selection == nil {
				t.Fatal("no guidance.packs written")
			}
			if diff := cmp.Diff(tc.want, *selection); diff != "" {
				t.Fatalf("guidance.packs (-want +got):\n%s", diff)
			}
			if installed[".guidance/packs/sample/base/guide.md"] == "" {
				t.Fatalf("adopted pack not installed: %v", installed)
			}
			if _, present := installed[".guidance/packs/other/pack/guide.md"]; present {
				t.Fatal("inapplicable pack installed")
			}
			for _, host := range []string{"CLAUDE.md", "AGENTS.md"} {
				routed, err := os.ReadFile(filepath.Join(root, host))
				if err != nil || !strings.Contains(string(routed), "[.guidance/index.md](.guidance/index.md)") {
					t.Fatalf("%s missing guidance route: %v", host, err)
				}
			}
		})
	}
}
