package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// documentationHierarchyTierBulletRegex matches a top-level tier
// bullet of the `## Documentation hierarchy` section's shape:
// `- **<Tier Name>** — <body>`. The tier name is whatever sits
// between the bold markers, letting the policy validate it against
// the closed set rather than assuming the exact wording up front.
var documentationHierarchyTierBulletRegex = regexp.MustCompile(`^- \*\*([^*]+)\*\*`)

// documentationHierarchyClosedTiers is the closed-set vocabulary
// AC-1 requires (M-0128): every tier bullet under the section must
// name one of these four, and all four must appear at least once.
var documentationHierarchyClosedTiers = map[string]bool{
	"Normative":       true,
	"Forward-looking": true,
	"Exploratory":     true,
	"Archival":        true,
}

// documentationHierarchySubtrees is the fixed, post-M-0127 set of
// active docs/ subtrees the section must account for. Per the
// epic's own out-of-scope note, this is a snapshot assertion against
// the layout as it exists today, not a live drift check against
// docs/'s actual contents — that's G-0092's deferred kernel-rule
// follow-on.
var documentationHierarchySubtrees = []string{
	"docs/adr/",
	"docs/design/",
	"docs/explorations/",
	"docs/research/",
	"docs/initiatives/",
	"docs/migration/",
	"docs/archive/",
}

// PolicyM0128DocumentationHierarchy asserts CLAUDE.md carries a
// `## Documentation hierarchy` section (per CLAUDE.md "Substring
// assertions are not structural assertions", scoped to that named
// section, not the file at large) that:
//
//   - names every active docs/ subtree in documentationHierarchySubtrees;
//   - tags each tier bullet with a name from the closed
//     documentationHierarchyClosedTiers set;
//   - covers all four tiers at least once.
//
// Pins M-0128/AC-1.
func PolicyM0128DocumentationHierarchy(root string) ([]Violation, error) {
	const relPath = "CLAUDE.md"
	const heading = "## Documentation hierarchy"
	abs := filepath.Join(root, relPath)

	var vs []Violation
	report := func(detail string) {
		vs = append(vs, Violation{
			Policy: "m0128-documentation-hierarchy",
			File:   relPath,
			Detail: detail,
		})
	}

	raw, err := os.ReadFile(abs)
	if err != nil {
		report(fmt.Sprintf("ReadFile failed: %v", err))
		return vs, nil
	}
	content := string(raw)

	body := markdownSection(content, heading)
	if body == "" {
		report(fmt.Sprintf("missing section heading %q", heading))
		return vs, nil
	}

	for _, subtree := range documentationHierarchySubtrees {
		if !strings.Contains(body, subtree) {
			report(fmt.Sprintf("section body does not mention active subtree %q", subtree))
		}
	}

	seenTiers := map[string]bool{}
	for _, line := range strings.Split(body, "\n") {
		m := documentationHierarchyTierBulletRegex.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		tier := m[1]
		seenTiers[tier] = true
		if !documentationHierarchyClosedTiers[tier] {
			report(fmt.Sprintf("tier bullet %q is not one of the closed-set tiers (normative / forward-looking / exploratory / archival)", tier))
		}
	}

	for tier := range documentationHierarchyClosedTiers {
		if !seenTiers[tier] {
			report(fmt.Sprintf("closed-set tier %q has no bullet in the section", tier))
		}
	}

	return vs, nil
}
