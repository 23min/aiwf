package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PolicyM0132DevcontainerReadme asserts that .devcontainer/README.md
// ships the four canonical operator-facing H2 sections — Build,
// Reopen in Container, Environment variables, Recovery prompt —
// each with non-empty body content.
//
// Pins M-0132/AC-5. Per CLAUDE.md's "substring assertions are not
// structural assertions" rule, the assertion walks the heading
// hierarchy and asserts each section by structural position; a
// canonical section title appearing inside a code fence or a
// different heading level doesn't count.
func PolicyM0132DevcontainerReadme(root string) ([]Violation, error) {
	const relPath = ".devcontainer/README.md"
	abs := filepath.Join(root, relPath)
	raw, err := os.ReadFile(abs)
	if err != nil {
		return []Violation{{
			Policy: "m0132-devcontainer-readme",
			File:   relPath,
			Detail: fmt.Sprintf("missing or unreadable: %v", err),
		}}, nil
	}

	wantSections := []string{
		"Build",
		"Reopen in Container",
		"Environment variables",
		"Recovery prompt",
	}
	wantSet := map[string]bool{}
	for _, s := range wantSections {
		wantSet[s] = true
	}

	// Walk lines: track code-fence state, collect H2 headings and the
	// body line counts between them.
	type section struct {
		title    string
		bodyLine int // first non-blank line under the heading, 0 if none
	}
	var sections []section
	var current *section
	inFence := false

	lines := strings.Split(string(raw), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Code-fence boundaries.
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		// H2 heading.
		if strings.HasPrefix(line, "## ") {
			title := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			s := section{title: title}
			sections = append(sections, s)
			current = &sections[len(sections)-1]
			continue
		}
		// Anything non-blank under the current section is "body content"
		// for our purposes (text, code-fence start lines that get
		// toggled below, list items, table rows).
		if current != nil && current.bodyLine == 0 {
			if trimmed != "" && !strings.HasPrefix(line, "# ") {
				current.bodyLine = i + 1
			}
		}
	}

	var vs []Violation
	report := func(detail string) {
		vs = append(vs, Violation{
			Policy: "m0132-devcontainer-readme",
			File:   relPath,
			Detail: detail,
		})
	}

	// Each canonical section must appear at H2 with non-empty body.
	foundTitles := map[string]*section{}
	for i := range sections {
		s := &sections[i]
		if wantSet[s.title] {
			foundTitles[s.title] = s
		}
	}
	var missing []string
	for _, want := range wantSections {
		s, ok := foundTitles[want]
		if !ok {
			missing = append(missing, want)
			continue
		}
		if s.bodyLine == 0 {
			report(fmt.Sprintf("section `## %s` has no body content (per CLAUDE.md, body sections are required, not optional)", want))
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		report(fmt.Sprintf("missing required H2 section(s): %s (canonical operator-facing sections per M-0132/AC-5)", strings.Join(missing, ", ")))
	}

	return vs, nil
}
