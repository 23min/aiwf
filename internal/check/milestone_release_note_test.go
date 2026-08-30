package check

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

// loadReleaseNoteTree loads a fixture tree. The package-external test files
// carry their own loader, which an in-package test cannot reach.
func loadReleaseNoteTree(t *testing.T, root string) *tree.Tree {
	t.Helper()
	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	return tr
}

// writeReleaseNoteFixture writes an epic and one milestone whose body is
// bodySections, returning the tree root. The milestone's status is caller-set so
// the rule's `done` scoping can be exercised from both sides.
func writeReleaseNoteFixture(t *testing.T, status, bodySections string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "work", "epics", "E-0001-subject")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	epic := "---\nid: E-0001\ntitle: Subject\nstatus: active\n---\n\n## Goal\n\nx\n"
	if err := os.WriteFile(filepath.Join(dir, "epic.md"), []byte(epic), 0o600); err != nil {
		t.Fatalf("write epic: %v", err)
	}
	ms := "---\nid: M-0001\ntitle: Subject\nparent: E-0001\nstatus: " + status + "\n---\n\n" + bodySections
	if err := os.WriteFile(filepath.Join(dir, "M-0001-subject.md"), []byte(ms), 0o600); err != nil {
		t.Fatalf("write milestone: %v", err)
	}
	return root
}

const releaseNoteFilled = "## Goal\n\nx\n\n## Release note\n\nThe verb now accepts a flag.\n"

// scaffoldOnly is what a milestone spec carries straight out of the template:
// the heading with its guidance comment and nothing an author wrote.
const scaffoldOnly = "## Goal\n\nx\n\n## Release note\n\n<!-- The user-visible delta of this milestone. -->\n"

// noSection is a spec with no such heading at all. It counts as unwritten:
// scoping to present-and-empty would make deleting the heading an escape.
const noSection = "## Goal\n\nx\n\n## Work log\n\n### AC-1 — x\n\ndone\n"

func TestMilestoneDoneEmptyReleaseNote(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status string
		body   string
		want   bool
	}{
		{"done with only the scaffold comment fires", "done", scaffoldOnly, true},
		{"done with an empty section fires", "done", "## Goal\n\nx\n\n## Release note\n", true},
		{"done with no such section fires", "done", noSection, true},
		{"done with a written note is clean", "done", releaseNoteFilled, false},
		// One case for the status gate, not one per non-done status: every
		// non-done status leaves through the same arm, so the rest would be
		// spellings of one rule rather than distinct rules.
		{"a milestone short of done is not yet due", "in_progress", noSection, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := writeReleaseNoteFixture(t, tc.status, tc.body)
			fs := milestoneDoneEmptyReleaseNote(loadReleaseNoteTree(t, root))
			if tc.want && len(fs) == 0 {
				t.Fatal("want a finding, got none")
			}
			if !tc.want && len(fs) != 0 {
				t.Fatalf("want no finding, got %+v", fs)
			}
			if !tc.want {
				return
			}
			f := fs[0]
			if f.Code != CodeMilestoneDoneEmptyReleaseNote {
				t.Errorf("Code = %q, want %q", f.Code, CodeMilestoneDoneEmptyReleaseNote)
			}
			if f.Severity != SeverityError {
				t.Errorf("Severity = %v, want error — the promote precondition gates on error severity", f.Severity)
			}
			if f.Path == "" {
				t.Error("Path is empty; the finding must name the file to look at")
			}
			if f.Field != "release_note" {
				t.Errorf("Field = %q, want release_note", f.Field)
			}
			if f.EntityID != "M-0001" {
				t.Errorf("EntityID = %q, want M-0001", f.EntityID)
			}
			if !strings.Contains(f.Message, "M-0001") {
				t.Errorf("Message = %q, want it to name the milestone", f.Message)
			}
		})
	}
}

// TestMilestoneDoneEmptyReleaseNote_SkipsArchived pins the ADR-0004 archive
// scoping: an archived milestone is historical state, not active drift, and
// every done milestone is swept there eventually. The live window the rule
// governs is the promote itself and the span before the sweep.
func TestMilestoneDoneEmptyReleaseNote_SkipsArchived(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// The ADR-0004 archive shape moves the whole epic directory under
	// `work/epics/archive/`; a per-milestone `archive/` subdirectory is not a
	// form the loader recognizes, so a fixture shaped that way loads no
	// milestone and would assert nothing.
	dir := filepath.Join(root, "work", "epics", "archive", "E-0001-subject")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	epic := "---\nid: E-0001\ntitle: Subject\nstatus: done\n---\n\n## Goal\n\nx\n"
	if err := os.WriteFile(filepath.Join(dir, "epic.md"), []byte(epic), 0o600); err != nil {
		t.Fatalf("write epic: %v", err)
	}
	ms := "---\nid: M-0001\ntitle: Subject\nparent: E-0001\nstatus: done\n---\n\n" + scaffoldOnly
	if err := os.WriteFile(filepath.Join(dir, "M-0001-subject.md"), []byte(ms), 0o600); err != nil {
		t.Fatalf("write milestone: %v", err)
	}
	tr := loadReleaseNoteTree(t, root)
	if tr.ByID("M-0001") == nil {
		t.Fatal("fixture did not load the archived milestone; the test would assert nothing")
	}
	if fs := milestoneDoneEmptyReleaseNote(tr); len(fs) != 0 {
		t.Fatalf("want no finding on an archived milestone, got %+v", fs)
	}
}

// TestMilestoneDoneEmptyReleaseNote_SkipsUnreadableBody pins the contract for a
// file that changes between the tree load and this rule's read: the rule reports
// nothing rather than guessing, because the load-error path already owns the
// file-cannot-be-read case. Both post-load failures are exercised — the file
// gone, and the file no longer splitting into frontmatter and body.
func TestMilestoneDoneEmptyReleaseNote_SkipsUnreadableBody(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		rewrite func(path string) error
	}{
		{"file removed after load", os.Remove},
		{"frontmatter destroyed after load", func(path string) error {
			return os.WriteFile(path, []byte("no frontmatter here\n"), 0o600)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := writeReleaseNoteFixture(t, "done", scaffoldOnly)
			tr := loadReleaseNoteTree(t, root)
			path := filepath.Join(root, "work", "epics", "E-0001-subject", "M-0001-subject.md")
			if err := tc.rewrite(path); err != nil {
				t.Fatalf("rewrite: %v", err)
			}
			if fs := milestoneDoneEmptyReleaseNote(tr); len(fs) != 0 {
				t.Fatalf("want no finding once the body is unreadable, got %+v", fs)
			}
		})
	}
}
