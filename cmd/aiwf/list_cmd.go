package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/tree"
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
		noTrunc  bool
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
  aiwf list --kind contract --format=json --pretty

  # Full titles even in a narrow terminal (default truncates the title column)
  aiwf list --kind milestone --no-trunc`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return wrapExitCode(runListCmd(root, kind, status, parent, archived, format, pretty, noTrunc))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root (default: discover via aiwf.yaml)")
	cmd.Flags().StringVar(&kind, "kind", "", "filter by entity kind (epic, milestone, adr, gap, decision, contract)")
	cmd.Flags().StringVar(&status, "status", "", "filter by entity status (kind-aware)")
	cmd.Flags().StringVar(&parent, "parent", "", "filter to entities whose parent is this id (e.g., milestones under E-13)")
	cmd.Flags().BoolVar(&archived, "archived", false, "include terminal-status entities (default: hide them)")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "indent JSON output (only with --format=json)")
	cmd.Flags().BoolVar(&noTrunc, "no-trunc", false, "do not truncate the title column when stdout is a terminal narrower than the row")

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

func runListCmd(root, kind, status, parent string, archived bool, format string, pretty, noTrunc bool) int {
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
		w := termTitleBudget(os.Stdout, noTrunc)
		renderListRowsText(os.Stdout, rows, w, render.ColorEnabled(os.Stdout))
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
	canonParent := entity.Canonicalize(parent)

	rows := make([]listSummary, 0, len(matched))
	for _, e := range matched {
		if parent != "" && entity.Canonicalize(e.Parent) != canonParent {
			continue
		}
		if !archived && entity.IsTerminal(e.Kind, e.Status) {
			continue
		}
		// Emitted ids are canonical per AC-3 in M-081 — display
		// surfaces are uniform-width regardless of on-disk filename.
		rows = append(rows, listSummary{
			ID:     entity.Canonicalize(e.ID),
			Kind:   string(e.Kind),
			Status: e.Status,
			Title:  e.Title,
			Parent: entity.Canonicalize(e.Parent),
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
//	ID      STATUS         TITLE                  PARENT
//	M-001   ○ draft        M one                  E-01
//	M-002   → in_progress  M two                  E-01
//
// Empty-result is the empty string (no header) — keeps the verb cheap
// to consume in shell pipelines and grep-friendly. The status column
// carries a 1-rune glyph + space prefix when the status maps to the
// G-0080 palette (every kernel status does); the glyph is content,
// not style, and appears in piped output the same as in a TTY.
//
// titleBudget caps the title column's rune width when stdout is a TTY
// narrower than a row's natural width — closes G-0080's tabwriter-wrap
// bug where long titles wrap into the id-column gutter. A non-positive
// budget disables truncation (piped output, --no-trunc, or a TTY wide
// enough to fit the row as-is). The cap is applied per-row before the
// tabwriter sees the input, so tabwriter's column alignment stays
// intact.
//
// colorEnabled toggles the ANSI-bold styling on the header row. It is
// the only place ANSI escapes enter this verb's output; row content
// stays escape-free so downstream tooling (grep, awk) sees plain text.
func renderListRowsText(w io.Writer, rows []listSummary, titleBudget int, colorEnabled bool) {
	if len(rows) == 0 {
		return
	}
	// Pre-compute the rendered status for each row so the truncation
	// budget measures the glyph-prefixed width, not the raw status. Two
	// runes per glyphed status ("X "); empty string for rows whose
	// status falls outside the palette (defensive — the kernel's status
	// vocabulary is closed and maps fully today).
	statuses := make([]string, len(rows))
	for i := range rows {
		if g := render.StatusGlyph(rows[i].Status); g != "" {
			statuses[i] = g + " " + rows[i].Status
		} else {
			statuses[i] = rows[i].Status
		}
	}
	titleMax := computeTitleBudget(rows, statuses, titleBudget)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, render.Bold("ID\tSTATUS\tTITLE\tPARENT", colorEnabled))
	for i, r := range rows {
		parent := r.Parent
		if parent == "" {
			parent = "-"
		}
		title := r.Title
		if titleMax > 0 {
			title = render.Truncate(title, titleMax)
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", r.ID, statuses[i], title, parent)
	}
	_ = tw.Flush()
}

// computeTitleBudget returns the per-row rune cap for the title column,
// or 0 to disable truncation. termWidth=0 means "no TTY / no-trunc /
// width unknown" — pass through as 0 and the caller skips truncation.
// renderedStatuses is the per-row status string as it will be written
// (glyph + space + status when the row matches the G-0080 palette);
// measured here so the budget reflects the actual column width.
//
// The math: tabwriter renders id|status|title|parent with 2-char
// padding between columns. Natural row width is
// id_w + 2 + status_w + 2 + title_w + 2 + parent_w. Set
// title_w = termWidth - (id_w + status_w + parent_w + 6) and floor at
// minTitleColumnRunes — if the remainder would be below the floor,
// truncation buys so little (and looks so ugly) that we return 0 and
// let the terminal wrap. The header row contributes its own width
// (ID/STATUS/TITLE/PARENT, all narrower than typical content) so we
// don't measure it.
func computeTitleBudget(rows []listSummary, renderedStatuses []string, termWidth int) int {
	if termWidth <= 0 || len(rows) == 0 {
		return 0
	}
	idW, statusW, parentW := len("ID"), len("STATUS"), len("PARENT")
	naturalTitleW := len("TITLE")
	for i := range rows {
		if n := utf8.RuneCountInString(rows[i].ID); n > idW {
			idW = n
		}
		var statusText string
		if i < len(renderedStatuses) {
			statusText = renderedStatuses[i]
		} else {
			statusText = rows[i].Status
		}
		if n := utf8.RuneCountInString(statusText); n > statusW {
			statusW = n
		}
		parent := rows[i].Parent
		if parent == "" {
			parent = "-"
		}
		if n := utf8.RuneCountInString(parent); n > parentW {
			parentW = n
		}
		if n := utf8.RuneCountInString(rows[i].Title); n > naturalTitleW {
			naturalTitleW = n
		}
	}
	natural := idW + statusW + parentW + naturalTitleW + 6 // 3 gaps × 2 chars
	if natural <= termWidth {
		return 0 // already fits — no truncation needed
	}
	budget := termWidth - (idW + statusW + parentW + 6)
	if budget < minTitleColumnRunes {
		return 0
	}
	return budget
}

// minTitleColumnRunes is the floor below which the title column is no
// longer worth truncating — at 10 runes most titles collapse to a few
// initial words plus "…", which is more annoying than useful. When the
// terminal is narrower than what's needed to leave this much room, we
// give up on truncation and let the terminal wrap as it always has.
const minTitleColumnRunes = 10

// termTitleBudget resolves the title-truncation budget for stdout. The
// returned width is the terminal width when stdout is a TTY and
// noTrunc is false; 0 otherwise. Callers feed this into
// computeTitleBudget to compute the per-column cap.
func termTitleBudget(f *os.File, noTrunc bool) int {
	if noTrunc {
		return 0
	}
	return render.TerminalWidth(f)
}
