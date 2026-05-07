package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/tree"
)

// TestEntityBodyEmpty_FiresPerKind_OneSectionEmpty pins M-066/AC-1:
// for each entity kind in the per-kind table, an entity whose
// frontmatter is fine and most body sections have prose, but one
// load-bearing section has nothing under its heading, surfaces a
// single `entity-body-empty` warning naming the empty section.
//
// The cases share the same tempdir-and-fixture pattern so each kind
// runs in isolation; the fixture builders below produce minimal
// frontmatter + the body shape the rule walks. The "AC sub-element"
// case lives under a milestone parent because ACs are not standalone
// files.
func TestEntityBodyEmpty_FiresPerKind_OneSectionEmpty(t *testing.T) {
	cases := []struct {
		name         string
		writeFixture func(root string) (entities []*entity.Entity, err error)
		wantEntityID string
		wantSubcode  string
		wantSection  string // appears in Message
	}{
		{
			name:         "epic with empty Scope",
			writeFixture: writeEpicFixture("Scope"),
			wantEntityID: "E-01",
			wantSubcode:  "epic",
			wantSection:  "Scope",
		},
		{
			name:         "milestone with empty Approach",
			writeFixture: writeMilestoneFixture("Approach"),
			wantEntityID: "M-001",
			wantSubcode:  "milestone",
			wantSection:  "Approach",
		},
		{
			name:         "AC body empty under heading",
			writeFixture: writeACFixture(),
			wantEntityID: "M-001/AC-1",
			wantSubcode:  "ac",
			wantSection:  "AC-1",
		},
		{
			name:         "gap with empty `Why it matters`",
			writeFixture: writeGapFixture("Why it matters"),
			wantEntityID: "G-001",
			wantSubcode:  "gap",
			wantSection:  "Why it matters",
		},
		{
			name:         "adr with empty Decision",
			writeFixture: writeADRFixture("Decision"),
			wantEntityID: "ADR-0001",
			wantSubcode:  "adr",
			wantSection:  "Decision",
		},
		{
			name:         "decision with empty Reasoning",
			writeFixture: writeDecisionFixture("Reasoning"),
			wantEntityID: "D-001",
			wantSubcode:  "decision",
			wantSection:  "Reasoning",
		},
		{
			name:         "contract with empty Stability",
			writeFixture: writeContractFixture("Stability"),
			wantEntityID: "C-001",
			wantSubcode:  "contract",
			wantSection:  "Stability",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			ents, err := tc.writeFixture(root)
			if err != nil {
				t.Fatalf("write fixture: %v", err)
			}
			tr := &tree.Tree{Root: root, Entities: ents}

			got := entityBodyEmpty(tr)
			if len(got) != 1 {
				t.Fatalf("entityBodyEmpty findings = %d, want 1: %+v", len(got), got)
			}
			f := got[0]
			if f.Code != "entity-body-empty" {
				t.Errorf("Code = %q, want entity-body-empty", f.Code)
			}
			if f.Severity != SeverityWarning {
				t.Errorf("Severity = %v, want warning", f.Severity)
			}
			if f.Subcode != tc.wantSubcode {
				t.Errorf("Subcode = %q, want %q", f.Subcode, tc.wantSubcode)
			}
			if f.EntityID != tc.wantEntityID {
				t.Errorf("EntityID = %q, want %q", f.EntityID, tc.wantEntityID)
			}
			if !contains(f.Message, tc.wantSection) {
				t.Errorf("Message %q should mention section %q", f.Message, tc.wantSection)
			}
			if f.Path == "" {
				t.Errorf("Path empty; finding must name the file path")
			}
		})
	}
}

// TestEntityBodyEmpty_CancelledACSkipped pins the cancelled-AC arm
// of the AC-body branch: when an AC is `status: cancelled`, the rule
// must not fire even if its body section is empty. Cancellation
// signals "this AC was ruled out"; surfacing an empty-body warning
// against it would be noise.
func TestEntityBodyEmpty_CancelledACSkipped(t *testing.T) {
	root := t.TempDir()
	path := "work/epics/E-01-foo/M-001-bar.md"
	body := `## Goal

Goal prose.

## Approach

Approach prose.

## Acceptance criteria

Each AC pins one observable behavior.

### AC-1 — Live AC

prose

### AC-2 — Cancelled AC

`
	fm := `---
id: M-001
title: Bar
status: in_progress
parent: E-01
tdd: none
acs:
    - id: AC-1
      title: Live AC
      status: open
    - id: AC-2
      title: Cancelled AC
      status: cancelled
---

`
	abs := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(fm+body), 0o644); err != nil {
		t.Fatal(err)
	}
	tr := &tree.Tree{
		Root: root,
		Entities: []*entity.Entity{{
			ID: "M-001", Kind: entity.KindMilestone, Title: "Bar",
			Status: "in_progress", Parent: "E-01", TDD: "none", Path: path,
			ACs: []entity.AcceptanceCriterion{
				{ID: "AC-1", Title: "Live AC", Status: "open"},
				{ID: "AC-2", Title: "Cancelled AC", Status: entity.StatusCancelled},
			},
		}},
	}
	got := entityBodyEmpty(tr)
	if len(got) != 0 {
		t.Errorf("cancelled AC with empty body should produce no finding; got %+v", got)
	}
}

// TestEntityBodyEmpty_ACWithoutBodyHeadingSkipped pins the
// "AC heading missing from body" arm: when an AC exists in
// frontmatter `acs[]` but has no matching `### AC-N` body heading,
// `acs-body-coherence/missing-heading` is the rule that owns the
// finding. entity-body-empty must stay silent so the operator gets
// one signal, not two redundant ones.
func TestEntityBodyEmpty_ACWithoutBodyHeadingSkipped(t *testing.T) {
	root := t.TempDir()
	path := "work/epics/E-01-foo/M-001-bar.md"
	// AC-1 exists in frontmatter but the body has only `## ` headings,
	// no `### AC-1` heading. acs-body-coherence reports the missing
	// heading; entity-body-empty stays silent.
	body := `## Goal

Goal prose.

## Approach

Approach prose.

## Acceptance criteria

Each AC pins one observable behavior.
`
	fm := `---
id: M-001
title: Bar
status: in_progress
parent: E-01
tdd: none
acs:
    - id: AC-1
      title: Orphaned in frontmatter
      status: open
---

`
	abs := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(fm+body), 0o644); err != nil {
		t.Fatal(err)
	}
	tr := &tree.Tree{
		Root: root,
		Entities: []*entity.Entity{{
			ID: "M-001", Kind: entity.KindMilestone, Title: "Bar",
			Status: "in_progress", Parent: "E-01", TDD: "none", Path: path,
			ACs: []entity.AcceptanceCriterion{
				{ID: "AC-1", Title: "Orphaned in frontmatter", Status: "open"},
			},
		}},
	}
	got := entityBodyEmpty(tr)
	if len(got) != 0 {
		t.Errorf("AC missing body heading should not surface entity-body-empty (acs-body-coherence handles it); got %+v", got)
	}
}

// TestEntityBodyEmpty_NonEmptyBodyClean confirms the rule stays silent
// when every required section has content. Same kinds as the firing
// test; serves as the negative control so a future bug that emits
// findings on healthy trees is caught fast.
func TestEntityBodyEmpty_NonEmptyBodyClean(t *testing.T) {
	root := t.TempDir()
	ents, err := writeFullyPopulatedFixture(root)
	if err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	tr := &tree.Tree{Root: root, Entities: ents}
	if got := entityBodyEmpty(tr); len(got) != 0 {
		t.Errorf("populated tree should produce no findings; got %+v", got)
	}
}

// --- fixture builders ---------------------------------------------------

func writeEpicFixture(emptySection string) func(root string) ([]*entity.Entity, error) {
	return func(root string) ([]*entity.Entity, error) {
		path := "work/epics/E-01-foo/epic.md"
		body := buildBody(map[string]string{
			"Goal":         "Goal prose.",
			"Scope":        "Scope prose.",
			"Out of scope": "Out-of-scope prose.",
		}, []string{"Goal", "Scope", "Out of scope"}, emptySection)
		fm := "---\nid: E-01\ntitle: Foo\nstatus: active\n---\n\n"
		return write1(root, path, fm+body, &entity.Entity{
			ID: "E-01", Kind: entity.KindEpic, Title: "Foo", Status: "active", Path: path,
		})
	}
}

func writeMilestoneFixture(emptySection string) func(root string) ([]*entity.Entity, error) {
	return func(root string) ([]*entity.Entity, error) {
		path := "work/epics/E-01-foo/M-001-bar.md"
		body := buildBody(map[string]string{
			"Goal":                "Goal prose.",
			"Approach":            "Approach prose.",
			"Acceptance criteria": "Each AC pins one observable behavior.",
		}, []string{"Goal", "Approach", "Acceptance criteria"}, emptySection)
		fm := "---\nid: M-001\ntitle: Bar\nstatus: in_progress\nparent: E-01\ntdd: none\n---\n\n"
		return write1(root, path, fm+body, &entity.Entity{
			ID: "M-001", Kind: entity.KindMilestone, Title: "Bar",
			Status: "in_progress", Parent: "E-01", TDD: "none", Path: path,
		})
	}
}

// writeACFixture builds a milestone whose AC-1 body is empty
// (heading present, no prose under it) and AC-2 body has prose.
// All three top-level milestone sections have prose.
func writeACFixture() func(root string) ([]*entity.Entity, error) {
	return func(root string) ([]*entity.Entity, error) {
		path := "work/epics/E-01-foo/M-001-bar.md"
		body := `## Goal

Goal prose.

## Approach

Approach prose.

## Acceptance criteria

Each AC pins one observable behavior.

### AC-1 — Empty AC

### AC-2 — Filled AC

ac-2 prose
`
		fm := `---
id: M-001
title: Bar
status: in_progress
parent: E-01
tdd: none
acs:
    - id: AC-1
      title: Empty AC
      status: open
    - id: AC-2
      title: Filled AC
      status: open
---

`
		return write1(root, path, fm+body, &entity.Entity{
			ID: "M-001", Kind: entity.KindMilestone, Title: "Bar",
			Status: "in_progress", Parent: "E-01", TDD: "none", Path: path,
			ACs: []entity.AcceptanceCriterion{
				{ID: "AC-1", Title: "Empty AC", Status: "open"},
				{ID: "AC-2", Title: "Filled AC", Status: "open"},
			},
		})
	}
}

func writeGapFixture(emptySection string) func(root string) ([]*entity.Entity, error) {
	return func(root string) ([]*entity.Entity, error) {
		path := "work/gaps/G-001-foo.md"
		body := buildBody(map[string]string{
			"What's missing": "Missing prose.",
			"Why it matters": "Matters prose.",
		}, []string{"What's missing", "Why it matters"}, emptySection)
		fm := "---\nid: G-001\ntitle: Foo\nstatus: open\n---\n\n"
		return write1(root, path, fm+body, &entity.Entity{
			ID: "G-001", Kind: entity.KindGap, Title: "Foo", Status: "open", Path: path,
		})
	}
}

func writeADRFixture(emptySection string) func(root string) ([]*entity.Entity, error) {
	return func(root string) ([]*entity.Entity, error) {
		path := "docs/adr/ADR-0001-foo.md"
		body := buildBody(map[string]string{
			"Context":      "Context prose.",
			"Decision":     "Decision prose.",
			"Consequences": "Consequences prose.",
		}, []string{"Context", "Decision", "Consequences"}, emptySection)
		fm := "---\nid: ADR-0001\ntitle: Foo\nstatus: proposed\n---\n\n"
		return write1(root, path, fm+body, &entity.Entity{
			ID: "ADR-0001", Kind: entity.KindADR, Title: "Foo", Status: "proposed", Path: path,
		})
	}
}

func writeDecisionFixture(emptySection string) func(root string) ([]*entity.Entity, error) {
	return func(root string) ([]*entity.Entity, error) {
		path := "work/decisions/D-001-foo.md"
		body := buildBody(map[string]string{
			"Question":  "Question prose.",
			"Decision":  "Decision prose.",
			"Reasoning": "Reasoning prose.",
		}, []string{"Question", "Decision", "Reasoning"}, emptySection)
		fm := "---\nid: D-001\ntitle: Foo\nstatus: proposed\n---\n\n"
		return write1(root, path, fm+body, &entity.Entity{
			ID: "D-001", Kind: entity.KindDecision, Title: "Foo", Status: "proposed", Path: path,
		})
	}
}

func writeContractFixture(emptySection string) func(root string) ([]*entity.Entity, error) {
	return func(root string) ([]*entity.Entity, error) {
		path := "work/contracts/C-001-foo/contract.md"
		body := buildBody(map[string]string{
			"Purpose":   "Purpose prose.",
			"Stability": "Stability prose.",
		}, []string{"Purpose", "Stability"}, emptySection)
		fm := "---\nid: C-001\ntitle: Foo\nstatus: proposed\n---\n\n"
		return write1(root, path, fm+body, &entity.Entity{
			ID: "C-001", Kind: entity.KindContract, Title: "Foo", Status: "proposed", Path: path,
		})
	}
}

// writeFullyPopulatedFixture builds a tree with one entity per kind,
// each with all required sections non-empty. Used by the negative
// control test.
func writeFullyPopulatedFixture(root string) ([]*entity.Entity, error) {
	type want struct {
		path string
		fm   string
		body string
		ent  *entity.Entity
	}
	all := []want{
		{
			path: "work/epics/E-01-foo/epic.md",
			fm:   "---\nid: E-01\ntitle: Foo\nstatus: active\n---\n\n",
			body: "## Goal\n\nGoal.\n\n## Scope\n\nScope.\n\n## Out of scope\n\nOOS.\n",
			ent:  &entity.Entity{ID: "E-01", Kind: entity.KindEpic, Title: "Foo", Status: "active", Path: "work/epics/E-01-foo/epic.md"},
		},
		{
			path: "work/epics/E-01-foo/M-001-bar.md",
			fm:   "---\nid: M-001\ntitle: Bar\nstatus: in_progress\nparent: E-01\ntdd: none\n---\n\n",
			body: "## Goal\n\nGoal.\n\n## Approach\n\nApproach.\n\n## Acceptance criteria\n\nEach AC pins one observable behavior.\n",
			ent:  &entity.Entity{ID: "M-001", Kind: entity.KindMilestone, Title: "Bar", Status: "in_progress", Parent: "E-01", TDD: "none", Path: "work/epics/E-01-foo/M-001-bar.md"},
		},
		{
			path: "work/gaps/G-001-foo.md",
			fm:   "---\nid: G-001\ntitle: Foo\nstatus: open\n---\n\n",
			body: "## What's missing\n\nMissing.\n\n## Why it matters\n\nMatters.\n",
			ent:  &entity.Entity{ID: "G-001", Kind: entity.KindGap, Title: "Foo", Status: "open", Path: "work/gaps/G-001-foo.md"},
		},
		{
			path: "docs/adr/ADR-0001-foo.md",
			fm:   "---\nid: ADR-0001\ntitle: Foo\nstatus: proposed\n---\n\n",
			body: "## Context\n\nContext.\n\n## Decision\n\nDecision.\n\n## Consequences\n\nConsequences.\n",
			ent:  &entity.Entity{ID: "ADR-0001", Kind: entity.KindADR, Title: "Foo", Status: "proposed", Path: "docs/adr/ADR-0001-foo.md"},
		},
		{
			path: "work/decisions/D-001-foo.md",
			fm:   "---\nid: D-001\ntitle: Foo\nstatus: proposed\n---\n\n",
			body: "## Question\n\nQuestion.\n\n## Decision\n\nDecision.\n\n## Reasoning\n\nReasoning.\n",
			ent:  &entity.Entity{ID: "D-001", Kind: entity.KindDecision, Title: "Foo", Status: "proposed", Path: "work/decisions/D-001-foo.md"},
		},
		{
			path: "work/contracts/C-001-foo/contract.md",
			fm:   "---\nid: C-001\ntitle: Foo\nstatus: proposed\n---\n\n",
			body: "## Purpose\n\nPurpose.\n\n## Stability\n\nStability.\n",
			ent:  &entity.Entity{ID: "C-001", Kind: entity.KindContract, Title: "Foo", Status: "proposed", Path: "work/contracts/C-001-foo/contract.md"},
		},
	}
	ents := make([]*entity.Entity, 0, len(all))
	for _, w := range all {
		abs := filepath.Join(root, filepath.FromSlash(w.path))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(abs, []byte(w.fm+w.body), 0o644); err != nil {
			return nil, err
		}
		ents = append(ents, w.ent)
	}
	return ents, nil
}

// buildBody constructs a body where every named section gets its
// prose from filled, except `empty` which is rendered as a bare
// heading with no content beneath it.
func buildBody(filled map[string]string, order []string, empty string) string {
	var b []byte
	for i, name := range order {
		b = append(b, []byte("## "+name+"\n\n")...)
		if name != empty {
			b = append(b, []byte(filled[name]+"\n")...)
		}
		if i < len(order)-1 {
			b = append(b, []byte("\n")...)
		}
	}
	return string(b)
}

// write1 writes one entity file and returns the entity slice for the
// tree. Couples the on-disk write with the in-memory entity so the
// fixture builders stay readable.
func write1(root, rel, content string, e *entity.Entity) ([]*entity.Entity, error) {
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		return nil, err
	}
	return []*entity.Entity{e}, nil
}
