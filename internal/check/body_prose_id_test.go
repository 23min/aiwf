package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// TestBodyProseID_Matrix walks the rule's classification space:
// malformed-shape, unresolved bare, unresolved composite parent,
// unresolved composite AC, and the silent positive controls.
// Per G-0184 — pins the id-shape chokepoint at the committed body
// prose layer.
func TestBodyProseID_Matrix(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		body        string
		wantSubcode string
		wantToken   string
		silent      bool
	}{
		// --- malformed-shape ---
		{
			name:        "single-lowercase-letter (triggering instance)",
			body:        "We depend on the milestone M-a.",
			wantSubcode: "malformed-shape",
			wantToken:   "M-a",
		},
		{
			name:        "lowercase-word suffix",
			body:        "The plan starts with the milestone M-alpha.",
			wantSubcode: "malformed-shape",
			wantToken:   "M-alpha",
		},
		{
			name:        "uppercase placeholder M-NNNN",
			body:        "Once allocated, promote to addressed --by M-NNNN.",
			wantSubcode: "malformed-shape",
			wantToken:   "M-NNNN",
		},
		{
			name:        "narrow-numeric milestone M-1 (conversational leak)",
			body:        "The first milestone is M-1.",
			wantSubcode: "malformed-shape",
			wantToken:   "M-1",
		},
		{
			name:        "narrow-numeric epic E-1",
			body:        "Scope leak through E-1's depends_on chain.",
			wantSubcode: "malformed-shape",
			wantToken:   "E-1",
		},
		{
			name:        "compound English word ADR-shaped",
			body:        "This is an ADR-shaped concern.",
			wantSubcode: "malformed-shape",
			wantToken:   "ADR-shaped",
		},

		// --- unresolved bare ---
		{
			name:        "unresolved well-formed milestone",
			body:        "See M-9999 for the proposed rule.",
			wantSubcode: "unresolved",
			wantToken:   "M-9999",
		},
		{
			name:        "unresolved well-formed ADR (4-digit canonical)",
			body:        "Per ADR-9999, the decision stands.",
			wantSubcode: "unresolved",
			wantToken:   "ADR-9999",
		},

		// --- unresolved composite ---
		{
			name:        "unresolved composite milestone",
			body:        "Cross-reference to M-9999/AC-1.",
			wantSubcode: "unresolved-milestone",
			wantToken:   "M-9999/AC-1",
		},
		{
			name:        "composite parent present, AC missing",
			body:        "Per M-0001/AC-9, the gap is closed.",
			wantSubcode: "unresolved-ac",
			wantToken:   "M-0001/AC-9",
		},

		// --- silent positive controls ---
		{
			name:   "well-formed resolved",
			body:   "Per M-0001, the rule applies.",
			silent: true,
		},
		{
			name:   "composite resolved",
			body:   "Per M-0001/AC-1, the AC holds.",
			silent: true,
		},
		{
			name:   "malformed inside inline code span",
			body:   "Discussion of `M-a` and `M-NNNN` shapes is fine in code spans.",
			silent: true,
		},
		{
			name:   "malformed inside fenced code block",
			body:   "Example:\n```\nM-a\nM-NNNN\n```\nDone.",
			silent: true,
		},
		{
			name:   "malformed inside tilde fenced code block",
			body:   "Example:\n~~~\nM-a\n~~~\nDone.",
			silent: true,
		},
		{
			name:   "unresolved well-formed inside backticks",
			body:   "Hypothetical id `M-9999` is OK in backticks.",
			silent: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			ents := writeBodyProseFixture(t, root, tc.body)
			tr := &tree.Tree{Root: root, Entities: ents}

			got := bodyProseID(tr)
			if tc.silent {
				if len(got) != 0 {
					t.Fatalf("expected silent, got %d findings: %+v", len(got), got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("findings = %d, want 1: %+v", len(got), got)
			}
			f := got[0]
			if f.Code != CodeBodyProseID {
				t.Errorf("Code = %q, want %q", f.Code, CodeBodyProseID)
			}
			if f.Severity != SeverityError {
				t.Errorf("Severity = %v, want error", f.Severity)
			}
			if f.Subcode != tc.wantSubcode {
				t.Errorf("Subcode = %q, want %q", f.Subcode, tc.wantSubcode)
			}
			if !strings.Contains(f.Message, tc.wantToken) {
				t.Errorf("Message %q should contain token %q", f.Message, tc.wantToken)
			}
			if f.Path == "" {
				t.Errorf("Path empty; finding must name the file path")
			}
		})
	}
}

// TestBodyProseID_DedupePerEntityToken pins the dedupe contract:
// repeated mentions of the same bad token in one entity body produce
// one finding, not one per occurrence.
func TestBodyProseID_DedupePerEntityToken(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	body := "M-a appears here. And M-a appears again. And once more: M-a."
	ents := writeBodyProseFixture(t, root, body)
	tr := &tree.Tree{Root: root, Entities: ents}

	got := bodyProseID(tr)
	if len(got) != 1 {
		t.Fatalf("dedupe broken: got %d findings, want 1: %+v", len(got), got)
	}
}

// TestBodyProseID_ArchivedEntitySkipped pins the archive-scoping
// contract per ADR-0004 §"Check shape rules". An archived entity's
// body is not scanned even if it contains malformed tokens.
func TestBodyProseID_ArchivedEntitySkipped(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	body := "Stale reference to M-a in an archived gap."
	path := "work/gaps/archive/G-0001-archived.md"
	abs := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	fm := "---\nid: G-0001\ntitle: Old\nstatus: addressed\n---\n\n## What's missing\n\n" + body + "\n## Why it matters\n\nDoes not matter.\n"
	if err := os.WriteFile(abs, []byte(fm), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	tr := &tree.Tree{Root: root, Entities: []*entity.Entity{{
		ID: "G-0001", Kind: entity.KindGap, Title: "Old", Status: "addressed", Path: path,
	}}}

	got := bodyProseID(tr)
	if len(got) != 0 {
		t.Fatalf("archived entity should be skipped, got %d findings: %+v", len(got), got)
	}
}

// TestBodyProseID_MultipleEntitiesEachReportSeparately pins per-entity
// scoping: two entities each containing the same malformed token
// produce two findings (one per entity), not a single deduped finding.
func TestBodyProseID_MultipleEntitiesEachReportSeparately(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ents := writeTwoGapsBodyProseFixture(t, root, "M-a appears.")
	tr := &tree.Tree{Root: root, Entities: ents}

	got := bodyProseID(tr)
	if len(got) != 2 {
		t.Fatalf("per-entity finding broken: got %d, want 2: %+v", len(got), got)
	}
}

// writeBodyProseFixture lays down a gap G-0001 with the supplied body
// prose under `## What's missing`, plus a milestone M-0001 with AC-1
// to back the composite-resolution positive controls. Both are loaded
// into the returned slice so the bodyProseID rule's id index sees them.
func writeBodyProseFixture(t *testing.T, root, prose string) []*entity.Entity {
	t.Helper()
	gapPath := "work/gaps/G-0002-fixture.md"
	gapBody := "---\nid: G-0002\ntitle: Fixture\nstatus: open\n---\n\n## What's missing\n\n" +
		prose + "\n\n## Why it matters\n\nIt matters.\n"
	mustWriteFile(t, root, gapPath, gapBody)

	mPath := "work/epics/E-0001-foo/M-0001-bar.md"
	mBody := `---
id: M-0001
title: Bar
status: in_progress
parent: E-0001
tdd: none
acs:
    - id: AC-1
      title: First AC
      status: open
---

## Goal

Goal prose.

## Approach

Approach prose.

## Acceptance criteria

Each AC pins one observable behavior.

### AC-1 — First AC

Body prose for AC-1.
`
	mustWriteFile(t, root, mPath, mBody)

	return []*entity.Entity{
		{ID: "G-0002", Kind: entity.KindGap, Title: "Fixture", Status: "open", Path: gapPath},
		{
			ID: "M-0001", Kind: entity.KindMilestone, Title: "Bar",
			Status: "in_progress", Parent: "E-0001", TDD: "none", Path: mPath,
			ACs: []entity.AcceptanceCriterion{{ID: "AC-1", Title: "First AC", Status: "open"}},
		},
	}
}

// writeTwoGapsBodyProseFixture is the per-entity-scoping fixture:
// two gaps with identical body prose so a per-(entity, token, subcode)
// dedupe surfaces both, while a global dedupe would mask one.
func writeTwoGapsBodyProseFixture(t *testing.T, root, prose string) []*entity.Entity {
	t.Helper()
	g1Path := "work/gaps/G-0002-fixture-a.md"
	g2Path := "work/gaps/G-0003-fixture-b.md"
	body := func(id string) string {
		return "---\nid: " + id + "\ntitle: Fixture\nstatus: open\n---\n\n## What's missing\n\n" +
			prose + "\n\n## Why it matters\n\nIt matters.\n"
	}
	mustWriteFile(t, root, g1Path, body("G-0002"))
	mustWriteFile(t, root, g2Path, body("G-0003"))
	return []*entity.Entity{
		{ID: "G-0002", Kind: entity.KindGap, Title: "Fixture", Status: "open", Path: g1Path},
		{ID: "G-0003", Kind: entity.KindGap, Title: "Fixture", Status: "open", Path: g2Path},
	}
}

func mustWriteFile(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", abs, err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", abs, err)
	}
}
