package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ritualsMarketplaceSlug is the GitHub slug of the companion plugin
// marketplace. Centralized so init's suggestion block, doctor's note,
// and any future surface stay consistent.
const ritualsMarketplaceSlug = "23min/ai-workflow-rituals"

// ritualsMarketplaceName is the marketplace's `name` field as declared
// in its marketplace.json — the suffix users append to plugin names
// when they install (e.g. `aiwf-extensions@ai-workflow-rituals`).
const ritualsMarketplaceName = "ai-workflow-rituals"

// printRitualsSuggestion writes the "recommended next steps" block to
// stdout. Used by `aiwf init` after a successful run, and by `aiwf
// doctor` when no rituals plugin is detected.
//
// The nudge steers operators to the interactive `/plugin` menu (no
// args) and asks them to install at PROJECT scope via the Discover
// tab. The CLI form `claude /plugin install <name>@<marketplace>`
// defaults to *user* scope, which does not satisfy `aiwf doctor`'s
// `doctor.recommended_plugins` check (that check polls project-scope
// installs only). Sending fresh operators down the CLI path leaves
// them in a state where doctor keeps warning silently — closes
// G-0069. The canonical procedure lives in CLAUDE.md's "Operator
// setup" section.
func printRitualsSuggestion() {
	lines := []string{
		"",
		"→ Recommended next step",
		"",
		"In a Claude Code session at this repo's root, add the marketplace and",
		"then open the interactive plugin menu:",
		"",
		"  /plugin marketplace add " + ritualsMarketplaceSlug,
		"  /plugin                     (no args — opens the menu)",
		"",
		"In the menu, go to the Discover tab and install BOTH plugins at",
		"PROJECT scope (not user scope — only project scope satisfies",
		"`aiwf doctor`'s recommended-plugins check):",
		"",
		"  - aiwf-extensions@" + ritualsMarketplaceName,
		"  - wf-rituals@" + ritualsMarketplaceName,
		"",
		"Then verify with:",
		"",
		"  aiwf doctor",
		"",
		"Once both plugins are project-scope installed, the",
		"`recommended-plugin-not-installed` warnings go silent.",
		"",
		"Note: the CLI form `claude /plugin install <name>@<marketplace>`",
		"defaults to user scope and will not silence the doctor warnings —",
		"use the interactive `/plugin` menu instead.",
	}
	for _, line := range lines {
		// stdout, not stderr — these are user-facing recommendations.
		_, _ = os.Stdout.WriteString(line + "\n")
	}
}

// ritualsPluginInstalled is a best-effort heuristic that returns true
// when `aiwf-extensions` appears in the consumer's project- or local-
// scope Claude Code settings. False negatives are expected for
// user-scope installs (those land in ~/.claude/settings.json which
// aiwf doesn't read), so callers treat a "not detected" result as a
// soft hint, not a finding.
func ritualsPluginInstalled(rootDir string) bool {
	for _, name := range []string{".claude/settings.json", ".claude/settings.local.json"} {
		path := filepath.Join(rootDir, name)
		content, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			// Unreadable file — treat as undetected; doctor's check is
			// soft, so we won't surface the read error.
			continue
		}
		if strings.Contains(string(content), "aiwf-extensions") {
			return true
		}
	}
	return false
}
