package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PolicyM0132ClaudeMdDevcontainerSection asserts that CLAUDE.md's
// `## Operator setup` section contains a `### Devcontainer`
// subsection with non-empty body content. G-0194 retired the
// marketplace-plugin transitional machinery; the shadow-mount /
// plugin-index workaround (claude-code#31388) was removed from this
// subsection because the plugin-index infrastructure is no longer
// relevant for rituals. The policy now pins only the presence of the
// subsection and its non-emptiness — the specific content is the
// devcontainer materialization description.
//
// Pins M-0132/AC-6 (narrowed post-G-0194).
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

	const wantMaterialize = "materialize"
	if !strings.Contains(strings.ToLower(body), wantMaterialize) {
		report("`### Devcontainer` subsection body should describe the ritualmaterialization mechanism (`aiwf init` / `aiwf update`) — the subsection's purpose post-G-0194 is confirming no separate install is needed inside the container")
	}

	return vs, nil
}
