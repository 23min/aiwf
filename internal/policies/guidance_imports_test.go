package policies

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/projectguidance"
)

// TestPolicy_GuidanceImportsInRepository holds the live tree to M-0335
// AC-1: neither host entry point imports a file from outside the
// repository.
func TestPolicy_GuidanceImportsInRepository(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyGuidanceImportsInRepository)
}

// TestOutsideImports pins the rule (M-0335 AC-1): an import under the home
// directory, by absolute path, or by a relative path that climbs out of the
// repository is reported against its entry point, naming the import, and an
// import of a repository file passes.
func TestOutsideImports(t *testing.T) {
	t.Parallel()
	vs := outsideImports(map[string]string{
		"CLAUDE.md": "@.claude/aiwf-guidance.md\n@~/.agents/guidance/go.md\n",
		"AGENTS.md": "See @/etc/global.md and @../outside/rules.md for more.\n",
	})
	want := map[string][]string{
		"AGENTS.md": {"@/etc/global.md", "@../outside/rules.md"},
		"CLAUDE.md": {"@~/.agents/guidance/go.md"},
	}
	got := map[string][]string{}
	for _, v := range vs {
		if v.Policy != "guidance-imports-in-repository" {
			t.Errorf("violation Policy = %q, want guidance-imports-in-repository", v.Policy)
		}
		got[v.File] = append(got[v.File], strings.Fields(v.Detail)[0])
	}
	for file, imports := range want {
		if strings.Join(got[file], " ") != strings.Join(imports, " ") {
			t.Errorf("%s: reported %v, want %v", file, got[file], imports)
		}
	}
	if len(got) != len(want) {
		t.Errorf("reported files %v, want only %v", got, want)
	}
}

// TestPolicyGuidanceImportsInRepository_ReadsManagedBlocks pins the
// policy's reading of the entry points (M-0335 AC-1): both files are read
// whole, so an outside import inside aiwf's managed block is reported as
// one in handwritten text is.
func TestPolicyGuidanceImportsInRepository_ReadsManagedBlocks(t *testing.T) {
	t.Parallel()
	start, end, _ := projectguidance.GuidanceMarkers()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, projectguidance.ClaudeFile), "# Repo\n\n"+start+"\n@/y.md\n"+end+"\n")
	mustWrite(t, filepath.Join(root, projectguidance.AgentsFile), "# Repo\n\n"+start+"\n@~/x.md\n"+end+"\n")
	vs, err := PolicyGuidanceImportsInRepository(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	var files []string
	for _, v := range vs {
		files = append(files, v.File)
	}
	if strings.Join(files, " ") != "AGENTS.md CLAUDE.md" {
		t.Errorf("reported %v, want one violation in each entry point's managed block", files)
	}
}
