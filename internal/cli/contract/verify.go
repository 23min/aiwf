package contract

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/version"
)

// newVerifyCmd builds `aiwf contract verify`. Runs the verify and
// evolve passes for every non-terminal contract binding declared in
// aiwf.yaml. Output respects the standard --format=text/json envelope
// and exit codes.
func newVerifyCmd() *cobra.Command {
	var (
		root   string
		format string
		pretty bool
	)
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Run the verify and evolve passes for every contract binding",
		Example: `  # Validate every contract binding
  aiwf contract verify

  # JSON envelope for CI scripts
  aiwf contract verify --format=json --pretty`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(Run(root, format, pretty))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root (default: discover via aiwf.yaml)")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "indent JSON output (only with --format=json)")
	cliutil.RegisterFormatCompletion(cmd)
	return cmd
}

// Run is the exported entry point for `aiwf contract verify`.
// `contract verify` is read-only — it loads the tree, runs the
// contractcheck (config correspondence) and contractverify
// (subprocess validators) passes, and prints findings. The
// internal/policies/read_only.go entry pins this path so a future
// regression that adds gitops.Commit / verb.Apply / os.WriteFile to
// the verify body fails CI.
func Run(root, format string, pretty bool) int {
	if format != "text" && format != "json" {
		fmt.Fprintf(os.Stderr, "aiwf contract verify: --format must be 'text' or 'json', got %q\n", format)
		return cliutil.ExitUsage
	}
	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract verify: %v\n", err)
		return cliutil.ExitUsage
	}
	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract verify: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}
	contracts, err := cliutil.LoadContractsBlock(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract verify: %v\n", err)
		return cliutil.ExitInternal
	}
	findings := RunValidation(ctx, tr, rootDir, contracts)
	ApplyHintsLikeRun(findings)
	check.SortFindings(findings)

	switch format {
	case "text":
		if err := render.Text(os.Stdout, findings); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf contract verify: writing output: %v\n", err)
			return cliutil.ExitInternal
		}
	case "json":
		env := render.Envelope{
			Tool:     "aiwf",
			Version:  version.Current().Version,
			Status:   render.StatusFor(findings),
			Findings: findings,
			Metadata: map[string]any{
				"root":     rootDir,
				"bindings": BindingCount(contracts),
				"findings": len(findings),
			},
		}
		if err := render.JSON(os.Stdout, env, pretty); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf contract verify: writing output: %v\n", err)
			return cliutil.ExitInternal
		}
	}
	if check.HasErrors(findings) {
		return cliutil.ExitFindings
	}
	return cliutil.ExitOK
}
