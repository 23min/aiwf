package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
)

// newWhoamiCmd builds `aiwf whoami`: prints the resolved actor for the
// current context, plus the source label that produced it. Useful to
// confirm what `aiwf-actor:` trailer the next mutating verb would write.
func newWhoamiCmd() *cobra.Command {
	var (
		root  string
		actor string
	)
	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Print the resolved actor and the source it came from",
		Example: `  # Show the actor derived from git config user.email
  aiwf whoami

  # Echo back an explicit actor (validates the role/identifier shape)
  aiwf whoami --actor human/peter`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(runWhoamiCmd(root, actor))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root (default: discover via aiwf.yaml)")
	cmd.Flags().StringVar(&actor, "actor", "", "actor override; echoes back if valid")
	return cmd
}

func runWhoamiCmd(root, actor string) int {
	rootDir, err := resolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf whoami: %v\n", err)
		return cliutil.ExitUsage
	}

	resolved, source, err := cliutil.ResolveActorWithSource(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf whoami: %v\n", err)
		return cliutil.ExitFindings
	}
	fmt.Printf("%s (source: %s)\n", resolved, source)
	return cliutil.ExitOK
}
