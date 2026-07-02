package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PolicyM0202DevcontainerOnboarding asserts that the devcontainer
// onboarding surface — .devcontainer/init.sh's post-install banner and
// .devcontainer/README.md — carries no instruction for the retired
// rituals marketplace/sibling-repo plugin-install flow, and that the
// init.sh banner points operators at the current verification path
// (`aiwf doctor`'s `rituals:` line).
//
// Rituals materialize into .claude/ when `aiwf init` runs (ADR-0014,
// E-0038); the upstream 23min/ai-workflow-rituals marketplace channel is
// archived (ADR-0016, G-0193). The banner and README used to tell the
// operator to install two plugins at PROJECT scope and verify via a
// `recommended-plugin-not-installed` doctor warning that no longer
// exists — a retired flow that fails on every container (re)open.
//
// Pins M-0202/AC-1 (corrected content) and AC-2 (the durable drift
// chokepoint). This is an aiwf-repo development invariant — the
// .devcontainer/ files exist only here — so it lives as a Go policy
// test, mirroring the sibling PolicyM0132* devcontainer policies, not as
// an `aiwf check` finding (which would be inert in a consumer tree).
func PolicyM0202DevcontainerOnboarding(root string) ([]Violation, error) {
	const (
		initRel   = ".devcontainer/init.sh"
		readmeRel = ".devcontainer/README.md"
	)

	// retiredMarkers are instruction fragments unique to the retired
	// plugin-install flow. None has a legitimate home in either file, so
	// any occurrence means the retired flow crept back in.
	retiredMarkers := []string{
		// /plugin marketplace add …
		"plugin marketplace",
		// /reload-plugins
		"reload-plugins",
		// the retired aiwf doctor warning
		"recommended-plugin-not-installed",
		// the manual-install framing
		"PROJECT scope",
		// the archived marketplace / sibling repo
		"ai-workflow-rituals",
	}

	var vs []Violation
	report := func(file, detail string) {
		vs = append(vs, Violation{
			Policy: "m0202-devcontainer-onboarding",
			File:   file,
			Detail: detail,
		})
	}

	contents := map[string]string{}
	for _, rel := range []string{initRel, readmeRel} {
		raw, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			report(rel, fmt.Sprintf("missing or unreadable: %v", err))
			continue
		}
		contents[rel] = string(raw)
	}

	for _, rel := range []string{initRel, readmeRel} {
		content, ok := contents[rel]
		if !ok {
			continue
		}
		for _, marker := range retiredMarkers {
			if strings.Contains(content, marker) {
				report(rel, fmt.Sprintf("reintroduces the retired rituals plugin-install flow: contains %q — rituals materialize via `aiwf init` (ADR-0014); the marketplace channel is archived (ADR-0016). Drop the manual-install instruction.", marker))
			}
		}
	}

	// The init.sh banner must point operators at the current verification
	// path: `aiwf doctor`'s `rituals:` line.
	if content, ok := contents[initRel]; ok {
		for _, needle := range []string{"aiwf doctor", "rituals:"} {
			if !strings.Contains(content, needle) {
				report(initRel, fmt.Sprintf("banner must direct the operator to the current verification path (looked for literal %q — `aiwf doctor`'s `rituals:` line confirms materialization)", needle))
			}
		}
	}

	return vs, nil
}
