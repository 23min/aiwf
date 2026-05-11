package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/manifest"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/verb"
)

// newImportCmd builds `aiwf import <manifest>`. Reads the manifest,
// runs the import verb against the tree, and either renders findings
// (no writes) or applies each plan (one commit per plan).
//
// Flags:
//
//	--root           consumer repo root
//	--actor          override the manifest's `actor` (and aiwf.yaml)
//	--on-collision   fail (default) | skip | update
//	--dry-run        validate the projection and print what would happen, no writes
func newImportCmd() *cobra.Command {
	var (
		root        string
		actor       string
		principal   string
		onCollision string
		dryRun      bool
	)
	cmd := &cobra.Command{
		Use:   "import <manifest>",
		Short: "Bulk-create entities from a YAML/JSON manifest (one commit by default)",
		Example: `  # Validate a manifest without writing
  aiwf import seed.yaml --dry-run

  # Apply, replacing entities with explicit ids that already exist
  aiwf import seed.yaml --on-collision update`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return wrapExitCode(runImportCmd(args[0], root, actor, principal, onCollision, dryRun))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer (overrides manifest and aiwf.yaml)")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; per-entity scope gating is deferred to G22 — bulk import currently only enforces principal coherence)")
	cmd.Flags().StringVar(&onCollision, "on-collision", verb.OnCollisionFail, "behavior when an explicit id already exists: fail|skip|update")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate the projection and print the would-be plan without writing")
	_ = cmd.RegisterFlagCompletionFunc("on-collision", cobra.FixedCompletions(
		[]string{verb.OnCollisionFail, verb.OnCollisionSkip, verb.OnCollisionUpdate},
		cobra.ShellCompDirectiveNoFileComp,
	))
	return cmd
}

func runImportCmd(manifestPath, root, actor, principal, onCollision string, dryRun bool) int {
	rootDir, err := resolveRoot(root)
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
	actorStr := actor
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
	if !dryRun {
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

	// Provenance coherence: when the operator is non-human, a principal
	// is required (the I2.5 trailer-coherence rule). Per-entity scope
	// gating (running Allow against each plan's CreationRefs) is
	// deferred to G22; bulk-import attribution lives there.
	principalStr := strings.TrimSpace(principal)
	actorIsNonHuman := actorStr != "" && !strings.HasPrefix(actorStr, "human/")
	if actorIsNonHuman && principalStr == "" {
		fmt.Fprintf(os.Stderr, "aiwf import: --principal human/<id> is required when --actor is non-human (got actor=%q)\n", actorStr)
		return exitUsage
	}
	if !actorIsNonHuman && principalStr != "" {
		fmt.Fprintln(os.Stderr, "aiwf import: --principal is forbidden when --actor is human/ (humans act directly)")
		return exitUsage
	}

	res, err := verb.Import(ctx, tr, m, actorStr, verb.ImportOptions{
		OnCollision:    onCollision,
		TitleMaxLength: configuredTitleMaxLength(rootDir),
	})
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

	if dryRun {
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
		if actorIsNonHuman {
			// Stamp the principal trailer on every per-entity plan
			// so the resulting commits satisfy CheckTrailerCoherence
			// (non-human actor requires a principal). Per-entity
			// scope authorization (aiwf-on-behalf-of /
			// aiwf-authorized-by) is G22.
			p.Trailers = append(p.Trailers, gitops.Trailer{
				Key:   gitops.TrailerPrincipal,
				Value: principalStr,
			})
		}
		if applyErr := verb.Apply(ctx, rootDir, p); applyErr != nil {
			fmt.Fprintf(os.Stderr, "aiwf import: applying plan %d: %v\n", i, applyErr)
			return exitInternal
		}
		fmt.Println(p.Subject)
	}
	return exitOK
}
