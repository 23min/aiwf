package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/manifest"
	"github.com/23min/ai-workflow-v2/tools/internal/render"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
	"github.com/23min/ai-workflow-v2/tools/internal/verb"
)

// runImport handles `aiwf import <manifest>`. Reads the manifest,
// runs the import verb against the tree, and either renders findings
// (no writes) or applies each plan (one commit per plan).
//
// Flags:
//
//	--root           consumer repo root
//	--actor          override the manifest's `actor` (and aiwf.yaml)
//	--on-collision   fail (default) | skip | update
//	--dry-run        validate the projection and print what would happen, no writes
func runImport(args []string) int {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	root := fs.String("root", "", "consumer repo root")
	actor := fs.String("actor", "", "actor for the commit trailer (overrides manifest and aiwf.yaml)")
	onCollision := fs.String("on-collision", verb.OnCollisionFail, "behavior when an explicit id already exists: fail|skip|update")
	dryRun := fs.Bool("dry-run", false, "validate the projection and print the would-be plan without writing")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(os.Stderr, "aiwf import: usage: aiwf import <manifest.yaml|manifest.json>")
		return exitUsage
	}
	manifestPath := rest[0]

	rootDir, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf import: %v\n", err)
		return exitUsage
	}

	m, err := manifest.ParseFile(manifestPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf import: %v\n", err)
		return exitUsage
	}

	// Actor resolution: --actor wins, then manifest.actor, then
	// aiwf.yaml derivation via resolveActor.
	actorStr := *actor
	if actorStr == "" {
		actorStr = m.Actor
	}
	if actorStr == "" {
		resolved, aErr := resolveActor("", rootDir)
		if aErr != nil {
			fmt.Fprintf(os.Stderr, "aiwf import: %v\n", aErr)
			return exitUsage
		}
		actorStr = resolved
	}

	// dry-run is read-only; lock only when we'd write.
	if !*dryRun {
		release, rc := acquireRepoLock(rootDir, "aiwf import")
		if release == nil {
			return rc
		}
		defer release()
	}

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf import: loading tree: %v\n", err)
		return exitInternal
	}

	res, err := verb.Import(tr, m, actorStr, verb.ImportOptions{OnCollision: *onCollision})
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf import: %v\n", err)
		return exitUsage
	}

	if check.HasErrors(res.Findings) {
		_ = render.Text(os.Stderr, res.Findings)
		return exitFindings
	}
	if len(res.Plans) == 0 {
		fmt.Println("aiwf import: manifest had no entities to import.")
		return exitOK
	}

	if *dryRun {
		fmt.Printf("aiwf import: dry-run — %d plan(s) would land:\n", len(res.Plans))
		for _, p := range res.Plans {
			fmt.Printf("  %s\n", p.Subject)
			for _, op := range p.Ops {
				fmt.Printf("    write %s (%d bytes)\n", op.Path, len(op.Content))
			}
		}
		fmt.Println("\naiwf import: dry-run complete. Re-run without --dry-run to apply.")
		return exitOK
	}

	for i, p := range res.Plans {
		if applyErr := verb.Apply(ctx, rootDir, p); applyErr != nil {
			fmt.Fprintf(os.Stderr, "aiwf import: applying plan %d: %v\n", i, applyErr)
			return exitInternal
		}
		fmt.Println(p.Subject)
	}
	return exitOK
}
