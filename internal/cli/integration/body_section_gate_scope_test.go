package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
)

// body_section_gate_scope_test.go — M-0331/AC-2. The push-seam membership gate
// is scoped to entities whose body *content* the range changed. An entity
// touched only by a frontmatter write or a file move is outside it, so ordinary
// work on an entity that already omits a required section is not blocked by
// debt that work did not create.
//
// Driven through `cli.Execute` so the real verbs do the writing and the real
// `aiwf check` does the judging — the gate's scope is a property of what those
// two do to a tree, not of any one function.

// incompleteGap omits `## Why it matters`, which the gap kind requires. It is
// written by hand and committed with plain git, because `aiwf add` refuses to
// create it — that refusal is the write seam, and this file exists to stand for
// the bodies already committed before either seam landed.
const incompleteGap = `---
id: G-0001
title: Something is missing
status: open
---
## What's missing

A body section, deliberately.
`

// seedIncompleteEntity writes the gap, commits it with plain git, and returns
// the SHA — the range base, so the omission predates everything the gate judges.
func seedIncompleteEntity(t *testing.T, root string) string {
	t.Helper()
	rel := "work/gaps/G-0001-something-is-missing.md"
	abs := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(incompleteGap), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := osExec(t, root, "git", "add", "-A"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := osExec(t, root, "git", "commit", "-q", "-m", "seed an entity missing a required section"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	return headOf(t, root)
}

func headOf(t *testing.T, root string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func TestBodySectionGate_OrdinaryVerbsOnAnIncompleteEntityStillSucceed(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	base := seedIncompleteEntity(t, root)

	// Each verb writes a different surface of the same incomplete entity:
	// retitle the title and the body H1, promote the status field, archive the
	// file's location. None of them is a body-section change.
	for _, step := range []struct {
		name string
		args []string
	}{
		{"retitle", []string{"retitle", "G-0001", "Something else is missing", "--root", root, "--actor", "human/test"}},
		{"promote", []string{"promote", "G-0001", "addressed", "--by-commit", base, "--root", root, "--actor", "human/test"}},
		{"archive", []string{"archive", "--apply", "--root", root, "--actor", "human/test"}},
	} {
		if rc := cli.Execute(step.args); rc != cliutil.ExitOK {
			t.Fatalf("%s against an entity missing a required section: rc = %d, want ExitOK", step.name, rc)
		}
	}

	if rc := cli.Execute([]string{"check", "--root", root, "--since", base}); rc != cliutil.ExitOK {
		t.Errorf("check after retitle+promote+archive: rc = %d, want ExitOK — "+
			"the gate is scoped to entities whose body content the range changed, "+
			"and none of those three changed one", rc)
	}
}

// TestBodySectionGate_RangeIsLiveInThisFixture is the control for the test
// above. Without it, a gate that never runs at all in this fixture — an
// unresolved range, a rule wired out — would read as the scope holding.
func TestBodySectionGate_RangeIsLiveInThisFixture(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	base := seedIncompleteEntity(t, root)

	rel := "work/gaps/G-0001-something-is-missing.md"
	stripped := strings.Replace(incompleteGap, "## What's missing\n\nA body section, deliberately.\n", "", 1)
	if err := os.WriteFile(filepath.Join(root, rel), []byte(stripped), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := osExec(t, root, "git", "add", "-A"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := osExec(t, root, "git", "commit", "-q", "-m", "drop the one section it had",
		"--trailer", "aiwf-verb: edit-body", "--trailer", "aiwf-entity: G-0001",
		"--trailer", "aiwf-actor: human/test"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	if rc := cli.Execute([]string{"check", "--root", root, "--since", base}); rc == cliutil.ExitOK {
		t.Error("check = ExitOK after a plain git commit dropped a required section; " +
			"the gate did not run in this fixture, so the scope test above proves nothing")
	}
}
