package policies

import (
	"bufio"
	"strings"
)

// extractMarkdownSubsection isolates the prose under a `### child`
// heading nested inside a `## parent` heading. Matching is
// heading-text-exact after trimming, and case-sensitive. Returns the
// child section's body (excluding both heading lines) and true when
// found; ("", false) when the parent or the nested child heading is
// absent. The child ends at the next `###`, `##` or `#` heading.
//
// entity.ParseBodySections (the kernel's own markdown-section reader)
// only tracks `## ` boundaries — everything under a `## ` heading,
// including its `### ` subsections, folds into one blob. That is
// correct for entity bodies, which never nest headings, but a caller
// needing exactly one subsection's own prose needs this instead.
func extractMarkdownSubsection(doc, parentHeading, childHeading string) (string, bool) {
	scanner := bufio.NewScanner(strings.NewReader(doc))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	inParent := false
	inChild := false
	var buf strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "## "):
			if inChild {
				return strings.TrimSpace(buf.String()), true
			}
			inParent = strings.TrimSpace(strings.TrimPrefix(line, "## ")) == parentHeading
			inChild = false
		case strings.HasPrefix(line, "### "):
			if inChild {
				return strings.TrimSpace(buf.String()), true
			}
			inChild = inParent && strings.TrimSpace(strings.TrimPrefix(line, "### ")) == childHeading
		case strings.HasPrefix(line, "# "):
			if inChild {
				return strings.TrimSpace(buf.String()), true
			}
			inParent = false
			inChild = false
		default:
			if inChild {
				buf.WriteString(line)
				buf.WriteString("\n")
			}
		}
	}
	if inChild {
		return strings.TrimSpace(buf.String()), true
	}
	return "", false
}
