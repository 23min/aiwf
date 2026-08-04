package cliutil

import "github.com/spf13/cobra"

// AnnotationVerbGroup marks a command as a verb group — a parent that
// namespaces subverbs and carries no operation of its own. MarkVerbGroup
// stamps it; IsVerbGroup reads it.
//
// The marker exists because a verb group is Runnable at the Cobra level
// without being a verb: the drift tests that ask "does this command need
// an Example, a --format flag, positional completion?" mean "does it
// perform an operation?", which Cobra's Runnable cannot answer on its own
// once a group carries a RunE.
const AnnotationVerbGroup = "aiwf:verb-group"

// MarkVerbGroup turns cmd into a verb group and returns it, for wrapping
// a command literal at its construction site:
//
//	cmd := cliutil.MarkVerbGroup(&cobra.Command{
//	    Use:   "worktree",
//	    Short: "Worktree-scoped verbs",
//	})
//	cmd.AddCommand(newAddCmd(correlationID))
//
// Naming the group without reaching a subverb prints its help and reports
// a usage error — nothing was done, the same answer `aiwf` itself gives
// when handed no verb, and `aiwf add` and `aiwf render` when handed no
// kind or format.
//
// The RunE is load-bearing rather than cosmetic. Cobra returns
// flag.ErrHelp for a command that is not Runnable before it validates
// arguments, so a group whose behavior lives entirely in its children has
// no reachable Args constraint: it answers a subverb it does not have
// with help and exit 0. Supplying RunE makes the group Runnable, which is
// what lets the NoArgs below reject that name (G-0528).
func MarkVerbGroup(cmd *cobra.Command) *cobra.Command {
	cmd.Args = cobra.NoArgs
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.RunE = func(c *cobra.Command, _ []string) error {
		// NoArgs has run by now, so the only way here is with no subverb
		// named at all.
		_ = c.Help()
		return WrapExitCode(ExitUsage)
	}
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[AnnotationVerbGroup] = ""
	return cmd
}

// IsVerbGroup reports whether cmd was built by MarkVerbGroup.
func IsVerbGroup(cmd *cobra.Command) bool {
	_, ok := cmd.Annotations[AnnotationVerbGroup]
	return ok
}
