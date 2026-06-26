package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// setupRequiredAreaRepo initializes a repo with a declared areas block
// (members: platform, billing) AND `required: true`. Returns the repo
// root. Mirrors setupAreaRepo (add_area_test.go) with the required knob on.
func setupRequiredAreaRepo(t *testing.T) string {
	t.Helper()
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	patched := string(raw) + "areas:\n  required: true\n  members:\n    - platform\n    - billing\n"
	if err := os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
	return root
}

// TestCheck_AreaRequiredExitsNonZero pins M-0178/AC-2 (end-to-end seam):
// `aiwf check` on a required:true fixture with an untagged entity exits
// non-zero (ExitFindings) and surfaces the area-required code. Catches the
// bug class where check.AreaRequired exists and is unit-tested but the CLI
// Run forgets to compose it (or passes the wrong config field).
func TestCheck_AreaRequiredExitsNonZero(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	// Create the untagged gap BEFORE areas.required is set — otherwise the
	// add dispatcher would refuse the untagged create (AC-5).
	mustRun(t, "add", "gap", "--title", "Leak", "--actor", "human/test", "--root", root)

	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	patched := string(raw) + "areas:\n  required: true\n  members:\n    - platform\n    - billing\n"
	if err := os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}

	rc, stdout, _ := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"check", "--root", root})
	})
	if rc != cliutil.ExitFindings {
		t.Errorf("rc = %d, want ExitFindings (%d)", rc, cliutil.ExitFindings)
	}
	if !strings.Contains(stdout, "area-required") {
		t.Errorf("expected area-required in check output; got:\n%s", stdout)
	}
}

// TestAdd_RefusesUntaggedWhenRequired pins M-0178/AC-5: under required:true,
// an untagged `add epic` refuses (no entity written, message names --area);
// `add epic --area <member>` succeeds; a milestone may be added untagged
// (its area derives from the parent epic); and a gap whose --discovered-in
// derives a non-empty area is NOT refused (the exemption — derivation runs
// before the guard). Dropping the refusal lets the untagged-add case write an
// entity; moving the guard above the derivation reddens the discovered-in case.
func TestAdd_RefusesUntaggedWhenRequired(t *testing.T) {
	root := setupRequiredAreaRepo(t)

	// 1. Untagged add epic is refused: ExitUsage, names --area, writes nothing.
	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"add", "epic", "--title", "Untagged", "--actor", "human/test", "--root", root})
	})
	if rc != cliutil.ExitUsage {
		t.Errorf("untagged add: rc = %d, want ExitUsage (%d)", rc, cliutil.ExitUsage)
	}
	if !strings.Contains(stderr, "--area") {
		t.Errorf("untagged add: stderr %q should name --area", stderr)
	}
	if matches, _ := filepath.Glob(filepath.Join(root, "work", "epics", "E-*", "epic.md")); len(matches) != 0 {
		t.Fatalf("untagged add should create nothing; found %v", matches)
	}

	// 2. Tagged add epic succeeds.
	mustRun(t, "add", "epic", "--title", "Tagged", "--area", "platform", "--actor", "human/test", "--root", root)

	// 3. A milestone may be added untagged (area derives from the parent epic).
	mustRun(t, "add", "milestone", "--epic", "E-0001", "--tdd", "none", "--title", "Child", "--actor", "human/test", "--root", root)

	// 4. A gap whose --discovered-in derives a non-empty area is exempt: the
	//    derivation (resolvedArea = ResolvedAreaByID(discoveredIn)) runs before
	//    the refusal guard, so a gap discovered in the tagged E-0001 resolves to
	//    "platform" and is NOT refused. Guards against a refactor that moves the
	//    guard above the derivation.
	mustRun(t, "add", "gap", "--title", "Found in tagged epic", "--discovered-in", "E-0001", "--actor", "human/test", "--root", root)
	if matches, _ := filepath.Glob(filepath.Join(root, "work", "gaps", "G-*.md")); len(matches) != 1 {
		t.Fatalf("discovered-in gap should be created (exempt); found %v", matches)
	}
}

// TestAdd_UntaggedAllowedWhenRequiredOff pins the parity half of AC-5:
// with required off (members declared but `required` absent → false), an
// untagged `add epic` succeeds — byte-for-byte the pre-knob behavior.
func TestAdd_UntaggedAllowedWhenRequiredOff(t *testing.T) {
	root := setupAreaRepo(t) // members declared, required not set → false
	mustRun(t, "add", "epic", "--title", "Untagged", "--actor", "human/test", "--root", root)
}

// TestCheck_AreaUnknownErrorsUnderRequired pins M-0178/AC-7 (end-to-end
// seam): a gap carrying a present-but-undeclared (typo'd) area under
// `areas.required: true` makes `aiwf check` exit ExitFindings with
// area-unknown escalated to error; with `required` off the same tree
// exits ExitOK (area-unknown stays a warning). Catches the bug class
// where check.ApplyAreaRequiredStrict exists and is unit-tested but the
// CLI Run forgets to compose it (or passes the wrong config field).
func TestCheck_AreaUnknownErrorsUnderRequired(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--title", "Leak", "--actor", "human/test", "--root", root)

	// Hand-edit the gap to carry an undeclared area (a typo of "platform").
	// Non-empty area, so area-required never fires — this isolates the
	// area-unknown escalation.
	matches, err := filepath.Glob(filepath.Join(root, "work", "gaps", "G-0001-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("locate gap file: matches=%v err=%v", matches, err)
	}
	gapRaw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read gap: %v", err)
	}
	gapPatched := strings.Replace(string(gapRaw), "status: open\n", "status: open\narea: platfrm\n", 1)
	if gapPatched == string(gapRaw) {
		t.Fatalf("failed to inject area into gap frontmatter:\n%s", gapRaw)
	}
	if err = os.WriteFile(matches[0], []byte(gapPatched), 0o644); err != nil {
		t.Fatalf("write gap: %v", err)
	}

	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}

	// 1. required OFF (members declared, `required` absent → false): the
	//    typo'd area fires area-unknown at warning. check exits ExitOK.
	off := string(raw) + "areas:\n  members:\n    - platform\n    - billing\n"
	if err = os.WriteFile(yamlPath, []byte(off), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml (off): %v", err)
	}
	rc, stdout, _ := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"check", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Errorf("required off: rc = %d, want ExitOK (%d) — area-unknown must stay a warning\n%s",
			rc, cliutil.ExitOK, stdout)
	}
	if !strings.Contains(stdout, "area-unknown") {
		t.Errorf("required off: expected area-unknown in output; got:\n%s", stdout)
	}

	// 2. required ON: the same typo'd area escalates to error. check exits
	//    ExitFindings.
	on := string(raw) + "areas:\n  required: true\n  members:\n    - platform\n    - billing\n"
	if err = os.WriteFile(yamlPath, []byte(on), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml (on): %v", err)
	}
	rc, stdout, _ = testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"check", "--root", root})
	})
	if rc != cliutil.ExitFindings {
		t.Errorf("required on: rc = %d, want ExitFindings (%d) — area-unknown must escalate to error\n%s",
			rc, cliutil.ExitFindings, stdout)
	}
	if !strings.Contains(stdout, "area-unknown") {
		t.Errorf("required on: expected area-unknown in output; got:\n%s", stdout)
	}
}
