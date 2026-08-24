package check_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/tree"
)

// writeGap writes a gap file with the given status and body, using the
// canonical on-disk shape the loader expects.
func writeGap(t *testing.T, root, id, status, body string) {
	t.Helper()
	dir := filepath.Join(root, "work", "gaps")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	content := "---\nid: " + id + "\ntitle: " + id + " subject\nstatus: " + status + "\n---\n" + body
	if err := os.WriteFile(filepath.Join(dir, id+"-subject.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", id, err)
	}
}

func loadTree(t *testing.T, root string) *tree.Tree {
	t.Helper()
	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	return tr
}

// TestCitersOf_NamesOpenRecordsThatCiteTheID is the notice's core: at a
// closure, the operator is handed the still-open records whose bodies
// name the entity being closed.
func TestCitersOf_NamesOpenRecordsThatCiteTheID(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeGap(t, root, "G-0001", "open", "## What's missing\n\nBlocked until G-0009 lands.\n")
	writeGap(t, root, "G-0002", "open", "## What's missing\n\nUnrelated to anything.\n")
	writeGap(t, root, "G-0009", "addressed", "## What's missing\n\nThe closed one.\n")

	got := check.CitersOf(loadTree(t, root), "G-0009")

	if len(got) != 1 {
		t.Fatalf("CitersOf named %d records, want 1: %+v", len(got), got)
	}
	// Exact values: the file:line pair is the whole operator-facing
	// payload, and the fixture determines it — writeGap emits five
	// frontmatter lines, a heading and a blank, so the citation is on
	// file line 8. Asserting only that Line is non-zero leaves both
	// fields free to be wrong.
	want := check.Citation{ID: "G-0001", Path: "work/gaps/G-0001-subject.md", Line: 8}
	if got[0] != want {
		t.Errorf("citation = %+v, want %+v", got[0], want)
	}
}

// TestCitersOf_SkipsRecordsThatAreThemselvesTerminal keeps the notice
// pointed at work someone may still act on. A closed record citing a
// closed record is history, and asking anyone to re-read it produces an
// edit that makes the record less true, not more.
func TestCitersOf_SkipsRecordsThatAreThemselvesTerminal(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeGap(t, root, "G-0001", "addressed", "## What's missing\n\nBlocked until G-0009 lands.\n")
	writeGap(t, root, "G-0009", "addressed", "## What's missing\n\nThe closed one.\n")

	if got := check.CitersOf(loadTree(t, root), "G-0009"); len(got) != 0 {
		t.Errorf("CitersOf named %d records, want 0 (the citer is itself terminal): %+v", len(got), got)
	}
}

// TestCitersOf_IgnoresTheClosedRecordsOwnBody stops the notice naming
// the entity just closed, which cites its own id in frontmatter and
// often in prose.
func TestCitersOf_IgnoresTheClosedRecordsOwnBody(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeGap(t, root, "G-0009", "open", "## What's missing\n\nG-0009 is about itself.\n")

	if got := check.CitersOf(loadTree(t, root), "G-0009"); len(got) != 0 {
		t.Errorf("CitersOf named %d records, want 0 (self-citation): %+v", len(got), got)
	}
}

// TestCitersOf_MatchesAcrossLegacyWidth pins that a citation written at
// a narrower legacy width names the same entity — which a reader must
// do whether or not the citation should have been written that way.
// Two cases produce one: a body citing a genuinely-narrow archived
// entity, correct as written because read tolerance is permanent
// (ADR-0008); and the write-side hole G-0518 is open on, where a body
// names a real entity at a width it does not have. Going quiet on the
// second would lose the notice exactly where the record is already
// wrong. Three digits is a gap's narrowest legal form — the floor is
// per-kind, and an epic's is two.
func TestCitersOf_MatchesAcrossLegacyWidth(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeGap(t, root, "G-0001", "open", "## What's missing\n\nWaiting on G-009 to land.\n")
	writeGap(t, root, "G-0009", "addressed", "## What's missing\n\nThe closed one.\n")

	got := check.CitersOf(loadTree(t, root), "G-0009")
	if len(got) != 1 {
		t.Fatalf("CitersOf named %d records, want 1 (G-009 is G-0009): %+v", len(got), got)
	}
}

// TestCitersOf_IgnoresFrontmatter keeps the notice on prose. A
// `discovered_in:` naming a finished epic is the field working
// correctly, not a claim anyone needs to re-read.
func TestCitersOf_IgnoresFrontmatter(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "work", "gaps")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "---\nid: G-0001\ntitle: G-0001 subject\nstatus: open\ndiscovered_in: E-0009\n---\n" +
		"## What's missing\n\nNothing cites anything here.\n"
	if err := os.WriteFile(filepath.Join(dir, "G-0001-subject.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	epicDir := filepath.Join(root, "work", "epics", "E-0009-subject")
	if err := os.MkdirAll(epicDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	epic := "---\nid: E-0009\ntitle: E-0009 subject\nstatus: done\n---\n## Goal\n\nDone.\n"
	if err := os.WriteFile(filepath.Join(epicDir, "epic.md"), []byte(epic), 0o644); err != nil {
		t.Fatalf("write epic: %v", err)
	}

	if got := check.CitersOf(loadTree(t, root), "E-0009"); len(got) != 0 {
		t.Errorf("CitersOf named %d records, want 0 (the only mention is frontmatter): %+v", len(got), got)
	}
}

// TestCitersOf_SkipsArchivedFiles pins the archive exclusion on the
// path rather than on status: an archived record is a frozen snapshot,
// so it is out of scope whatever status its frontmatter carries.
func TestCitersOf_SkipsArchivedFiles(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "work", "gaps", "archive")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "---\nid: G-0001\ntitle: G-0001 subject\nstatus: open\n---\n" +
		"## What's missing\n\nCites G-0009 from the archive.\n"
	if err := os.WriteFile(filepath.Join(dir, "G-0001-subject.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	writeGap(t, root, "G-0009", "addressed", "## What's missing\n\nThe closed one.\n")

	if got := check.CitersOf(loadTree(t, root), "G-0009"); len(got) != 0 {
		t.Errorf("CitersOf named %d records, want 0 (the only citer is archived): %+v", len(got), got)
	}
}

// TestCitersOf_IgnoresOtherIDsAndOrdersByID covers two things one
// fixture shows better than two: a token naming some other entity is
// passed over, and multiple citers come back in id order so the notice
// reads the same way twice.
func TestCitersOf_IgnoresOtherIDsAndOrdersByID(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeGap(t, root, "G-0002", "open", "## What's missing\n\nSecond citer of G-0009.\n")
	writeGap(t, root, "G-0001", "open", "## What's missing\n\nCites G-0005 and G-0009.\n")
	writeGap(t, root, "G-0005", "open", "## What's missing\n\nA live decoy.\n")
	writeGap(t, root, "G-0009", "addressed", "## What's missing\n\nThe closed one.\n")

	got := check.CitersOf(loadTree(t, root), "G-0009")
	if len(got) != 2 {
		t.Fatalf("CitersOf named %d records, want 2: %+v", len(got), got)
	}
	if got[0].ID != "G-0001" || got[1].ID != "G-0002" {
		t.Errorf("order = %s, %s; want G-0001, G-0002", got[0].ID, got[1].ID)
	}
}

// TestCitersOf_CountsBacktickedCitations pins the wider mask. A body
// naming an entity inside a code span is citing it; body-prose-id masks
// those out because there a backticked id-shape is how a body discusses
// syntax, and reading that mask here would drop real citations.
func TestCitersOf_CountsBacktickedCitations(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeGap(t, root, "G-0001", "open", "## What's missing\n\nWaits on `G-0009` to land.\n")
	writeGap(t, root, "G-0009", "addressed", "## What's missing\n\nThe closed one.\n")

	if got := check.CitersOf(loadTree(t, root), "G-0009"); len(got) != 1 {
		t.Errorf("CitersOf named %d records, want 1 (a backticked id names an entity): %+v", len(got), got)
	}
}

// TestCitersOf_CompositeTokenNamesItsParent pins the one-directional
// widening: a body resting on an acceptance criterion rests on the
// milestone that owns it, so closing the milestone reaches that body.
func TestCitersOf_CompositeTokenNamesItsParent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeGap(t, root, "G-0001", "open", "## What's missing\n\nRests on M-0001/AC-1 landing.\n")
	epicDir := filepath.Join(root, "work", "epics", "E-0001-subject")
	if err := os.MkdirAll(epicDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	epic := "---\nid: E-0001\ntitle: E-0001 subject\nstatus: active\n---\n## Goal\n\nActive.\n"
	if err := os.WriteFile(filepath.Join(epicDir, "epic.md"), []byte(epic), 0o644); err != nil {
		t.Fatalf("write epic: %v", err)
	}
	ms := "---\nid: M-0001\ntitle: M-0001 subject\nstatus: done\nparent: E-0001\ntdd: none\n---\n## Goal\n\nDone.\n"
	if err := os.WriteFile(filepath.Join(epicDir, "M-0001-subject.md"), []byte(ms), 0o644); err != nil {
		t.Fatalf("write milestone: %v", err)
	}

	got := check.CitersOf(loadTree(t, root), "M-0001")
	if len(got) != 1 {
		t.Fatalf("CitersOf named %d records, want 1 (M-0001/AC-1 names M-0001): %+v", len(got), got)
	}
	if got[0].ID != "G-0001" {
		t.Errorf("named %q, want G-0001", got[0].ID)
	}
}
