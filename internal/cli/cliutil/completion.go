package cliutil

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// RegisterFormatCompletion wires `--format=` shell completion to the
// closed set {text, json}. Called by every read-only verb that
// accepts --format so the shell-completion experience is uniform
// across the surface (E-14's auto-completion-friendliness rule).
func RegisterFormatCompletion(cmd *cobra.Command) {
	_ = cmd.RegisterFlagCompletionFunc("format", cobra.FixedCompletions(
		[]string{"text", "json"},
		cobra.ShellCompDirectiveNoFileComp,
	))
}

// AllKindNames returns the entity-kind names as strings, in the
// canonical iteration order from entity.AllKinds(). Used by the
// `aiwf add` and `aiwf schema` / `aiwf template` completion functions.
func AllKindNames() []string {
	all := entity.AllKinds()
	names := make([]string, len(all))
	for i, k := range all {
		names[i] = string(k)
	}
	return names
}

// StatusesForID returns the closed set of statuses that an entity's
// kind allows, derived from the id's prefix without loading the
// repo's tree. Used as the static-completion source for `aiwf promote
// <id> <new-status>`. Returns nil for ids whose kind isn't recognized
// (composite ids, malformed input) — the completion source then falls
// back to file completion at the shell level.
func StatusesForID(id string) []string {
	if id == "" || entity.IsCompositeID(id) {
		return nil
	}
	k, ok := entity.KindFromID(id)
	if !ok {
		return nil
	}
	return entity.AllowedStatuses(k)
}

// CompleteEntityIDs returns the live ids in the consumer repo's
// planning tree, optionally filtered to a single kind. Designed for
// use as a Cobra ValidArgsFunction or RegisterFlagCompletionFunc body:
// failures (no aiwf.yaml, malformed tree, unreadable disk) collapse
// to an empty list rather than spamming the user's shell with errors,
// satisfying M-054 AC-2's graceful-no-op rule.
func CompleteEntityIDs(filter entity.Kind) ([]string, cobra.ShellCompDirective) {
	rootDir, err := ResolveRoot("")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	tr, _, err := tree.Load(context.Background(), rootDir)
	if err != nil || tr == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ids := make([]string, 0, len(tr.Entities))
	for _, e := range tr.Entities {
		if filter != "" && e.Kind != filter {
			continue
		}
		// Emit canonical ids so completion always offers the canonical
		// width, regardless of on-disk filename width (AC-3 in M-081).
		// Inputs at narrow width are still accepted everywhere
		// downstream via tree.ByID's lookup-side canonicalization.
		ids = append(ids, entity.Canonicalize(e.ID))
	}
	return ids, cobra.ShellCompDirectiveNoFileComp
}

// CompleteEntityIDFlag is the standard Cobra flag-completion adapter
// over CompleteEntityIDs. Callers wire it via
// `cmd.RegisterFlagCompletionFunc(name, cliutil.CompleteEntityIDFlag(kind))`
// where kind is either "" for all kinds or a specific entity.Kind.
func CompleteEntityIDFlag(filter entity.Kind) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return CompleteEntityIDs(filter)
	}
}

// CompleteAreaFlag is the Cobra flag-completion adapter for `aiwf add
// --area`: it offers exactly the declared `aiwf.yaml: areas.members`
// (E-0043, M-0173/AC-4), the same closed set the write-time validation
// and the M-0172 area-unknown check read. Failures (no aiwf.yaml, no
// areas block) collapse to an empty list rather than erroring in the
// shell, matching the graceful-no-op rule the entity-id completions
// follow.
func CompleteAreaFlag() func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		rootDir, err := ResolveRoot("")
		if err != nil { //coverage:ignore ResolveRoot("") only errors if os.Getwd fails (unreachable in practice; it falls back to cwd on a missing aiwf.yaml); mirrors CompleteEntityIDs
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		// The --area value space (set-time and read-filter) is a settable
		// area value: declared members PLUS the reserved AreaGlobal sentinel
		// (M-0184). rename-area's <old> is a different space (renameable
		// members only) and stays on CompleteAreaArg.
		return areaValueCompletions(rootDir), cobra.ShellCompDirectiveNoFileComp
	}
}

// CompleteAreaArg is the positional-arg completion adapter for `aiwf
// rename-area <old> <new>`: it offers exactly the declared
// `aiwf.yaml: areas.members` at the given position and nothing
// elsewhere. Wired as `cmd.ValidArgsFunction = cliutil.CompleteAreaArg(0)`,
// it completes `<old>` (position 0) to the declared member set and
// offers nothing for `<new>` (position 1) — a brand-new name has no
// closed set to suggest. Mirrors CompleteEntityIDArg's position-
// respecting shape and CompleteAreaFlag's member source; failures (no
// aiwf.yaml, no areas block) collapse to an empty list rather than
// erroring in the shell.
func CompleteAreaArg(position int) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return completeAreaArg(position, false)
}

// CompleteAreaValueArg is the positional-arg completion adapter for a
// settable area VALUE — set-area's `<member>` positional (M-0184). Unlike
// CompleteAreaArg (renameable members only), it offers the reserved
// AreaGlobal sentinel alongside the declared members, since set-area
// accepts `global` as the affirmative cross-cutting escape valve. Wired as
// `CompleteAreaValueArg(1)` from set-area's composed ValidArgsFunction
// (position 0 is the entity id). Same position-respecting and
// graceful-no-op shape as CompleteAreaArg.
func CompleteAreaValueArg(position int) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return completeAreaArg(position, true)
}

// completeAreaArg is the shared body for the two positional area completers.
// includeGlobal toggles whether the reserved AreaGlobal sentinel is offered
// alongside the declared members: rename-area's <old> (a renameable member)
// passes false; set-area's <member> (a settable value) passes true.
func completeAreaArg(position int, includeGlobal bool) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) != position {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		rootDir, err := ResolveRoot("")
		if err != nil { //coverage:ignore ResolveRoot("") only errors if os.Getwd fails (unreachable in practice; it falls back to cwd on a missing aiwf.yaml); mirrors CompleteAreaFlag
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		if includeGlobal {
			return areaValueCompletions(rootDir), cobra.ShellCompDirectiveNoFileComp
		}
		return ConfiguredAreaMembers(rootDir), cobra.ShellCompDirectiveNoFileComp
	}
}

// areaValueCompletions returns the declared members plus the reserved
// AreaGlobal sentinel — the completion value set for any surface that
// accepts a settable area value (`add --area`, set-area's `<member>`, and
// the list/show/status read filters). Keeping the append here means the
// "global is a valid area value" fact lives next to the SSOT predicate's
// consumers, not duplicated at each call site (M-0184).
//
// global is offered only when an areas block is declared: with no block the
// `area` field is inert (M-0171) and `add --area global` is a usage error
// (M-0184/AC-4), so suggesting global there would mislead. No block →
// nothing, preserving the graceful-no-op completion outside a configured
// project.
func areaValueCompletions(rootDir string) []string {
	members := ConfiguredAreaMembers(rootDir)
	if len(members) == 0 {
		return nil
	}
	return append(members, entity.AreaGlobal)
}

// CompleteEntityIDArg is the standard Cobra positional-arg completion
// adapter over CompleteEntityIDs. Callers assign it as a command's
// ValidArgsFunction. Unlike the flag adapter, this version respects
// the args slice — if the positional in question isn't the first one,
// it returns no suggestions (so e.g. `aiwf promote E-01 <TAB>` doesn't
// re-suggest entity ids when the second positional is the new-status).
func CompleteEntityIDArg(filter entity.Kind, position int) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) != position {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return CompleteEntityIDs(filter)
	}
}
