package check

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
)

// enumerateRegisteredVerbs walks the Cobra command tree rooted at
// root (typically `aiwf` from c.Root() at RunE time) and returns the
// closed set of trailer-value strings that match registered verbs:
// every command's path joined by hyphens, top-level + subverbs.
//
// Examples (from the live tree at G-0150's land time):
//
//	"add"                       (top-level: aiwf add)
//	"add-ac"                    (subverb:   aiwf add ac)
//	"render-roadmap"            (subverb:   aiwf render roadmap)
//	"milestone-depends-on"      (subverb:   aiwf milestone depends-on)
//
// The set is the source of truth for `check.RunTrailerVerbUnknown`.
// Sourcing it from the running binary's command tree means a future
// verb addition is recognized as soon as it's wired into the tree —
// no separate registry to keep in sync.
//
// Cobra's `Execute()` lazily adds top-level `help` and `completion`
// commands to every root. `NewRootCmd` records the set of verbs the
// binary explicitly registers in the root command's
// `Annotations[cliutil.AnnotationRegisteredVerbs]` BEFORE Execute
// runs, so this enumerator filters Cobra's auto-adds out
// structurally — no hardcoded list of auto-add names.
//
// When the annotation is absent (test fixtures that build minimal
// Cobra trees without going through `NewRootCmd`), the enumerator
// falls back to walking every top-level command. That preserves the
// straightforward "build a tree, enumerate it" expectation for
// tests.
//
// Hidden commands and the root itself are excluded. A nil root
// returns nil.
//
// Closes G-0150.
func enumerateRegisteredVerbs(root *cobra.Command) map[string]struct{} {
	if root == nil {
		return nil
	}
	explicit := explicitVerbSet(root)
	out := make(map[string]struct{})
	for _, sub := range root.Commands() {
		if explicit != nil && !explicit[sub.Name()] {
			continue
		}
		walkVerbs(sub, "", out)
	}
	return out
}

// explicitVerbSet returns the set of top-level verb names recorded
// by `NewRootCmd` in the root's Annotations, or nil when no such
// annotation is present. A nil return tells `enumerateRegisteredVerbs`
// to fall back to walking every top-level command (the test-fixture
// path).
func explicitVerbSet(root *cobra.Command) map[string]bool {
	// Reading from a nil map is legal in Go and returns ("", false),
	// so the nil-Annotations and missing-key cases collapse into one.
	raw, ok := root.Annotations[cliutil.AnnotationRegisteredVerbs]
	if !ok {
		return nil
	}
	set := make(map[string]bool)
	for _, name := range strings.Split(raw, "\n") {
		if name != "" {
			set[name] = true
		}
	}
	return set
}

// walkVerbs is enumerateRegisteredVerbs' recursive worker. prefix
// carries the hyphen-joined ancestry; the current command's Name() is
// appended unless the command is hidden.
func walkVerbs(c *cobra.Command, prefix string, out map[string]struct{}) {
	if c == nil || c.Hidden {
		return
	}
	name := c.Name()
	if name == "" {
		return
	}
	current := name
	if prefix != "" {
		current = prefix + "-" + name
	}
	out[current] = struct{}{}
	for _, sub := range c.Commands() {
		walkVerbs(sub, current, out)
	}
}
