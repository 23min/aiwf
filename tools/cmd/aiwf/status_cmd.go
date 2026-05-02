package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/render"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
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
	Date           string             `json:"date"`
	InFlightEpics  []statusEpic       `json:"in_flight_epics"`
	PlannedEpics   []statusEpic       `json:"planned_epics"`
	OpenDecisions  []statusEntity     `json:"open_decisions"`
	OpenGaps       []statusGap        `json:"open_gaps"`
	Warnings       []statusFinding    `json:"warnings"`
	RecentActivity []HistoryEvent     `json:"recent_activity"`
	Health         statusHealthCounts `json:"health"`
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

// runStatus handles `aiwf status`: a project-wide snapshot of in-flight
// work, open decisions, open gaps, and recent activity. Read-only;
// produces no commit. Use it to answer "what's next?", "where are we?",
// "what are we working on?".
func runStatus(args []string) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	root := fs.String("root", "", "consumer repo root (default: discover via aiwf.yaml)")
	format := fs.String("format", "text", "output format: text, json, or md")
	pretty := fs.Bool("pretty", false, "indent JSON output (only with --format=json)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if *format != "text" && *format != "json" && *format != "md" {
		fmt.Fprintf(os.Stderr, "aiwf status: --format must be 'text', 'json', or 'md', got %q\n", *format)
		return exitUsage
	}

	rootDir, err := resolveRoot(*root)
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

	switch *format {
	case "text":
		if err := renderStatusText(os.Stdout, &report); err != nil {
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
		if err := render.JSON(os.Stdout, env, *pretty); err != nil {
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

	// In-flight epics: status == "active". Sorted by id.
	epics := append([]*entity.Entity(nil), tr.ByKind(entity.KindEpic)...)
	sort.SliceStable(epics, func(i, j int) bool { return epics[i].ID < epics[j].ID })

	milestonesByParent := map[string][]*entity.Entity{}
	for _, m := range tr.ByKind(entity.KindMilestone) {
		milestonesByParent[m.Parent] = append(milestonesByParent[m.Parent], m)
	}
	for _, ms := range milestonesByParent {
		sort.SliceStable(ms, func(i, j int) bool { return ms[i].ID < ms[j].ID })
	}

	for _, e := range epics {
		if e.Status != entity.StatusActive && e.Status != entity.StatusProposed {
			continue
		}
		se := statusEpic{
			ID:     e.ID,
			Title:  e.Title,
			Status: e.Status,
		}
		for _, m := range milestonesByParent[e.ID] {
			se.Milestones = append(se.Milestones, statusMilestone{
				ID:     m.ID,
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
	for _, e := range tr.ByKind(entity.KindADR) {
		if e.Status != entity.StatusProposed {
			continue
		}
		r.OpenDecisions = append(r.OpenDecisions, statusEntity{
			ID:     e.ID,
			Title:  e.Title,
			Status: e.Status,
			Kind:   string(entity.KindADR),
		})
	}
	for _, e := range tr.ByKind(entity.KindDecision) {
		if e.Status != entity.StatusProposed {
			continue
		}
		r.OpenDecisions = append(r.OpenDecisions, statusEntity{
			ID:     e.ID,
			Title:  e.Title,
			Status: e.Status,
			Kind:   string(entity.KindDecision),
		})
	}
	sort.SliceStable(r.OpenDecisions, func(i, j int) bool { return r.OpenDecisions[i].ID < r.OpenDecisions[j].ID })

	// Open gaps: status == "open".
	for _, e := range tr.ByKind(entity.KindGap) {
		if e.Status != entity.StatusOpen {
			continue
		}
		r.OpenGaps = append(r.OpenGaps, statusGap{
			ID:           e.ID,
			Title:        e.Title,
			DiscoveredIn: e.DiscoveredIn,
		})
	}
	sort.SliceStable(r.OpenGaps, func(i, j int) bool { return r.OpenGaps[i].ID < r.OpenGaps[j].ID })

	// Health: errors and warnings from a single check.Run. Warning
	// detail is surfaced inline; errors stay summarised — if there are
	// any, the user should run `aiwf check` for the full report.
	findings := check.Run(tr, loadErrs)
	for i := range findings {
		switch findings[i].Severity {
		case check.SeverityError:
			r.Health.Errors++
		case check.SeverityWarning:
			r.Health.Warnings++
			r.Warnings = append(r.Warnings, statusFinding{
				Code:     findings[i].Code,
				EntityID: findings[i].EntityID,
				Path:     findings[i].Path,
				Message:  findings[i].Message,
			})
		}
	}
	r.Health.Entities = len(tr.Entities)

	return r
}

// readRecentActivity returns the last `limit` commits whose message
// carries any `aiwf-verb:` trailer. Cross-entity, no filter — used by
// `aiwf status` to answer "what changed lately?" across the whole
// project. Events come back newest-first.
func readRecentActivity(ctx context.Context, root string, limit int) ([]HistoryEvent, error) {
	if !hasCommits(ctx, root) {
		return nil, nil
	}
	const sep = "\x1f"
	const recSep = "\x1e\n"
	cmd := exec.CommandContext(ctx, "git", "log",
		"-n", fmt.Sprintf("%d", limit),
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
		events = append(events, HistoryEvent{
			Commit: shortHash(parts[0]),
			Date:   parts[1],
			Detail: strings.TrimSpace(parts[2]),
			Verb:   strings.TrimSpace(parts[3]),
			Actor:  strings.TrimSpace(parts[4]),
			Body:   stripTrailers(strings.TrimSpace(parts[5])),
		})
	}
	return events, nil
}

// writeStatusEpicText writes one epic plus its milestones in the
// terminal-friendly shape shared by the In flight and Roadmap sections.
func writeStatusEpicText(b *strings.Builder, e statusEpic) {
	fmt.Fprintf(b, "  %s — %s    [%s]\n", e.ID, e.Title, e.Status)
	if len(e.Milestones) == 0 {
		b.WriteString("       (no milestones)\n")
	}
	for _, m := range e.Milestones {
		marker := "   "
		switch m.Status {
		case entity.StatusInProgress:
			marker = " → "
		case entity.StatusDone:
			marker = " ✓ "
		}
		suffix := ""
		if progress := renderACProgress(m.ACs); progress != "" {
			suffix = "    · " + progress
		}
		if m.TDD != "" {
			suffix += "    · tdd: " + m.TDD
		}
		fmt.Fprintf(b, "    %s%s — %s    [%s]%s\n", marker, m.ID, m.Title, m.Status, suffix)
	}
}

// renderStatusText writes the human-readable status report to w. The
// in-progress milestone gets a `→` prefix; done a `✓`; everything else
// blank-prefix so the row aligns. Empty sections render with a
// parenthesised "(none)" so a glance can see "yes there are open
// decisions" without counting bullets. Builds the full output in a
// strings.Builder and writes once so the only error to surface is the
// final write.
func renderStatusText(w io.Writer, r *statusReport) error {
	var b strings.Builder
	fmt.Fprintf(&b, "aiwf status — %s\n\n", r.Date)

	b.WriteString("In flight\n")
	if len(r.InFlightEpics) == 0 {
		b.WriteString("  (no active epics)\n")
	}
	for _, e := range r.InFlightEpics {
		writeStatusEpicText(&b, e)
	}
	b.WriteByte('\n')

	b.WriteString("Roadmap\n")
	if len(r.PlannedEpics) == 0 {
		b.WriteString("  (nothing planned)\n")
	}
	for _, e := range r.PlannedEpics {
		writeStatusEpicText(&b, e)
	}
	b.WriteByte('\n')

	b.WriteString("Open decisions\n")
	if len(r.OpenDecisions) == 0 {
		b.WriteString("  (none)\n")
	}
	for _, d := range r.OpenDecisions {
		fmt.Fprintf(&b, "  %s — %s    [%s]\n", d.ID, d.Title, d.Status)
	}
	b.WriteByte('\n')

	b.WriteString("Open gaps\n")
	if len(r.OpenGaps) == 0 {
		b.WriteString("  (none)\n")
	}
	for _, g := range r.OpenGaps {
		if g.DiscoveredIn != "" {
			fmt.Fprintf(&b, "  %s — %s    (discovered in %s)\n", g.ID, g.Title, g.DiscoveredIn)
		} else {
			fmt.Fprintf(&b, "  %s — %s\n", g.ID, g.Title)
		}
	}
	b.WriteByte('\n')

	b.WriteString("Warnings\n")
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

	b.WriteString("Recent activity\n")
	if len(r.RecentActivity) == 0 {
		b.WriteString("  (none)\n")
	}
	for _, ev := range r.RecentActivity {
		date := ev.Date
		if len(date) >= 10 {
			date = date[:10]
		}
		fmt.Fprintf(&b, "  %s  %-16s  %-10s  %s\n", date, ev.Actor, ev.Verb, ev.Detail)
	}
	b.WriteByte('\n')

	b.WriteString("Health\n")
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
		for _, ev := range r.RecentActivity {
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
