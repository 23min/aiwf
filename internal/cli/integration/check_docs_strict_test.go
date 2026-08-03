package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
)

// TestCheck_DocsStrict_EscalatesBothDocRules pins the dispatcher seam for
// M-0289. The rules' own unit tests cover their logic, and a policy test
// asserts this repo's docs are clean — but both call the rule functions
// directly, so neither observes whether the CLI still invokes them.
//
// Three edits are all invisible without this: deleting either rule call, and
// moving check.ApplyDocsStrict above the two appends. That last one is the
// realistic failure — the surrounding code groups its Apply* calls together,
// so a tidy-up that joins them silently disables docs.strict for every
// consumer, with nothing to compile-fail or redden.
func TestCheck_DocsStrict_EscalatesBothDocRules(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := cli.Execute([]string{"add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("add epic: %d", rc)
	}

	// Append rather than overwrite, so init's comment header survives — the
	// shape a real consumer's file has.
	cfgPath := filepath.Join(root, "aiwf.yaml")
	base, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	docPath := filepath.Join(root, "GUIDE.md")
	seed := func(body, cfgTail string) {
		t.Helper()
		if err := os.WriteFile(docPath, []byte(body), 0o644); err != nil {
			t.Fatalf("seeding doc: %v", err)
		}
		if err := os.WriteFile(cfgPath, append(base, []byte(cfgTail)...), 0o644); err != nil {
			t.Fatalf("rewrite aiwf.yaml: %v", err)
		}
	}
	const advisory = "\ndocs:\n  paths:\n    - GUIDE.md\n"
	const strict = "\ndocs:\n  paths:\n    - GUIDE.md\n  strict: true\n"

	// Each rule gets a document only IT can fire on, so deleting either call
	// leaves its scenario passing at exit 0 rather than being masked by the
	// other rule's finding. `E-01` carries no slug, so the slug rule cannot
	// see it; the citation below is canonical-width, so the width rule cannot.
	widthOnly := "# Guide\n\nSee E-01 for the shape.\n"
	slugOnly := "# Guide\n\nSee work/epics/E-0001-not-the-real-slug/epic.md.\n"

	for _, tc := range []struct {
		name string
		body string
	}{
		{"doc-id-width", widthOnly},
		{"doc-id-slug", slugOnly},
	} {
		t.Run(tc.name, func(t *testing.T) {
			seed(tc.body, advisory)
			if rc := cli.Execute([]string{"check", "--root", root}); rc != cliutil.ExitOK {
				t.Errorf("check without docs.strict = %d, want %d — doc findings ship advisory",
					rc, cliutil.ExitOK)
			}
			seed(tc.body, strict)
			if rc := cli.Execute([]string{"check", "--root", root}); rc != cliutil.ExitFindings {
				t.Errorf("check with docs.strict = %d, want %d — this rule is not reaching the exit code",
					rc, cliutil.ExitFindings)
			}
		})
	}
}
