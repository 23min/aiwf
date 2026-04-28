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
	OpenDecisions  []statusEntity     `json:"open_decisions"`
	OpenGaps       []statusGap        `json:"open_gaps"`
	RecentActivity []HistoryEvent     `json:"recent_activity"`
	Health         statusHealthCounts `json:"health"`
}

// statusEpic is one in-flight epic plus every milestone under it.
type statusEpic struct {
	ID         string            `json:"id"`
	Title      string            `json:"title"`
	Status     string            `json:"status"`
	Milestones []statusMilestone `json:"milestones"`
}

// statusMilestone is one milestone under an in-flight epic, with the
// in-progress one identifiable by Status.
type statusMilestone struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
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
	format := fs.String("format", "text", "output format: text or json")
	pretty := fs.Bool("pretty", false, "indent JSON output (only with --format=json)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintf(os.Stderr, "aiwf status: --format must be 'text' or 'json', got %q\n", *format)
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
		if e.Status != "active" {
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
			})
		}
		r.InFlightEpics = append(r.InFlightEpics, se)
	}

	// Open decisions: ADRs and Decision entities with status == "proposed".
	for _, e := range tr.ByKind(entity.KindADR) {
		if e.Status != "proposed" {
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
		if e.Status != "proposed" {
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
		if e.Status != "open" {
			continue
		}
		r.OpenGaps = append(r.OpenGaps, statusGap{
			ID:           e.ID,
			Title:        e.Title,
			DiscoveredIn: e.DiscoveredIn,
		})
	}
	sort.SliceStable(r.OpenGaps, func(i, j int) bool { return r.OpenGaps[i].ID < r.OpenGaps[j].ID })

	// Health: errors and warnings from a single check.Run.
	findings := check.Run(tr, loadErrs)
	for i := range findings {
		switch findings[i].Severity {
		case check.SeverityError:
			r.Health.Errors++
		case check.SeverityWarning:
			r.Health.Warnings++
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
		"--pretty=tformat:%H"+sep+"%aI"+sep+"%s"+sep+"%(trailers:key=aiwf-verb,valueonly=true,unfold=true)"+sep+"%(trailers:key=aiwf-actor,valueonly=true,unfold=true)\x1e",
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
		parts := strings.SplitN(rec, sep, 5)
		if len(parts) < 5 {
			continue
		}
		events = append(events, HistoryEvent{
			Commit: shortHash(parts[0]),
			Date:   parts[1],
			Detail: strings.TrimSpace(parts[2]),
			Verb:   strings.TrimSpace(parts[3]),
			Actor:  strings.TrimSpace(parts[4]),
		})
	}
	return events, nil
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
		fmt.Fprintf(&b, "  %s — %s    [%s]\n", e.ID, e.Title, e.Status)
		if len(e.Milestones) == 0 {
			b.WriteString("       (no milestones)\n")
		}
		for _, m := range e.Milestones {
			marker := "   "
			switch m.Status {
			case "in_progress":
				marker = " → "
			case "done":
				marker = " ✓ "
			}
			fmt.Fprintf(&b, "    %s%s — %s    [%s]\n", marker, m.ID, m.Title, m.Status)
		}
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
