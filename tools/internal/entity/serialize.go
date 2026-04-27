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
// Non-ASCII characters are dropped (e.g., "Café" becomes "caf"). Tighten
// to a Unicode-aware mapping later if real consumers ask for it; the
// PoC's audience is ASCII titles.
func Slugify(title string) string {
	var b strings.Builder
	lastWasHyphen := true
	for _, r := range strings.ToLower(title) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastWasHyphen = false
		default:
			if !lastWasHyphen {
				b.WriteByte('-')
				lastWasHyphen = true
			}
		}
	}
	return strings.TrimRight(b.String(), "-")
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
