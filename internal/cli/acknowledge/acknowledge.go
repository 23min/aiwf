// Package acknowledge implements the `aiwf acknowledge` verb group — the
// sovereign-acknowledgement family. Each subcommand records a sovereign act (a
// human actor plus a written --reason) as an empty commit carrying
// aiwf-verb / aiwf-actor / aiwf-reason trailers, telling a kernel audit rule to
// stop flagging a thing the operator has reviewed and accepted.
//
//	aiwf acknowledge illegal <sha>   — exempt a historical commit from the
//	                                   FSM-history / provenance audit rules
//	                                   (M-0136).
//
// Regrouped from the former top-level `acknowledge-illegal` verb (M-0181/AC-5).
// The CLI surface now groups the family, but each act's aiwf-verb trailer value
// is unchanged: the command path `acknowledge illegal` enumerates to the same
// `acknowledge-illegal` string via the hyphen-join walker (internal/cli/check/
// verbs.go), so history, the trailer-verb-unknown rule, and the commit-msg hook
// all keep validating without a back-compat shim. The mistag sibling
// (`aiwf acknowledge mistag`) lands alongside in M-0181/AC-6.
//
// The parent is non-Runnable — `aiwf acknowledge` with no subcommand prints
// help — mirroring the `aiwf contract` topical group.
package acknowledge

import (
	"github.com/spf13/cobra"
)

// NewCmd builds the `aiwf acknowledge` parent command. Non-Runnable; the
// subcommands carry the behavior.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "acknowledge",
		Short:         "Sovereign acknowledgement of a flagged commit or entity",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.AddCommand(newIllegalCmd())
	return cmd
}
