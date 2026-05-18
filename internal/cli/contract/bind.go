package contract

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/verb"
)

// newBindCmd builds `aiwf contract bind <C-id> --validator <name>
// --schema <path> --fixtures <path> [--force]`.
func newBindCmd() *cobra.Command {
	var (
		root      string
		actor     string
		validator string
		schema    string
		fixtures  string
		force     bool
	)
	cmd := &cobra.Command{
		Use:   "bind <C-id>",
		Short: "Add or replace a contract binding in aiwf.yaml",
		Example: `  # Bind a validator, schema, and fixtures atomically
  aiwf contract bind C-001 --validator render --schema schemas/render.cue --fixtures fixtures/render

  # Replace an existing binding (different values)
  aiwf contract bind C-001 --validator render --schema schemas/v2.cue --fixtures fixtures/render --force`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(runBind(args[0], root, actor, validator, schema, fixtures, force))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&validator, "validator", "", "validator name (must be declared in aiwf.yaml.contracts.validators)")
	cmd.Flags().StringVar(&schema, "schema", "", "repo-relative path to the schema file")
	cmd.Flags().StringVar(&fixtures, "fixtures", "", "repo-relative path to the fixtures-tree root")
	cmd.Flags().BoolVar(&force, "force", false, "replace an existing binding even when values differ")
	cmd.ValidArgsFunction = cliutil.CompleteEntityIDArg(entity.KindContract, 0)
	_ = cmd.RegisterFlagCompletionFunc("validator", completeDeclaredValidators)
	return cmd
}

func runBind(id, root, actor, validator, schema, fixtures string, force bool) int {
	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract bind: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract bind: %v\n", err)
		return cliutil.ExitUsage
	}

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf contract bind")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract bind: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}
	doc, contracts, err := cliutil.LoadContractsDoc(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract bind: %v\n", err)
		return cliutil.ExitUsage
	}

	result, err := verb.ContractBind(ctx, tr, doc, contracts, id, actorStr, rootDir, verb.ContractBindOptions{
		Validator: validator,
		Schema:    schema,
		Fixtures:  fixtures,
		Force:     force,
	})
	return cliutil.FinishVerb(ctx, rootDir, "aiwf contract bind", result, err)
}
