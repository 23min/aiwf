package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
)

// body_section_gate_scope_test.go — M-0331/AC-2. The push-seam membership gate
// judges an entity file that differs between the start of the range and HEAD,
// and reports only a section it carried at the start and lacks at HEAD. A
// frontmatter write or a file move changes no section, so ordinary work on an
// entity that already omits a required section is not blocked by debt that work
// did not create.
//
// Driven through `cli.Execute` so the real verbs do the writing and the real
// `aiwf check` does the judging — the gate's scope is a property of what those
// two do to a tree, not of any one function.

// incompleteGap omits `## Why it matters`, which the gap kind requires. It is
// written by hand and committed with plain git, because `aiwf add` refuses to
// create it — that refusal is the write seam, and this file exists to stand for
// the bodies already committed before any seam landed.
func incompleteGap(status string) string {
	return "---\nid: G-0001\ntitle: Something is missing\nstatus: " + status +
		"\n---\n## What's missing\n\nA body section, deliberately.\n"
}

// seedIncompleteEntity writes the gap, commits it with plain git, and returns
// the SHA — the range base, so the omission predates everything the gate judges.
func seedIncompleteEntity(t *testing.T, root, status string) string {
	t.Helper()
	rel := "work/gaps/G-0001-something-is-missing.md"
	abs := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(incompleteGap(status)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := osExec(t, root, "git", "add", "-A"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := osExec(t, root, "git", "commit", "-q", "-m", "seed an entity missing a required section"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	return headSHA(t, root)
}

func TestBodySectionGate_OrdinaryVerbsOnAnIncompleteEntityStillSucceed(t *testing.T) {
	t.Parallel()
	// Each verb writes a different surface of an incomplete entity — retitle the
	// title and the slug; promote the status field; archive the
	// file's location — and none of them is a body-section change. Each runs in
	// its own repository, so one verb's outcome cannot stand in for another's.
	cases := []struct {
		name   string
		status string
		args   func(root, base string) []string
	}{
		{"retitle", "open", func(root, _ string) []string {
			return []string{"retitle", "G-0001", "Something else is missing", "--root", root, "--actor", "human/test"}
		}},
		{"promote", "open", func(root, base string) []string {
			return []string{"promote", "G-0001", "addressed", "--by-commit", base, "--root", root, "--actor", "human/test"}
		}},
		{"archive", "wontfix", func(root, _ string) []string {
			return []string{"archive", "--apply", "--root", root, "--actor", "human/test"}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := setupCLITestRepo(t)
			base := seedIncompleteEntity(t, root, tc.status)
			if rc := cli.Execute(tc.args(root, base)); rc != cliutil.ExitOK {
				t.Fatalf("%s against an entity missing a required section: rc = %d, want ExitOK", tc.name, rc)
			}
			if rc := cli.Execute([]string{"check", "--root", root, "--since", base}); rc != cliutil.ExitOK {
				t.Errorf("check after %s: rc = %d, want ExitOK — the gate reports only a section the "+
					"entity carried at the start of the range, and this one never carried it", tc.name, rc)
			}
		})
	}
}

// TestBodySectionGate_RangeIsLiveInThisFixture is the control for the test
// above. Without it, a gate that never runs at all in this fixture — an
// unresolved range, a rule wired out — would read as the scope holding.
func TestBodySectionGate_RangeIsLiveInThisFixture(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	base := seedIncompleteEntity(t, root, "open")

	rel := "work/gaps/G-0001-something-is-missing.md"
	stripped := strings.Replace(incompleteGap("open"), "## What's missing\n\nA body section, deliberately.\n", "", 1)
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
