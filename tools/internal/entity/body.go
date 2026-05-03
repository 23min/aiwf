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
