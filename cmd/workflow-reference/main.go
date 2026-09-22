// Command workflow-reference regenerates this repository's workflow reference.
// It is development tooling, not part of the installed aiwf command tree.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/pathutil"
	"github.com/23min/aiwf/internal/workflows/spec"
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:]))
}

func run(ctx context.Context, args []string) int {
	var destination string
	exitCode := cliutil.ExitUsage
	cmd := &cobra.Command{
		Use:           "workflow-reference",
		Short:         "Regenerate the declared workflow reference from the repository root",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			exitCode = cliutil.ExitInternal
			if err := cmd.Context().Err(); err != nil {
				return fmt.Errorf("generating workflow reference: %w", err)
			}
			if err := pathutil.AtomicWriteFile(destination, spec.RenderReference(), 0o644); err != nil {
				return fmt.Errorf("writing workflow reference to %s (choose an existing parent directory with --out): %w", destination, err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&destination, "out", "docs/reference/workflow-legality.md", "output file; parent directory must exist")
	cmd.SetArgs(args)
	if err := cmd.ExecuteContext(ctx); err != nil {
		cliutil.Errorln(err.Error())
		return exitCode
	}
	return cliutil.ExitOK
}
