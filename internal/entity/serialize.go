package entity

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Split returns the YAML frontmatter bytes and the markdown body bytes
// from a complete entity file. Returns (nil, nil, false) when the input
// has no frontmatter delimiter, mirroring Parse's tolerance.
//
// Mutating verbs use Split to preserve body prose during frontmatter
// edits: read → split → modify entity → Serialize(entity, body) →
// write.
func Split(content []byte) (frontmatter, body []byte, ok bool) {
	content = bytes.TrimPrefix(content, []byte("\xef\xbb\xbf"))

	if !bytes.HasPrefix(content, []byte("---\n")) && !bytes.HasPrefix(content, []byte("---\r\n")) {
		return nil, nil, false
	}

	nl := bytes.IndexByte(content, '\n') + 1
	rest := content[nl:]

	var fm bytes.Buffer
	for {
		idx := bytes.IndexByte(rest, '\n')
		var line []byte
		if idx < 0 {
			line = rest
			rest = nil
		} else {
			line = rest[:idx]
			rest = rest[idx+1:]
		}
		if bytes.Equal(bytes.TrimRight(line, "\r"), []byte("---")) {
			return fm.Bytes(), rest, true
		}
		fm.Write(line)
		if idx < 0 {
			return nil, nil, false
		}
		fm.WriteByte('\n')
	}
}

// Serialize composes an entity file's bytes: the opening "---" line,
// the entity's YAML frontmatter, the closing "---" line, and the body
// bytes verbatim. Use Split's body output as the body argument when
// editing an existing file, or BodyTemplate(kind) for newly-created
// entities.
//
// Field order in the YAML follows the Entity struct definition: id,
// title, status, then per-kind fields (which appear only when set,
// thanks to `omitempty`). This makes output deterministic across runs.
func Serialize(e *Entity, body []byte) ([]byte, error) {
	yamlBytes, err := yaml.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("marshaling frontmatter: %w", err)
	}
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(yamlBytes)
	buf.WriteString("---\n")
	buf.Write(body)
	return buf.Bytes(), nil
}

// Slugify converts a title into a kebab-case slug suitable for use in
// filenames and directory names. Lowercases ASCII letters, keeps digits,
// collapses runs of non-alphanumerics into single hyphens, and trims
// leading and trailing hyphens.
//
// Non-ASCII characters are dropped (e.g., "Café" becomes "caf"). Use
// SlugifyDetailed if you need to know which characters were dropped
// so the user can be warned.
func Slugify(title string) string {
	slug, _ := SlugifyDetailed(title)
	return slug
}

// SlugifyDetailed is Slugify plus the list of input runes that were
// silently dropped because they were non-ASCII letters/digits. The
// dropped list is empty when the title is purely ASCII (or contained
// only ASCII alphanumerics + punctuation that legitimately collapses
// to hyphens). Callers in the verb dispatcher use the dropped list to
// surface a one-line notice to the user.
func SlugifyDetailed(title string) (slug string, dropped []rune) {
	var b strings.Builder
	lastWasHyphen := true
	for _, r := range strings.ToLower(title) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastWasHyphen = false
		default:
			if isMeaningfulNonASCII(r) {
				dropped = append(dropped, r)
			}
			if !lastWasHyphen {
				b.WriteByte('-')
				lastWasHyphen = true
			}
		}
	}
	return strings.TrimRight(b.String(), "-"), dropped
}

// isMeaningfulNonASCII flags runes whose loss would surprise the
// user — non-ASCII letters and digits. Pure punctuation runs that
// collapse to hyphens (e.g., "!!!", "---") are intentional and not
// reported as dropped.
func isMeaningfulNonASCII(r rune) bool {
	return r > 127
}

// ValidateTitle reports whether title is within the consumer's
// configured length cap (`entities.title_max_length` in aiwf.yaml;
// kernel default 80, per G-0102). Returns nil when title is
// acceptable, or a typed error explaining the cap and pointing the
// operator at `--body-file` for elaboration that doesn't belong in
// the title.
//
// A non-positive maxLength is a no-op (validation always passes), so
// callers in tests or paths that don't thread a config can pass 0.
//
// The same cap also applies to slugs (see ValidateSlug) — title and
// slug share a length budget so on-disk filenames and frontmatter
// titles stay in sync. This is what makes the kernel surfaces
// (CLI tables, HTML render, git-log subjects, `aiwf history`,
// filesystem) all degrade uniformly rather than diverging.
func ValidateTitle(title string, maxLength int) error {
	if maxLength <= 0 {
		return nil
	}
	if len(title) <= maxLength {
		return nil
	}
	return fmt.Errorf(
		"title length %d exceeds the configured cap of %d (entities.title_max_length); "+
			"shorten the title and put elaboration in the entity body (`--body-file` at create time, or `aiwf edit-body` after)",
		len(title), maxLength,
	)
}

// ValidateSlug reports whether slug is within the configured length
// cap. Same cap as ValidateTitle — title and slug share a budget so
// filenames and frontmatter stay in sync (G-0102). Used by
// `aiwf rename`, where the operator supplies a slug directly with no
// title context.
//
// A non-positive maxLength is a no-op. Validation runs on the
// post-slugify form (after `SlugifyDetailed`) because that's what
// ends up on disk; pre-slugify length may include characters
// (whitespace, punctuation) that get dropped.
func ValidateSlug(slug string, maxLength int) error {
	if maxLength <= 0 {
		return nil
	}
	if len(slug) <= maxLength {
		return nil
	}
	return fmt.Errorf(
		"slug length %d exceeds the configured cap of %d (entities.title_max_length); "+
			"choose a shorter slug",
		len(slug), maxLength,
	)
}

// BodyTemplate returns the per-kind starter body that `aiwf add`
// writes after the frontmatter. Sections are scaffolds; bodies are
// not validated by `aiwf check`.
func BodyTemplate(k Kind) []byte {
	switch k {
	case KindEpic:
		return []byte("\n## Goal\n\n## Scope\n\n## Out of scope\n")
	case KindMilestone:
		return []byte("\n## Goal\n\n## Acceptance criteria\n")
	case KindADR:
		return []byte("\n## Context\n\n## Decision\n\n## Consequences\n")
	case KindGap:
		return []byte("\n## What's missing\n\n## Why it matters\n")
	case KindDecision:
		return []byte("\n## Question\n\n## Decision\n\n## Reasoning\n")
	case KindContract:
		return []byte("\n## Purpose\n\n## Stability\n")
	}
	return []byte("\n")
}
