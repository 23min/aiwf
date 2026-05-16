package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/contractcheck"
	"github.com/23min/aiwf/internal/contractverify"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/recipe"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/verb"
)

// newContractCmd builds the `aiwf contract` parent command. Five direct
// children (verify, bind, unbind, recipes, recipe) plus the recipe
// sub-tree (show, install, remove). The parent itself is non-Runnable
// — `aiwf contract` with no subcommand prints help.
func newContractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "contract",
		Short:         "Manage contract entities and their validators",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.AddCommand(newContractVerifyCmd())
	cmd.AddCommand(newContractBindCmd())
	cmd.AddCommand(newContractUnbindCmd())
	cmd.AddCommand(newContractRecipesCmd())
	cmd.AddCommand(newContractRecipeCmd())
	return cmd
}

// newContractVerifyCmd builds `aiwf contract verify`. Runs the verify
// and evolve passes for every non-terminal contract binding declared
// in aiwf.yaml. Output respects the standard --format=text/json
// envelope and exit codes.
func newContractVerifyCmd() *cobra.Command {
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
			return cliutil.WrapExitCode(runContractVerifyCmd(root, format, pretty))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root (default: discover via aiwf.yaml)")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "indent JSON output (only with --format=json)")
	registerFormatCompletion(cmd)
	return cmd
}

func runContractVerifyCmd(root, format string, pretty bool) int {
	if format != "text" && format != "json" {
		fmt.Fprintf(os.Stderr, "aiwf contract verify: --format must be 'text' or 'json', got %q\n", format)
		return cliutil.ExitUsage
	}
	rootDir, err := resolveRoot(root)
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
	contracts, err := loadContractsBlock(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract verify: %v\n", err)
		return cliutil.ExitInternal
	}
	findings := runContractValidation(ctx, tr, rootDir, contracts)
	applyHintsLikeRun(findings)
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
			Version:  Version,
			Status:   render.StatusFor(findings),
			Findings: findings,
			Metadata: map[string]any{
				"root":     rootDir,
				"bindings": bindingCount(contracts),
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

// runContractValidation is the shared entry point for both the CLI
// `aiwf contract verify` and the pre-push integration in `aiwf check`.
// It runs contractcheck (config correspondence) plus contractverify
// (subprocess validators) and returns the combined findings slice
// (un-sorted, hints not applied — caller composes).
//
// A nil contracts argument is treated as "no contracts configured":
// the function returns nil. Terminal-state contract entities
// (rejected, retired) are excluded from verification.
func runContractValidation(ctx context.Context, tr *tree.Tree, rootDir string, contracts *aiwfyaml.Contracts) []check.Finding {
	if contracts == nil {
		return nil
	}
	configFindings := contractcheck.Run(tr, contracts, rootDir)

	// Build skip set keyed by canonical id so a narrow legacy binding
	// `id: C-001` matches the canonical-width terminal contract entity
	// `C-0001` (AC-2 in M-081). The verify path also canonicalizes
	// before lookup.
	skip := make(map[string]bool)
	for _, e := range tr.ByKind(entity.KindContract) {
		if e.Status == entity.StatusRejected || e.Status == entity.StatusRetired {
			skip[entity.Canonicalize(e.ID)] = true
		}
	}
	verifyResults := contractverify.Run(ctx, contractverify.Options{
		RepoRoot:  rootDir,
		Contracts: contracts,
		SkipIDs:   skip,
	})

	out := append([]check.Finding(nil), configFindings...)
	for _, r := range verifyResults {
		out = append(out, resultToFinding(r, contracts.StrictValidators))
	}
	return out
}

// applyHintsLikeRun fills the Hint field on every finding from the
// shared hint table. Mirrors the post-processing check.Run does for
// the entity-level findings; we inline it here because we don't go
// through check.Run for contract verify.
func applyHintsLikeRun(findings []check.Finding) {
	for i := range findings {
		f := &findings[i]
		if f.Hint != "" {
			continue
		}
		f.Hint = check.HintFor(f.Code, f.Subcode)
	}
}

// bindingCount is a small helper for the JSON envelope's metadata.
func bindingCount(c *aiwfyaml.Contracts) int {
	if c == nil {
		return 0
	}
	return len(c.Entries)
}

// loadContractsBlock reads aiwf.yaml from rootDir and returns the
// contracts: block (nil if absent or if the file itself is absent).
// A malformed contracts: block is an internal error — the verb can't
// proceed without trustworthy bindings.
func loadContractsBlock(rootDir string) (*aiwfyaml.Contracts, error) {
	cfgPath := filepath.Join(rootDir, config.FileName)
	if _, err := os.Stat(cfgPath); err != nil {
		return nil, nil
	}
	_, contracts, err := aiwfyaml.Read(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("reading aiwf.yaml: %w", err)
	}
	return contracts, nil
}

// loadContractsDoc reads aiwf.yaml and returns both the editable
// Doc and the parsed contracts block. Used by mutating verbs that
// need to splice the block back into the source.
func loadContractsDoc(rootDir string) (*aiwfyaml.Doc, *aiwfyaml.Contracts, error) {
	cfgPath := filepath.Join(rootDir, config.FileName)
	if _, err := os.Stat(cfgPath); err != nil {
		return nil, nil, fmt.Errorf("aiwf.yaml not found at %s; run 'aiwf init' first", cfgPath)
	}
	doc, contracts, err := aiwfyaml.Read(cfgPath)
	if err != nil {
		return nil, nil, fmt.Errorf("reading aiwf.yaml: %w", err)
	}
	return doc, contracts, nil
}

// newContractBindCmd builds `aiwf contract bind <C-id> --validator
// <name> --schema <path> --fixtures <path> [--force]`.
func newContractBindCmd() *cobra.Command {
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
			return cliutil.WrapExitCode(runContractBindCmd(args[0], root, actor, validator, schema, fixtures, force))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&validator, "validator", "", "validator name (must be declared in aiwf.yaml.contracts.validators)")
	cmd.Flags().StringVar(&schema, "schema", "", "repo-relative path to the schema file")
	cmd.Flags().StringVar(&fixtures, "fixtures", "", "repo-relative path to the fixtures-tree root")
	cmd.Flags().BoolVar(&force, "force", false, "replace an existing binding even when values differ")
	cmd.ValidArgsFunction = completeEntityIDArg(entity.KindContract, 0)
	_ = cmd.RegisterFlagCompletionFunc("validator", completeDeclaredValidators)
	return cmd
}

func runContractBindCmd(id, root, actor, validator, schema, fixtures string, force bool) int {
	rootDir, err := resolveRoot(root)
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
	doc, contracts, err := loadContractsDoc(rootDir)
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

// newContractUnbindCmd builds `aiwf contract unbind <C-id>`.
func newContractUnbindCmd() *cobra.Command {
	var (
		root  string
		actor string
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
			return cliutil.WrapExitCode(runContractUnbindCmd(args[0], root, actor))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.ValidArgsFunction = completeEntityIDArg(entity.KindContract, 0)
	return cmd
}

func runContractUnbindCmd(id, root, actor string) int {
	rootDir, err := resolveRoot(root)
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
	doc, contracts, err := loadContractsDoc(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract unbind: %v\n", err)
		return cliutil.ExitUsage
	}

	result, err := verb.ContractUnbind(ctx, doc, contracts, id, actorStr)
	return cliutil.FinishVerb(ctx, rootDir, "aiwf contract unbind", result, err)
}

// newContractRecipesCmd builds `aiwf contract recipes`. Lists embedded
// recipes plus the validators currently declared in aiwf.yaml so the
// user (or LLM) can see both sides at a glance.
func newContractRecipesCmd() *cobra.Command {
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
			return cliutil.WrapExitCode(runContractRecipesCmd(root))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	return cmd
}

func runContractRecipesCmd(root string) int {
	rootDir, err := resolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipes: %v\n", err)
		return cliutil.ExitUsage
	}

	embedded, err := recipe.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipes: %v\n", err)
		return cliutil.ExitInternal
	}

	contracts, err := loadContractsBlock(rootDir)
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
		sortStrings(names)
		for _, n := range names {
			v := contracts.Validators[n]
			fmt.Printf("  %s — %s\n", n, v.Command)
		}
	}
	return cliutil.ExitOK
}

// newContractRecipeCmd builds `aiwf contract recipe`. Three children:
// show, install, remove. The parent itself is non-Runnable.
func newContractRecipeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "recipe",
		Short:         "Manage validators (show / install / remove a recipe)",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.AddCommand(newContractRecipeShowCmd())
	cmd.AddCommand(newContractRecipeInstallCmd())
	cmd.AddCommand(newContractRecipeRemoveCmd())
	return cmd
}

// newContractRecipeShowCmd builds `aiwf contract recipe show <name>`.
// Prints the embedded recipe's full markdown to stdout.
func newContractRecipeShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Print an embedded recipe's markdown",
		Example: `  # Read the render recipe (no install)
  aiwf contract recipe show render`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(runContractRecipeShowCmd(args[0]))
		},
	}
	cmd.ValidArgsFunction = completeEmbeddedRecipeNamesArg
	return cmd
}

func runContractRecipeShowCmd(name string) int {
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

// newContractRecipeInstallCmd builds `aiwf contract recipe install
// <name>` and `aiwf contract recipe install --from <path>`. The two
// flag shapes are mutually exclusive: the positional name reads the
// embedded recipe set; `--from` reads a custom-validator YAML file.
func newContractRecipeInstallCmd() *cobra.Command {
	var (
		root  string
		actor string
		from  string
		force bool
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
			return cliutil.WrapExitCode(runContractRecipeInstallCmd(args, root, actor, from, force))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&from, "from", "", "path to a custom-validator YAML file")
	cmd.Flags().BoolVar(&force, "force", false, "replace an existing validator with a different definition")
	cmd.ValidArgsFunction = completeEmbeddedRecipeNamesArg
	return cmd
}

func runContractRecipeInstallCmd(args []string, root, actor, from string, force bool) int {
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

	rootDir, err := resolveRoot(root)
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

	doc, contracts, err := loadContractsDoc(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipe install: %v\n", err)
		return cliutil.ExitUsage
	}

	ctx := context.Background()
	result, err := verb.RecipeInstall(ctx, doc, contracts, r.Name, r.Validator, actorStr, verb.RecipeInstallOptions{Force: force})
	return cliutil.FinishVerb(ctx, rootDir, "aiwf contract recipe install", result, err)
}

// newContractRecipeRemoveCmd builds `aiwf contract recipe remove
// <name>`. Removes a declared validator; errors when bindings still
// reference it.
func newContractRecipeRemoveCmd() *cobra.Command {
	var (
		root  string
		actor string
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
			return cliutil.WrapExitCode(runContractRecipeRemoveCmd(args[0], root, actor))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.ValidArgsFunction = completeDeclaredValidatorsArg
	return cmd
}

func runContractRecipeRemoveCmd(name, root, actor string) int {
	rootDir, err := resolveRoot(root)
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

	doc, contracts, err := loadContractsDoc(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipe remove: %v\n", err)
		return cliutil.ExitUsage
	}

	ctx := context.Background()
	result, err := verb.RecipeRemove(ctx, doc, contracts, name, actorStr)
	return cliutil.FinishVerb(ctx, rootDir, "aiwf contract recipe remove", result, err)
}

// completeEmbeddedRecipeNamesArg is a Cobra ValidArgsFunction that
// returns the names of every embedded recipe. Used by `recipe show`
// (read-only) and `recipe install` (positional name; mutex with
// --from). Failures collapse to the empty list per the M-054 graceful-
// no-op rule.
func completeEmbeddedRecipeNamesArg(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	rs, err := recipe.List()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	names := make([]string, 0, len(rs))
	for _, r := range rs {
		names = append(names, r.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeDeclaredValidatorsArg returns the validator names currently
// declared under aiwf.yaml.contracts.validators. Used by `recipe
// remove` (you remove what's declared, not what's in the embedded
// catalog). Cleanly returns empty when aiwf.yaml is absent or the
// block is empty.
func completeDeclaredValidatorsArg(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return declaredValidatorNames(), cobra.ShellCompDirectiveNoFileComp
}

// completeDeclaredValidators is the flag-side adapter. Used by
// `contract bind --validator <TAB>` so the user picks from the
// already-declared set rather than free-typing.
func completeDeclaredValidators(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return declaredValidatorNames(), cobra.ShellCompDirectiveNoFileComp
}

func declaredValidatorNames() []string {
	rootDir, err := resolveRoot("")
	if err != nil {
		return nil
	}
	contracts, err := loadContractsBlock(rootDir)
	if err != nil || contracts == nil || len(contracts.Validators) == 0 {
		return nil
	}
	names := make([]string, 0, len(contracts.Validators))
	for n := range contracts.Validators {
		names = append(names, n)
	}
	sortStrings(names)
	return names
}

// sortStrings is the local insertion-sort used to keep the listing
// output deterministic without pulling in the sort package.
func sortStrings(ss []string) {
	for i := 1; i < len(ss); i++ {
		for j := i; j > 0 && ss[j-1] > ss[j]; j-- {
			ss[j-1], ss[j] = ss[j], ss[j-1]
		}
	}
}

// resultToFinding converts a contractverify.Result into the Finding
// shape the render layer expects. Most codes are errors; the
// per-machine `validator-unavailable` code is a warning by default,
// upgraded to an error by strictValidators. The path is the fixture
// path when present, otherwise empty (the user locates the issue by
// entity id).
func resultToFinding(r contractverify.Result, strictValidators bool) check.Finding {
	severity := check.SeverityError
	code := r.Code
	subcode := ""
	if r.Code == contractverify.CodeValidatorUnavailable {
		// Render as a contract-config finding with subcode so the
		// hint table and the rest of the user-facing surface treat
		// it consistently with other contract-config findings.
		code = "contract-config"
		subcode = "validator-unavailable"
		if !strictValidators {
			severity = check.SeverityWarning
		}
	}
	return check.Finding{
		Code:     code,
		Severity: severity,
		Subcode:  subcode,
		Message:  r.Message,
		Path:     r.FixturePath,
		EntityID: r.EntityID,
	}
}
