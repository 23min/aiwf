package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/verb"
)

// newArchiveCmd builds `aiwf archive [--apply] [--kind <kind>] [--root <path>]`.
//
// Default invocation (no `--apply`) is dry-run: the verb computes a
// Plan and the dispatcher prints the planned moves. `--apply` runs
// verb.Apply against the same Plan, producing exactly one git commit
// per kernel principle #7 with trailer `aiwf-verb: archive` (no
// `aiwf-entity:` trailer — multi-entity sweep, same shape as
// `aiwf rewidth`).
//
// Per ADR-0004:
//   - The verb sweeps terminal-status entities into per-kind archive/
//     subdirectories.
//   - There is no positional id argument — the verb sweeps by status,
//     not by id (the rejected per-id housekeeping alternative).
//   - Idempotent: re-runs on a clean tree are no-ops.
//   - Reversal is deliberately not implemented (ADR-0004 §"Reversal").
//     If a closed entity needs revisiting, file a new entity that
//     references the archived one.
func newArchiveCmd() *cobra.Command {
	var (
		actor     string
		principal string
		root      string
		apply     bool
		dryRun    bool
		kind      string
	)
	cmd := &cobra.Command{
		Use:   "archive [--apply | --dry-run] [--kind <kind>]",
		Short: "Sweep terminal-status entities into per-kind archive/ subdirs (per ADR-0004)",
		Long: `Sweep terminal-status entities into their per-kind archive/
subdirectories per ADR-0004. Default is dry-run; --apply commits the
sweep as a single commit with trailer aiwf-verb: archive. --dry-run is
an explicit alias for the default behavior (mutually exclusive with
--apply) so finding hints and ad-hoc invocations can name it directly.
The verb sweeps by status, not by id — there is no positional id
argument.

Per-kind storage layout (per ADR-0004 §"Storage — per-kind layout"):

  Epic      work/epics/<epic>/                 -> work/epics/archive/<epic>/
  Milestone (rides with parent epic — does not archive independently)
  Contract  work/contracts/<contract>/         -> work/contracts/archive/<contract>/
  Gap       work/gaps/G-NNNN-<slug>.md         -> work/gaps/archive/G-NNNN-<slug>.md
  Decision  work/decisions/D-NNNN-<slug>.md    -> work/decisions/archive/D-NNNN-<slug>.md
  ADR       docs/adr/ADR-NNNN-<slug>.md        -> docs/adr/archive/ADR-NNNN-<slug>.md

Idempotent: re-runs on a clean tree produce zero commits and exit 0.
The reverse path is intentionally not implemented (ADR-0004 §"Reversal");
file a new entity that references the archived one if a closed entity
needs revisiting.

The same verb covers both the bulk first-run sweep against a pre-
ADR-0004 tree and the routine ongoing sweeps that follow.`,
		Example: `  # Preview the sweep (dry-run is the default)
  aiwf archive

  # Same, named explicitly
  aiwf archive --dry-run

  # Commit the sweep as a single commit
  aiwf archive --apply

  # Scope the sweep to one kind
  aiwf archive --apply --kind gap`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			if apply && dryRun {
				fmt.Fprintln(os.Stderr, "aiwf archive: --apply and --dry-run are mutually exclusive")
				return wrapExitCode(exitUsage)
			}
			return wrapExitCode(runArchiveCmd(actor, principal, root, kind, apply))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().BoolVar(&apply, "apply", false, "commit the sweep; without this flag the verb is dry-run")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "explicit alias for the default dry-run behavior; mutually exclusive with --apply")
	cmd.Flags().StringVar(&kind, "kind", "", "scope the sweep to one kind (epic, contract, gap, decision, adr); milestones do not archive independently")

	_ = cmd.RegisterFlagCompletionFunc("kind", cobra.FixedCompletions(
		archiveKindCompletions(),
		cobra.ShellCompDirectiveNoFileComp,
	))

	return cmd
}

// archiveKindCompletions returns the closed set of kinds the --kind
// flag accepts. Milestone is excluded by design (per ADR-0004's
// storage table — milestones don't archive independently). The
// remaining five kinds match entity.AllKinds() minus milestone.
func archiveKindCompletions() []string {
	return []string{"epic", "contract", "gap", "decision", "adr"}
}

func runArchiveCmd(actor, principal, root, kind string, apply bool) int {
	rootDir, err := resolveRoot(root)
	if err != nil { //coverage:ignore resolveRoot only fails on missing aiwf.yaml + non-existent --root path
		fmt.Fprintf(os.Stderr, "aiwf archive: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(actor, rootDir)
	if err != nil { //coverage:ignore resolveActor only fails when actor cannot be derived from any source
		fmt.Fprintf(os.Stderr, "aiwf archive: %v\n", err)
		return exitUsage
	}

	// Provenance coherence check: a non-human actor needs a principal;
	// a human actor must not carry one. Mirrors `aiwf rewidth` and
	// `aiwf import`'s shape — bulk-sweep, no per-entity scope gating.
	principalStr := strings.TrimSpace(principal)
	actorIsNonHuman := actorStr != "" && !strings.HasPrefix(actorStr, "human/")
	if actorIsNonHuman && principalStr == "" {
		fmt.Fprintf(os.Stderr, "aiwf archive: --principal human/<id> is required when --actor is non-human (got actor=%q)\n", actorStr)
		return exitUsage
	}
	if !actorIsNonHuman && principalStr != "" {
		fmt.Fprintln(os.Stderr, "aiwf archive: --principal is forbidden when --actor is human/ (humans act directly)")
		return exitUsage
	}

	// Validate --kind early so a typo doesn't wait for the verb.
	kindStr := strings.TrimSpace(kind)
	if kindStr != "" {
		if !validArchiveKind(kindStr) {
			fmt.Fprintf(os.Stderr, "aiwf archive: --kind %q is not one of %s\n", kindStr, strings.Join(archiveKindCompletions(), ", "))
			return exitUsage
		}
	}

	// Dry-run is read-only; lock only when we'd write.
	if apply {
		release, rc := acquireRepoLock(rootDir, "aiwf archive")
		if release == nil { //coverage:ignore acquireRepoLock only returns nil on lock contention from a concurrent verb
			return rc
		}
		defer release()
	}

	ctx := context.Background()

	result, err := verb.Archive(ctx, rootDir, actorStr, kindStr)
	if err != nil { //coverage:ignore verb.Archive only errors on filesystem failures
		fmt.Fprintf(os.Stderr, "aiwf archive: %v\n", err)
		return exitInternal
	}
	if result == nil { //coverage:ignore Archive always returns a non-nil Result on success
		fmt.Fprintln(os.Stderr, "aiwf archive: no result returned")
		return exitInternal
	}

	if result.NoOp {
		fmt.Println(result.NoOpMessage)
		return exitOK
	}
	if result.Plan == nil { //coverage:ignore non-NoOp result without a Plan is unreachable today
		fmt.Fprintln(os.Stderr, "aiwf archive: validation passed but no plan produced")
		return exitInternal
	}

	if !apply {
		printArchiveDryRun(result.Plan)
		return exitOK
	}

	// Stamp principal trailer when the operator is non-human, mirroring
	// rewidth's bulk-sweep shape.
	if actorIsNonHuman {
		result.Plan.Trailers = append(result.Plan.Trailers, gitops.Trailer{
			Key:   gitops.TrailerPrincipal,
			Value: principalStr,
		})
	}

	if applyErr := verb.Apply(ctx, rootDir, result.Plan); applyErr != nil { //coverage:ignore Apply only errors on git mv/commit failures
		fmt.Fprintf(os.Stderr, "aiwf archive: %v\n", applyErr)
		return exitInternal
	}
	if len(result.Findings) > 0 { //coverage:ignore Archive currently never populates Findings
		_ = render.Text(os.Stderr, result.Findings)
	}
	fmt.Println(result.Plan.Subject)
	return exitOK
}

// validArchiveKind reports whether s is one of the kinds the --kind
// flag accepts. The closed set lives next to archiveKindCompletions
// so the completion list and the validator stay in step.
func validArchiveKind(s string) bool {
	for _, k := range archiveKindCompletions() {
		if k == s {
			return true
		}
	}
	return false
}

// printArchiveDryRun prints a human-readable summary of the planned
// moves. Stdout, not stderr — the user reads this to decide whether
// to re-run with --apply.
func printArchiveDryRun(p *verb.Plan) {
	fmt.Println(p.Subject + " (dry-run; re-run with --apply to commit)")
	if p.Body != "" {
		fmt.Println()
		fmt.Print(p.Body)
	}
	moves := 0
	for _, op := range p.Ops {
		if op.Type == verb.OpMove {
			moves++
		}
	}
	fmt.Println()
	fmt.Printf("Moves (%d):\n", moves)
	for _, op := range p.Ops {
		if op.Type == verb.OpMove {
			fmt.Printf("  %s -> %s\n", op.Path, op.NewPath)
		}
	}
}
