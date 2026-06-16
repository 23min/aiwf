package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
)

// TestRun_Guidance_InitWiresAndUpdateSelfHeals drives the full verb seam
// (cli.Execute → init/update.Run → initrepo → ensureGuidanceImport):
// `aiwf init` wires the CLAUDE.md guidance import, and `aiwf update`
// self-heals it after the operator removes it (M-0164, automagical model).
func TestRun_Guidance_InitWiresAndUpdateSelfHeals(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: rc=%d", rc)
	}
	claudePath := filepath.Join(root, "CLAUDE.md")
	if b, _ := os.ReadFile(claudePath); !strings.Contains(string(b), "@.claude/aiwf-guidance.md") {
		t.Fatalf("init did not wire the guidance import:\n%s", b)
	}

	// Operator removes the block entirely; `aiwf update` must self-heal it.
	if err := os.WriteFile(claudePath, []byte("# just my notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := cli.Execute([]string{"update", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("update: rc=%d", rc)
	}
	b, _ := os.ReadFile(claudePath)
	if !strings.Contains(string(b), "@.claude/aiwf-guidance.md") {
		t.Errorf("update did not self-heal the guidance import:\n%s", b)
	}
	if !strings.Contains(string(b), "# just my notes") {
		t.Errorf("update clobbered user content:\n%s", b)
	}
}

// TestRun_Guidance_ConfigOptOut: with guidance.wire_claudemd=false in
// aiwf.yaml, the verb path does not wire CLAUDE.md (M-0164 opt-out).
func TestRun_Guidance_ConfigOptOut(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("guidance:\n  wire_claudemd: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: rc=%d", rc)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "CLAUDE.md")); strings.Contains(string(b), "@.claude/aiwf-guidance.md") {
		t.Errorf("opt-out: CLAUDE.md should not be wired:\n%s", b)
	}
}
