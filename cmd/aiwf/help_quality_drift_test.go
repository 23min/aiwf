package main

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// M-069 AC-7 — Help-quality drift asserts Example present and no
// migration prose.
//
// `aiwf <verb> --help` is the AI-discoverability surface (CLAUDE.md
// §"Kernel functionality must be AI-discoverable"). Cobra renders
// that output from each verb's Long / Short / Example fields, so
// help quality reduces to those three strings being present, current,
// and timeless. E-14's migration to native Cobra is closed; help
// text written during the migration that references the migration
// itself ("version verb migrated", "newly migrated to Cobra") is now
// drift — it leaks an implementation-history detail into the
// user-facing surface and dates the help text in a way that becomes
// misleading as the reference recedes.
//
// This file holds two paired tests:
//
//  1. TestPolicy_ExamplePresent — walks every Runnable command and
//     asserts `Example` is non-empty.
//  2. TestPolicy_NoMigrationProse — walks Long/Short/Example of every
//     command and asserts none contain the case-insensitive substring
//     `migrat` (catches migrated, migration, migrating, migrate). The
//     pattern is deliberately narrow: Cobra (the framework name) and
//     entity ids like E-14 are legitimate references and pass.

// migrationProsePattern catches the four migrate-class verb forms
// (migrate, migrated, migrating, migration) case-insensitively.
// Narrower than a full audit-prose regex on purpose — broader
// patterns (Cobra, E-14, etc.) are legitimate references in help
// text and would create false positives.
var migrationProsePattern = regexp.MustCompile(`(?i)\bmigrat(e|ed|ing|ion)\b`)

// helpFieldOptOuts names commands intentionally without an Example
// block. Non-Runnable parents dispatch to children; Cobra-generated
// commands (completion, help) belong to the framework.
var helpFieldOptOuts = map[string]string{
	"aiwf":                       "root command; no single-verb Example to ship",
	"aiwf contract":              "non-Runnable parent; dispatches to children",
	"aiwf contract recipe":       "non-Runnable parent; dispatches to children",
	"aiwf contract recipes":      "trivial read-only listing; no Example needed",
	"aiwf completion":            "Cobra-generated; framework owns the help text",
	"aiwf completion bash":       "Cobra-generated; framework owns the help text",
	"aiwf completion zsh":        "Cobra-generated; framework owns the help text",
	"aiwf completion fish":       "Cobra-generated; framework owns the help text",
	"aiwf completion powershell": "Cobra-generated; framework owns the help text",
	"aiwf help":                  "Cobra-default help command; framework owns the help text",
	"aiwf render help":           "hidden help alias; no Example needed",
	"aiwf render":                "non-Runnable parent in subverb mode (subcommand or --format=html); the html branch has its own Example via cmd.Long",
	"aiwf add":                   "non-Runnable parent; each kind subcommand carries its own Example",
}

// TestPolicy_ExamplePresent (M-069 AC-7) walks every Runnable command
// in newRootCmd()'s tree and asserts `Example` is non-empty. A
// regression where a new verb lands without an Example block fails
// CI here.
func TestPolicy_ExamplePresent(t *testing.T) {
	root := newRootCmd()

	var failures []string
	walkCommands(root, func(cmd *cobra.Command) {
		path := cmd.CommandPath()
		if _, ok := helpFieldOptOuts[path]; ok {
			return
		}
		// Non-Runnable parents (no RunE, has subcommands) opt out by
		// definition — they dispatch to children. The opt-out map
		// lists the handful of these explicitly so a regression that
		// turns a parent into a Runnable verb without an Example
		// surfaces here.
		if !cmd.Runnable() {
			return
		}
		if strings.TrimSpace(cmd.Example) == "" {
			failures = append(failures, path)
		}
	})
	if len(failures) > 0 {
		sort.Strings(failures)
		t.Errorf("%d command(s) missing Example block (E-14 / M-069 AC-7):\n  %s\n\n"+
			"Every Runnable verb must ship an Example block — that's the AI-discoverability surface "+
			"per CLAUDE.md. Either add an Example to the cobra.Command literal, or extend "+
			"helpFieldOptOuts in help_quality_drift_test.go with a one-line rationale.",
			len(failures), joinFailures(failures))
	}
}

// TestPolicy_NoMigrationProse (M-069 AC-7) walks Long / Short /
// Example of every command and asserts none contain the
// case-insensitive substring `migrat` (catches migrated, migration,
// migrating, migrate). The pattern is narrow on purpose — broader
// terms like "Cobra" or entity ids are legitimate references in
// help text and would generate false positives.
func TestPolicy_NoMigrationProse(t *testing.T) {
	root := newRootCmd()

	type hit struct {
		path  string
		field string
		match string
	}
	var hits []hit
	walkCommands(root, func(cmd *cobra.Command) {
		for field, val := range map[string]string{
			"Long":    cmd.Long,
			"Short":   cmd.Short,
			"Example": cmd.Example,
		} {
			if m := migrationProsePattern.FindString(val); m != "" {
				hits = append(hits, hit{path: cmd.CommandPath(), field: field, match: m})
			}
		}
	})
	if len(hits) > 0 {
		sort.Slice(hits, func(i, j int) bool {
			if hits[i].path != hits[j].path {
				return hits[i].path < hits[j].path
			}
			return hits[i].field < hits[j].field
		})
		var lines []string
		for _, h := range hits {
			lines = append(lines, h.path+" ."+h.field+" — matched "+h.match)
		}
		t.Errorf("%d migration-prose hit(s) in user-facing help (E-14 / M-069 AC-7):\n  %s\n\n"+
			"Help text written during the E-14 migration sometimes referenced the migration itself "+
			"('version verb migrated', 'newly migrated to Cobra'). That prose is now drift — it leaks "+
			"an implementation-history detail into the user-facing surface and dates the help text. "+
			"Replace migration-flavored language with timeless prose; if you need a literal example "+
			"that legitimately mentions migration (e.g., the hook-migration runtime feature), move "+
			"it out of Long/Short/Example and into a fmt.Println in the runner.",
			len(hits), joinFailures(lines))
	}
}
