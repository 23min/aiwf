package contract

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// newUnbindCmd builds `aiwf contract unbind <C-id>`.
func newUnbindCmd() *cobra.Command {
	var (
		root  string
		actor string
		out   *cliutil.OutputFormat
	)
	cmd := &cobra.Command{
		Use:   "unbind <C-id>",
		Short: "Remove a contract binding from aiwf.yaml (entity status untouched)",
		Example: `  # Drop the binding without changing the contract entity's status
  aiwf contract unbind C-001`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(runUnbind(args[0], root, actor, *out))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	out = cliutil.AddFormatFlags(cmd)
	cmd.ValidArgsFunction = cliutil.CompleteEntityIDArg(entity.KindContract, 0)
	return cmd
}

func runUnbind(id, root, actor string, out cliutil.OutputFormat) int {
	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract unbind: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract unbind: %v\n", err)
		return cliutil.ExitUsage
	}

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf contract unbind")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	doc, contracts, err := cliutil.LoadContractsDoc(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract unbind: %v\n", err)
		return cliutil.ExitUsage
	}

	result, err := verb.ContractUnbind(ctx, doc, contracts, id, actorStr)
	return cliutil.FinishVerb(ctx, rootDir, "aiwf contract unbind", result, err, out)
}
