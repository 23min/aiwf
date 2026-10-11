package policies

import (
	"strings"
	"testing"
)

// TestPolicy_GuidanceImportsInRepository holds the live tree to M-0335
// AC-1: neither host entry point imports a file from outside the
// repository.
func TestPolicy_GuidanceImportsInRepository(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyGuidanceImportsInRepository)
}

// TestOutsideImports pins the rule (M-0335 AC-1): an import under the home
// directory or by absolute path is reported against its entry point, naming
// the import, and an import of a repository file passes.
func TestOutsideImports(t *testing.T) {
	t.Parallel()
	vs := outsideImports(map[string]string{
		"CLAUDE.md": "@.claude/aiwf-guidance.md\n@~/.agents/guidance/go.md\n",
		"AGENTS.md": "See @/etc/global.md for more.\n",
	})
	want := map[string]string{"AGENTS.md": "@/etc/global.md", "CLAUDE.md": "@~/.agents/guidance/go.md"}
	if len(vs) != len(want) {
		t.Fatalf("violations = %+v, want one per entry point", vs)
	}
	for _, v := range vs {
		if v.Policy != "guidance-imports-in-repository" || !strings.Contains(v.Detail, want[v.File]) {
			t.Errorf("violation %+v, want guidance-imports-in-repository naming %s", v, want[v.File])
		}
	}
}
