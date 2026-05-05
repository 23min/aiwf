package entity

import (
	"bytes"
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

// TestSlugifyDetailed reports both the slug and the runes that
// were dropped. The dropped list lets verbs surface a notice when
// a non-ASCII title silently loses characters in the slug.
func TestSlugifyDetailed(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		wantSlug    string
		wantDropped []rune
	}{
		{"pure ascii", "Hello World", "hello-world", nil},
		{"non-ascii single", "Café", "caf", []rune{'é'}},
		{"non-ascii multiple", "München-Frühling", "m-nchen-fr-hling", []rune{'ü', 'ü'}},
		{"all non-ascii drops to empty", "日本語", "", []rune{'日', '本', '語'}},
		{"empty input", "", "", nil},
		{"punctuation only is not dropped", "!!!", "", nil},
		{"mixed letters digits ascii", "Pi 3.14", "pi-3-14", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSlug, gotDropped := SlugifyDetailed(tt.title)
			if gotSlug != tt.wantSlug {
				t.Errorf("slug = %q, want %q", gotSlug, tt.wantSlug)
			}
			if diff := cmp.Diff(tt.wantDropped, gotDropped); diff != "" {
				t.Errorf("dropped mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestSlugify_StaysCompatibleWithSlugifyDetailed: the simple
// Slugify wrapper must agree with SlugifyDetailed's slug return.
func TestSlugify_StaysCompatibleWithSlugifyDetailed(t *testing.T) {
	for _, title := range []string{"Hello World", "Café", "München-Frühling", "", "日本語"} {
		want, _ := SlugifyDetailed(title)
		got := Slugify(title)
		if got != want {
			t.Errorf("Slugify(%q) = %q, SlugifyDetailed slug = %q", title, got, want)
		}
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
	if !bytes.Equal(body, body2) {
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

// TestSerialize_RoundTripACsAndTDD confirms a milestone carrying I2's
// new fields (`tdd:` policy and `acs[]` with per-AC `tdd_phase`)
// survives marshal+unmarshal without losing or reordering data. The
// inner `tdd_phase` field is `omitempty`; an AC without a phase round-
// trips with an empty string, not a nil-vs-empty distinction.
func TestSerialize_RoundTripACsAndTDD(t *testing.T) {
	original := []byte(`---
id: M-007
title: Engine warning surface
status: in_progress
parent: E-03
tdd: required
acs:
    - id: AC-1
      title: Engine emits warning
      status: open
      tdd_phase: red
    - id: AC-2
      title: Pack receives result
      status: met
      tdd_phase: done
---

## Goal
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

	e2, err := Parse("test.md", out)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if diff := cmp.Diff(e, e2); diff != "" {
		t.Errorf("entity round-trip mismatch (-want +got):\n%s", diff)
	}
}

// TestSerialize_OmitsAbsentTDDPhase covers the empty-string sentinel:
// when an AC has no phase, `tdd_phase` must not appear in the serialized
// YAML (otherwise we'd write `tdd_phase: ""` which conflicts with the
// closed-set membership rule).
func TestSerialize_OmitsAbsentTDDPhase(t *testing.T) {
	e := &Entity{
		ID:     "M-008",
		Title:  "No-TDD",
		Status: "draft",
		Parent: "E-03",
		ACs: []AcceptanceCriterion{
			{ID: "AC-1", Title: "Something", Status: "open"},
		},
	}
	out, err := Serialize(e, []byte("\nbody\n"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out, []byte("tdd_phase")) {
		t.Errorf("serialized output should omit tdd_phase when empty:\n%s", out)
	}
	if bytes.Contains(out, []byte("\ntdd:")) {
		t.Errorf("serialized output should omit tdd: when empty:\n%s", out)
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
