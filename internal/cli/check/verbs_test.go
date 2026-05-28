package check

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
)

// TestEnumerateRegisteredVerbs_TopLevelAndSubverbs pins the
// hyphen-joined path enumeration: subverbs surface as
// `<verb>-<sub>`, matching the trailer-value shape historical
// commits use (e.g. `milestone-depends-on`, `render-roadmap`).
//
// Closes G-0150.
func TestEnumerateRegisteredVerbs_TopLevelAndSubverbs(t *testing.T) {
	t.Parallel()

	add := &cobra.Command{Use: "add"}
	add.AddCommand(&cobra.Command{Use: "ac"})

	milestone := &cobra.Command{Use: "milestone"}
	milestone.AddCommand(&cobra.Command{Use: "depends-on"})

	render := &cobra.Command{Use: "render"}
	render.AddCommand(&cobra.Command{Use: "roadmap"})

	promote := &cobra.Command{Use: "promote"}

	root := &cobra.Command{Use: "aiwf"}
	root.AddCommand(add, milestone, render, promote)

	got := enumerateRegisteredVerbs(root)
	want := []string{
		"add", "add-ac",
		"milestone", "milestone-depends-on",
		"render", "render-roadmap",
		"promote",
	}
	if len(got) != len(want) {
		t.Errorf("len = %d, want %d; got %v", len(got), len(want), got)
	}
	for _, w := range want {
		if _, ok := got[w]; !ok {
			t.Errorf("missing %q in %v", w, got)
		}
	}
	// The root itself must not appear.
	if _, ok := got["aiwf"]; ok {
		t.Error("root command must not appear in the verb set")
	}
}

// TestEnumerateRegisteredVerbs_HiddenCommandsExcluded asserts that a
// hidden command (and its descendants) are not enumerated — hidden
// commands are intentionally not part of the user-visible verb
// surface.
func TestEnumerateRegisteredVerbs_HiddenCommandsExcluded(t *testing.T) {
	t.Parallel()

	hidden := &cobra.Command{Use: "debug-only", Hidden: true}
	hidden.AddCommand(&cobra.Command{Use: "deep"}) // descendant must also be skipped
	visible := &cobra.Command{Use: "promote"}

	root := &cobra.Command{Use: "aiwf"}
	root.AddCommand(hidden, visible)

	got := enumerateRegisteredVerbs(root)
	if _, ok := got["debug-only"]; ok {
		t.Error("hidden command must not appear")
	}
	if _, ok := got["debug-only-deep"]; ok {
		t.Error("descendants of hidden commands must not appear")
	}
	if _, ok := got["promote"]; !ok {
		t.Error("visible command must appear")
	}
}

// TestEnumerateRegisteredVerbs_NilRootIsSafe guards against a
// RunE-time caller that somehow lacks a root command — the function
// returns nil rather than panicking.
func TestEnumerateRegisteredVerbs_NilRootIsSafe(t *testing.T) {
	t.Parallel()
	if got := enumerateRegisteredVerbs(nil); got != nil {
		t.Errorf("got %v, want nil for nil root", got)
	}
}

// TestEnumerateRegisteredVerbs_EmptyUseSkipped guards against a
// malformed Cobra command with `Use: ""` polluting the set —
// cobra.Command.Name() returns "" in that case, and the walker
// short-circuits rather than ascending its descendants.
func TestEnumerateRegisteredVerbs_EmptyUseSkipped(t *testing.T) {
	t.Parallel()
	broken := &cobra.Command{Use: ""}
	broken.AddCommand(&cobra.Command{Use: "deep"}) // would otherwise surface as `-deep`
	visible := &cobra.Command{Use: "promote"}
	root := &cobra.Command{Use: "aiwf"}
	root.AddCommand(broken, visible)

	got := enumerateRegisteredVerbs(root)
	if _, ok := got[""]; ok {
		t.Error("empty-Use command must not enter the set")
	}
	if _, ok := got["-deep"]; ok {
		t.Error("descendants of empty-Use commands must not be reached")
	}
	if _, ok := got["promote"]; !ok {
		t.Error("sibling visible command must still appear")
	}
}

// TestEnumerateRegisteredVerbs_AnnotationFiltersOutAutoAdds pins the
// principled mechanism: when the root carries
// `cliutil.AnnotationRegisteredVerbs`, commands not in that explicit
// set are filtered out — regardless of name. This is how the running
// binary distinguishes its own verbs from Cobra's auto-added `help`
// and `completion` (the names are not hardcoded anywhere in the
// enumeration logic; the annotation is the single source of truth).
func TestEnumerateRegisteredVerbs_AnnotationFiltersOutAutoAdds(t *testing.T) {
	t.Parallel()
	// Simulate the production setup: 3 top-level commands, but only
	// `promote` is in the explicit-verb annotation. The other two
	// stand in for Cobra's auto-adds.
	help := &cobra.Command{Use: "help"}
	completion := &cobra.Command{Use: "completion"}
	promote := &cobra.Command{Use: "promote"}
	root := &cobra.Command{Use: "aiwf"}
	root.AddCommand(help, completion, promote)
	root.Annotations = map[string]string{
		cliutil.AnnotationRegisteredVerbs: "promote",
	}

	got := enumerateRegisteredVerbs(root)
	if _, ok := got["help"]; ok {
		t.Error("`help` is not in the explicit-verb annotation; must be filtered")
	}
	if _, ok := got["completion"]; ok {
		t.Error("`completion` is not in the explicit-verb annotation; must be filtered")
	}
	if _, ok := got["promote"]; !ok {
		t.Error("`promote` is in the explicit-verb annotation; must remain")
	}
}

// TestEnumerateRegisteredVerbs_FallsBackToWalkAllWhenNoAnnotation
// pins the fallback path: when the root has no
// `AnnotationRegisteredVerbs` (test fixtures that build minimal Cobra
// trees without going through `NewRootCmd`), every top-level command
// is enumerated. This keeps the "build a tree, enumerate it" shape
// straightforward for tests.
func TestEnumerateRegisteredVerbs_FallsBackToWalkAllWhenNoAnnotation(t *testing.T) {
	t.Parallel()
	promote := &cobra.Command{Use: "promote"}
	extra := &cobra.Command{Use: "extra"}
	root := &cobra.Command{Use: "aiwf"}
	root.AddCommand(promote, extra)
	// No Annotations set.

	got := enumerateRegisteredVerbs(root)
	if _, ok := got["promote"]; !ok {
		t.Error("no annotation → walk-all fallback should include promote")
	}
	if _, ok := got["extra"]; !ok {
		t.Error("no annotation → walk-all fallback should include extra")
	}
}

// TestEnumerateRegisteredVerbs_AnnotationPresentButEmpty asserts the
// defensive shape: an empty annotation (key set, value empty) means
// "no commands are explicit," so every command is filtered out. This
// is distinct from the no-annotation case (fallback to walk-all);
// callers that want walk-all behavior should not set the annotation
// at all.
func TestEnumerateRegisteredVerbs_AnnotationPresentButEmpty(t *testing.T) {
	t.Parallel()
	root := &cobra.Command{Use: "aiwf"}
	root.AddCommand(&cobra.Command{Use: "promote"})
	root.Annotations = map[string]string{
		cliutil.AnnotationRegisteredVerbs: "",
	}

	got := enumerateRegisteredVerbs(root)
	if _, ok := got["promote"]; ok {
		t.Error("empty annotation should filter every command out (caller asked for explicit-only)")
	}
}

// TestEnumerateRegisteredVerbs_AnnotationDoesNotAffectSubverbs
// pins that the annotation gates only the top-level walk: once a
// listed top-level verb is admitted, its full sub-tree is
// enumerated.
func TestEnumerateRegisteredVerbs_AnnotationDoesNotAffectSubverbs(t *testing.T) {
	t.Parallel()
	milestone := &cobra.Command{Use: "milestone"}
	milestone.AddCommand(&cobra.Command{Use: "depends-on"})
	root := &cobra.Command{Use: "aiwf"}
	root.AddCommand(milestone)
	root.Annotations = map[string]string{
		cliutil.AnnotationRegisteredVerbs: "milestone",
	}

	got := enumerateRegisteredVerbs(root)
	if _, ok := got["milestone"]; !ok {
		t.Error("explicit top-level verb must be admitted")
	}
	if _, ok := got["milestone-depends-on"]; !ok {
		t.Error("sub-command of an admitted verb must be enumerated")
	}
}

// TestEnumerateRegisteredVerbs_EmptyTreeIsEmpty asserts a root with
// no children returns an empty (non-nil) map — the rule layer treats
// nil and empty equivalently, but the function consistently returns
// a map for non-nil inputs.
func TestEnumerateRegisteredVerbs_EmptyTreeIsEmpty(t *testing.T) {
	t.Parallel()
	root := &cobra.Command{Use: "aiwf"}
	got := enumerateRegisteredVerbs(root)
	if got == nil {
		t.Error("got nil, want empty map for empty tree")
	}
	if len(got) != 0 {
		t.Errorf("got %v, want empty map", got)
	}
}
