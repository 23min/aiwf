package contract

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/recipe"
	"github.com/23min/aiwf/internal/verb"
)

// newRecipesCmd builds `aiwf contract recipes`. Lists embedded
// recipes plus the validators currently declared in aiwf.yaml so the
// user (or LLM) can see both sides at a glance.
func newRecipesCmd() *cobra.Command {
	var root string
	cmd := &cobra.Command{
		Use:   "recipes",
		Short: "List embedded validator recipes and currently declared validators",
		Example: `  # Survey what's available and what's already wired
  aiwf contract recipes`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(runRecipes(root))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	return cmd
}

func runRecipes(root string) int {
	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipes: %v\n", err)
		return cliutil.ExitUsage
	}

	embedded, err := recipe.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipes: %v\n", err)
		return cliutil.ExitInternal
	}

	contracts, err := cliutil.LoadContractsBlock(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipes: %v\n", err)
		return cliutil.ExitInternal
	}

	fmt.Println("Embedded recipes (install via `aiwf contract recipe install <name>`):")
	for _, r := range embedded {
		fmt.Printf("  %s\n", r.Name)
	}
	fmt.Println()
	fmt.Println("Currently declared validators in aiwf.yaml.contracts.validators:")
	if contracts == nil || len(contracts.Validators) == 0 {
		fmt.Println("  (none)")
	} else {
		names := make([]string, 0, len(contracts.Validators))
		for n := range contracts.Validators {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			v := contracts.Validators[n]
			fmt.Printf("  %s — %s\n", n, v.Command)
		}
	}
	return cliutil.ExitOK
}

// newRecipeCmd builds `aiwf contract recipe`. Three children: show,
// install, remove. The parent itself is non-Runnable.
func newRecipeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "recipe",
		Short:         "Manage validators (show / install / remove a recipe)",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.AddCommand(newRecipeShowCmd())
	cmd.AddCommand(newRecipeInstallCmd())
	cmd.AddCommand(newRecipeRemoveCmd())
	return cmd
}

// newRecipeShowCmd builds `aiwf contract recipe show <name>`. Prints
// the embedded recipe's full markdown to stdout.
func newRecipeShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Print an embedded recipe's markdown",
		Example: `  # Read the render recipe (no install)
  aiwf contract recipe show render`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(runRecipeShow(args[0]))
		},
	}
	cmd.ValidArgsFunction = completeEmbeddedRecipeNamesArg
	return cmd
}

func runRecipeShow(name string) int {
	r, err := recipe.Get(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipe show: %v\n", err)
		return cliutil.ExitUsage
	}
	if _, err := os.Stdout.Write(r.Markdown); err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipe show: %v\n", err)
		return cliutil.ExitInternal
	}
	return cliutil.ExitOK
}

// newRecipeInstallCmd builds `aiwf contract recipe install <name>`
// and `aiwf contract recipe install --from <path>`. The two flag
// shapes are mutually exclusive: the positional name reads the
// embedded recipe set; `--from` reads a custom-validator YAML file.
func newRecipeInstallCmd() *cobra.Command {
	var (
		root  string
		actor string
		from  string
		force bool
		out   *cliutil.OutputFormat
	)
	cmd := &cobra.Command{
		Use:   "install <name|--from <path>>",
		Short: "Install a validator from the embedded set or from a YAML file",
		Example: `  # Install one of the embedded recipes
  aiwf contract recipe install render

  # Install a custom recipe from a file
  aiwf contract recipe install --from custom-validator.yaml`,
		Args:          cobra.MaximumNArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(runRecipeInstall(args, root, actor, from, force, *out))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&from, "from", "", "path to a custom-validator YAML file")
	cmd.Flags().BoolVar(&force, "force", false, "replace an existing validator with a different definition")
	out = cliutil.AddFormatFlags(cmd)
	cmd.ValidArgsFunction = completeEmbeddedRecipeNamesArg
	return cmd
}

func runRecipeInstall(args []string, root, actor, from string, force bool, out cliutil.OutputFormat) int {
	var (
		r       recipe.Recipe
		loadErr error
	)
	switch {
	case from != "" && len(args) > 0:
		fmt.Fprintln(os.Stderr, "aiwf contract recipe install: pass either <name> or --from <path>, not both")
		return cliutil.ExitUsage
	case from != "":
		r, loadErr = recipe.ParseFile(from)
	case len(args) == 1:
		r, loadErr = recipe.Get(args[0])
	default:
		fmt.Fprintln(os.Stderr, "aiwf contract recipe install: usage: aiwf contract recipe install <name> | --from <path>")
		return cliutil.ExitUsage
	}
	if loadErr != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipe install: %v\n", loadErr)
		return cliutil.ExitUsage
	}

	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipe install: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipe install: %v\n", err)
		return cliutil.ExitUsage
	}

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf contract recipe install")
	if release == nil {
		return rc
	}
	defer release()

	doc, contracts, err := cliutil.LoadContractsDoc(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipe install: %v\n", err)
		return cliutil.ExitUsage
	}

	ctx := context.Background()
	result, err := verb.RecipeInstall(ctx, doc, contracts, r.Name, r.Validator, actorStr, verb.RecipeInstallOptions{Force: force})
	return cliutil.FinishVerb(ctx, rootDir, "aiwf contract recipe install", result, err, out)
}

// newRecipeRemoveCmd builds `aiwf contract recipe remove <name>`.
// Removes a declared validator; errors when bindings still reference
// it.
func newRecipeRemoveCmd() *cobra.Command {
	var (
		root  string
		actor string
		out   *cliutil.OutputFormat
	)
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a declared validator (errors when bindings still reference it)",
		Example: `  # Drop a validator that no contract is bound to
  aiwf contract recipe remove render`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(runRecipeRemove(args[0], root, actor, *out))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	out = cliutil.AddFormatFlags(cmd)
	cmd.ValidArgsFunction = completeDeclaredValidatorsArg
	return cmd
}

func runRecipeRemove(name, root, actor string, out cliutil.OutputFormat) int {
	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipe remove: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipe remove: %v\n", err)
		return cliutil.ExitUsage
	}

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf contract recipe remove")
	if release == nil {
		return rc
	}
	defer release()

	doc, contracts, err := cliutil.LoadContractsDoc(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipe remove: %v\n", err)
		return cliutil.ExitUsage
	}

	ctx := context.Background()
	result, err := verb.RecipeRemove(ctx, doc, contracts, name, actorStr)
	return cliutil.FinishVerb(ctx, rootDir, "aiwf contract recipe remove", result, err, out)
}
