// Package importcmd implements the `aiwf import` verb (per-verb subpackage of M-0116;
// directory and package are `importcmd` because `import` is a Go reserved word).
package importcmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/logger"
	"github.com/23min/aiwf/internal/manifest"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/verb"
	"github.com/23min/aiwf/internal/version"
)

// NewCmd builds `aiwf import <manifest>`. Reads the manifest,
// runs the import verb against the tree, and either renders findings
// (no writes) or applies each plan (one commit per plan).
//
// Flags:
//
//	--root           consumer repo root
//	--actor          override the manifest's `actor` (and aiwf.yaml)
//	--on-collision   fail (default) | skip | update
//	--dry-run        validate the projection and print what would happen, no writes
func NewCmd(correlationID string) *cobra.Command {
	var (
		root        string
		actor       string
		principal   string
		onCollision string
		dryRun      bool
		out         *cliutil.OutputFormat
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
			return cliutil.WrapExitCode(Run(args[0], root, actor, principal, onCollision, dryRun, *out))
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
	out = cliutil.AddFormatFlags(cmd)
	out.CorrelationID = correlationID
	return cmd
}

// Run executes `aiwf import`. Returns one of the cliutil.Exit* codes.
func Run(manifestPath, root, actor, principal, onCollision string, dryRun bool, out cliutil.OutputFormat) (code int) {
	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil { //coverage:ignore ResolveRoot only wraps filepath.Abs (explicit --root) or os.Getwd (no --root) — neither fails in a healthy test harness; a missing aiwf.yaml is tolerated, not an error
		return failImport(out, err.Error(), cliutil.ExitUsage)
	}

	m, err := manifest.ParseFile(manifestPath)
	if err != nil {
		return failImport(out, err.Error(), cliutil.ExitUsage)
	}

	// Actor resolution: --actor wins, then manifest.actor, then
	// aiwf.yaml derivation via cliutil.ResolveActor.
	actorStr := actor
	if actorStr == "" {
		actorStr = m.Actor
	}
	if actorStr == "" {
		resolved, aErr := cliutil.ResolveActor("", rootDir)
		if aErr != nil {
			return failImport(out, aErr.Error(), cliutil.ExitUsage)
		}
		actorStr = resolved
	}

	ctx := context.Background()

	// M-0249: diagnostic-logging wiring, mirroring cancel.Run's own
	// M-0238/AC-5 pattern. import can batch multiple entities into one
	// invocation, so entity stays empty (like add/archive) — every
	// created id is already in entityIDs/the JSON envelope.
	diagLog, closeDiagLog := cliutil.ResolveLogger(rootDir, os.Getenv)
	defer func() { _ = closeDiagLog() }()
	if diagLog.Enabled(ctx, slog.LevelInfo) {
		runID := out.CorrelationID
		if runID == "" {
			runID = logger.NewRunID()
		}
		diagLog = logger.WithVerb(diagLog, "import", "", actorStr, runID)
	}
	var sha string
	defer func() { cliutil.EmitVerbOutcome(diagLog, "verb", code, sha) }()

	// dry-run is read-only; lock only when we'd write.
	if !dryRun {
		release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf import", out)
		if release == nil {
			return rc
		}
		defer release()
	}

	// LoadTreeWithTrunk (not bare tree.Load) so the verb-time
	// body-prose-id scan sees TrunkIDs: an imported body referencing an
	// entity allocated on trunk but absent from this branch's tree must
	// not refuse the import (G-0241). Matches add/check/reallocate/
	// rewidth.
	tr, _, err := cliutil.LoadTreeWithTrunk(ctx, rootDir)
	if err != nil {
		return failImport(out, fmt.Sprintf("loading tree: %v", err), cliutil.ExitInternal)
	}

	// Provenance coherence: when the operator is non-human, a principal
	// is required (the I2.5 trailer-coherence rule). Per-entity scope
	// gating (running Allow against each plan's CreationRefs) is
	// deferred to G22; bulk-import attribution lives there.
	principalStr := strings.TrimSpace(principal)
	actorIsNonHuman := actorStr != "" && !strings.HasPrefix(actorStr, "human/")
	if actorIsNonHuman && principalStr == "" {
		return failImport(out, fmt.Sprintf("--principal human/<id> is required when --actor is non-human (got actor=%q)", actorStr), cliutil.ExitUsage)
	}
	if !actorIsNonHuman && principalStr != "" {
		return failImport(out, "--principal is forbidden when --actor is human/ (humans act directly)", cliutil.ExitUsage)
	}

	res, err := verb.Import(ctx, tr, m, actorStr, verb.ImportOptions{
		OnCollision:    onCollision,
		TitleMaxLength: cliutil.ConfiguredTitleMaxLength(rootDir),
	})
	if err != nil {
		return failImport(out, err.Error(), cliutil.ExitUsage)
	}

	if check.HasErrors(res.Findings) {
		if out.JSON() {
			env := render.Envelope{
				Tool:     "aiwf",
				Version:  version.Current().Version,
				Status:   render.StatusFor(res.Findings),
				Findings: res.Findings,
				Metadata: out.Metadata(nil),
			}
			_ = render.JSON(os.Stdout, env, out.Pretty)
			return cliutil.ExitFindings
		}
		_ = render.Text(os.Stderr, res.Findings)
		return cliutil.ExitFindings
	}
	if len(res.Plans) == 0 {
		if out.JSON() {
			emitImportEnvelope(out, "aiwf import: manifest had no entities to import.", nil)
			return cliutil.ExitOK
		}
		cliutil.Println("aiwf import: manifest had no entities to import.")
		return cliutil.ExitOK
	}

	// entityIDs comes from each plan's aiwf-entity trailers, not
	// len(res.Plans): the default commit mode batches every entity
	// into one plan carrying N entity trailers, while
	// manifest.CommitPerEntity produces one plan per entity — either
	// way, the trailers are the single source of truth for "how many
	// entities" independent of how they're grouped into commits.
	entityIDs := importedEntityIDs(res.Plans)

	if dryRun {
		if out.JSON() {
			emitImportEnvelope(out, fmt.Sprintf("aiwf import: dry-run — %d entities would land", len(entityIDs)), map[string]any{"imported_count": len(entityIDs), "entity_ids": entityIDs})
			return cliutil.ExitOK
		}
		cliutil.Printf("aiwf import: dry-run — %d plan(s) would land:\n", len(res.Plans))
		for _, p := range res.Plans {
			cliutil.Printf("  %s\n", p.Subject)
			for _, op := range p.Ops {
				cliutil.Printf("    write %s (%d bytes)\n", op.Path, len(op.Content))
			}
		}
		cliutil.Println("\naiwf import: dry-run complete. Re-run without --dry-run to apply.")
		return cliutil.ExitOK
	}

	var lastSHA string
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
		planSHA, applyErr := verb.Apply(ctx, rootDir, p)
		if applyErr != nil {
			return failImport(out, fmt.Sprintf("applying plan %d: %v", i, applyErr), cliutil.ExitInternal)
		}
		lastSHA = planSHA
		if !out.JSON() {
			cliutil.Println(p.Subject)
		}
	}
	sha = lastSHA
	if out.JSON() {
		// One envelope for the whole batch (M-0239/AC-2): import can
		// produce more than one commit (manifest.CommitPerEntity is a
		// deliberate exception to the one-verb-one-commit norm), so
		// commit_sha here is the batch's LAST commit, not "the"
		// commit — entity_ids carries every imported id so a caller
		// can still resolve each one's own commit via `aiwf history`.
		emitImportEnvelope(out, fmt.Sprintf("aiwf import: %d entities created", len(entityIDs)), withCommitSHA(map[string]any{"imported_count": len(entityIDs), "entity_ids": entityIDs}, lastSHA))
	}
	return cliutil.ExitOK
}

// importedEntityIDs collects every aiwf-entity trailer value across
// all of plans, in order. The single source of truth for "how many/
// which entities" independent of commit-batching mode (see the call
// site's comment).
func importedEntityIDs(plans []*verb.Plan) []string {
	var ids []string
	for _, p := range plans {
		for _, tr := range p.Trailers {
			if tr.Key == gitops.TrailerEntity {
				ids = append(ids, tr.Value)
			}
		}
	}
	return ids
}

// failImport reports a terminal error in the chosen output format.
// Mirrors archive/rewidth's identically-shaped helper.
func failImport(out cliutil.OutputFormat, message string, code int) int {
	if !out.JSON() {
		cliutil.Errorf("aiwf import: %s\n", message)
		return code
	}
	env := render.Envelope{
		Tool:    "aiwf",
		Version: version.Current().Version,
		Status:  "error",
		Error:   &render.EnvelopeError{Message: message},
	}
	_ = render.JSON(os.Stdout, env, out.Pretty)
	return code
}

// emitImportEnvelope writes a status:"ok" envelope carrying subject
// and metadata (correlation_id merged in via out.Metadata). Called
// only when out.JSON() is true.
func emitImportEnvelope(out cliutil.OutputFormat, subject string, metadata map[string]any) {
	env := render.Envelope{
		Tool:     "aiwf",
		Version:  version.Current().Version,
		Status:   "ok",
		Result:   map[string]any{"subject": subject},
		Metadata: out.Metadata(metadata),
	}
	_ = render.JSON(os.Stdout, env, out.Pretty)
}

// withCommitSHA returns a copy of md with "commit_sha" set to sha,
// without mutating md. Mirrors archive/rewidth's identical helper
// (unexported, cross-package — each verb that builds its own envelope
// needs its own copy). sha is always non-empty at this function's one
// call site; render.Envelope's Metadata field carries omitempty,
// which treats a zero-length map the same as nil, so no empty/nil
// special-casing is needed here.
func withCommitSHA(md map[string]any, sha string) map[string]any {
	out := make(map[string]any, len(md)+1)
	for k, v := range md {
		out[k] = v
	}
	if sha != "" {
		out["commit_sha"] = sha
	}
	return out
}
