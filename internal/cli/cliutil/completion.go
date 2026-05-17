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
