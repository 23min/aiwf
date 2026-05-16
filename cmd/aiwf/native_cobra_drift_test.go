package main

import (
	"sort"
	"testing"

	"github.com/spf13/cobra"
)

// M-069 AC-6 — Native-Cobra drift test fails CI on
// passthrough-adapter regression.
//
// E-14 migrated every verb from a hand-rolled passthrough adapter
// (manual argv parsing wrapped around legacy verbs) to native Cobra
// commands with declarative flag binding. The migration's
// user-visible payoff is the AI-discoverability and shell-completion
// guarantees in CLAUDE.md: `aiwf <verb> --help` is authoritative
// because Cobra generates it from the same flag-binding source the
// runtime uses, and tab-completion works because every value-taking
// flag has a completion function bound to that same source.
//
// `TestPolicy_FlagsHaveCompletion` pins the completion-wiring half.
// It does *not* pin the migration's structural half: a regression
// where a contributor sets `DisableFlagParsing: true` on a Cobra
// command and walks `os.Args` themselves would silently break flag
// binding, completion, and help generation. The bypassed command's
// flags never reach `cmd.Flags()` for the existing walker to find,
// so the regression would land green.
//
// This file holds two tests:
//
//  1. TestPolicy_NoPassthroughAdapters — walks every command in
//     newRootCmd()'s tree and asserts DisableFlagParsing and
//     DisableFlagsInUseLine are both false.
//  2. TestPolicy_NoPassthroughAdapters_DetectsRegression — constructs
//     a synthetic tree with DisableFlagParsing: true on one node,
//     runs the same walker, and asserts the violation is reported.
//     This pins both directions: production doesn't trip the rule,
//     and the rule actually fires when it should — closes the silent-
//     no-op trap.

// passthroughViolation is one entry in the drift report. Encapsulated
// so the rule-test-test pair can compare violations without parsing
// strings.
type passthroughViolation struct {
	Path   string
	Reason string
}

// findPassthroughAdapters walks the command tree rooted at root and
// returns any commands that opt out of native Cobra flag parsing.
// The shared walker between the production check and the
// rule-test-test pair, so the synthetic-violation test exercises the
// same code path the production check does.
func findPassthroughAdapters(root *cobra.Command) []passthroughViolation {
	var out []passthroughViolation
	walkCommands(root, func(cmd *cobra.Command) {
		if cmd.DisableFlagParsing {
			out = append(out, passthroughViolation{
				Path:   cmd.CommandPath(),
				Reason: "DisableFlagParsing=true (passthrough adapter — bypasses Cobra flag binding, completion, and help generation)",
			})
		}
		if cmd.DisableFlagsInUseLine {
			out = append(out, passthroughViolation{
				Path:   cmd.CommandPath(),
				Reason: "DisableFlagsInUseLine=true (typically paired with manual argv parsing; suspicious)",
			})
		}
	})
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Reason < out[j].Reason
	})
	return out
}

// TestPolicy_NoPassthroughAdapters asserts the production command
// tree contains no passthrough adapters. A regression where any verb
// is wired with DisableFlagParsing:true (or DisableFlagsInUseLine:true)
// fails CI with the verb's command path.
func TestPolicy_NoPassthroughAdapters(t *testing.T) {
	t.Parallel()
	violations := findPassthroughAdapters(newRootCmd())
	if len(violations) > 0 {
		var lines []string
		for _, v := range violations {
			lines = append(lines, v.Path+" — "+v.Reason)
		}
		t.Errorf("native-Cobra drift detected (E-14 / M-069 AC-6) — %d passthrough adapter(s) reintroduced:\n  %s\n\n"+
			"Every verb in newRootCmd() must use native Cobra flag binding (no DisableFlagParsing). "+
			"If a verb genuinely needs to consume raw args, route them through a positional Args validator and "+
			"a ValidArgsFunction so completion still works.", len(violations), joinFailures(lines))
	}
}

// TestPolicy_NoPassthroughAdapters_DetectsRegression is the rule-test-
// test pair. It constructs a tiny synthetic command tree with
// DisableFlagParsing:true on one node, runs the same finder the
// production check uses, and asserts the violation is reported.
//
// Without this test, a future refactor that broke walkCommands or
// findPassthroughAdapters in a silent way (early return, dropped
// loop body) would let TestPolicy_NoPassthroughAdapters keep passing
// forever even though the rule had stopped firing. This test pins
// the rule actually fires when it should.
func TestPolicy_NoPassthroughAdapters_DetectsRegression(t *testing.T) {
	t.Parallel()
	// Synthetic tree: root → [clean-child, dirty-child].
	root := &cobra.Command{Use: "synth"}
	clean := &cobra.Command{Use: "clean", Run: func(*cobra.Command, []string) {}}
	dirty := &cobra.Command{
		Use:                "dirty",
		DisableFlagParsing: true,
		Run:                func(*cobra.Command, []string) {},
	}
	cosmetic := &cobra.Command{
		Use:                   "cosmetic",
		DisableFlagsInUseLine: true,
		Run:                   func(*cobra.Command, []string) {},
	}
	root.AddCommand(clean, dirty, cosmetic)

	got := findPassthroughAdapters(root)
	if len(got) != 2 {
		t.Fatalf("rule did not fire on synthetic violations: got %d violations, want 2\n  violations: %+v", len(got), got)
	}

	// Cardinality + path pin: the synthetic tree's two dirty nodes
	// must surface; the clean node must not.
	wantPaths := map[string]bool{"synth dirty": true, "synth cosmetic": true}
	for _, v := range got {
		if !wantPaths[v.Path] {
			t.Errorf("rule reported unexpected violation on %q (reason: %s)", v.Path, v.Reason)
		}
	}
	for _, v := range got {
		if v.Path == "synth clean" {
			t.Errorf("rule reported a violation on the clean synthetic node, which has neither flag set")
		}
	}
}
