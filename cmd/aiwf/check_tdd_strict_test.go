package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
)

// TestCheck_TDDStrict_EscalatesEntityBodyEmpty pins M-066/AC-2's
// dispatcher seam: the same fixture tree (one epic with the
// scaffolded empty `## Goal`/`## Scope`/`## Out of scope` sections)
// produces a clean exit when `aiwf.yaml: tdd.strict: true` is
// absent (warnings, exit 0) and a findings exit when it is set
// (errors, exit cliutil.ExitFindings). The unit test on
// check.ApplyTDDStrict already covers the bumper logic in
// isolation; this test exercises the seam where main.go's
// dispatcher reads config and applies the bumper, so a future
// refactor can't silently drop the wiring.
func TestCheck_TDDStrict_EscalatesEntityBodyEmpty(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	// `aiwf add epic` scaffolds an epic body with bare `## Goal`,
	// `## Scope`, `## Out of scope` headings — exactly the shape
	// the entity-body-empty rule fires on.
	if rc := run([]string{"add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("add epic: %d", rc)
	}

	// Without tdd.strict (the aiwf.yaml init writes is the
	// commented-header default — no tdd block at all). The rule
	// fires at warning severity; check exits 0.
	if rc := run([]string{"check", "--root", root}); rc != cliutil.ExitOK {
		t.Errorf("check without tdd.strict = %d, want cliutil.ExitOK (%d) — warnings should not block",
			rc, cliutil.ExitOK)
	}

	// Append tdd.strict: true to aiwf.yaml. Append rather than
	// overwrite so the comment header (and any other content init
	// landed) is preserved — that's how a real consumer would
	// edit the file.
	cfgPath := filepath.Join(root, "aiwf.yaml")
	current, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	if err := os.WriteFile(cfgPath, append(current, []byte("\ntdd:\n  strict: true\n")...), 0o644); err != nil {
		t.Fatalf("rewrite aiwf.yaml: %v", err)
	}

	// Same tree, same scaffolded entity, but tdd.strict is now
	// true. The bumper escalates entity-body-empty findings to
	// error severity; check exits with cliutil.ExitFindings.
	if rc := run([]string{"check", "--root", root}); rc != cliutil.ExitFindings {
		t.Errorf("check with tdd.strict = %d, want cliutil.ExitFindings (%d) — strict must escalate the rule to error",
			rc, cliutil.ExitFindings)
	}
}
