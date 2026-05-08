package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/render"
	"github.com/23min/ai-workflow-v2/internal/tree"
)

// listSummary is the per-entity row emitted by `aiwf list`. The shape
// is the JSON envelope's `result` element verbatim and is the contract
// downstream tooling depends on; keep it stable across V1 evolutions.
type listSummary struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
	Title  string `json:"title"`
	Parent string `json:"parent,omitempty"`
	Path   string `json:"path,omitempty"`
}

// listCounts is the per-kind count payload for the no-args invocation.
// Keys are kind names; the renderer iterates entity.AllKinds() so the
// order is canonical and a future kind picks up automatically.
type listCounts map[string]int

// newListCmd builds `aiwf list`: the AI-first read primitive over the
// planning tree. Read-only; no commit. Default semantic is "non-
// terminal entities" (forward-compat with ADR-0004); --archived widens
// to include terminal-status entities.
func newListCmd() *cobra.Command {
	var (
		root     string
		kind     string
		status   string
		parent   string
		archived bool
		format   string
		pretty   bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Filter the planning tree (the hot-path read primitive for AI assistants)",
		Example: `  # Per-kind counts of non-terminal entities
  aiwf list

  # Every milestone with status 'draft'
  aiwf list --kind milestone --status draft

  # Children of a specific epic
  aiwf list --kind milestone --parent E-13

  # Include terminal-status entities (forward-compat with ADR-0004)
  aiwf list --archived

  # JSON envelope for downstream tooling
  aiwf list --kind contract --format=json --pretty`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return wrapExitCode(runListCmd(root, kind, status, parent, archived, format, pretty))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root (default: discover via aiwf.yaml)")
	cmd.Flags().StringVar(&kind, "kind", "", "filter by entity kind (epic, milestone, adr, gap, decision, contract)")
	cmd.Flags().StringVar(&status, "status", "", "filter by entity status (kind-aware)")
	cmd.Flags().StringVar(&parent, "parent", "", "filter to entities whose parent is this id (e.g., milestones under E-13)")
	cmd.Flags().BoolVar(&archived, "archived", false, "include terminal-status entities (default: hide them)")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "indent JSON output (only with --format=json)")

	_ = cmd.RegisterFlagCompletionFunc("kind", cobra.FixedCompletions(
		allKindNames(),
		cobra.ShellCompDirectiveNoFileComp,
	))
	_ = cmd.RegisterFlagCompletionFunc("status", func(c *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		k, _ := c.Flags().GetString("kind")
		if k == "" {
			return unionAllStatuses(), cobra.ShellCompDirectiveNoFileComp
		}
		return entity.AllowedStatuses(entity.Kind(k)), cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("parent", completeEntityIDFlag(""))
	registerFormatCompletion(cmd)

	return cmd
}

// unionAllStatuses returns every status string any kind allows, sorted
// and de-duplicated. Used as the --status completion fallback when
// --kind has not been set yet.
func unionAllStatuses() []string {
	seen := map[string]struct{}{}
	var out []string
	for _, k := range entity.AllKinds() {
		for _, s := range entity.AllowedStatuses(k) {
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

func runListCmd(root, kind, status, parent string, archived bool, format string, pretty bool) int {
	if format != "text" && format != "json" {
		fmt.Fprintf(os.Stderr, "aiwf list: --format must be 'text' or 'json', got %q\n", format)
		return exitUsage
	}
	if kind != "" && !isKnownKind(kind) {
		fmt.Fprintf(os.Stderr, "aiwf list: --kind must be one of %v, got %q\n", allKindNames(), kind)
		return exitUsage
	}

	rootDir, err := resolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf list: %v\n", err)
		return exitUsage
	}

	tr, _, err := tree.Load(context.Background(), rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf list: loading tree: %v\n", err)
		return exitInternal
	}

	noArgs := kind == "" && status == "" && parent == "" && !archived
	if noArgs {
		counts := buildListCounts(tr)
		switch format {
		case "text":
			renderListCountsText(os.Stdout, counts)
		case "json":
			env := render.Envelope{
				Tool:    "aiwf",
				Version: resolvedVersion(),
				Status:  "ok",
				Result:  counts,
				Metadata: map[string]any{
					"root": rootDir,
				},
			}
			if err := render.JSON(os.Stdout, env, pretty); err != nil {
				fmt.Fprintf(os.Stderr, "aiwf list: writing output: %v\n", err)
				return exitInternal
			}
		}
		return exitOK
	}

	rows := buildListRows(tr, kind, status, parent, archived)
	switch format {
	case "text":
		renderListRowsText(os.Stdout, rows)
	case "json":
		env := render.Envelope{
			Tool:    "aiwf",
			Version: resolvedVersion(),
			Status:  "ok",
			Result:  rows,
			Metadata: map[string]any{
				"root":  rootDir,
				"count": len(rows),
			},
		}
		if err := render.JSON(os.Stdout, env, pretty); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf list: writing output: %v\n", err)
			return exitInternal
		}
	}
	return exitOK
}

// isKnownKind validates --kind input against the closed kind set
// before the verb walks the tree. Cheap usage-error check.
func isKnownKind(s string) bool {
	for _, k := range entity.AllKinds() {
		if string(k) == s {
			return true
		}
	}
	return false
}

// buildListRows applies the V1 filter axes to tr and returns the
// matched entities as summary rows in id-ascending order. Default
// semantic excludes terminal-status entities; archived=true widens.
//
// The kind+status filter routes through tree.FilterByKindStatuses so
// `aiwf list --kind gap --status open` and `aiwf status`'s Open gaps
// section share one source of truth (M-072 AC-6).
func buildListRows(tr *tree.Tree, kind, status, parent string, archived bool) []listSummary {
	var statuses []string
	if status != "" {
		statuses = []string{status}
	}
	matched := tr.FilterByKindStatuses(entity.Kind(kind), statuses...)

	rows := make([]listSummary, 0, len(matched))
	for _, e := range matched {
		if parent != "" && e.Parent != parent {
			continue
		}
		if !archived && entity.IsTerminal(e.Kind, e.Status) {
			continue
		}
		rows = append(rows, listSummary{
			ID:     e.ID,
			Kind:   string(e.Kind),
			Status: e.Status,
			Title:  e.Title,
			Parent: e.Parent,
			Path:   e.Path,
		})
	}
	return rows
}

// buildListCounts returns the per-kind count of non-terminal entities
// for the no-args invocation. Iteration order follows entity.AllKinds.
func buildListCounts(tr *tree.Tree) listCounts {
	out := listCounts{}
	for _, k := range entity.AllKinds() {
		out[string(k)] = 0
	}
	for _, e := range tr.Entities {
		if entity.IsTerminal(e.Kind, e.Status) {
			continue
		}
		out[string(e.Kind)]++
	}
	return out
}

// renderListCountsText emits the per-kind summary line in the order
// dictated by entity.AllKinds. Format:
//
//	5 epics · 47 milestones · 12 ADRs · 14 gaps · 3 decisions · 1 contract
func renderListCountsText(w io.Writer, counts listCounts) {
	parts := make([]string, 0, len(entity.AllKinds()))
	for _, k := range entity.AllKinds() {
		n := counts[string(k)]
		parts = append(parts, fmt.Sprintf("%d %s", n, pluralKindLabel(k, n)))
	}
	_, _ = fmt.Fprintln(w, strings.Join(parts, " · "))
}

// pluralKindLabel returns the human label for a kind, pluralized when
// n != 1. ADR is the only kind whose canonical display is uppercased
// (consistent with `aiwf status` and the kernel docs).
func pluralKindLabel(k entity.Kind, n int) string {
	label := string(k)
	if k == entity.KindADR {
		label = "ADR"
	}
	if n != 1 {
		label += "s"
	}
	return label
}

// renderListRowsText emits one row per entity with aligned columns:
//
//	ID      STATUS    TITLE                  PARENT
//	M-001   draft     M one                  E-01
//
// Empty-result is the empty string (no header) — keeps the verb cheap
// to consume in shell pipelines and grep-friendly.
func renderListRowsText(w io.Writer, rows []listSummary) {
	if len(rows) == 0 {
		return
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "ID\tSTATUS\tTITLE\tPARENT")
	for _, r := range rows {
		parent := r.Parent
		if parent == "" {
			parent = "-"
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", r.ID, r.Status, r.Title, parent)
	}
	_ = tw.Flush()
}
