// Package history implements the `aiwf history` verb (per-verb subpackage of
// M-0116; cmd/aiwf/main.go's newRootCmd wires it via NewCmd). The neutral,
// Cobra-free HistoryEvent parsing lives in internal/entityview; this
// package holds the Cobra wiring plus the text-rendering helpers
// (RenderTo, RenderActor, RenderScopeChips) and the scope-map guard
// (ScopeMapFor) that composes entityview's predicates with cliutil.
package history

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entityview"
	"github.com/23min/aiwf/internal/logger"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/version"
)

// NewCmd builds `aiwf history <id>`: filters git log for the
// entity's structured trailers and prints one line per event.
func NewCmd(correlationID string) *cobra.Command {
	var (
		root     string
		format   string
		pretty   bool
		showAuth bool
	)
	cmd := &cobra.Command{
		Use:   "history <id>",
		Short: "Show the entity's lifecycle from git log trailers",
		Example: `  # Print one line per lifecycle event
  aiwf history E-01

  # Render the full provenance chain as JSON
  aiwf history M-007 --format=json --pretty`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(Run(args[0], root, format, pretty, showAuth, correlationID))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "indent JSON output (only with --format=json)")
	cmd.Flags().BoolVar(&showAuth, "show-authorization", false, "include the full aiwf-authorized-by SHA on scope-authorized rows (text format only)")
	cliutil.RegisterFormatCompletion(cmd)
	cmd.ValidArgsFunction = cliutil.CompleteEntityIDArg("", 0)
	return cmd
}

// Run executes `aiwf history`. Returns one of the cliutil.Exit* codes.
func Run(id, root, format string, pretty, showAuth bool, correlationID string) (code int) {
	if format != "text" && format != "json" {
		cliutil.Errorf("aiwf history: --format must be text or json, got %q\n", format)
		return cliutil.ExitUsage
	}

	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil { //coverage:ignore cliutil.ResolveRoot only fails on missing aiwf.yaml + non-existent --root path
		cliutil.Errorf("aiwf history: %v\n", err)
		return cliutil.ExitUsage
	}

	ctx := context.Background()

	// M-0249 follow-up: diagnostic-logging wiring, mirroring show.Run's
	// own read-only rationale — history has no --actor flag, so actor
	// resolution is best-effort only and never fails the verb
	// (ADR-0017).
	diagLog, closeDiagLog := cliutil.ResolveLogger(rootDir, os.Getenv)
	defer func() { _ = closeDiagLog() }()
	if diagLog.Enabled(ctx, slog.LevelInfo) {
		actorStr, actorErr := cliutil.ResolveActor("", rootDir)
		if actorErr != nil {
			actorStr = ""
		}
		runID := correlationID
		if runID == "" {
			runID = logger.NewRunID()
		}
		diagLog = logger.WithVerb(diagLog, "history", id, actorStr, runID)
	}
	defer func() { cliutil.EmitVerbOutcome(diagLog, "verb", code, "") }()

	// Resolve the queried id through prior_ids lineage so a query for
	// an old id returns the same chronological chain as a query for
	// the entity's current id. The chain is the union of (a) the
	// queried id itself, (b) the canonical entity's current id when
	// distinct, and (c) every id in the canonical entity's PriorIDs.
	// readHistory greps git log once for the union — pre-rename
	// commits (matching aiwf-entity: <old>), the rename commit
	// itself (matching aiwf-prior-entity: <old> against the queried
	// id), and post-rename commits (matching aiwf-entity: <new>) all
	// arrive in one chronological pass.
	chain := []string{id}
	if tr, _, terr := tree.Load(ctx, rootDir); terr == nil && tr != nil {
		if e := tr.ResolveByCurrentOrPriorID(id); e != nil {
			seen := map[string]bool{id: true}
			for _, p := range e.PriorIDs {
				if !seen[p] {
					chain = append(chain, p)
					seen[p] = true
				}
			}
			if !seen[e.ID] {
				chain = append(chain, e.ID)
			}
		}
	}

	events, err := entityview.ReadHistoryChain(ctx, rootDir, chain)
	if err != nil {
		//coverage:ignore ReadHistoryChain's `git log` only fails for a
		// genuine git/environmental fault once cliutil.HasCommits
		// (called first, using the same root) has already succeeded —
		// not reachable through a clean deterministic fixture; each id
		// is regexp.QuoteMeta-escaped before reaching the --grep
		// pattern (entity.IDGrepAlternation), so a malformed id cannot
		// break the regex either.
		cliutil.Errorf("aiwf history: %v\n", err)
		return cliutil.ExitInternal
	}

	switch format {
	case "text":
		if len(events) == 0 {
			cliutil.Printf("no history for %s\n", id)
			return cliutil.ExitOK
		}
		// Resolve authorize-SHA → scope-entity for the chip labels, but
		// only when the loaded events actually reference a scope (E-0054 /
		// M-0223 guard — see ScopeMapFor).
		scopeEntities := ScopeMapFor(ctx, rootDir, events)
		for i := range events {
			e := &events[i]
			cliutil.Printf("%s  %-16s  %-10s  %-12s  %s  %s%s\n",
				e.Date, RenderActor(*e), e.Verb, RenderTo(e.To), e.Detail, e.Commit,
				RenderScopeChips(*e, scopeEntities, showAuth))
			if e.Force != "" {
				cliutil.Printf("    [forced: %s]\n", e.Force)
			}
			if e.AuditOnly != "" {
				cliutil.Printf("    [audit-only: %s]\n", e.AuditOnly)
			}
			if e.Reason != "" {
				cliutil.Printf("    [reason: %s]\n", e.Reason)
			}
			if e.Body != "" {
				for _, line := range strings.Split(e.Body, "\n") {
					cliutil.Printf("    %s\n", line)
				}
			}
		}
	case "json":
		env := render.Envelope{
			Tool:    "aiwf",
			Version: version.Current().Version,
			Status:  "ok",
			Result:  map[string]any{"id": id, "events": events},
			Metadata: map[string]any{
				"root":   rootDir,
				"events": len(events),
			},
		}
		if err := render.JSON(os.Stdout, env, pretty); err != nil { //coverage:ignore render.JSON to os.Stdout fails only on a write fault (broken pipe, closed fd); not deterministically reproducible.
			cliutil.Errorf("aiwf history: %v\n", err)
			return cliutil.ExitInternal
		}
	}
	return cliutil.ExitOK
}

// RenderTo formats the target-status column in `aiwf history` text
// output. Empty (the absent-trailer case for non-promote events and
// pre-I2 promote commits) renders as "-"; a populated value is shown
// with a leading arrow so the column reads as a transition target.
func RenderTo(to string) string {
	if to == "" {
		return "-"
	}
	return "→ " + to
}

// RenderActor formats the actor column. When a non-human principal
// is present and differs from the actor (the agent-acts-for-human
// case from I2.5), the column reads `principal via agent` so the
// human is visually attributed first. Direct human acts (no
// principal) render the actor verbatim.
func RenderActor(e entityview.HistoryEvent) string {
	if e.Principal == "" || e.Principal == e.Actor {
		return e.Actor
	}
	return e.Principal + " via " + e.Actor
}

// RenderScopeChips assembles the trailing chip block for one history
// row. For `aiwf authorize` rows, a `[<scope> <event>]` chip names
// the lifecycle event (`opened` / `paused` / `resumed`). For
// scope-authorized rows, a `[<scope-entity> <auth-short>]` chip
// names the authorizing scope. For terminal-promote rows that ended
// one or more scopes, one `[<scope-entity> ended]` chip per ended
// scope.
//
// scopeEntities maps full auth-SHA to scope-entity id. showAuth
// flips on the full SHA inline (the --show-authorization flag).
//
// The output begins with a leading "  " when non-empty so it sits
// flush against the Commit column the caller already printed.
func RenderScopeChips(e entityview.HistoryEvent, scopeEntities map[string]string, showAuth bool) string {
	var chips []string
	if e.Verb == "authorize" && e.Scope != "" {
		chips = append(chips, fmt.Sprintf("[scope: %s]", e.Scope))
	}
	if e.AuthorizedBy != "" {
		scopeEntity := scopeEntities[e.AuthorizedBy]
		if scopeEntity == "" {
			scopeEntity = "?"
		}
		sha := entityview.ShortHash(e.AuthorizedBy)
		if showAuth {
			sha = e.AuthorizedBy
		}
		chips = append(chips, fmt.Sprintf("[%s %s]", scopeEntity, sha))
	}
	for _, sha := range e.ScopeEnds {
		scopeEntity := scopeEntities[sha]
		if scopeEntity == "" {
			scopeEntity = entityview.ShortHash(sha)
		}
		chips = append(chips, fmt.Sprintf("[%s ended]", scopeEntity))
	}
	if len(chips) == 0 {
		return ""
	}
	return "  " + strings.Join(chips, " ")
}
