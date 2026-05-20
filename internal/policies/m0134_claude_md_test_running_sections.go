package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PolicyM0134ClaudeMdTestRunningSections asserts that CLAUDE.md's
// `## Go conventions → ### Testing` area carries the post-M-0134
// shape: a devcontainer-primary subsection precedes a macOS-host
// fallback subsection, the body content under each names the
// expected concepts (Linux + no-wrapper claim on the primary;
// sign-and-run.sh + make test + G-0127/G-0128/G-0133 diagnostic
// ids on the fallback), the "Defaults, not a chokepoint" caveat
// is scoped to the fallback block (not floating or in the primary
// block), and the stale "Structural fix (Linux devcontainer) is
// parked." phrase is absent (it shipped in M-0132).
//
// Assertions resolve to specific parsed sub-trees under each
// named heading (line-scan walker, not a real markdown parser —
// goldmark deferred per M-0134 scoping conversation). Per
// CLAUDE.md *"Substring assertions are not structural
// assertions"* the substring matches for body contents are
// scoped to the section body, not the file as a whole.
//
// Pins M-0134/AC-1.
func PolicyM0134ClaudeMdTestRunningSections(root string) ([]Violation, error) {
	const relPath = "CLAUDE.md"
	abs := filepath.Join(root, relPath)

	raw, err := os.ReadFile(abs)
	if err != nil {
		return []Violation{{
			Policy: "m0134-claude-md-test-running-sections",
			File:   relPath,
			Detail: fmt.Sprintf("ReadFile failed: %v", err),
		}}, nil
	}
	content := string(raw)

	var vs []Violation
	report := func(detail string) {
		vs = append(vs, Violation{
			Policy: "m0134-claude-md-test-running-sections",
			File:   relPath,
			Detail: detail,
		})
	}

	const (
		devcontainerHeading = "#### Running tests in the devcontainer (primary)"
		macosHeading        = "#### Running tests on macOS host (fallback)"
		staleParkedPhrase   = "Structural fix (Linux devcontainer) is parked."
	)

	devcontainerStart := strings.Index(content, "\n"+devcontainerHeading+"\n")
	macosStart := strings.Index(content, "\n"+macosHeading+"\n")

	if devcontainerStart < 0 {
		report(fmt.Sprintf("missing subsection heading %q (post-M-0134 expects this as the primary path)", devcontainerHeading))
	}
	if macosStart < 0 {
		report(fmt.Sprintf("missing subsection heading %q (post-M-0134 expects the macOS guidance demoted to a fallback subsection)", macosHeading))
	}

	if devcontainerStart >= 0 && macosStart >= 0 {
		if devcontainerStart > macosStart {
			report(fmt.Sprintf("subsection order wrong: %q (line offset %d) must appear BEFORE %q (line offset %d)",
				devcontainerHeading, devcontainerStart, macosHeading, macosStart))
		}
	}

	devcontainerBody := markdownSection(content, devcontainerHeading)
	macosBody := markdownSection(content, macosHeading)

	// Devcontainer subsection: must indicate explicitly that no
	// wrapper is required on Linux (the whole point of the section).
	if devcontainerBody != "" {
		if !strings.Contains(devcontainerBody, "Linux") {
			report("devcontainer subsection body does not mention `Linux` (the OS context that obviates the wrapper)")
		}
		hasNoWrapperClaim := strings.Contains(devcontainerBody, "no signing") ||
			strings.Contains(devcontainerBody, "no wrapper") ||
			strings.Contains(devcontainerBody, "unwrapped")
		if !hasNoWrapperClaim {
			report("devcontainer subsection body does not claim no-wrapper-required (expected one of: \"no signing\", \"no wrapper\", \"unwrapped\")")
		}
	}

	// macOS-host subsection: must carry the demoted wrapper-discipline
	// content (sign-and-run.sh, make test, diagnostic gap ids).
	if macosBody != "" {
		if !strings.Contains(macosBody, "sign-and-run.sh") {
			report("macOS-host fallback subsection body does not mention `sign-and-run.sh` (the wrapper)")
		}
		if !strings.Contains(macosBody, "make test") {
			report("macOS-host fallback subsection body does not mention `make test` (the recommended Do invocation)")
		}
		diagnosticGaps := []string{"G-0127", "G-0128", "G-0133"}
		hasDiagnostic := false
		for _, g := range diagnosticGaps {
			if strings.Contains(macosBody, g) {
				hasDiagnostic = true
				break
			}
		}
		if !hasDiagnostic {
			report(fmt.Sprintf("macOS-host fallback subsection body does not mention any of the diagnostic gap ids %v", diagnosticGaps))
		}
	}

	// "Defaults, not a chokepoint" caveat must be scoped to the
	// macOS-host fallback subsection — it doesn't apply in the
	// container and shouldn't float at file scope.
	const caveatPhrase = "Defaults, not a chokepoint"
	if macosBody != "" {
		if !strings.Contains(macosBody, caveatPhrase) {
			report(fmt.Sprintf("macOS-host fallback subsection body does not contain the %q caveat (expected scoped under this subsection)", caveatPhrase))
		}
	}
	if devcontainerBody != "" {
		if strings.Contains(devcontainerBody, caveatPhrase) {
			report(fmt.Sprintf("devcontainer subsection body contains the %q caveat (the caveat applies only to the macOS host; move it into the fallback subsection)", caveatPhrase))
		}
	}

	// Stale "parked" phrase must be absent — the structural fix
	// shipped in M-0132.
	if strings.Contains(content, staleParkedPhrase) {
		report(fmt.Sprintf("contains stale phrase %q — the structural fix (Linux devcontainer) shipped in M-0132; remove the sentence", staleParkedPhrase))
	}

	return vs, nil
}

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
