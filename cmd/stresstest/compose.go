package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/stresstest"
)

func newComposeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compose <raw-report-path>",
		Short: "Render a human-readable summary from a raw-report JSONL file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCompose(args[0], cmd.OutOrStdout())
		},
	}
	return cmd
}

func runCompose(path string, out io.Writer) error {
	result, err := stresstest.Compose(path)
	if err != nil {
		return fmt.Errorf("composing report: %w", err)
	}
	_, _ = fmt.Fprintf(out, "%d event(s)", len(result.Events))
	if result.Truncated {
		_, _ = fmt.Fprint(out, " (trailing line truncated — dropped)")
	}
	_, _ = fmt.Fprintln(out)
	for i, raw := range result.Events {
		_, _ = fmt.Fprintf(out, "  [%d] %s\n", i, raw)
	}
	return nil
}
