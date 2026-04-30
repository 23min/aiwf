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
)

// runContract is the dispatcher for `aiwf contract <subcommand>`.
// The verify subcommand is the only one in I1.4; bind/unbind and the
// recipe family land in later increments.
func runContract(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "aiwf contract: missing subcommand. Try 'aiwf contract verify'.")
		return exitUsage
	}
	switch args[0] {
	case "verify":
		return runContractVerify(args[1:])
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
