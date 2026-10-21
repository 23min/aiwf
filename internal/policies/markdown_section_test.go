package policies

import "strings"

// markdownSection returns the markdown body of the section opened by
// `heading` — i.e., everything between the heading line and the
// next heading at the same or higher level (fewer or equal `#`
// markers). Returns the empty string if the heading isn't found.
//
// Heading level is derived from the count of leading `#` chars in
// `heading`. The walker is line-based; it does not parse markdown
// content (no goldmark dependency).
func markdownSection(content, heading string) string {
	needle := "\n" + heading + "\n"
	idx := strings.Index(content, needle)
	if idx < 0 {
		if strings.HasPrefix(content, heading+"\n") {
			idx = 0
			needle = heading + "\n"
		} else {
			return ""
		}
	}
	bodyStart := idx + len(needle)

	// Count leading `#` chars in the heading to learn its level.
	headingLevel := 0
	for headingLevel < len(heading) && heading[headingLevel] == '#' {
		headingLevel++
	}
	if headingLevel == 0 {
		return ""
	}

	lines := strings.Split(content[bodyStart:], "\n")
	var bodyLines []string
	for _, line := range lines {
		trim := strings.TrimRight(line, " \t\r")
		if strings.HasPrefix(trim, "#") {
			level := 0
			for level < len(trim) && trim[level] == '#' {
				level++
			}
			if level > 0 && level <= 6 && level <= headingLevel {
				if level < len(trim) && trim[level] == ' ' {
					break
				}
			}
		}
		bodyLines = append(bodyLines, line)
	}
	return strings.Join(bodyLines, "\n")
}
