package main

import "github.com/spf13/cobra"

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "stresstest",
		Short: "On-demand correctness stress harness for aiwf (dev-only; E-0062)",
	}
	root.AddCommand(newRunCmd())
	root.AddCommand(newComposeCmd())
	root.AddCommand(newListCmd())
	return root
}
