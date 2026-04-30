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
	"github.com/23min/ai-workflow-v2/tools/internal/render"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
	"github.com/23min/ai-workflow-v2/tools/internal/verb"
)

// runContract is the dispatcher for `aiwf contract <subcommand>`.
// I1.4 shipped `verify`; I1.5 adds `bind` and `unbind`. The recipe
// family lands in I1.6.
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
	default:
		fmt.Fprintf(os.Stderr, "aiwf contract: unknown subcommand %q\n", args[0])
		return exitUsage
	}
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

	findings := append([]check.Finding(nil), configFindings...)
	for _, r := range verifyResults {
		findings = append(findings, resultToFinding(r))
	}
	applyHintsLikeRun(findings)

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

	result, err := verb.ContractBind(tr, doc, contracts, id, actorStr, verb.ContractBindOptions{
		Validator: *validator,
		Schema:    *schema,
		Fixtures:  *fixtures,
		Force:     *force,
	})
	return finishVerb(ctx, rootDir, "aiwf contract bind", result, err)
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

	ctx := context.Background()
	doc, contracts, err := loadContractsDoc(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf contract unbind: %v\n", err)
		return exitUsage
	}

	result, err := verb.ContractUnbind(doc, contracts, id, actorStr)
	return finishVerb(ctx, rootDir, "aiwf contract unbind", result, err)
}

// resultToFinding converts a contractverify.Result into the Finding
// shape the render layer expects. All contract-verify codes are
// errors; the path is the fixture path when present, otherwise empty
// (the user locates the issue by entity id).
func resultToFinding(r contractverify.Result) check.Finding {
	return check.Finding{
		Code:     r.Code,
		Severity: check.SeverityError,
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
