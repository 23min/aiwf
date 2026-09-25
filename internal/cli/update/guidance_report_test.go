package update_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/gitops"
)

// guidanceSection returns the lines of the guidance section, which must be the
// last thing update prints.
func guidanceSection(t *testing.T, output string) []string {
	t.Helper()
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i] == "Guidance:" {
			return lines[i+1:]
		}
	}
	t.Fatalf("output has no guidance section:\n%s", output)
	return nil
}

func sourceCommit(t *testing.T, source string) string {
	t.Helper()
	out, err := exec.CommandContext(t.Context(), "git", "-C", source, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func TestBinary_UpdateEndsWithGuidanceSection(t *testing.T) {
	t.Parallel()
	root, source := t.TempDir(), adoptionSource(t, "*.xyz", "*.abc")
	if err := gitops.Init(t.Context(), root); err != nil {
		t.Fatal(err)
	}
	commit := sourceCommit(t, source)
	configuration := "hosts: [codex]\nguidance:\n  source: " + source + "\n  ignored: [other/pack]\n"
	guidanceWrite(t, root, config.FileName, configuration)
	guidanceWrite(t, root, "notes.txt", "notes")

	steps := []struct {
		name   string
		before func()
		want   []string
	}{
		{
			name: "no applicable pack",
			want: []string{
				"  selected:  not adopted (guidance.packs unset)",
				"  adopted:   none",
				"  installed: none",
				"  ignored:   other/pack",
			},
		},
		{
			name:   "first adoption",
			before: func() { guidanceWrite(t, root, "source.xyz", "project source") },
			want: []string{
				"  selected:  sample/base",
				"  adopted:   sample/base",
				"  installed: sample/base at " + commit,
				"  ignored:   other/pack",
			},
		},
		{
			name: "nothing new to adopt",
			want: []string{
				"  selected:  sample/base",
				"  adopted:   none",
				"  installed: sample/base at " + commit,
				"  ignored:   other/pack",
			},
		},
		{
			name: "maintenance disabled",
			before: func() {
				guidanceWrite(t, root, config.FileName, "hosts: [codex]\nguidance:\n  enabled: false\n  source: "+source+"\n  packs: [sample/base]\n")
				guidanceWrite(t, root, "new.abc", "matches other/pack")
			},
			want: []string{
				"  selected:  sample/base",
				"  maintenance disabled; installed policy remains in use",
				"  adopted:   none",
				"  installed: sample/base at " + commit,
				"  ignored:   none",
			},
		},
	}
	// The steps share one repository, so they run in order.
	for _, step := range steps {
		if step.before != nil {
			step.before()
		}
		got := guidanceSection(t, runUpdateBinary(t, root))
		if diff := cmp.Diff(step.want, got); diff != "" {
			t.Fatalf("%s: guidance section (-want +got):\n%s", step.name, diff)
		}
	}
}

func TestBinary_UpdateReportsSelectedPacksTheCatalogueNoLongerHas(t *testing.T) {
	t.Parallel()
	root, source := t.TempDir(), adoptionSource(t, "*.xyz", "*.abc")
	if err := gitops.Init(t.Context(), root); err != nil {
		t.Fatal(err)
	}
	// Claude hosts end update through the hook and statusline steps.
	guidanceWrite(t, root, config.FileName, "hosts: [claude-code]\nguidance:\n  source: "+source+"\n  packs: [sample/base, gone/pack]\n")
	guidanceWrite(t, root, "source.xyz", "project source")
	got := guidanceSection(t, runUpdateBinary(t, root))
	want := "  missing:   gone/pack; the catalogue no longer lists it: remove it from guidance.packs or move it to guidance.ignored"
	if !cmp.Equal(got[len(got)-1], want) {
		t.Fatalf("last guidance line = %q, want %q\n%s", got[len(got)-1], want, strings.Join(got, "\n"))
	}
}
