package policies

import (
	"os"
	"path/filepath"
	"strings"
)

// PolicyClaudeMdTestDisciplineSection asserts that CLAUDE.md
// contains a `### Test discipline` section under the `## Go
// conventions` parent heading. Walks the heading hierarchy so the
// section is anchored to the right place — a top-level `## Test
// discipline` (in the wrong scope) or a `### Test discipline` under
// a different parent heading fails this check.
//
// Pins M-0093/AC-1. The section content is the human-readable
// convention; PolicyTestSetupPresence (M-0093/AC-2) is the
// mechanical chokepoint that enforces what this section documents.
// Both ship together so a contributor seeing a CI failure from
// PolicyTestSetupPresence can find the explanation; and a future
// change that drops the section without updating the policy fires
// this check instead.
func PolicyClaudeMdTestDisciplineSection(root string) ([]Violation, error) {
	path := filepath.Join(root, "CLAUDE.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Walk the heading hierarchy: track the most recent ## heading
	// and look for `### Test discipline` only while inside `## Go
	// conventions`. Code-block boundaries are honored so a `###`
	// inside a fenced ```markdown``` example doesn't false-positive.
	inGoConventions := false
	inCodeFence := false
	foundSection := false
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "```") {
			inCodeFence = !inCodeFence
			continue
		}
		if inCodeFence {
			continue
		}
		switch {
		case strings.HasPrefix(line, "## "):
			inGoConventions = strings.TrimSpace(strings.TrimPrefix(line, "## ")) == "Go conventions"
		case strings.HasPrefix(line, "### ") && inGoConventions:
			if strings.TrimSpace(strings.TrimPrefix(line, "### ")) == "Test discipline" {
				foundSection = true
			}
		case strings.HasPrefix(line, "# "):
			// A top-level # heading would re-anchor the hierarchy.
			// CLAUDE.md uses a single top-level header at file
			// start; this branch is defensive for unusual edits.
			inGoConventions = false
		}
	}

	if !foundSection {
		return []Violation{{
			Policy: "claude-md-test-discipline-section",
			File:   "CLAUDE.md",
			Detail: "missing `### Test discipline` section under `## Go conventions` (M-0093/AC-1 documents the parallel-by-default + setup_test.go convention there; PolicyTestSetupPresence is the mechanical chokepoint that enforces it)",
		}}, nil
	}
	return nil, nil
}
