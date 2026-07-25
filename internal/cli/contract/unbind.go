package contract

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/verb"
)

// newUnbindCmd builds `aiwf contract unbind <C-id>`.
func newUnbindCmd(correlationID string) *cobra.Command {
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
	out.CorrelationID = correlationID
	cmd.ValidArgsFunction = cliutil.CompleteEntityIDArg(entity.KindContract, 0)
	return cmd
}

func runUnbind(id, root, actor string, out cliutil.OutputFormat) (code int) {
	rootDir, actorStr, code, ok := cliutil.ResolvePrelude("aiwf contract unbind", root, actor)
	if !ok {
		return code
	}

	ctx := context.Background()

	finish := cliutil.BeginVerbDiag(rootDir, "contract-unbind", id, actorStr, out.CorrelationID)
	var sha string
	defer finish(&code, &sha)

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf contract unbind", out)
	if release == nil {
		return rc
	}
	defer release()

	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil { //coverage:ignore tree.Load errors only on filesystem IO failure (e.g. a permission fault) or context cancellation; malformed entities surface as load findings, not an error here.
		cliutil.Errorf("aiwf contract unbind: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}
	doc, contracts, err := cliutil.LoadContractsDoc(rootDir)
	if err != nil {
		cliutil.Errorf("aiwf contract unbind: %v\n", err)
		return cliutil.ExitUsage
	}

	result, err := verb.ContractUnbind(ctx, tr, doc, contracts, id, actorStr, rootDir)
	code, sha = cliutil.FinishVerb(ctx, rootDir, "aiwf contract unbind", result, err, out)
	return code
}
