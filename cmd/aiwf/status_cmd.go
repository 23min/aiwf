package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/tree"
)

// recentActivityLimit is the number of recent commits surfaced by
// `aiwf status`'s "Recent activity" section. Five fits in a glance and
// answers "what changed lately?" without scrolling. For longer history,
// fall through to `aiwf history <id>`.
const recentActivityLimit = 5

// statusReport is the pure-data payload for `aiwf status`. The text and
// JSON renderers consume the same struct; buildStatus produces it from
// a loaded tree. Lives alongside the CLI dispatcher rather than under
// internal/ because it is purely a presentational read view — adding a
// package boundary would be over-engineering for one verb.
type statusReport struct {
	Date           string              `json:"date"`
	InFlightEpics  []statusEpic        `json:"in_flight_epics"`
	PlannedEpics   []statusEpic        `json:"planned_epics"`
	OpenDecisions  []statusEntity      `json:"open_decisions"`
	OpenGaps       []statusGap         `json:"open_gaps"`
	Warnings       []statusFinding     `json:"warnings"`
	RecentActivity []HistoryEvent      `json:"recent_activity"`
	SweepPending   *statusSweepPending `json:"sweep_pending,omitempty"`
	Health         statusHealthCounts  `json:"health"`
}

// statusSweepPending is the tree-health one-liner for terminal-status
// entities still living in active directories. Per ADR-0004 §"Display
// surfaces": "The tree-health section gains a one-liner when sweep is
// pending: 'Sweep pending: N terminal entities not yet archived (run
// `aiwf archive --dry-run` to preview).' Hidden when 0."
//
// Populated from the `archive-sweep-pending` aggregate finding
// (M-0086); nil when the count is zero so the renderer can skip the
// section with a single nil-check. Lifted out of statusReport.Warnings
// on purpose — the aggregate belongs in the tree-health section, not
// mixed in with body-empty / resolver-missing warnings.
type statusSweepPending struct {
	Count   int    `json:"count"`
	Message string `json:"message"`
}

// statusFinding is one warning surfaced inline in the status report.
// Mirrors the load-bearing fields of check.Finding without coupling the
// JSON shape to the validator package's internal schema.
type statusFinding struct {
	Code     string `json:"code"`
	EntityID string `json:"entity_id,omitempty"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
}

// statusEpic is one in-flight epic plus every milestone under it.
type statusEpic struct {
	ID         string            `json:"id"`
	Title      string            `json:"title"`
	Status     string            `json:"status"`
	Milestones []statusMilestone `json:"milestones"`
}

// statusMilestone is one milestone under an in-flight epic, with the
// in-progress one identifiable by Status. The TDD and ACs fields
// carry the I2 acceptance-criteria surface; ACs is omitted from JSON
// when the milestone carries none (zero progress).
type statusMilestone struct {
	ID     string            `json:"id"`
	Title  string            `json:"title"`
	Status string            `json:"status"`
	TDD    string            `json:"tdd,omitempty"`
	ACs    *statusACProgress `json:"acs,omitempty"`
}

// statusACProgress is the per-status count of a milestone's ACs.
// `Total` includes cancelled entries (they remain in the list per
// the position-stability rule); `InScope` excludes them, so that's
// the denominator the renderers use for "M/T met" progress.
type statusACProgress struct {
	Total     int `json:"total"`
	InScope   int `json:"in_scope"`
	Open      int `json:"open"`
	Met       int `json:"met"`
	Deferred  int `json:"deferred"`
	Cancelled int `json:"cancelled"`
}

// summarizeACs returns the per-status counts for a milestone's acs[].
// Returns nil when the slice is empty so the renderer can skip the
// "ACs: …" suffix entirely on milestones that don't carry any.
func summarizeACs(acs []entity.AcceptanceCriterion) *statusACProgress {
	if len(acs) == 0 {
		return nil
	}
	p := &statusACProgress{Total: len(acs)}
	for i := range acs {
		switch acs[i].Status {
		case entity.StatusOpen:
			p.Open++
		case entity.StatusMet:
			p.Met++
		case entity.StatusDeferred:
			p.Deferred++
		case entity.StatusCancelled:
			p.Cancelled++
		}
	}
	p.InScope = p.Total - p.Cancelled
	return p
}

// renderACProgress formats the AC progress badge appended to a
// milestone row. Returns "" when there are no ACs (so the renderer
// can skip the separator). Format:
//
//	"ACs 2/3 met"           — typical case, in-scope total ≥ 1
//	"ACs 1/2 met (1 open)"  — when there are still open ACs
//	"ACs all cancelled"     — every AC was cancelled (in-scope = 0)
func renderACProgress(p *statusACProgress) string {
	if p == nil {
		return ""
	}
	if p.InScope == 0 {
		return "ACs all cancelled"
	}
	out := fmt.Sprintf("ACs %d/%d met", p.Met, p.InScope)
	if p.Open > 0 {
		out += fmt.Sprintf(" (%d open)", p.Open)
	}
	return out
}

// statusEntity is the shared shape for ADRs and decisions in the
// "Open decisions" section.
type statusEntity struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Kind   string `json:"kind"`
}

// statusGap is one open gap with the milestone or epic it was
// discovered in (if any).
type statusGap struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	DiscoveredIn string `json:"discovered_in,omitempty"`
}

// statusHealthCounts summarizes the tree's current validation state
// without re-running expensive checks; pulled from a single check.Run.
type statusHealthCounts struct {
	Entities int `json:"entities"`
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
}

// newStatusCmd builds `aiwf status`: a project-wide snapshot of in-flight
// work, open decisions, open gaps, and recent activity. Read-only;
// produces no commit. Use it to answer "what's next?", "where are we?",
// "what are we working on?".
func newStatusCmd() *cobra.Command {
	var (
		root    string
		format  string
		pretty  bool
		noTrunc bool
	)
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Project snapshot: in-flight work, open decisions, gaps, recent activity",
		Example: `  # One-screen project snapshot
  aiwf status

  # Markdown form (the same output committed to STATUS.md)
  aiwf status --format=md

  # Full titles even in a narrow terminal (default truncates long titles
  # when stdout is a TTY narrower than the row)
  aiwf status --no-trunc`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return wrapExitCode(runStatusCmd(root, format, pretty, noTrunc))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root (default: discover via aiwf.yaml)")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text, json, or md")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "indent JSON output (only with --format=json)")
	cmd.Flags().BoolVar(&noTrunc, "no-trunc", false, "do not truncate long titles when stdout is a terminal narrower than the row")
	_ = cmd.RegisterFlagCompletionFunc("format", cobra.FixedCompletions(
		[]string{"text", "json", "md"},
		cobra.ShellCompDirectiveNoFileComp,
	))
	return cmd
}

func runStatusCmd(root, format string, pretty, noTrunc bool) int {
	if format != "text" && format != "json" && format != "md" {
		fmt.Fprintf(os.Stderr, "aiwf status: --format must be 'text', 'json', or 'md', got %q\n", format)
		return exitUsage
	}

	rootDir, err := resolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf status: %v\n", err)
		return exitUsage
	}

	ctx := context.Background()
	tr, loadErrs, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf status: loading tree: %v\n", err)
		return exitInternal
	}

	report := buildStatus(tr, loadErrs)

	recent, err := readRecentActivity(ctx, rootDir, recentActivityLimit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf status: reading recent activity: %v\n", err)
		return exitInternal
	}
	report.RecentActivity = recent

	switch format {
	case "text":
		termWidth := 0
		if !noTrunc {
			termWidth = render.TerminalWidth(os.Stdout)
		}
		if err := renderStatusText(os.Stdout, &report, termWidth, render.ColorEnabled(os.Stdout)); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf status: writing output: %v\n", err)
			return exitInternal
		}
	case "json":
		env := render.Envelope{
			Tool:    "aiwf",
			Version: Version,
			Status:  "ok",
			Result:  &report,
			Metadata: map[string]any{
				"root":     rootDir,
				"entities": report.Health.Entities,
			},
		}
		if err := render.JSON(os.Stdout, env, pretty); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf status: writing output: %v\n", err)
			return exitInternal
		}
	case "md":
		if err := renderStatusMarkdown(os.Stdout, &report); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf status: writing output: %v\n", err)
			return exitInternal
		}
	}
	return exitOK
}

// buildStatus returns the project status payload for tree tr, with
// loadErrs surfaced as part of the health counts. The returned report
// has RecentActivity unset; the caller fills it via readRecentActivity
// (which needs git access; this function stays pure for testability).
func buildStatus(tr *tree.Tree, loadErrs []tree.LoadError) statusReport {
	r := statusReport{
		Date: time.Now().UTC().Format("2006-01-02"),
	}

	// In-flight epics: active or proposed. The shared filter helper
	// gives us both buckets pre-sorted; the loop below splits them.
	// `aiwf list --kind epic --status active` routes through the same
	// helper, so the two verbs cannot drift on identical queries
	// (M-072 AC-6).
	epics := tr.FilterByKindStatuses(entity.KindEpic, entity.StatusActive, entity.StatusProposed)

	// Group milestones by canonicalized parent so a narrow parent ref
	// (E-22) and a canonical parent ref (E-0022) bucket together — AC-2
	// in M-081.
	milestonesByParent := map[string][]*entity.Entity{}
	for _, m := range tr.ByKind(entity.KindMilestone) {
		key := entity.Canonicalize(m.Parent)
		milestonesByParent[key] = append(milestonesByParent[key], m)
	}
	for _, ms := range milestonesByParent {
		sort.SliceStable(ms, func(i, j int) bool { return ms[i].ID < ms[j].ID })
	}

	for _, e := range epics {
		canonEpic := entity.Canonicalize(e.ID)
		se := statusEpic{
			ID:     canonEpic,
			Title:  e.Title,
			Status: e.Status,
		}
		for _, m := range milestonesByParent[canonEpic] {
			se.Milestones = append(se.Milestones, statusMilestone{
				ID:     entity.Canonicalize(m.ID),
				Title:  m.Title,
				Status: m.Status,
				TDD:    m.TDD,
				ACs:    summarizeACs(m.ACs),
			})
		}
		switch e.Status {
		case entity.StatusActive:
			r.InFlightEpics = append(r.InFlightEpics, se)
		case entity.StatusProposed:
			r.PlannedEpics = append(r.PlannedEpics, se)
		}
	}

	// Open decisions: ADRs and Decision entities with status == "proposed".
	// Each kind is one helper call; the merge needs an outer sort to
	// interleave the two id namespaces deterministically.
	for _, e := range tr.FilterByKindStatuses(entity.KindADR, entity.StatusProposed) {
		r.OpenDecisions = append(r.OpenDecisions, statusEntity{
			ID:     entity.Canonicalize(e.ID),
			Title:  e.Title,
			Status: e.Status,
			Kind:   string(entity.KindADR),
		})
	}
	for _, e := range tr.FilterByKindStatuses(entity.KindDecision, entity.StatusProposed) {
		r.OpenDecisions = append(r.OpenDecisions, statusEntity{
			ID:     entity.Canonicalize(e.ID),
			Title:  e.Title,
			Status: e.Status,
			Kind:   string(entity.KindDecision),
		})
	}
	sort.SliceStable(r.OpenDecisions, func(i, j int) bool { return r.OpenDecisions[i].ID < r.OpenDecisions[j].ID })

	// Open gaps: status == "open". Helper returns id-sorted; no extra
	// sort needed.
	for _, e := range tr.FilterByKindStatuses(entity.KindGap, entity.StatusOpen) {
		r.OpenGaps = append(r.OpenGaps, statusGap{
			ID:           entity.Canonicalize(e.ID),
			Title:        e.Title,
			DiscoveredIn: entity.Canonicalize(e.DiscoveredIn),
		})
	}

	// Health: errors and warnings from a single check.Run. Warning
	// detail is surfaced inline; errors stay summarised — if there are
	// any, the user should run `aiwf check` for the full report.
	//
	// `archive-sweep-pending` (M-0086 aggregate) is lifted out of the
	// general warnings list into r.SweepPending — per ADR-0004
	// §"Display surfaces", the sweep-pending one-liner belongs in the
	// tree-health section, not in the general warnings stream. The
	// per-file `terminal-entity-not-archived` warnings stay in
	// r.Warnings alongside other finding codes.
	findings := check.Run(tr, loadErrs)
	for i := range findings {
		switch findings[i].Severity {
		case check.SeverityError:
			r.Health.Errors++
		case check.SeverityWarning:
			r.Health.Warnings++
			if findings[i].Code == "archive-sweep-pending" {
				r.SweepPending = parseSweepPending(findings[i].Message)
				continue
			}
			r.Warnings = append(r.Warnings, statusFinding{
				Code:     findings[i].Code,
				EntityID: entity.Canonicalize(findings[i].EntityID),
				Path:     findings[i].Path,
				Message:  findings[i].Message,
			})
		}
	}
	r.Health.Entities = len(tr.Entities)

	return r
}

// parseSweepPending extracts the count from an `archive-sweep-pending`
// finding message and packages it into a statusSweepPending. The
// rule's message format ("%d terminal entities awaiting `aiwf archive
// --apply`...") is the upstream contract; this function is the
// consumer-side parser, so a future format change must update both
// sites. The friendlier render-side message names the dry-run verb
// per ADR-0004's worded example ("run `aiwf archive --dry-run` to
// preview").
//
// Returns nil if the message doesn't begin with a digit, which would
// only happen if the upstream finding-rule produced an empty count;
// the rule itself returns nil at zero so this branch shouldn't fire
// in practice.
func parseSweepPending(message string) *statusSweepPending {
	var count int
	if _, err := fmt.Sscanf(message, "%d", &count); err != nil || count <= 0 {
		return nil
	}
	return &statusSweepPending{
		Count: count,
		Message: fmt.Sprintf("Sweep pending: %d terminal entit%s not yet archived (run `aiwf archive --dry-run` to preview)",
			count, plural(count, "y", "ies")),
	}
}

// plural picks between singular and plural noun endings; small,
// inlined to avoid a dependency for one call site.
func plural(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

// readRecentActivity returns the last `limit` commits whose message
// carries any `aiwf-verb:` trailer. Cross-entity, no filter — used by
// `aiwf status` to answer "what changed lately?" across the whole
// project. Events come back newest-first.
//
// The `--grep` is an I/O-narrowing pre-filter; correctness is gated
// on Git's structured trailer parser (`%(trailers:key=…,valueonly=…)`).
// Hand-authored prose that wraps such that a line happens to start
// with `aiwf-verb:` would match the grep but produce an empty parsed
// Verb column — those records are skipped (G30). The grep over a
// long history is also asked for more rows than `limit` so the
// post-filter doesn't silently shrink the result; we then truncate.
func readRecentActivity(ctx context.Context, root string, limit int) ([]HistoryEvent, error) {
	if !hasCommits(ctx, root) {
		return nil, nil
	}
	const sep = "\x1f"
	const recSep = "\x1e\n"
	// Over-fetch so post-filtering can't drop us below `limit`. Four
	// times the requested count handles repos with a heavy ratio of
	// prose-mention false-positives without unbounded scanning; if a
	// repo has more than `3*limit` consecutive false-positives in its
	// most recent history, the user will see a shorter table — that's
	// surface noise, not correctness loss.
	fetchN := limit * 4
	if fetchN < limit {
		fetchN = limit
	}
	cmd := exec.CommandContext(ctx, "git", "log",
		"-n", fmt.Sprintf("%d", fetchN),
		"--grep", "^aiwf-verb: ",
		"--pretty=tformat:%H"+sep+"%aI"+sep+"%s"+sep+"%(trailers:key=aiwf-verb,valueonly=true,unfold=true)"+sep+"%(trailers:key=aiwf-actor,valueonly=true,unfold=true)"+sep+"%b\x1e",
	)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git log: %w\n%s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git log: %w", err)
	}

	var events []HistoryEvent
	for _, rec := range strings.Split(string(out), recSep) {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, sep, 6)
		if len(parts) < 6 {
			continue
		}
		verb := strings.TrimSpace(parts[3])
		if verb == "" {
			// Prose-mention false-positive: --grep matched a wrapped
			// line that starts with "aiwf-verb:" but Git's trailer
			// parser found no real trailer. Skip.
			continue
		}
		events = append(events, HistoryEvent{
			Commit: shortHash(parts[0]),
			Date:   parts[1],
			Detail: strings.TrimSpace(parts[2]),
			Verb:   verb,
			Actor:  strings.TrimSpace(parts[4]),
			Body:   stripTrailers(strings.TrimSpace(parts[5])),
		})
		if len(events) >= limit {
			break
		}
	}
	return events, nil
}

// writeStatusEpicText writes one epic plus its milestones in the
// terminal-friendly shape shared by the In flight and Roadmap sections.
// termWidth caps title widths so long titles don't wrap into the next
// row's column-zero (the G-0080 visual-scan bug); 0 disables truncation.
//
// Milestone rows lead with a glyph from the G-0080 palette so every
// row carries a visible state marker — the in-progress and done
// glyphs (→ ✓) have always been present; this function also emits ○
// for draft and ✗ for cancelled so the four-glyph palette is uniformly
// applied across all milestone states.
func writeStatusEpicText(b *strings.Builder, e statusEpic, termWidth int) {
	epicPrefix := fmt.Sprintf("  %s — ", e.ID)
	epicTail := fmt.Sprintf("    [%s]", e.Status)
	epicTitle := truncStatusTitle(e.Title, termWidth, epicPrefix, epicTail)
	fmt.Fprintf(b, "%s%s%s\n", epicPrefix, epicTitle, epicTail)
	if len(e.Milestones) == 0 {
		b.WriteString("       (no milestones)\n")
	}
	for _, m := range e.Milestones {
		// 3-rune marker keeps the milestone id at a fixed column across
		// every status; the glyph is centred so " ○ M-0001" lines up
		// with " → M-0002" and " ✓ M-0003" regardless of state.
		marker := "   "
		if g := render.StatusGlyph(m.Status); g != "" {
			marker = " " + g + " "
		}
		suffix := ""
		if progress := renderACProgress(m.ACs); progress != "" {
			suffix = "    · " + progress
		}
		if m.TDD != "" {
			suffix += "    · tdd: " + m.TDD
		}
		msPrefix := fmt.Sprintf("    %s%s — ", marker, m.ID)
		msTail := fmt.Sprintf("    [%s]%s", m.Status, suffix)
		msTitle := truncStatusTitle(m.Title, termWidth, msPrefix, msTail)
		fmt.Fprintf(b, "%s%s%s\n", msPrefix, msTitle, msTail)
	}
}

// truncStatusTitle caps title to fit termWidth given the non-title
// prefix/tail of the line. Returns title unchanged when termWidth is 0
// (no TTY / --no-trunc) or when the available room would fall below
// the per-G-0080 minimum useful column width. The minimum (10 runes)
// is the same floor renderListRowsText uses — see minTitleColumnRunes.
func truncStatusTitle(title string, termWidth int, prefix, tail string) string {
	if termWidth <= 0 {
		return title
	}
	other := utf8.RuneCountInString(prefix) + utf8.RuneCountInString(tail)
	avail := termWidth - other
	if avail < minTitleColumnRunes {
		return title
	}
	return render.Truncate(title, avail)
}

// renderStatusText writes the human-readable status report to w. The
// in-progress milestone gets a `→` prefix; done a `✓`; draft a `○`;
// cancelled a `✗` — the four-glyph G-0080 palette, applied so every
// milestone row carries a visible state marker. Empty sections render
// with a parenthesised "(none)" so a glance can see "yes there are
// open decisions" without counting bullets. Builds the full output in
// a strings.Builder and writes once so the only error to surface is
// the final write.
//
// termWidth caps title widths to keep rows on one visual line when
// stdout is a TTY narrower than the natural row; pass 0 to disable
// truncation (default in tests, in pipes, and under --no-trunc).
// colorEnabled toggles ANSI-bold section labels; row content stays
// escape-free so downstream tooling (grep, awk) sees plain text. The
// glyph palette is content, not style, and appears regardless.
func renderStatusText(w io.Writer, r *statusReport, termWidth int, colorEnabled bool) error {
	var b strings.Builder
	fmt.Fprintf(&b, "aiwf status — %s\n\n", r.Date)

	b.WriteString(render.Bold("In flight", colorEnabled) + "\n")
	if len(r.InFlightEpics) == 0 {
		b.WriteString("  (no active epics)\n")
	}
	for _, e := range r.InFlightEpics {
		writeStatusEpicText(&b, e, termWidth)
	}
	b.WriteByte('\n')

	b.WriteString(render.Bold("Roadmap", colorEnabled) + "\n")
	if len(r.PlannedEpics) == 0 {
		b.WriteString("  (nothing planned)\n")
	}
	for _, e := range r.PlannedEpics {
		writeStatusEpicText(&b, e, termWidth)
	}
	b.WriteByte('\n')

	b.WriteString(render.Bold("Open decisions", colorEnabled) + "\n")
	if len(r.OpenDecisions) == 0 {
		b.WriteString("  (none)\n")
	}
	for _, d := range r.OpenDecisions {
		prefix := fmt.Sprintf("  %s — ", d.ID)
		tail := fmt.Sprintf("    [%s]", d.Status)
		title := truncStatusTitle(d.Title, termWidth, prefix, tail)
		fmt.Fprintf(&b, "%s%s%s\n", prefix, title, tail)
	}
	b.WriteByte('\n')

	b.WriteString(render.Bold("Open gaps", colorEnabled) + "\n")
	if len(r.OpenGaps) == 0 {
		b.WriteString("  (none)\n")
	}
	for _, g := range r.OpenGaps {
		prefix := fmt.Sprintf("  %s — ", g.ID)
		tail := ""
		if g.DiscoveredIn != "" {
			tail = fmt.Sprintf("    (discovered in %s)", g.DiscoveredIn)
		}
		title := truncStatusTitle(g.Title, termWidth, prefix, tail)
		fmt.Fprintf(&b, "%s%s%s\n", prefix, title, tail)
	}
	b.WriteByte('\n')

	b.WriteString(render.Bold("Warnings", colorEnabled) + "\n")
	if len(r.Warnings) == 0 {
		b.WriteString("  (none)\n")
	}
	for _, w := range r.Warnings {
		switch {
		case w.EntityID != "":
			fmt.Fprintf(&b, "  %s  [%s]  %s\n", w.Code, w.EntityID, w.Message)
		case w.Path != "":
			fmt.Fprintf(&b, "  %s  (%s)  %s\n", w.Code, w.Path, w.Message)
		default:
			fmt.Fprintf(&b, "  %s  %s\n", w.Code, w.Message)
		}
	}
	b.WriteByte('\n')

	b.WriteString(render.Bold("Recent activity", colorEnabled) + "\n")
	if len(r.RecentActivity) == 0 {
		b.WriteString("  (none)\n")
	}
	for i := range r.RecentActivity {
		ev := &r.RecentActivity[i]
		date := ev.Date
		if len(date) >= 10 {
			date = date[:10]
		}
		fmt.Fprintf(&b, "  %s  %-16s  %-10s  %s\n", date, ev.Actor, ev.Verb, ev.Detail)
	}
	b.WriteByte('\n')

	b.WriteString(render.Bold("Health", colorEnabled) + "\n")
	// Sweep-pending one-liner lives in the Health section per
	// ADR-0004 §"Display surfaces". Nil-checked at the top of the
	// section so an absent SweepPending stays silent (AC-2).
	if r.SweepPending != nil {
		fmt.Fprintf(&b, "  %s\n", r.SweepPending.Message)
	}
	suffix := ""
	if r.Health.Errors > 0 || r.Health.Warnings > 0 {
		suffix = " · run `aiwf check` for details"
	}
	fmt.Fprintf(&b, "  %d entities · %d errors · %d warnings%s\n",
		r.Health.Entities, r.Health.Errors, r.Health.Warnings, suffix)

	_, err := io.WriteString(w, b.String())
	return err
}

// renderStatusMarkdown writes the status report as a self-contained
// markdown document, with mermaid `flowchart` blocks for in-flight and
// roadmap epics. The output renders unchanged in any markdown viewer
// that supports mermaid (GitHub web, VSCode, Obsidian, glow + mermaid
// extension, etc.). Plain markdown — no HTML, no JS.
func renderStatusMarkdown(w io.Writer, r *statusReport) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# aiwf status — %s\n\n", r.Date)

	suffix := ""
	if r.Health.Errors > 0 || r.Health.Warnings > 0 {
		suffix = " · run `aiwf check` for details"
	}
	fmt.Fprintf(&b, "_%d entities · %d errors · %d warnings%s_\n\n",
		r.Health.Entities, r.Health.Errors, r.Health.Warnings, suffix)
	// Sweep-pending one-liner — same Health-section placement as the
	// text renderer. Quoted as markdown blockquote so the line stands
	// out visually inline with the health summary.
	if r.SweepPending != nil {
		fmt.Fprintf(&b, "> %s\n\n", r.SweepPending.Message)
	}

	b.WriteString("## In flight\n\n")
	if len(r.InFlightEpics) == 0 {
		b.WriteString("_(no active epics)_\n\n")
	}
	for _, e := range r.InFlightEpics {
		writeStatusEpicMarkdown(&b, e)
	}

	b.WriteString("## Roadmap\n\n")
	if len(r.PlannedEpics) == 0 {
		b.WriteString("_(nothing planned)_\n\n")
	}
	for _, e := range r.PlannedEpics {
		writeStatusEpicMarkdown(&b, e)
	}

	b.WriteString("## Open decisions\n\n")
	if len(r.OpenDecisions) == 0 {
		b.WriteString("_(none)_\n\n")
	} else {
		b.WriteString("| ID | Kind | Title | Status |\n")
		b.WriteString("|----|------|-------|--------|\n")
		for _, d := range r.OpenDecisions {
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
				d.ID, d.Kind, mdEscape(d.Title), d.Status)
		}
		b.WriteByte('\n')
	}

	b.WriteString("## Open gaps\n\n")
	if len(r.OpenGaps) == 0 {
		b.WriteString("_(none)_\n\n")
	} else {
		b.WriteString("| ID | Title | Discovered in |\n")
		b.WriteString("|----|-------|---------------|\n")
		for _, g := range r.OpenGaps {
			fmt.Fprintf(&b, "| %s | %s | %s |\n",
				g.ID, mdEscape(g.Title), g.DiscoveredIn)
		}
		b.WriteByte('\n')
	}

	b.WriteString("## Warnings\n\n")
	if len(r.Warnings) == 0 {
		b.WriteString("_(none)_\n\n")
	} else {
		b.WriteString("| Code | Entity | Path | Message |\n")
		b.WriteString("|------|--------|------|---------|\n")
		for _, ww := range r.Warnings {
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
				ww.Code, ww.EntityID, ww.Path, mdEscape(ww.Message))
		}
		b.WriteByte('\n')
	}

	b.WriteString("## Recent activity\n\n")
	if len(r.RecentActivity) == 0 {
		b.WriteString("_(none)_\n\n")
	} else {
		b.WriteString("| Date | Actor | Verb | Detail |\n")
		b.WriteString("|------|-------|------|--------|\n")
		for i := range r.RecentActivity {
			ev := &r.RecentActivity[i]
			date := ev.Date
			if len(date) >= 10 {
				date = date[:10]
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
				date, ev.Actor, ev.Verb, mdEscape(ev.Detail))
		}
		b.WriteByte('\n')
	}

	_, err := io.WriteString(w, b.String())
	return err
}

// writeStatusEpicMarkdown writes one epic — header, milestone list, and
// a mermaid `flowchart LR` keyed by milestone status — into b. Empty
// milestone lists render an explicit "(no milestones)" line so the
// section stays visually balanced and the diagram is omitted (mermaid
// barfs on a flowchart with one node and no edges).
func writeStatusEpicMarkdown(b *strings.Builder, e statusEpic) {
	fmt.Fprintf(b, "### %s — %s _(%s)_\n\n", e.ID, mdEscape(e.Title), e.Status)
	if len(e.Milestones) == 0 {
		b.WriteString("_(no milestones)_\n\n")
		return
	}
	for _, m := range e.Milestones {
		marker := ""
		switch m.Status {
		case entity.StatusInProgress:
			marker = "→ "
		case entity.StatusDone:
			marker = "✓ "
		}
		suffix := ""
		if progress := renderACProgress(m.ACs); progress != "" {
			suffix = " — " + progress
		}
		if m.TDD != "" {
			suffix += " — tdd: " + m.TDD
		}
		fmt.Fprintf(b, "- %s**%s** — %s _(%s)_%s\n", marker, m.ID, mdEscape(m.Title), m.Status, suffix)
	}
	b.WriteByte('\n')

	b.WriteString("```mermaid\nflowchart LR\n")
	fmt.Fprintf(b, "  %s[\"%s<br/>%s\"]:::epic_%s\n",
		mermaidID(e.ID), e.ID, mdEscape(e.Title), e.Status)
	for _, m := range e.Milestones {
		// Append "(M/T)" badge to the mermaid label when the milestone
		// has any in-scope ACs. Cancelled-only milestones get no badge
		// (the design's "all cancelled" case isn't useful at a glance).
		acBadge := ""
		if m.ACs != nil && m.ACs.InScope > 0 {
			acBadge = fmt.Sprintf(" (%d/%d)", m.ACs.Met, m.ACs.InScope)
		}
		fmt.Fprintf(b, "  %s[\"%s%s<br/>%s\"]:::ms_%s\n",
			mermaidID(m.ID), m.ID, acBadge, mdEscape(m.Title), m.Status)
		fmt.Fprintf(b, "  %s --> %s\n", mermaidID(e.ID), mermaidID(m.ID))
	}
	b.WriteString("  classDef epic_active fill:#d6eaff,stroke:#1a73e8,color:#000\n")
	b.WriteString("  classDef epic_proposed fill:#f4f4f4,stroke:#888,color:#000\n")
	b.WriteString("  classDef ms_done fill:#d8f5d8,stroke:#2a8a2a,color:#000\n")
	b.WriteString("  classDef ms_in_progress fill:#fff3c4,stroke:#caa400,color:#000\n")
	b.WriteString("  classDef ms_draft fill:#f4f4f4,stroke:#888,color:#000\n")
	b.WriteString("  classDef ms_cancelled fill:#fbeaea,stroke:#c33,color:#000\n")
	b.WriteString("```\n\n")
}

// mermaidID converts an entity id to a mermaid-safe node id by
// replacing the "-" with "_" (mermaid treats "-" as a metacharacter in
// some contexts). The id stays unique because the inverse mapping is
// trivial and the original id is shown in the node label.
func mermaidID(id string) string {
	return strings.ReplaceAll(id, "-", "_")
}

// mdEscape escapes the four characters that break a markdown table row
// or a mermaid label: pipe, backtick, the bracket pair (mermaid uses
// them for node syntax), and the double-quote (mermaid uses it to
// delimit labels). Newlines are stripped so a single field stays on
// one row. Conservative; not a full markdown sanitizer.
func mdEscape(s string) string {
	r := strings.NewReplacer(
		"|", "\\|",
		"`", "\\`",
		"\"", "'",
		"\n", " ",
		"\r", " ",
	)
	return r.Replace(s)
}
