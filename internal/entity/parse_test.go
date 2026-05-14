package entity

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParse_Minimal(t *testing.T) {
	t.Parallel()
	content := []byte(`---
id: E-01
title: Platform foundations
status: active
---

## Goal

Set up the platform.
`)
	got, err := Parse("work/epics/E-01-platform/epic.md", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := &Entity{
		ID:     "E-01",
		Title:  "Platform foundations",
		Status: "active",
		Path:   "work/epics/E-01-platform/epic.md",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Parse mismatch (-want +got):\n%s", diff)
	}
}

func TestParse_MilestoneWithRefs(t *testing.T) {
	t.Parallel()
	content := []byte(`---
id: M-007
title: Cache warmup
status: in_progress
parent: E-01
depends_on:
  - M-002
  - M-005
---

body
`)
	got, err := Parse("work/epics/E-01-platform/M-007-cache.md", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.ID != "M-007" || got.Parent != "E-01" {
		t.Errorf("got %+v", got)
	}
	if diff := cmp.Diff([]string{"M-002", "M-005"}, got.DependsOn); diff != "" {
		t.Errorf("depends_on mismatch (-want +got):\n%s", diff)
	}
}

// TestParse_MilestoneAbsentACsAndTDD pins the load-bearing absent-field
// defaults: a milestone with no `acs:` and no `tdd:` keys must parse to
// a zero-value slice and an empty TDD policy. Step 6's checks treat
// these as `[]` and `none` respectively; a parse-side regression here
// would silently break that contract before the check runs.
func TestParse_MilestoneAbsentACsAndTDD(t *testing.T) {
	t.Parallel()
	content := []byte(`---
id: M-001
title: Pre-I2 milestone
status: in_progress
parent: E-01
---
`)
	got, err := Parse("work/epics/E-01-platform/M-001-pre.md", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.TDD != "" {
		t.Errorf("absent tdd: parsed as %q, want empty string", got.TDD)
	}
	if got.ACs != nil {
		t.Errorf("absent acs: parsed as %v, want nil", got.ACs)
	}
}

// TestParse_MilestoneWithACsAndTDD covers the full positive path: a
// `tdd: required` milestone with two ACs, the first open in red and
// the second met with phase done. The `KnownFields(true)` decoder will
// reject the new keys until the Entity struct carries them.
func TestParse_MilestoneWithACsAndTDD(t *testing.T) {
	t.Parallel()
	content := []byte(`---
id: M-007
title: Engine warning surface
status: in_progress
parent: E-03
tdd: required
acs:
  - id: AC-1
    title: Engine emits warning on bad input
    status: open
    tdd_phase: red
  - id: AC-2
    title: Pack receives canonical OpResult
    status: met
    tdd_phase: done
---
`)
	got, err := Parse("work/epics/E-03/M-007-warnings.md", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.TDD != "required" {
		t.Errorf("tdd = %q, want required", got.TDD)
	}
	want := []AcceptanceCriterion{
		{ID: "AC-1", Title: "Engine emits warning on bad input", Status: "open", TDDPhase: "red"},
		{ID: "AC-2", Title: "Pack receives canonical OpResult", Status: "met", TDDPhase: "done"},
	}
	if diff := cmp.Diff(want, got.ACs); diff != "" {
		t.Errorf("ACs mismatch (-want +got):\n%s", diff)
	}
}

// TestParse_ACWithoutTDDPhase confirms an AC without `tdd_phase:` parses
// as TDDPhase == "" — the absent-field sentinel. This is the shape an
// `aiwf add ac` invocation produces when the parent milestone is `tdd:
// none` (or absent).
func TestParse_ACWithoutTDDPhase(t *testing.T) {
	t.Parallel()
	content := []byte(`---
id: M-008
title: No-TDD milestone
status: draft
parent: E-03
acs:
  - id: AC-1
    title: Something to do
    status: open
---
`)
	got, err := Parse("work/epics/E-03/M-008.md", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(got.ACs) != 1 {
		t.Fatalf("len(ACs) = %d, want 1", len(got.ACs))
	}
	if got.ACs[0].TDDPhase != "" {
		t.Errorf("ACs[0].TDDPhase = %q, want empty string (absent)", got.ACs[0].TDDPhase)
	}
}

func TestParse_ContractFields(t *testing.T) {
	t.Parallel()
	content := []byte(`---
id: C-003
title: Orders API
status: accepted
linked_adrs:
  - ADR-0001
  - ADR-0002
---
`)
	got, err := Parse("work/contracts/C-003-orders/contract.md", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Status != "accepted" {
		t.Errorf("status = %q, want %q", got.Status, "accepted")
	}
	if diff := cmp.Diff([]string{"ADR-0001", "ADR-0002"}, got.LinkedADRs); diff != "" {
		t.Errorf("linked_adrs mismatch (-want +got):\n%s", diff)
	}
}

func TestParse_NoFrontmatter(t *testing.T) {
	t.Parallel()
	content := []byte("# Just a markdown file\n\nNo frontmatter here.\n")
	_, err := Parse("notes.md", content)
	if !errors.Is(err, ErrNoFrontmatter) {
		t.Errorf("err = %v, want ErrNoFrontmatter", err)
	}
}

func TestParse_UnclosedFrontmatter(t *testing.T) {
	t.Parallel()
	content := []byte("---\nid: E-01\ntitle: Foo\n")
	_, err := Parse("foo.md", content)
	if !errors.Is(err, ErrNoFrontmatter) {
		t.Errorf("err = %v, want ErrNoFrontmatter", err)
	}
}

func TestParse_UnknownField(t *testing.T) {
	t.Parallel()
	content := []byte(`---
id: E-01
title: Foo
status: active
mystery_field: nope
---
`)
	_, err := Parse("foo.md", content)
	if err == nil || !strings.Contains(err.Error(), "field mystery_field") {
		t.Errorf("err = %v, want a 'field mystery_field' error", err)
	}
}

func TestParse_BOMTolerant(t *testing.T) {
	t.Parallel()
	content := []byte("\xef\xbb\xbf---\nid: E-01\ntitle: Foo\nstatus: active\n---\n")
	got, err := Parse("foo.md", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.ID != "E-01" {
		t.Errorf("ID = %q, want E-01", got.ID)
	}
}

func TestParse_CRLFTolerant(t *testing.T) {
	t.Parallel()
	content := []byte("---\r\nid: E-01\r\ntitle: Foo\r\nstatus: active\r\n---\r\nbody\r\n")
	got, err := Parse("foo.md", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.ID != "E-01" {
		t.Errorf("ID = %q, want E-01", got.ID)
	}
}
