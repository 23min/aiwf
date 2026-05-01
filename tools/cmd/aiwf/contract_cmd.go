package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/23min/ai-workflow-v2/tools/internal/aiwfyaml"
	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/config"
	"github.com/23min/ai-workflow-v2/tools/internal/contractcheck"
	"github.com/23min/ai-workflow-v2/tools/internal/contractverify"
	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/recipe"
	"github.com/23min/ai-workflow-v2/tools/internal/render"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
	"github.com/23min/ai-workflow-v2/tools/internal/verb"
)

// runContract is the dispatcher for `aiwf contract <subcommand>`.
// I1.4 shipped `verify`; I1.5 added `bind`/`unbind`; I1.6 adds the
// recipe family.
func runContract(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "aiwf contract: missing subcommand. Try 'aiwf contract verify'.")
		return exitUsage
	}
	switch args[0] {
	case "verify":
		return runContractVerify(args[1:])
	case "bind":
		return runContractBind(args[1:])
	case "unbind":
		return runContractUnbind(args[1:])
	case "recipes":
		return runContractRecipes(args[1:])
	case "recipe":
		return runContractRecipe(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "aiwf contract: unknown subcommand %q\n", args[0])
		return exitUsage
	}
}

// runContractRecipe is the second-level dispatcher for
// `aiwf contract recipe <subcommand>`.
func runContractRecipe(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "aiwf contract recipe: missing subcommand. Try 'aiwf contract recipe show <name>'.")
		return exitUsage
	}
	switch args[0] {
	case "show":
		return runContractRecipeShow(args[1:])
	case "install":
		return runContractRecipeInstall(args[1:])
	case "remove":
		return runContractRecipeRemove(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "aiwf contract recipe: unknown subcommand %q\n", args[0])
		return exitUsage
	}
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

	skip := make(map[string]bool)
	for _, e := range tr.ByKind(entity.KindContract) {
		if e.Status == "rejected" || e.Status == "retired" {
			skip[e.ID] = true
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

// runContractVerify runs the verify and evolve passes for every
// non-terminal contract binding in aiwf.yaml. Output respects the
// standard --format=text/json envelope and exit codes.
func runContractVerify(args []string) int {
	fs := flag.NewFlagSet("contract verify", flag.ContinueOnError)
	root := fs.String("root", "", "consumer repo root (default: discover via aiwf.yaml)")
	format := fs.String("format", "text", "output format: text or json")
	pretty := fs.Bool("pretty", false, "indent JSON output (only with --format=json)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintf(os.Stderr, "aiwf contract verify: --format must be 'text' or 'json', got %q\n", *format)
		return exitUsage
	}

	rootDir, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract verify: %v\n", err)
		return exitUsage
	}

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract verify: loading tree: %v\n", err)
		return exitInternal
	}

	contracts, err := loadContractsBlock(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract verify: %v\n", err)
		return exitInternal
	}

	findings := runContractValidation(ctx, tr, rootDir, contracts)
	applyHintsLikeRun(findings)
	check.SortFindings(findings)

	switch *format {
	case "text":
		if err := render.Text(os.Stdout, findings); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf contract verify: writing output: %v\n", err)
			return exitInternal
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
		if err := render.JSON(os.Stdout, env, *pretty); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf contract verify: writing output: %v\n", err)
			return exitInternal
		}
	}

	if check.HasErrors(findings) {
		return exitFindings
	}
	return exitOK
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

// runContractBind handles `aiwf contract bind <C-id> --validator
// <name> --schema <path> --fixtures <path> [--force]`.
func runContractBind(args []string) int {
	fs := flag.NewFlagSet("contract bind", flag.ContinueOnError)
	root := fs.String("root", "", "consumer repo root")
	actor := fs.String("actor", "", "actor for the commit trailer")
	validator := fs.String("validator", "", "validator name (must be declared in aiwf.yaml.contracts.validators)")
	schema := fs.String("schema", "", "repo-relative path to the schema file")
	fixtures := fs.String("fixtures", "", "repo-relative path to the fixtures-tree root")
	force := fs.Bool("force", false, "replace an existing binding even when values differ")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(reorderFlagsFirst(args, []string{"root", "actor", "validator", "schema", "fixtures", "force"})); err != nil {
		return exitUsage
	}
	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(os.Stderr, "aiwf contract bind: usage: aiwf contract bind <C-id> --validator <name> --schema <path> --fixtures <path> [--force]")
		return exitUsage
	}
	id := rest[0]

	rootDir, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract bind: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(*actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract bind: %v\n", err)
		return exitUsage
	}

	release, rc := acquireRepoLock(rootDir, "aiwf contract bind")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract bind: loading tree: %v\n", err)
		return exitInternal
	}
	doc, contracts, err := loadContractsDoc(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract bind: %v\n", err)
		return exitUsage
	}

	result, err := verb.ContractBind(ctx, tr, doc, contracts, id, actorStr, verb.ContractBindOptions{
		Validator: *validator,
		Schema:    *schema,
		Fixtures:  *fixtures,
		Force:     *force,
	})
	return finishVerb(ctx, rootDir, "aiwf contract bind", result, err)
}

// runContractRecipes handles `aiwf contract recipes`. Lists embedded
// recipes plus the validators currently declared in aiwf.yaml so the
// user (or LLM) can see both sides at a glance.
func runContractRecipes(args []string) int {
	fs := flag.NewFlagSet("contract recipes", flag.ContinueOnError)
	root := fs.String("root", "", "consumer repo root")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	rootDir, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipes: %v\n", err)
		return exitUsage
	}

	embedded, err := recipe.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipes: %v\n", err)
		return exitInternal
	}

	contracts, err := loadContractsBlock(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipes: %v\n", err)
		return exitInternal
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
	return exitOK
}

// runContractRecipeShow handles `aiwf contract recipe show <name>`.
// Prints the embedded recipe's full markdown to stdout.
func runContractRecipeShow(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "aiwf contract recipe show: usage: aiwf contract recipe show <name>")
		return exitUsage
	}
	r, err := recipe.Get(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipe show: %v\n", err)
		return exitUsage
	}
	if _, err := os.Stdout.Write(r.Markdown); err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipe show: %v\n", err)
		return exitInternal
	}
	return exitOK
}

// runContractRecipeInstall handles `aiwf contract recipe install
// <name>` and `aiwf contract recipe install --from <path>`. The two
// flag shapes are mutually exclusive: the positional name reads the
// embedded recipe set; `--from` reads a custom-validator YAML file.
func runContractRecipeInstall(args []string) int {
	fs := flag.NewFlagSet("contract recipe install", flag.ContinueOnError)
	root := fs.String("root", "", "consumer repo root")
	actor := fs.String("actor", "", "actor for the commit trailer")
	from := fs.String("from", "", "path to a custom-validator YAML file")
	force := fs.Bool("force", false, "replace an existing validator with a different definition")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(reorderFlagsFirst(args, []string{"root", "actor", "from", "force"})); err != nil {
		return exitUsage
	}
	rest := fs.Args()

	var (
		r       recipe.Recipe
		loadErr error
	)
	switch {
	case *from != "" && len(rest) > 0:
		fmt.Fprintln(os.Stderr, "aiwf contract recipe install: pass either <name> or --from <path>, not both")
		return exitUsage
	case *from != "":
		r, loadErr = recipe.ParseFile(*from)
	case len(rest) == 1:
		r, loadErr = recipe.Get(rest[0])
	default:
		fmt.Fprintln(os.Stderr, "aiwf contract recipe install: usage: aiwf contract recipe install <name> | --from <path>")
		return exitUsage
	}
	if loadErr != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipe install: %v\n", loadErr)
		return exitUsage
	}

	rootDir, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipe install: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(*actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipe install: %v\n", err)
		return exitUsage
	}

	release, rc := acquireRepoLock(rootDir, "aiwf contract recipe install")
	if release == nil {
		return rc
	}
	defer release()

	doc, contracts, err := loadContractsDoc(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipe install: %v\n", err)
		return exitUsage
	}

	ctx := context.Background()
	result, err := verb.RecipeInstall(ctx, doc, contracts, r.Name, r.Validator, actorStr, verb.RecipeInstallOptions{Force: *force})
	return finishVerb(ctx, rootDir, "aiwf contract recipe install", result, err)
}

// runContractRecipeRemove handles `aiwf contract recipe remove <name>`.
func runContractRecipeRemove(args []string) int {
	fs := flag.NewFlagSet("contract recipe remove", flag.ContinueOnError)
	root := fs.String("root", "", "consumer repo root")
	actor := fs.String("actor", "", "actor for the commit trailer")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(reorderFlagsFirst(args, []string{"root", "actor"})); err != nil {
		return exitUsage
	}
	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(os.Stderr, "aiwf contract recipe remove: usage: aiwf contract recipe remove <name>")
		return exitUsage
	}
	name := rest[0]

	rootDir, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipe remove: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(*actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipe remove: %v\n", err)
		return exitUsage
	}

	release, rc := acquireRepoLock(rootDir, "aiwf contract recipe remove")
	if release == nil {
		return rc
	}
	defer release()

	doc, contracts, err := loadContractsDoc(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract recipe remove: %v\n", err)
		return exitUsage
	}

	ctx := context.Background()
	result, err := verb.RecipeRemove(ctx, doc, contracts, name, actorStr)
	return finishVerb(ctx, rootDir, "aiwf contract recipe remove", result, err)
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

// runContractUnbind handles `aiwf contract unbind <C-id>`.
func runContractUnbind(args []string) int {
	fs := flag.NewFlagSet("contract unbind", flag.ContinueOnError)
	root := fs.String("root", "", "consumer repo root")
	actor := fs.String("actor", "", "actor for the commit trailer")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(reorderFlagsFirst(args, []string{"root", "actor"})); err != nil {
		return exitUsage
	}
	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(os.Stderr, "aiwf contract unbind: usage: aiwf contract unbind <C-id>")
		return exitUsage
	}
	id := rest[0]

	rootDir, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract unbind: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(*actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract unbind: %v\n", err)
		return exitUsage
	}

	release, rc := acquireRepoLock(rootDir, "aiwf contract unbind")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	doc, contracts, err := loadContractsDoc(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract unbind: %v\n", err)
		return exitUsage
	}

	result, err := verb.ContractUnbind(ctx, doc, contracts, id, actorStr)
	return finishVerb(ctx, rootDir, "aiwf contract unbind", result, err)
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
