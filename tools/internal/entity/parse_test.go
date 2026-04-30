package entity

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParse_Minimal(t *testing.T) {
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

func TestParse_ContractFields(t *testing.T) {
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
	content := []byte("# Just a markdown file\n\nNo frontmatter here.\n")
	_, err := Parse("notes.md", content)
	if !errors.Is(err, ErrNoFrontmatter) {
		t.Errorf("err = %v, want ErrNoFrontmatter", err)
	}
}

func TestParse_UnclosedFrontmatter(t *testing.T) {
	content := []byte("---\nid: E-01\ntitle: Foo\n")
	_, err := Parse("foo.md", content)
	if !errors.Is(err, ErrNoFrontmatter) {
		t.Errorf("err = %v, want ErrNoFrontmatter", err)
	}
}

func TestParse_UnknownField(t *testing.T) {
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
	content := []byte("---\r\nid: E-01\r\ntitle: Foo\r\nstatus: active\r\n---\r\nbody\r\n")
	got, err := Parse("foo.md", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.ID != "E-01" {
		t.Errorf("ID = %q, want E-01", got.ID)
	}
}
