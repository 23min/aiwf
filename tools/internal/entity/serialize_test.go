package entity

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{"Cache warmup", "cache-warmup"},
		{"  Many   spaces  ", "many-spaces"},
		{"Mixed-CASE_Title!", "mixed-case-title"},
		{"Already-kebab-case", "already-kebab-case"},
		{"Numbers 123 included", "numbers-123-included"},
		{"Punctuation: yes! and: no?", "punctuation-yes-and-no"},
		{"---hyphen-prefix", "hyphen-prefix"},
		{"trailing---", "trailing"},
		{"", ""},
		{"Café au lait", "caf-au-lait"}, // non-ASCII dropped
	}
	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			if got := Slugify(tt.title); got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.title, got, tt.want)
			}
		})
	}
}

func TestBodyTemplate_Sections(t *testing.T) {
	expectedSections := map[Kind][]string{
		KindEpic:      {"## Goal", "## Scope", "## Out of scope"},
		KindMilestone: {"## Goal", "## Acceptance criteria"},
		KindADR:       {"## Context", "## Decision", "## Consequences"},
		KindGap:       {"## What's missing", "## Why it matters"},
		KindDecision:  {"## Question", "## Decision", "## Reasoning"},
		KindContract:  {"## Purpose", "## Stability"},
	}
	for k, sections := range expectedSections {
		t.Run(string(k), func(t *testing.T) {
			body := string(BodyTemplate(k))
			for _, s := range sections {
				if !strings.Contains(body, s) {
					t.Errorf("BodyTemplate(%s) missing section %q\nbody: %q", k, s, body)
				}
			}
		})
	}
}

func TestSplit_RoundTrip(t *testing.T) {
	original := []byte(`---
id: M-007
title: Cache warmup
status: in_progress
parent: E-01
---

## Goal

Build a cache warmer.

## Acceptance criteria

It warms the cache.
`)
	fm, body, ok := Split(original)
	if !ok {
		t.Fatal("Split returned !ok")
	}
	if !strings.Contains(string(fm), "id: M-007") {
		t.Errorf("frontmatter missing id: %q", fm)
	}
	if !strings.HasPrefix(string(body), "\n## Goal") {
		t.Errorf("body should start with leading newline + ## Goal, got: %q", body)
	}
}

func TestSplit_NoFrontmatter(t *testing.T) {
	_, _, ok := Split([]byte("# Just markdown\n"))
	if ok {
		t.Error("Split should fail without frontmatter")
	}
}

func TestSerialize_RoundTrip(t *testing.T) {
	original := []byte(`---
id: M-007
title: Cache warmup
status: in_progress
parent: E-01
depends_on:
    - M-002
    - M-005
---

## Goal

Build a cache warmer.
`)
	e, err := Parse("test.md", original)
	if err != nil {
		t.Fatal(err)
	}
	_, body, _ := Split(original)

	out, err := Serialize(e, body)
	if err != nil {
		t.Fatal(err)
	}

	// Re-parse the serialized form; entity should equal the original.
	e2, err := Parse("test.md", out)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if diff := cmp.Diff(e, e2); diff != "" {
		t.Errorf("entity round-trip mismatch (-want +got):\n%s", diff)
	}

	// Body should also round-trip identically.
	_, body2, _ := Split(out)
	if string(body) != string(body2) {
		t.Errorf("body mismatch:\nwant: %q\ngot:  %q", body, body2)
	}
}

func TestSerialize_ModifyAndWrite(t *testing.T) {
	original := []byte(`---
id: M-007
title: Cache warmup
status: draft
parent: E-01
---

body unchanged
`)
	e, _ := Parse("test.md", original)
	_, body, _ := Split(original)

	// Promote.
	e.Status = "in_progress"

	out, err := Serialize(e, body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "status: in_progress") {
		t.Errorf("missing new status: %q", out)
	}
	if !strings.Contains(string(out), "body unchanged") {
		t.Errorf("body lost: %q", out)
	}
}

func TestSerialize_EmptyBodyForNewEntity(t *testing.T) {
	e := &Entity{
		ID:     "E-01",
		Title:  "Foundations",
		Status: "active",
		Kind:   KindEpic,
	}
	out, err := Serialize(e, BodyTemplate(KindEpic))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(out), "---\n") {
		t.Errorf("missing opening delimiter: %q", out)
	}
	if !strings.Contains(string(out), "## Goal") {
		t.Errorf("missing template section: %q", out)
	}
	// Round-trip parse confirms shape.
	if _, err := Parse("E-01.md", out); err != nil {
		t.Errorf("round-trip parse: %v", err)
	}
}
