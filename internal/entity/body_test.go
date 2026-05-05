package entity

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSectionSlug(t *testing.T) {
	cases := []struct {
		heading, want string
	}{
		{"Goal", "goal"},
		{"Out of scope", "out_of_scope"},
		{"What's missing", "what_s_missing"},
		{"  Padded heading  ", "padded_heading"},
		{"Question?", "question"},
		{"AC-1 description", "ac_1_description"},
		{"___", ""},
	}
	for _, tc := range cases {
		t.Run(tc.heading, func(t *testing.T) {
			if got := SectionSlug(tc.heading); got != tc.want {
				t.Errorf("SectionSlug(%q) = %q, want %q", tc.heading, got, tc.want)
			}
		})
	}
}

func TestParseBodySections_KindTemplates(t *testing.T) {
	// Cover every kind's BodyTemplate by parsing a populated version
	// of each template — the load-bearing case is "the slugs the
	// renderer relies on are the slugs the parser produces."
	cases := map[Kind]struct {
		body    string
		want    map[string]string
		wantNil bool
	}{
		KindEpic: {
			body: "\n## Goal\n\nthe goal\n\n## Scope\n\nin scope\n\n## Out of scope\n\nout of it\n",
			want: map[string]string{
				"goal":         "the goal",
				"scope":        "in scope",
				"out_of_scope": "out of it",
			},
		},
		KindMilestone: {
			body: "\n## Goal\n\nship it\n\n## Acceptance criteria\n\n### AC-1 — first\nfirst body\n",
			want: map[string]string{
				"goal":                "ship it",
				"acceptance_criteria": "### AC-1 — first\nfirst body",
			},
		},
		KindADR: {
			body: "\n## Context\n\nbecause\n\n## Decision\n\ndo this\n\n## Consequences\n\nfine\n",
			want: map[string]string{
				"context":      "because",
				"decision":     "do this",
				"consequences": "fine",
			},
		},
		KindGap: {
			body: "\n## What's missing\n\nthing\n\n## Why it matters\n\nbreaks stuff\n",
			want: map[string]string{
				"what_s_missing": "thing",
				"why_it_matters": "breaks stuff",
			},
		},
		KindDecision: {
			body: "\n## Question\n\nq?\n\n## Decision\n\nyes\n\n## Reasoning\n\nbecause\n",
			want: map[string]string{
				"question":  "q?",
				"decision":  "yes",
				"reasoning": "because",
			},
		},
		KindContract: {
			body: "\n## Purpose\n\nfor things\n\n## Stability\n\nstable\n",
			want: map[string]string{
				"purpose":   "for things",
				"stability": "stable",
			},
		},
	}
	for k, tc := range cases {
		t.Run(string(k), func(t *testing.T) {
			got := ParseBodySections([]byte(tc.body))
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ParseBodySections(%s) mismatch (-want +got):\n%s", k, diff)
			}
		})
	}
}

func TestParseBodySections_EdgeCases(t *testing.T) {
	t.Run("empty body returns nil", func(t *testing.T) {
		if got := ParseBodySections(nil); got != nil {
			t.Errorf("ParseBodySections(nil) = %v, want nil", got)
		}
		if got := ParseBodySections([]byte("")); got != nil {
			t.Errorf("ParseBodySections(\"\") = %v, want nil", got)
		}
	})
	t.Run("body with no level-2 heading returns nil", func(t *testing.T) {
		got := ParseBodySections([]byte("\n# top\n\nprose only\n"))
		if got != nil {
			t.Errorf("ParseBodySections(prose) = %v, want nil", got)
		}
	})
	t.Run("prose before first heading is dropped", func(t *testing.T) {
		got := ParseBodySections([]byte("ignore me\n\n## Goal\n\nbody\n"))
		want := map[string]string{"goal": "body"}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
	t.Run("level-1 heading terminates current section", func(t *testing.T) {
		got := ParseBodySections([]byte("## Goal\n\nbody\n# Aside\n\nignored\n## Scope\n\nthen this\n"))
		want := map[string]string{"goal": "body", "scope": "then this"}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
	t.Run("duplicate slugs collapse to last", func(t *testing.T) {
		got := ParseBodySections([]byte("## Goal\n\nfirst\n\n## Goal\n\nsecond\n"))
		want := map[string]string{"goal": "second"}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestParseACSections(t *testing.T) {
	body := []byte(`
## Goal

ship it

## Acceptance criteria

### AC-1 — first AC

first body

with a blank line

### AC-2 — second AC

second body
`)
	got := ParseACSections(body)
	want := map[string]string{
		"AC-1": "first body\n\nwith a blank line",
		"AC-2": "second body",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ParseACSections mismatch (-want +got):\n%s", diff)
	}
}

func TestParseACSections_EdgeCases(t *testing.T) {
	t.Run("no AC headings returns nil", func(t *testing.T) {
		got := ParseACSections([]byte("## Goal\n\nbody\n"))
		if got != nil {
			t.Errorf("ParseACSections(no AC) = %v, want nil", got)
		}
	})
	t.Run("empty body returns nil", func(t *testing.T) {
		if got := ParseACSections(nil); got != nil {
			t.Errorf("ParseACSections(nil) = %v, want nil", got)
		}
	})
	t.Run("non-AC h3 headings are ignored", func(t *testing.T) {
		body := []byte("### Notes\n\nfree-form\n\n### AC-1 — only AC\n\nthe body\n")
		want := map[string]string{"AC-1": "the body"}
		got := ParseACSections(body)
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
	t.Run("level-2 heading terminates AC section", func(t *testing.T) {
		body := []byte("### AC-1 — first\n\nin AC-1\n\n## Other section\n\nignored\n\n### AC-2 — second\n\nin AC-2\n")
		want := map[string]string{"AC-1": "in AC-1", "AC-2": "in AC-2"}
		got := ParseACSections(body)
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
}
