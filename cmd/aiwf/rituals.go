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
// doctor` when no rituals plugin is detected. Two-tier: aiwf-extensions
// is recommended (aiwf without it is a planning data store, not a
// workflow); wf-rituals is optional (opinionated TDD/review/lint
// rituals — bring your own if you prefer).
func printRitualsSuggestion() {
	lines := []string{
		"",
		"→ Recommended next step",
		"",
		"In a Claude Code session, install the companion rituals plugin:",
		"",
		"  /plugin marketplace add " + ritualsMarketplaceSlug,
		"  /plugin install aiwf-extensions@" + ritualsMarketplaceName,
		"",
		"This adds milestone-lifecycle skills and four role agents (planner, builder,",
		"reviewer, deployer) that compose with aiwf for an end-to-end workflow.",
		"Without it, aiwf is just the planning data layer — useful but bare.",
		"",
		"→ Optional",
		"",
		"  /plugin install wf-rituals@" + ritualsMarketplaceName,
		"",
		"Generic engineering rituals (TDD cycle, code review, doc-lint). Repo-agnostic.",
		"Skip if you have your own.",
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
