package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// list.go — M-0249/AC-3: `list` enumerates every scenario name
// `run --scenario` can select, so an operator can discover what's
// runnable without reading Go source.

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Enumerate every scenario name run --scenario can select",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.OutOrStdout())
		},
	}
	return cmd
}

// runList prints every registered catalog name, one per line, in
// catalog order.
func runList(out io.Writer) error {
	for _, name := range scenarioNames() {
		if _, err := fmt.Fprintln(out, name); err != nil { //coverage:ignore not portably triggerable: writing to the caller-supplied out (stdout in production, a bytes.Buffer in tests) has no realistic failure mode either path exercises
			return err
		}
	}
	return nil
}
