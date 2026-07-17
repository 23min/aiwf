// Package list implements the `aiwf list` verb (per-verb subpackage of M-0116;
// cmd/aiwf/main.go newRootCmd wires it via NewCmd).
package list

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/logger"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/trunk"
	"github.com/23min/aiwf/internal/version"
)

// ListSummary is the per-entity row emitted by `aiwf list`. The shape
// is the JSON envelope's `result` element verbatim and is the contract
// downstream tooling depends on; keep it stable across V1 evolutions.
type ListSummary struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
	Title  string `json:"title"`
	Parent string `json:"parent,omitempty"`
	Path   string `json:"path,omitempty"`

	// CrossBranchRef is the ref a row's content was resolved from
	// (M-0260/AC-1) when the id misses the local working tree.
	// Non-empty only for a resolved cross-branch row; empty for an
	// ordinary local row and for a cross-branch-collision row
	// (CrossBranchCollision covers that case instead).
	CrossBranchRef string `json:"cross_branch_ref,omitempty"`
	// CrossBranchCollision is true when the id is known cross-branch
	// but its content diverges across refs (M-0259/AC-3); Status/
	// Title/Parent are left empty — aiwf list declines to pick a side
	// (M-0260/AC-3). CrossBranchRefs names every candidate ref.
	CrossBranchCollision bool     `json:"cross_branch_collision,omitempty"`
	CrossBranchRefs      []string `json:"cross_branch_refs,omitempty"`

	// Priority carries the entity's own `priority` frontmatter
	// (G-0078, E-0066, M-0263) — empty for a kind that never carries
	// one (entity.CarriesOwnPriority) or a gap/decision with no
	// priority set.
	Priority string `json:"priority,omitempty"`
}

// ListCounts is the per-kind count payload for the no-args invocation.
// Keys are kind names; the renderer iterates entity.AllKinds() so the
// order is canonical and a future kind picks up automatically.
type ListCounts map[string]int

// NewCmd builds `aiwf list`: the AI-first read primitive over the
// planning tree. Read-only; no commit. Default semantic is "non-
// terminal entities" (forward-compat with ADR-0004); --archived widens
// to include terminal-status entities.
func NewCmd(correlationID string) *cobra.Command {
	var (
		root     string
		kind     string
		status   string
		parent   string
		area     string
		priority string
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
			return cliutil.WrapExitCode(Run(root, kind, status, parent, area, priority, archived, format, pretty, noTrunc, correlationID))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root (default: discover via aiwf.yaml)")
	cmd.Flags().StringVar(&kind, "kind", "", "filter by entity kind (epic, milestone, adr, gap, decision, contract)")
	cmd.Flags().StringVar(&status, "status", "", "filter by entity status (kind-aware)")
	cmd.Flags().StringVar(&parent, "parent", "", "filter to entities whose parent is this id (e.g., milestones under E-13)")
	cmd.Flags().StringVar(&area, "area", "", "filter to entities whose effective area equals this workstream tag (E-0043)")
	cmd.Flags().StringVar(&priority, "priority", "", "filter to gaps/decisions whose priority equals this closed-set level (urgent|high|medium|low) (G-0078, E-0066)")
	cmd.Flags().BoolVar(&archived, "archived", false, "include terminal-status entities (default: hide them)")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "indent JSON output (only with --format=json)")
	cmd.Flags().BoolVar(&noTrunc, "no-trunc", false, "do not truncate the title column when stdout is a terminal narrower than the row")

	_ = cmd.RegisterFlagCompletionFunc("kind", cobra.FixedCompletions(
		cliutil.AllKindNames(),
		cobra.ShellCompDirectiveNoFileComp,
	))
	_ = cmd.RegisterFlagCompletionFunc("status", func(c *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		k, _ := c.Flags().GetString("kind")
		if k == "" {
			return UnionAllStatuses(), cobra.ShellCompDirectiveNoFileComp
		}
		return entity.AllowedStatuses(entity.Kind(k)), cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("parent", cliutil.CompleteEntityIDFlag(""))
	_ = cmd.RegisterFlagCompletionFunc("area", cliutil.CompleteAreaFlag())
	_ = cmd.RegisterFlagCompletionFunc("priority", cobra.FixedCompletions(
		entity.AllowedPriorityLevels(),
		cobra.ShellCompDirectiveNoFileComp,
	))
	cliutil.RegisterFormatCompletion(cmd)

	return cmd
}

// UnionAllStatuses returns every status string any kind allows, sorted
// and de-duplicated. Used as the --status completion fallback when
// --kind has not been set yet.
func UnionAllStatuses() []string {
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

// Run executes `aiwf list`. Returns one of the cliutil.Exit* codes.
func Run(root, kind, status, parent, area, priority string, archived bool, format string, pretty, noTrunc bool, correlationID string) (code int) {
	if format != "text" && format != "json" {
		cliutil.Errorf("aiwf list: --format must be 'text' or 'json', got %q\n", format)
		return cliutil.ExitUsage
	}
	if kind != "" && !IsKnownKind(kind) {
		cliutil.Errorf("aiwf list: --kind must be one of %v, got %q\n", cliutil.AllKindNames(), kind)
		return cliutil.ExitUsage
	}
	// Priority is a closed, Go-hardcoded set (unlike --area's operator-
	// declared members), so an out-of-range value is a hard usage error —
	// mirroring --kind's validation, not --area's undeclared-value note
	// (M-0263 constraint: "a bad --priority value is a usage error, not
	// a silent empty result").
	if priority != "" && !entity.IsAllowedPriorityLevel(priority) {
		cliutil.Errorf("aiwf list: --priority must be one of %v, got %q\n", entity.AllowedPriorityLevels(), priority)
		return cliutil.ExitUsage
	}

	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil { //coverage:ignore cliutil.ResolveRoot only fails on missing aiwf.yaml + non-existent --root path
		cliutil.Errorf("aiwf list: %v\n", err)
		return cliutil.ExitUsage
	}

	ctx := context.Background()

	// M-0249 follow-up: diagnostic-logging wiring, mirroring show.Run's
	// own read-only rationale — list has no --actor flag, so actor
	// resolution is best-effort only and never fails the verb
	// (ADR-0017). No single target id (a filter/query verb), so entity
	// stays empty, matching add/archive/rewidth's own rationale.
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
		diagLog = logger.WithVerb(diagLog, "list", "", actorStr, runID)
	}
	defer func() { cliutil.EmitVerbOutcome(diagLog, "verb", code, "") }()

	// Advisory note when --area names a value that isn't declared
	// (E-0043, M-0174/AC-5). The filter below stays mechanical; the note
	// only tells the operator the value they typed isn't one they
	// declared. To stderr so it never pollutes the (stdout) result.
	if note := cliutil.UndeclaredAreaNote(rootDir, area); note != "" {
		cliutil.Errorln(note)
	}

	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil { //coverage:ignore tree.Load errors only on filesystem IO failure (e.g. a permission fault) or context cancellation; malformed entities surface as load findings, not an error here.
		cliutil.Errorf("aiwf list: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}

	noArgs := kind == "" && status == "" && parent == "" && area == "" && priority == "" && !archived
	if noArgs {
		counts := BuildListCounts(tr)
		switch format {
		case "text":
			RenderListCountsText(os.Stdout, counts)
		case "json":
			env := render.Envelope{
				Tool:    "aiwf",
				Version: version.Current().Version,
				Status:  "ok",
				Result:  counts,
				Metadata: map[string]any{
					"root": rootDir,
				},
			}
			if err := render.JSON(os.Stdout, env, pretty); err != nil { //coverage:ignore render.JSON to os.Stdout fails only on a write fault (broken pipe, closed fd); not deterministically reproducible.
				cliutil.Errorf("aiwf list: writing output: %v\n", err)
				return cliutil.ExitInternal
			}
		}
		return cliutil.ExitOK
	}

	rows := BuildListRows(ctx, tr, kind, status, parent, area, priority, archived)
	switch format {
	case "text":
		w := termTitleBudget(os.Stdout, noTrunc)
		RenderListRowsText(os.Stdout, rows, w, render.ColorEnabled(os.Stdout))
	case "json":
		env := render.Envelope{
			Tool:    "aiwf",
			Version: version.Current().Version,
			Status:  "ok",
			Result:  rows,
			Metadata: map[string]any{
				"root":  rootDir,
				"count": len(rows),
			},
		}
		if err := render.JSON(os.Stdout, env, pretty); err != nil { //coverage:ignore render.JSON to os.Stdout fails only on a write fault (broken pipe, closed fd); not deterministically reproducible.
			cliutil.Errorf("aiwf list: writing output: %v\n", err)
			return cliutil.ExitInternal
		}
	}
	return cliutil.ExitOK
}

// IsKnownKind validates --kind input against the closed kind set
// before the verb walks the tree. Cheap usage-error check.
func IsKnownKind(s string) bool {
	for _, k := range entity.AllKinds() {
		if string(k) == s {
			return true
		}
	}
	return false
}

// BuildListRows applies the V1 filter axes to tr and returns the
// matched entities as summary rows in id-ascending order. Default
// semantic excludes terminal-status entities; archived=true widens.
//
// The kind+status filter routes through tree.FilterByKindStatuses so
// `aiwf list --kind gap --status open` and `aiwf status`'s Open gaps
// section share one source of truth (M-072 AC-6).
//
// The area axis (E-0043, M-0174/AC-1) is an independent AND-ed filter:
// when non-empty, only entities whose effective area (tree.ResolvedArea
// — root kinds by their own field, milestones by parent-epic derivation)
// equals area survive. Untagged entities (effective area "") never match
// a specific area, so they are excluded from `--area X` (AC-6).
//
// M-0260/AC-1/AC-2: after the local rows above, ids known cross-branch
// (visible on another local or remote-tracking ref) but absent from
// the local tree also participate, labeled distinctly via
// CrossBranchRef/CrossBranchCollision — see crossBranchListRows for
// the filtering policy that governs them.
//
// The priority axis (G-0078, E-0066, M-0263/AC-1) is an independent
// AND-ed filter, mirroring area's shape: when non-empty, only entities
// whose own Priority field equals priority survive. A kind that never
// carries a priority (entity.CarriesOwnPriority) always has an empty
// Priority, so it never matches a specific level — no separate
// kind-gate needed here, same as an untagged gap/decision.
func BuildListRows(ctx context.Context, tr *tree.Tree, kind, status, parent, area, priority string, archived bool) []ListSummary {
	var statuses []string
	if status != "" {
		statuses = []string{status}
	}
	matched := tr.FilterByKindStatuses(entity.Kind(kind), statuses...)
	canonParent := entity.Canonicalize(parent)

	rows := make([]ListSummary, 0, len(matched))
	for _, e := range matched {
		if parent != "" && entity.Canonicalize(e.Parent) != canonParent {
			continue
		}
		if !archived && entity.IsTerminal(e.Kind, e.Status) {
			continue
		}
		if area != "" && tr.ResolvedArea(e) != area {
			continue
		}
		if priority != "" && e.Priority != priority {
			continue
		}
		// Emitted ids are canonical per AC-3 in M-081 — display
		// surfaces are uniform-width regardless of on-disk filename.
		rows = append(rows, ListSummary{
			ID:       entity.Canonicalize(e.ID),
			Kind:     string(e.Kind),
			Status:   e.Status,
			Title:    e.Title,
			Parent:   entity.Canonicalize(e.Parent),
			Path:     e.Path,
			Priority: e.Priority,
		})
	}

	rows = append(rows, crossBranchListRows(ctx, tr, kind, status, parent, area, priority, archived)...)
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows
}

// crossBranchListRows appends rows for ids known cross-branch
// (visible on another local or remote-tracking ref) but absent from
// the local tree (M-0260/AC-1/AC-2, ADR-0030's read-side extension
// point). Scans live only from within a filtered listing — never from
// the no-args counts path (BuildListCounts), so the common bare
// `aiwf list` invocation pays no extra subprocess cost.
//
// Filtering policy (decided during implementation, M-0260): a
// resolved (non-collision) hit reads its real content via
// gitops.BlobReader, so --status/--parent/--area/--archived apply to
// it exactly like a local row — the read is already paid to produce a
// real Title for AC-2's labeling, so there is no reason to skip the
// same checks. A collision hit (M-0259/AC-3's shape) has no real
// status/parent/area to check — resolving one would be exactly the
// arbitration AC-3 forbids — so it participates only in a kind-only
// (or unfiltered) query; --status/--parent/--area each exclude it (a
// filter claiming a match on data that doesn't exist yet would be a
// false positive, worse than omitting the row). --archived never
// excludes a collision row: that flag only controls default
// suppression of terminal-status entities, and an unresolved ambiguity
// should stay visible by default rather than risk being silently
// hidden as if it were routine terminal state. --priority (M-0263)
// joins the same resolved-vs-collision split as --area.
func crossBranchListRows(ctx context.Context, tr *tree.Tree, kind, status, parent, area, priority string, archived bool) []ListSummary {
	if tr.Root == "" {
		// An in-memory tree built directly (bare &tree.Tree{...}, no
		// tree.Load) never scans cross-branch: exec.Cmd treats an
		// empty Dir as "inherit the caller's own working directory,"
		// which would otherwise run these git subprocesses against
		// whatever directory the test process happens to be running
		// in — never what a bare in-memory fixture intends. This
		// mirrors every other cross-branch-aware field's documented
		// degrade for in-memory trees (nil CrossBranchHits, etc).
		return nil
	}
	all := append(trunk.LocalRefHits(ctx, tr.Root), trunk.RemoteRefHits(ctx, tr.Root)...)
	if len(all) == 0 {
		return nil
	}
	byID := make(map[string][]trunk.RefHit, len(all))
	for _, h := range all {
		canon := entity.Canonicalize(h.ID)
		byID[canon] = append(byID[canon], h)
	}
	collisions := trunk.DetectCollisions(ctx, tr.Root, all)
	canonParent := entity.Canonicalize(parent)

	var rows []ListSummary
	var br *gitops.BlobReader
	brAttempted := false
	defer func() {
		if br != nil {
			_ = br.Close()
		}
	}()

	for canon, hits := range byID {
		if tr.ByID(canon) != nil {
			continue // present locally — the local loop above already handled it
		}
		if kind != "" && string(hits[0].Kind) != kind {
			continue
		}
		if collisions[canon] {
			if status != "" || parent != "" || area != "" || priority != "" {
				continue
			}
			rows = append(rows, ListSummary{
				ID:                   canon,
				Kind:                 string(hits[0].Kind),
				CrossBranchCollision: true,
				CrossBranchRefs:      trunk.DistinctRefs(hits),
			})
			continue
		}

		if !brAttempted {
			brAttempted = true
			br, _ = gitops.NewBlobReader(ctx, tr.Root) // best-effort; nil on failure degrades below
		}
		if br == nil { //coverage:ignore by this point LocalRefHits/RemoteRefHits already confirmed tr.Root is a real repo (gitops.IsRepo); NewBlobReader failing here needs the repo to break between that scan and this construction — not reproducible against a healthy subprocess, same class as gitops.NewBlobReader's own internal coverage:ignore branches
			continue
		}

		hit := hits[0]
		content, err := br.Read(hit.Ref, hit.Path)
		if err != nil { //coverage:ignore hit.Path was just confirmed present at hit.Ref by the LsTreePaths scan that produced this RefHit; a subsequent blob read at the same ref:path failing needs the object store to change mid-request (repack/gc race), not reproducible in a unit test
			continue
		}
		e, err := entity.Parse(hit.Path, content)
		if err != nil {
			continue
		}
		e.Kind = hit.Kind

		if status != "" && e.Status != status {
			continue
		}
		if !archived && entity.IsTerminal(e.Kind, e.Status) {
			continue
		}
		if parent != "" && entity.Canonicalize(e.Parent) != canonParent {
			continue
		}
		if area != "" && tr.ResolvedArea(e) != area {
			continue
		}
		if priority != "" && e.Priority != priority {
			continue
		}
		rows = append(rows, ListSummary{
			ID:             entity.Canonicalize(e.ID),
			Kind:           string(e.Kind),
			Status:         e.Status,
			Title:          e.Title,
			Parent:         entity.Canonicalize(e.Parent),
			Path:           e.Path,
			Priority:       e.Priority,
			CrossBranchRef: hit.Ref,
		})
	}
	return rows
}

// BuildListCounts returns the per-kind count of non-terminal entities
// for the no-args invocation. Iteration order follows entity.AllKinds.
func BuildListCounts(tr *tree.Tree) ListCounts {
	out := ListCounts{}
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

// RenderListCountsText emits the per-kind summary line in the order
// dictated by entity.AllKinds. Format:
//
//	5 epics · 47 milestones · 12 ADRs · 14 gaps · 3 decisions · 1 contract
func RenderListCountsText(w io.Writer, counts ListCounts) {
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

// RenderListRowsText emits one row per entity with aligned columns:
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
func RenderListRowsText(w io.Writer, rows []ListSummary, titleBudget int, colorEnabled bool) {
	if len(rows) == 0 {
		return
	}
	// Pre-compute the rendered status for each row so the truncation
	// budget measures the glyph-prefixed width, not the raw status. Two
	// runes per glyphed status ("X "); empty string for rows whose
	// status falls outside the palette (defensive — the kernel's status
	// vocabulary is closed and maps fully today).
	//
	// M-0260/AC-1/AC-2: a cross-branch row's status column is marked
	// distinctly — " ⇄" appended for a resolved row (real status still
	// shown), "⇄ collision" in place of a status a collision row
	// doesn't have — so it never reads as an ordinary local row.
	statuses := make([]string, len(rows))
	for i := range rows {
		switch {
		case rows[i].CrossBranchCollision:
			statuses[i] = "⇄ collision"
		case rows[i].CrossBranchRef != "":
			if g := render.StatusGlyph(rows[i].Status); g != "" {
				statuses[i] = g + " " + rows[i].Status + " ⇄"
			} else { //coverage:ignore defensive: the kernel's status vocabulary is closed and maps fully today, so a resolved cross-branch row's real status always has a glyph; not reachable via any currently-legal entity status
				statuses[i] = rows[i].Status + " ⇄"
			}
		default:
			if g := render.StatusGlyph(rows[i].Status); g != "" {
				statuses[i] = g + " " + rows[i].Status
			} else { //coverage:ignore defensive: the kernel's status vocabulary is closed and maps fully today; not reachable via any currently-legal entity status
				statuses[i] = rows[i].Status
			}
		}
	}
	titleMax := ComputeTitleBudget(rows, statuses, titleBudget)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, render.Bold("ID\tSTATUS\tTITLE\tPARENT", colorEnabled))
	for i := range rows {
		parent := rows[i].Parent
		if parent == "" {
			parent = "-"
		}
		title := rows[i].Title
		if titleMax > 0 {
			title = render.Truncate(title, titleMax)
		}
		if rows[i].CrossBranchCollision {
			title = "(declines to render — refs diverge)"
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", rows[i].ID, statuses[i], title, parent)
	}
	_ = tw.Flush()
}

// ComputeTitleBudget returns the per-row rune cap for the title column,
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
// MinTitleColumnRunes — if the remainder would be below the floor,
// truncation buys so little (and looks so ugly) that we return 0 and
// let the terminal wrap. The header row contributes its own width
// (ID/STATUS/TITLE/PARENT, all narrower than typical content) so we
// don't measure it.
func ComputeTitleBudget(rows []ListSummary, renderedStatuses []string, termWidth int) int {
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
	if budget < MinTitleColumnRunes {
		return 0
	}
	return budget
}

// MinTitleColumnRunes is the floor below which the title column is no
// longer worth truncating — at 10 runes most titles collapse to a few
// initial words plus "…", which is more annoying than useful. When the
// terminal is narrower than what's needed to leave this much room, we
// give up on truncation and let the terminal wrap as it always has.
const MinTitleColumnRunes = 10

// termTitleBudget resolves the title-truncation budget for stdout. The
// returned width is the terminal width when stdout is a TTY and
// noTrunc is false; 0 otherwise. Callers feed this into
// ComputeTitleBudget to compute the per-column cap.
func termTitleBudget(f *os.File, noTrunc bool) int {
	if noTrunc {
		return 0
	}
	return render.TerminalWidth(f)
}
