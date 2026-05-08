package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M-077/AC-1: `aiwf retitle <id> "<new-title>" [--reason ...]` updates
// the entity's frontmatter `title:` field for any of the six top-level
// kinds. Title only — no body changes, no slug renames. Closes the
// top-level half of G-065.

// retitleSetup gives every test in this file a freshly-init'd repo
// with one entity per top-level kind so the verb has live targets to
// retitle.
func retitleSetup(t *testing.T) string {
	t.Helper()
	root := setupCLITestRepo(t)
	if rc := run([]string{"add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--epic", "E-01", "--tdd", "none", "--title", "First Milestone", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add milestone: %d", rc)
	}
	if rc := run([]string{"add", "adr", "--title", "First ADR", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add adr: %d", rc)
	}
	if rc := run([]string{"add", "gap", "--title", "First Gap", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add gap: %d", rc)
	}
	if rc := run([]string{"add", "decision", "--title", "First Decision", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add decision: %d", rc)
	}
	if rc := run([]string{"add", "contract", "--title", "First Contract", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add contract: %d", rc)
	}
	return root
}

// TestRetitle_AllKinds pins AC-1: retitle works for every top-level
// kind, updating the frontmatter `title:` field in one commit.
func TestRetitle_AllKinds(t *testing.T) {
	cases := []struct {
		name     string
		id       string
		newTitle string
		path     string
	}{
		{"epic", "E-01", "Refocused Foundations", "work/epics/E-01-foundations/epic.md"},
		{"milestone", "M-001", "Refocused First Milestone", "work/epics/E-01-foundations/M-001-first-milestone.md"},
		{"adr", "ADR-0001", "Refocused First ADR", "docs/adr/ADR-0001-first-adr.md"},
		{"gap", "G-001", "Refocused First Gap", "work/gaps/G-001-first-gap.md"},
		{"decision", "D-001", "Refocused First Decision", "work/decisions/D-001-first-decision.md"},
		{"contract", "C-001", "Refocused First Contract", "work/contracts/C-001-first-contract/contract.md"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := retitleSetup(t)

			rc := run([]string{
				"retitle", tc.id, tc.newTitle,
				"--actor", "human/test",
				"--root", root,
			})
			if rc != exitOK {
				t.Fatalf("retitle %s: %d", tc.id, rc)
			}

			body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(tc.path)))
			if err != nil {
				t.Fatalf("read %s: %v", tc.path, err)
			}
			want := "title: " + tc.newTitle
			if !strings.Contains(string(body), want) {
				t.Errorf("frontmatter missing %q after retitle:\n%s", want, body)
			}
		})
	}
}

// TestRetitle_Reason pins AC-1's --reason flag: the prose lands in
// the commit body (visible to `aiwf history`).
func TestRetitle_Reason(t *testing.T) {
	root := retitleSetup(t)

	rc := run([]string{
		"retitle", "E-01", "Reasoned Title",
		"--reason", "scope absorbed M-076 work",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitOK {
		t.Fatalf("retitle with --reason: %d", rc)
	}

	// History should land — the trailer chain reached git.
	if rc := run([]string{"history", "E-01", "--root", root}); rc != exitOK {
		t.Errorf("history E-01: %d", rc)
	}
}

// TestRetitle_EmptyTitleRejected pins the empty-title guard.
func TestRetitle_EmptyTitleRejected(t *testing.T) {
	root := retitleSetup(t)

	rc := run([]string{
		"retitle", "E-01", "   ",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitUsage {
		t.Errorf("retitle E-01 with whitespace title = %d, want %d", rc, exitUsage)
	}
}

// TestRetitle_SameTitleRejected pins the no-op guard: passing the
// current title produces a clear error so the operator notices the
// typo (no commit lands).
func TestRetitle_SameTitleRejected(t *testing.T) {
	root := retitleSetup(t)

	rc := run([]string{
		"retitle", "E-01", "Foundations",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitUsage {
		t.Errorf("retitle E-01 with same title = %d, want %d", rc, exitUsage)
	}
}

// TestRetitle_UnknownIdRejected pins the missing-target guard.
func TestRetitle_UnknownIdRejected(t *testing.T) {
	root := retitleSetup(t)

	rc := run([]string{
		"retitle", "E-99", "Whatever",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitUsage {
		t.Errorf("retitle E-99 = %d, want %d (E-99 doesn't exist)", rc, exitUsage)
	}
}

// TestRetitle_AC_FrontmatterAndBody pins AC-2: composite-id retitle
// updates BOTH the parent milestone's acs[i].title AND the matching
// `### AC-N — <title>` body heading, atomically in one commit.
func TestRetitle_AC_FrontmatterAndBody(t *testing.T) {
	root := retitleSetup(t)
	if rc := run([]string{"add", "ac", "M-001", "--title", "original ac title", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add ac: %d", rc)
	}

	rc := run([]string{
		"retitle", "M-001/AC-1", "refocused ac title",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitOK {
		t.Fatalf("retitle M-001/AC-1: %d", rc)
	}

	mPath := filepath.Join(root, "work", "epics", "E-01-foundations", "M-001-first-milestone.md")
	body, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	content := string(body)

	if !strings.Contains(content, "title: refocused ac title") {
		t.Errorf("acs[].title not updated:\n%s", content)
	}
	if !strings.Contains(content, "### AC-1 — refocused ac title") {
		t.Errorf("body heading not regenerated:\n%s", content)
	}
	if strings.Contains(content, "### AC-1 — original ac title") {
		t.Errorf("body still carries old AC-1 heading:\n%s", content)
	}
}

// TestRetitle_AC_UnknownRejected pins the missing-AC guard.
func TestRetitle_AC_UnknownRejected(t *testing.T) {
	root := retitleSetup(t)

	rc := run([]string{
		"retitle", "M-001/AC-9", "Whatever",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitUsage {
		t.Errorf("retitle M-001/AC-9 = %d, want %d (no AC-9)", rc, exitUsage)
	}
}

// TestRetitle_DispatcherSeam_TopLevel is the seam test for AC-6 on
// the top-level path, per CLAUDE.md "Test the seam, not just the
// layer". Drives the dispatcher end-to-end and asserts both the on-
// disk title change AND the trailered commit landed (history finds it).
func TestRetitle_DispatcherSeam_TopLevel(t *testing.T) {
	root := retitleSetup(t)

	rc := run([]string{
		"retitle", "E-01", "Seam-tested Foundations",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitOK {
		t.Fatalf("retitle (seam top-level): %d", rc)
	}

	body, err := os.ReadFile(filepath.Join(root, "work", "epics", "E-01-foundations", "epic.md"))
	if err != nil {
		t.Fatalf("read epic: %v", err)
	}
	if !strings.Contains(string(body), "title: Seam-tested Foundations") {
		t.Errorf("epic frontmatter missing new title (seam):\n%s", body)
	}
	if rc := run([]string{"history", "E-01", "--root", root}); rc != exitOK {
		t.Errorf("aiwf history E-01 (seam): %d", rc)
	}
}

// TestRetitle_DispatcherSeam_Composite is the seam test for AC-6 on
// the composite-id path.
func TestRetitle_DispatcherSeam_Composite(t *testing.T) {
	root := retitleSetup(t)
	if rc := run([]string{"add", "ac", "M-001", "--title", "original", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add ac: %d", rc)
	}

	rc := run([]string{
		"retitle", "M-001/AC-1", "seam-tested",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitOK {
		t.Fatalf("retitle (seam composite): %d", rc)
	}
	if rc := run([]string{"history", "M-001/AC-1", "--root", root}); rc != exitOK {
		t.Errorf("aiwf history M-001/AC-1 (seam): %d", rc)
	}
}
