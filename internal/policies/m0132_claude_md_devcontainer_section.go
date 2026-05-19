package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PolicyM0132ClaudeMdDevcontainerSection asserts that CLAUDE.md's
// `## Operator setup` section contains a `### Devcontainer`
// subsection whose body cites the claude-code#31388 URL and names
// the "shadow-mount" concept.
//
// Per CLAUDE.md's "substring assertions are not structural
// assertions" rule, the URL and concept literals must appear inside
// the subsection's body — not anywhere else in CLAUDE.md. A
// future change that puts the URL in some other section while
// removing the subsection-scoped reference still fires this check.
//
// Pins M-0132/AC-6.
func PolicyM0132ClaudeMdDevcontainerSection(root string) ([]Violation, error) {
	const relPath = "CLAUDE.md"
	abs := filepath.Join(root, relPath)
	raw, err := os.ReadFile(abs)
	if err != nil {
		return []Violation{{
			Policy: "m0132-claude-md-devcontainer-section",
			File:   relPath,
			Detail: fmt.Sprintf("missing or unreadable: %v", err),
		}}, nil
	}

	// Walk the heading hierarchy: track the most recent ## heading
	// and the most recent ### heading under it; collect body lines
	// under the targeted (## Operator setup, ### Devcontainer) span.
	lines := strings.Split(string(raw), "\n")
	inH2OperatorSetup := false
	inH3Devcontainer := false
	inFence := false
	var bodyLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Code-fence boundaries (honored inside Devcontainer body).
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			if inH3Devcontainer {
				bodyLines = append(bodyLines, line)
			}
			inFence = !inFence
			continue
		}
		if inFence {
			if inH3Devcontainer {
				bodyLines = append(bodyLines, line)
			}
			continue
		}
		switch {
		case strings.HasPrefix(line, "# "):
			// A top-level # heading would re-anchor the hierarchy.
			inH2OperatorSetup = false
			inH3Devcontainer = false
		case strings.HasPrefix(line, "## "):
			inH2OperatorSetup = strings.TrimSpace(strings.TrimPrefix(line, "## ")) == "Operator setup"
			inH3Devcontainer = false
		case strings.HasPrefix(line, "### "):
			if inH2OperatorSetup {
				inH3Devcontainer = strings.TrimSpace(strings.TrimPrefix(line, "### ")) == "Devcontainer"
			} else {
				inH3Devcontainer = false
			}
		default:
			if inH3Devcontainer {
				bodyLines = append(bodyLines, line)
			}
		}
	}

	var vs []Violation
	report := func(detail string) {
		vs = append(vs, Violation{
			Policy: "m0132-claude-md-devcontainer-section",
			File:   relPath,
			Detail: detail,
		})
	}

	if len(bodyLines) == 0 {
		report("missing `### Devcontainer` subsection under `## Operator setup` (M-0132/AC-6 requires the shadow-mount workaround documented in prose adjacent to the existing plugin-install instructions)")
		return vs, nil
	}

	body := strings.Join(bodyLines, "\n")
	hasNonBlank := false
	for _, l := range bodyLines {
		if strings.TrimSpace(l) != "" {
			hasNonBlank = true
			break
		}
	}
	if !hasNonBlank {
		report("`### Devcontainer` subsection has no body content (header without prose; readers can't pick up the workaround context)")
	}

	const wantURL = "https://github.com/anthropics/claude-code/issues/31388"
	if !strings.Contains(body, wantURL) {
		report(fmt.Sprintf("`### Devcontainer` subsection body missing %s (URL must appear in the subsection, not elsewhere in CLAUDE.md)", wantURL))
	}

	const wantConcept1 = "shadow-mount"
	const wantConcept2 = "plugin index shadow"
	if !strings.Contains(body, wantConcept1) && !strings.Contains(body, wantConcept2) {
		report(fmt.Sprintf("`### Devcontainer` subsection body missing the concept name (%q or %q) — readers shouldn't need to chase the upstream issue to understand what the workaround is", wantConcept1, wantConcept2))
	}

	return vs, nil
}
