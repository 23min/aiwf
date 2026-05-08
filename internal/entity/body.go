package entity

import (
	"bytes"
	"regexp"
	"strings"
)

// ParseBodySections walks `## `-level headings in body and returns a
// map from slugified heading to the section's prose, trimmed of leading
// and trailing whitespace. Sub-headings (`### `, `#### `, …) and prose
// inside the section are returned verbatim under the parent `## `.
//
// The slug lowercases the heading, replaces every run of non-alphanumeric
// runes with a single `_`, and trims leading/trailing `_`. So:
//
//	## Goal             → "goal"
//	## Out of scope     → "out_of_scope"
//	## What's missing   → "what_s_missing"
//
// Multiple `## ` sections with the same slug collapse to the last one
// (last write wins). Body content before the first `## ` is dropped —
// the templates always start with a `## ` heading. Returns nil when no
// `## ` heading is present.
//
// This is the read-side companion to BodyTemplate. It is intentionally
// not a full markdown parser: only `## ` boundaries matter, and a `# `
// or EOF terminates the current section.
func ParseBodySections(body []byte) map[string]string {
	if len(body) == 0 {
		return nil
	}
	lines := bytes.Split(body, []byte("\n"))
	out := map[string]string{}
	var currentSlug string
	var currentBuf []string
	flush := func() {
		if currentSlug == "" {
			return
		}
		out[currentSlug] = strings.TrimSpace(strings.Join(currentBuf, "\n"))
	}
	for _, line := range lines {
		switch {
		case bytes.HasPrefix(line, []byte("## ")):
			flush()
			currentSlug = SectionSlug(string(line[len("## "):]))
			currentBuf = nil
		case bytes.HasPrefix(line, []byte("# ")):
			// Top-level heading terminates the current `## ` section.
			flush()
			currentSlug = ""
			currentBuf = nil
		default:
			if currentSlug != "" {
				currentBuf = append(currentBuf, string(line))
			}
		}
	}
	flush()
	if len(out) == 0 {
		return nil
	}
	return out
}

// BodySection is one `## ` section with its display heading
// preserved alongside the slugified key. Heading is what the
// markdown source actually wrote (no slug round-trip — apostrophes,
// spaces, and capitalization come back verbatim); Slug is the
// ParseBodySections key; Content is the section prose, trimmed.
//
// Used by ParseBodySectionsOrdered when the caller cares about
// document order (the HTML renderer per G35/G36) instead of just
// the slug → content map.
type BodySection struct {
	Slug    string
	Heading string
	Content string
}

// ParseBodySectionsOrdered walks `## `-level headings in body and
// returns the sections in source order, with each section's display
// heading preserved alongside the slug. Same semantics as
// ParseBodySections for heading boundaries (`# ` or EOF terminates
// the current `## ` section; content before the first `## ` is
// dropped); duplicate slugs collapse to the last occurrence so
// callers can rely on slug uniqueness.
//
// Returns nil when body is empty or has no `## ` heading.
func ParseBodySectionsOrdered(body []byte) []BodySection {
	if len(body) == 0 {
		return nil
	}
	lines := bytes.Split(body, []byte("\n"))
	indexBySlug := map[string]int{}
	var sections []BodySection
	currentIdx := -1
	var currentBuf []string
	flush := func() {
		if currentIdx < 0 {
			return
		}
		sections[currentIdx].Content = strings.TrimSpace(strings.Join(currentBuf, "\n"))
	}
	for _, line := range lines {
		switch {
		case bytes.HasPrefix(line, []byte("## ")):
			flush()
			heading := strings.TrimSpace(string(line[len("## "):]))
			slug := SectionSlug(heading)
			if existing, ok := indexBySlug[slug]; ok {
				// Duplicate slug: rewrite the existing entry so
				// last-write-wins matches ParseBodySections.
				currentIdx = existing
				sections[currentIdx].Heading = heading
			} else {
				sections = append(sections, BodySection{Slug: slug, Heading: heading})
				currentIdx = len(sections) - 1
				indexBySlug[slug] = currentIdx
			}
			currentBuf = nil
		case bytes.HasPrefix(line, []byte("# ")):
			flush()
			currentIdx = -1
			currentBuf = nil
		default:
			if currentIdx >= 0 {
				currentBuf = append(currentBuf, string(line))
			}
		}
	}
	flush()
	if len(sections) == 0 {
		return nil
	}
	return sections
}

// sectionSlugReplaceRE matches every run of non-alphanumeric runes; we
// collapse each run to a single `_` to avoid double-underscore slugs
// for headings like `What's missing` (apostrophe + space).
var sectionSlugReplaceRE = regexp.MustCompile(`[^a-z0-9]+`)

// SectionSlug derives the body-section key from a `## ` heading. See
// ParseBodySections for the rule.
func SectionSlug(heading string) string {
	s := strings.ToLower(strings.TrimSpace(heading))
	s = sectionSlugReplaceRE.ReplaceAllString(s, "_")
	return strings.Trim(s, "_")
}

// acHeadingRE matches the `### AC-N — <title>` headings that
// `aiwf add ac` writes inside a milestone body. The `—` is the em-dash
// the writer emits; we accept either em-dash or `--` for forward-
// compat with hand-edited files. Anchored at start-of-line; the
// matched group 1 is the AC id.
var acHeadingRE = regexp.MustCompile(`^### (AC-\d+)\b`)

// ParseACSections walks `### AC-N` headings in body and returns a map
// from AC id to that section's prose, trimmed of leading and trailing
// whitespace. The heading line itself is not included; only the prose
// under it. A subsequent `### AC-N` heading or any `## ` / `# ` heading
// terminates the current section.
//
// Unrecognized `### ` headings (anything that does not start with
// `AC-<digits>`) are skipped — the section's body is not captured. This
// keeps the parser predictable when authors add free-form `### ` notes
// outside the AC structure. Returns nil when no `### AC-N` heading is
// present.
func ParseACSections(body []byte) map[string]string {
	if len(body) == 0 {
		return nil
	}
	lines := bytes.Split(body, []byte("\n"))
	out := map[string]string{}
	var currentID string
	var currentBuf []string
	flush := func() {
		if currentID == "" {
			return
		}
		out[currentID] = strings.TrimSpace(strings.Join(currentBuf, "\n"))
	}
	for _, line := range lines {
		switch {
		case bytes.HasPrefix(line, []byte("### ")):
			flush()
			if m := acHeadingRE.FindSubmatch(line); m != nil {
				currentID = string(m[1])
				currentBuf = nil
			} else {
				currentID = ""
				currentBuf = nil
			}
		case bytes.HasPrefix(line, []byte("## ")), bytes.HasPrefix(line, []byte("# ")):
			flush()
			currentID = ""
			currentBuf = nil
		default:
			if currentID != "" {
				currentBuf = append(currentBuf, string(line))
			}
		}
	}
	flush()
	if len(out) == 0 {
		return nil
	}
	return out
}
