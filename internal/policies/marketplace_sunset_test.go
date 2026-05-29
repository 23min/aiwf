package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mdSection returns the body of the `## <heading>` section of a markdown
// document — from just after the heading line up to (but not including)
// the next level-2 (`## `) heading. Level-3 (`### `) subsections are
// included. The bool is false when the heading is absent. Used to scope
// assertions to a named section rather than grepping flat over the whole
// file (CLAUDE.md § "Substring assertions are not structural assertions").
func mdSection(md, heading string) (string, bool) {
	lines := strings.Split(md, "\n")
	start := -1
	for i, l := range lines {
		if l == "## "+heading {
			start = i + 1
			break
		}
	}
	if start == -1 {
		return "", false
	}
	var b strings.Builder
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			break
		}
		b.WriteString(lines[i])
		b.WriteByte('\n')
	}
	return b.String(), true
}

// TestM0152_OperatorSetupDocRewritten covers M-0152 AC-3: the CLAUDE.md
// "Operator setup" section describes the one-command embed-and-
// materialize flow and no longer the retired marketplace `/plugin`
// install. The assertion is scoped to the named section.
func TestM0152_OperatorSetupDocRewritten(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	section, ok := mdSection(string(raw), "Operator setup")
	if !ok {
		t.Fatal("CLAUDE.md has no `## Operator setup` section")
	}
	for _, want := range []string{"aiwf init", "aiwf update", "materialize"} {
		if !strings.Contains(section, want) {
			t.Errorf("Operator setup section should describe the one-command flow (missing %q):\n%s", want, section)
		}
	}
	for _, forbidden := range []string{
		"/plugin marketplace add",
		"recommended_plugins",
		"recommended-plugin-not-installed",
	} {
		if strings.Contains(section, forbidden) {
			t.Errorf("Operator setup section still references the retired marketplace flow (%q):\n%s", forbidden, section)
		}
	}
}

// TestM0152_DefaultYamlDropsRecommendedPlugins covers M-0152 AC-3: this
// repo's own aiwf.yaml no longer declares doctor.recommended_plugins
// (the marketplace recommendation is retired, D-0016).
func TestM0152_DefaultYamlDropsRecommendedPlugins(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "aiwf.yaml"))
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	for _, forbidden := range []string{"recommended_plugins", "doctor:"} {
		if strings.Contains(string(raw), forbidden) {
			t.Errorf("aiwf.yaml still carries %q after marketplace retirement:\n%s", forbidden, raw)
		}
	}
}
