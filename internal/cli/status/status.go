// Package status implements the `aiwf status` verb (per-verb subpackage of M-0116).
package status

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

	"github.com/23min/aiwf/internal/areagroup"
	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/history"
	"github.com/23min/aiwf/internal/cli/list"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/version"
)

// RecentActivityLimit is the number of recent commits surfaced by
// `aiwf status`'s "Recent activity" section. Five fits in a glance and
// answers "what changed lately?" without scrolling. For longer history,
// fall through to `aiwf history <id>`.
const RecentActivityLimit = 5

// StatusReport is the pure-data payload for `aiwf status`. The text and
// JSON renderers consume the same struct; BuildStatus produces it from
// a loaded tree. Lives alongside the CLI dispatcher rather than under
// internal/ because it is purely a presentational read view — adding a
// package boundary would be over-engineering for one verb.
type StatusReport struct {
	Date           string                 `json:"date"`
	InFlightEpics  []StatusEpic           `json:"in_flight_epics"`
	PlannedEpics   []StatusEpic           `json:"planned_epics"`
	OpenDecisions  []StatusEntity         `json:"open_decisions"`
	OpenGaps       []StatusGap            `json:"open_gaps"`
	Warnings       []StatusFinding        `json:"warnings"`
	RecentActivity []history.HistoryEvent `json:"recent_activity"`
	SweepPending   *StatusSweepPending    `json:"sweep_pending,omitempty"`
	Health         StatusHealthCounts     `json:"health"`
	// TodayDigest / ReleaseDigest carry the G-0380 activity digests
	// rendered by the markdown renderer's "Today's work" / "Since
	// last release" sections. Always populated by BuildActivityDigests
	// (never the zero value in a real report) — included here even
	// though only the md renderer displays them today, per this
	// package's one-source-of-truth convention for StatusReport.
	TodayDigest   ActivityDigest `json:"today_digest"`
	ReleaseDigest ActivityDigest `json:"release_digest"`
	// Worktrees is populated only when `--worktrees` is set (G-0122).
	// Always omitted from the JSON envelope when nil so the default
	// shape stays unchanged for existing JSON consumers.
	Worktrees []WorktreeView `json:"worktrees,omitempty"`

	// AreaMembers / AreaDefault carry the aiwf.yaml: areas config to the
	// text and markdown renderers so they group the epic sections by area
	// when an areas block exists (E-0043, M-0175). Render hints, not report
	// data — excluded from the JSON envelope (json:"-"); JSON consumers read
	// each epic's `area` field instead. Empty AreaMembers => flat rendering
	// (zero-migration; AC-6).
	AreaMembers []string `json:"-"`
	AreaDefault string   `json:"-"`
}

// StatusSweepPending is the tree-health one-liner for terminal-status
// entities still living in active directories. Per ADR-0004 §"Display
// surfaces": "The tree-health section gains a one-liner when sweep is
// pending: 'Sweep pending: N terminal entities not yet archived (run
// `aiwf archive --dry-run` to preview).' Hidden when 0."
//
// Populated from the `archive-sweep-pending` aggregate finding
// (M-0086); nil when the count is zero so the renderer can skip the
// section with a single nil-check. Lifted out of StatusReport.Warnings
// on purpose — the aggregate belongs in the tree-health section, not
// mixed in with body-empty / resolver-missing warnings.
type StatusSweepPending struct {
	Count   int    `json:"count"`
	Message string `json:"message"`
}

// StatusFinding is one warning surfaced inline in the status report.
// Mirrors the load-bearing fields of check.Finding without coupling the
// JSON shape to the validator package's internal schema.
type StatusFinding struct {
	Code     string `json:"code"`
	EntityID string `json:"entity_id,omitempty"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
}

// StatusEpic is one in-flight epic plus every milestone under it.
type StatusEpic struct {
	ID         string            `json:"id"`
	Title      string            `json:"title"`
	Status     string            `json:"status"`
	Area       string            `json:"area,omitempty"`
	Milestones []StatusMilestone `json:"milestones"`
}

// StatusMilestone is one milestone under an in-flight epic, with the
// in-progress one identifiable by Status. The TDD and ACs fields
// carry the I2 acceptance-criteria surface; ACs is omitted from JSON
// when the milestone carries none (zero progress).
type StatusMilestone struct {
	ID     string            `json:"id"`
	Title  string            `json:"title"`
	Status string            `json:"status"`
	TDD    string            `json:"tdd,omitempty"`
	ACs    *StatusACProgress `json:"acs,omitempty"`
	// WorktreeDivergence is set when a sibling epic-branch worktree
	// reports a different status for this milestone than the copy
	// loaded from the current checkout (G-0277) — nil when no sibling
	// epic worktree drives this milestone's epic, or its copy agrees.
	WorktreeDivergence *StatusWorktreeDivergence `json:"worktree_divergence,omitempty"`
}

// StatusWorktreeDivergence names the more current status a milestone
// carries on a sibling epic-branch worktree, when it disagrees with
// the status loaded from the current checkout (G-0277).
type StatusWorktreeDivergence struct {
	Status string `json:"status"`
	Label  string `json:"label"`
}

// StatusACProgress is the per-status count of a milestone's ACs.
// `Total` includes cancelled entries (they remain in the list per
// the position-stability rule); `InScope` excludes them, so that's
// the denominator the renderers use for "M/T met" progress.
type StatusACProgress struct {
	Total     int `json:"total"`
	InScope   int `json:"in_scope"`
	Open      int `json:"open"`
	Met       int `json:"met"`
	Deferred  int `json:"deferred"`
	Cancelled int `json:"cancelled"`
}

// SummarizeACs returns the per-status counts for a milestone's acs[].
// Returns nil when the slice is empty so the renderer can skip the
// "ACs: …" suffix entirely on milestones that don't carry any.
func SummarizeACs(acs []entity.AcceptanceCriterion) *StatusACProgress {
	if len(acs) == 0 {
		return nil
	}
	p := &StatusACProgress{Total: len(acs)}
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

// RenderACProgress formats the AC progress badge appended to a
// milestone row. Returns "" when there are no ACs (so the renderer
// can skip the separator). Format:
//
//	"ACs 2/3 met"           — typical case, in-scope total ≥ 1
//	"ACs 1/2 met (1 open)"  — when there are still open ACs
//	"ACs all cancelled"     — every AC was cancelled (in-scope = 0)
func RenderACProgress(p *StatusACProgress) string {
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

// StatusEntity is the shared shape for ADRs and decisions in the
// "Open decisions" section.
type StatusEntity struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Kind   string `json:"kind"`
}

// StatusGap is one open gap with the milestone or epic it was
// discovered in (if any).
type StatusGap struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	DiscoveredIn string `json:"discovered_in,omitempty"`
}

// StatusHealthCounts summarizes the tree's current validation state
// without re-running expensive checks; pulled from a single check.Run.
type StatusHealthCounts struct {
	Entities int `json:"entities"`
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
}

// ActivityDigest is a day- or release-scoped activity summary (G-0380):
// what accumulated in a commit range, bucketed into the three facts a
// reader most wants at a glance. RangeLabel supplies the section's own
// header text ("Today's work" / "Yesterday's work" / "Since last
// release (vX.Y.Z)" / "Since project start") so the renderer stays a
// pure function of the report rather than re-deriving the label.
type ActivityDigest struct {
	RangeLabel  string        `json:"range_label"`
	GapsOpened  []DigestEntry `json:"gaps_opened"`
	GapsClosed  []DigestEntry `json:"gaps_closed"`
	ADRsCreated []DigestEntry `json:"adrs_created"`
}

// DigestEntry is one entity line inside an ActivityDigest bucket.
// Title is resolved best-effort from the loaded tree — empty when the
// entity can't be found there (e.g. a cross-branch commit). Status
// means different things per bucket: for GapsClosed it's the gap's
// target status (the commit's aiwf-to trailer); for ADRsCreated it's
// the ADR's CURRENT status read from the tree (a commit's trailers
// never carry status); GapsOpened entries carry no status.
type DigestEntry struct {
	ID     string `json:"id"`
	Title  string `json:"title,omitempty"`
	Status string `json:"status,omitempty"`
}

// NewCmd builds `aiwf status`: a project-wide snapshot of in-flight
// work, open decisions, open gaps, and recent activity. Read-only;
// produces no commit. Use it to answer "what's next?", "where are we?",
// "what are we working on?". With --worktrees, swaps the output to a
// worktree-organized layout (G-0122): per-worktree section, full epic
// expansion for epic-branch worktrees, stale/trunk catch-alls.
func NewCmd() *cobra.Command {
	var (
		root      string
		format    string
		pretty    bool
		noTrunc   bool
		worktrees bool
		area      string
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
  aiwf status --no-trunc

  # Worktree-organized view (G-0122): per-worktree section with full
  # epic expansion when a worktree is on an epic branch
  aiwf status --worktrees`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(Run(root, format, area, pretty, noTrunc, worktrees))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root (default: discover via aiwf.yaml)")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text, json, or md")
	cmd.Flags().StringVar(&area, "area", "", "scope the snapshot to one workstream by effective area (E-0043)")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "indent JSON output (only with --format=json)")
	cmd.Flags().BoolVar(&noTrunc, "no-trunc", false, "do not truncate long titles when stdout is a terminal narrower than the row")
	cmd.Flags().BoolVar(&worktrees, "worktrees", false, "render worktree-organized layout: per-worktree section, epic expansion, stale/trunk catch-alls (G-0122)")
	_ = cmd.RegisterFlagCompletionFunc("format", cobra.FixedCompletions(
		[]string{"text", "json", "md"},
		cobra.ShellCompDirectiveNoFileComp,
	))
	_ = cmd.RegisterFlagCompletionFunc("area", cliutil.CompleteAreaFlag())
	return cmd
}

// Run executes `aiwf status`. Returns one of the cliutil.Exit* codes.
// When worktrees is true, the output switches to the G-0122 worktree-
// organized layout (text format only renders the worktree sections;
// json format adds a `worktrees` key to the result envelope; md
// format ignores the flag for now).
func Run(root, format, area string, pretty, noTrunc, worktrees bool) int {
	if format != "text" && format != "json" && format != "md" {
		cliutil.Errorf("aiwf status: --format must be 'text', 'json', or 'md', got %q\n", format)
		return cliutil.ExitUsage
	}

	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		cliutil.Errorf("aiwf status: %v\n", err)
		return cliutil.ExitUsage
	}

	// Advisory note when --area names an undeclared value (M-0174/AC-5);
	// to stderr so it never pollutes the (stdout) report.
	if note := cliutil.UndeclaredAreaNote(rootDir, area); note != "" {
		cliutil.Errorln(note)
	}

	ctx := context.Background()
	tr, loadErrs, err := tree.Load(ctx, rootDir)
	if err != nil {
		cliutil.Errorf("aiwf status: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}

	now := time.Now()
	report := BuildStatus(tr, loadErrs, now)
	// Scope the entity-derived sections to one workstream when --area is
	// set (M-0174/AC-2). Recent activity, warnings, and health stay
	// global — they are cross-cutting tree-health signals, not per-area
	// concepts. A no-op when area is empty.
	FilterStatusByArea(tr, &report, area)

	// Area-grouping config for the text/md renderers (M-0175). Empty
	// members => flat rendering (zero-migration). Skipped under --area:
	// a filter already narrows to one workstream, so grouping is moot
	// (filter and grouping are alternative views, not combined).
	if area == "" {
		report.AreaMembers, report.AreaDefault = cliutil.ConfiguredAreas(rootDir)
	}

	recent, err := ReadRecentActivity(ctx, rootDir, RecentActivityLimit)
	if err != nil {
		cliutil.Errorf("aiwf status: reading recent activity: %v\n", err)
		return cliutil.ExitInternal
	}
	report.RecentActivity = recent

	// G-0380: the day-scoped ("today"/"yesterday") and release-scoped
	// activity digests behind STATUS.md's "Today's work" / "Since last
	// release" sections. Computed unconditionally (like RecentActivity
	// and Health) — these are cross-cutting tree-health signals, not
	// per-area concepts, so --area never scopes them (FilterStatusByArea
	// above already left them untouched).
	today, release, digestErr := BuildActivityDigests(ctx, rootDir, tr, now)
	if digestErr != nil {
		//coverage:ignore mirrors the (also untested) ReadRecentActivity
		// error branch just above: BuildActivityDigests only fails for
		// a genuine git/environmental fault, not reachable through a
		// clean deterministic fixture — see its own package-level notes.
		cliutil.Errorf("aiwf status: building activity digest: %v\n", digestErr)
		return cliutil.ExitInternal
	}
	report.TodayDigest = today
	report.ReleaseDigest = release

	// G-0122: populate Worktrees on every status call (regardless of
	// --worktrees flag) so default text + JSON output surface the
	// worktree summaries. The Worktrees field is `omitempty`-tagged
	// so single-worktree projects see no JSON delta. --worktrees
	// remains meaningful as "swap text output to the full
	// worktree-organized layout."
	views, vErr := BuildWorktreeViews(ctx, rootDir, tr)
	if vErr != nil {
		cliutil.Errorf("aiwf status: building worktree view: %v\n", vErr)
		return cliutil.ExitInternal
	}
	report.Worktrees = views
	// G-0277: the default view otherwise presents each milestone's
	// current-checkout status as if it were authoritative even when a
	// sibling epic-branch worktree already carries a more current one.
	AnnotateWorktreeDivergence(&report, views, rootDir)
	if worktrees && format == "text" {
		if rErr := RenderWorktreeViews(os.Stdout, views, render.ColorEnabled(os.Stdout)); rErr != nil {
			cliutil.Errorf("aiwf status: writing output: %v\n", rErr)
			return cliutil.ExitInternal
		}
		return cliutil.ExitOK
	}

	switch format {
	case "text":
		termWidth := 0
		if !noTrunc {
			termWidth = render.TerminalWidth(os.Stdout)
		}
		if err := RenderStatusText(os.Stdout, &report, termWidth, render.ColorEnabled(os.Stdout)); err != nil {
			cliutil.Errorf("aiwf status: writing output: %v\n", err)
			return cliutil.ExitInternal
		}
	case "json":
		env := render.Envelope{
			Tool:    "aiwf",
			Version: version.Current().Version,
			Status:  "ok",
			Result:  &report,
			Metadata: map[string]any{
				"root":     rootDir,
				"entities": report.Health.Entities,
			},
		}
		if err := render.JSON(os.Stdout, env, pretty); err != nil {
			cliutil.Errorf("aiwf status: writing output: %v\n", err)
			return cliutil.ExitInternal
		}
	case "md":
		if err := RenderStatusMarkdown(os.Stdout, &report); err != nil {
			cliutil.Errorf("aiwf status: writing output: %v\n", err)
			return cliutil.ExitInternal
		}
	}
	return cliutil.ExitOK
}

// BuildStatus returns the project status payload for tree tr, with
// loadErrs surfaced as part of the health counts, stamped with the
// supplied now. The caller acquires the clock at the edge and passes it
// in, so the report is a pure function of its inputs — StatusReport.Date
// is deterministic under test. The returned report has RecentActivity
// unset; the caller fills it via ReadRecentActivity, which needs git
// access.
func BuildStatus(tr *tree.Tree, loadErrs []tree.LoadError, now time.Time) StatusReport {
	r := StatusReport{
		Date: now.UTC().Format("2006-01-02"),
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
		se := StatusEpic{
			ID:     canonEpic,
			Title:  e.Title,
			Status: e.Status,
			Area:   tr.ResolvedArea(e),
		}
		for _, m := range milestonesByParent[canonEpic] {
			se.Milestones = append(se.Milestones, StatusMilestone{
				ID:     entity.Canonicalize(m.ID),
				Title:  m.Title,
				Status: m.Status,
				TDD:    m.TDD,
				ACs:    SummarizeACs(m.ACs),
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
		r.OpenDecisions = append(r.OpenDecisions, StatusEntity{
			ID:     entity.Canonicalize(e.ID),
			Title:  e.Title,
			Status: e.Status,
			Kind:   string(entity.KindADR),
		})
	}
	for _, e := range tr.FilterByKindStatuses(entity.KindDecision, entity.StatusProposed) {
		r.OpenDecisions = append(r.OpenDecisions, StatusEntity{
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
		r.OpenGaps = append(r.OpenGaps, StatusGap{
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
			if findings[i].Code == check.CodeArchiveSweepPending {
				r.SweepPending = ParseSweepPending(findings[i].Message)
				continue
			}
			r.Warnings = append(r.Warnings, StatusFinding{
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

// FilterStatusByArea scopes a status report's entity-derived sections to
// a single workstream (E-0043, M-0174/AC-2): in-flight epics, planned
// epics, open decisions, and open gaps are kept only when their effective
// area (tree.ResolvedAreaByID — root kinds by their own field, epics
// carrying their derived milestones) equals area. Untagged entities
// (effective area "") never match a named area, so they drop out (AC-6).
//
// A no-op when area is empty. Recent activity, warnings, health, the
// sweep-pending one-liner, and the worktree views are left untouched —
// they are cross-cutting tree-health signals, not per-area concepts.
// Mutates r in place; the caller (Run) passes &report, and the render
// resolver never calls this (it wants the full report).
func FilterStatusByArea(tr *tree.Tree, r *StatusReport, area string) {
	if area == "" {
		return
	}
	r.InFlightEpics = keepEpicsInArea(tr, r.InFlightEpics, area)
	r.PlannedEpics = keepEpicsInArea(tr, r.PlannedEpics, area)

	decisions := r.OpenDecisions[:0:0]
	for _, d := range r.OpenDecisions {
		if tr.ResolvedAreaByID(d.ID) == area {
			decisions = append(decisions, d)
		}
	}
	r.OpenDecisions = decisions

	gaps := r.OpenGaps[:0:0]
	for _, g := range r.OpenGaps {
		if tr.ResolvedAreaByID(g.ID) == area {
			gaps = append(gaps, g)
		}
	}
	r.OpenGaps = gaps
}

// keepEpicsInArea returns the epics whose effective area equals area.
// Milestones ride along with their kept epic — an epic's area is the
// single source for its milestones' derived area, so no per-milestone
// filtering is needed.
func keepEpicsInArea(tr *tree.Tree, epics []StatusEpic, area string) []StatusEpic {
	kept := epics[:0:0]
	for _, e := range epics {
		if tr.ResolvedAreaByID(e.ID) == area {
			kept = append(kept, e)
		}
	}
	return kept
}

// ParseSweepPending extracts the count from an `archive-sweep-pending`
// finding message and packages it into a StatusSweepPending. The
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
func ParseSweepPending(message string) *StatusSweepPending {
	var count int
	if _, err := fmt.Sscanf(message, "%d", &count); err != nil || count <= 0 {
		return nil
	}
	return &StatusSweepPending{
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

// ReadRecentActivity returns the last `limit` commits whose message
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
func ReadRecentActivity(ctx context.Context, root string, limit int) ([]history.HistoryEvent, error) {
	if !cliutil.HasCommits(ctx, root) {
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
	cmd := exec.CommandContext(
		ctx, "git", "log",
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

	var events []history.HistoryEvent
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
		events = append(events, history.HistoryEvent{
			Commit: history.ShortHash(parts[0]),
			Date:   parts[1],
			Detail: strings.TrimSpace(parts[2]),
			Verb:   verb,
			Actor:  strings.TrimSpace(parts[4]),
			Body:   history.StripTrailers(strings.TrimSpace(parts[5])),
		})
		if len(events) >= limit {
			break
		}
	}
	return events, nil
}

// digestCommit is one parsed commit record feeding the G-0380 activity
// digests: the trailer set BuildActivityDigests needs (verb, entity,
// promote target) plus the author date used for day-bucketing. A
// sibling of history.HistoryEvent rather than a shared type — the two
// digest sections need aiwf-entity/aiwf-to, which ReadRecentActivity's
// consumers (the "Recent activity" table) have no use for.
type digestCommit struct {
	AuthorDate string // %aI (strict ISO 8601, RFC3339-compatible)
	Verb       string
	EntityID   string
	To         string
}

// readDigestCommits runs `git log` with the trailer set the G-0380
// digests need (aiwf-verb, aiwf-entity, aiwf-to) plus the author date,
// following the same tformat + \x1f/\x1e separator convention and
// prose-mention guard as ReadRecentActivity. extraArgs is appended
// verbatim after the fixed options — a revision range ("<tag>..HEAD"),
// a "--since=<bound>" pre-filter, or nothing (full history).
//
// Returns (nil, nil) on a repo with no commits yet, matching
// ReadRecentActivity — a fresh repo is a legitimate state, not a fault.
func readDigestCommits(ctx context.Context, root string, extraArgs ...string) ([]digestCommit, error) {
	if !cliutil.HasCommits(ctx, root) {
		return nil, nil
	}
	const sep = "\x1f"
	const recSep = "\x1e\n"
	args := []string{
		"log",
		"--grep", "^aiwf-verb: ",
		"--pretty=tformat:%aI" + sep +
			"%(trailers:key=aiwf-verb,valueonly=true,unfold=true)" + sep +
			"%(trailers:key=aiwf-entity,valueonly=true,unfold=true)" + sep +
			"%(trailers:key=aiwf-to,valueonly=true,unfold=true)\x1e",
	}
	args = append(args, extraArgs...)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git log: %w\n%s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		// cmd.Output() failing with something other than an
		// *exec.ExitError (e.g. the git binary itself missing) can't be
		// exercised without breaking the identical HasCommits guard
		// above first, which would short-circuit to the nil-nil return
		// instead; mirrors ReadRecentActivity's identically-untested
		// fallback.
		return nil, fmt.Errorf("git log: %w", err) //coverage:ignore non-ExitError git-log failure unreachable without breaking the HasCommits guard above (see comment)
	}

	var commits []digestCommit
	for _, rec := range strings.Split(string(out), recSep) {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, sep, 4)
		if len(parts) < 4 {
			//coverage:ignore the tformat string above bakes in exactly 3
			// literal separator bytes per record regardless of trailer
			// content, so strings.SplitN always yields 4 parts for any
			// record git itself produced; unreachable without corrupted
			// git output.
			continue
		}
		verb := strings.TrimSpace(parts[1])
		if verb == "" {
			// Prose-mention false-positive — see ReadRecentActivity's
			// identical guard (G30).
			continue
		}
		commits = append(commits, digestCommit{
			AuthorDate: strings.TrimSpace(parts[0]),
			Verb:       verb,
			EntityID:   strings.TrimSpace(parts[2]),
			To:         strings.TrimSpace(parts[3]),
		})
	}
	return commits, nil
}

// latestReleaseTag returns the most recent "vX.Y.Z"-shaped tag
// reachable from HEAD, or "" when none exists — a brand-new repo that
// hasn't cut a release yet, which BuildActivityDigests renders as
// "Since project start" rather than a release-scoped range.
//
// Best-effort: `git describe` exiting non-zero (no matching tag
// reachable from HEAD, or no commits at all) yields "" rather than an
// error — an absent release tag is a legitimate repo state, not a
// fault worth surfacing.
func latestReleaseTag(ctx context.Context, root string) string {
	if !cliutil.HasCommits(ctx, root) {
		return ""
	}
	cmd := exec.CommandContext(ctx, "git", "describe", "--tags", "--abbrev=0", "--match", "v*")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// BuildActivityDigests computes the two G-0380 activity digests: a
// day-scoped digest (today's work, falling back to yesterday's — see
// buildDayDigest) and a release-scoped digest (since the latest v*
// release tag, or since project start when none exists). ctx and root
// drive the git log reads; tr resolves each digest entry's title and
// (for ADRs) current status. now is the caller's injected clock — the
// same no-time.Now-in-core convention BuildStatus follows.
func BuildActivityDigests(ctx context.Context, root string, tr *tree.Tree, now time.Time) (today, release ActivityDigest, err error) {
	today, err = buildDayDigest(ctx, root, tr, now)
	if err != nil {
		//coverage:ignore buildDayDigest's own git-log invocation only
		// fails for genuine environmental faults (see readDigestCommits'
		// identical note); not reachable through a clean deterministic
		// fixture.
		return ActivityDigest{}, ActivityDigest{}, err
	}
	release, err = buildReleaseDigest(ctx, root, tr)
	if err != nil {
		//coverage:ignore mirrors the today-digest branch immediately
		// above — buildReleaseDigest's git-log invocation uses
		// internally-derived, always-well-formed args.
		return ActivityDigest{}, ActivityDigest{}, err
	}
	return today, release, nil
}

// buildDayDigest returns today's activity digest, or yesterday's when
// today has zero qualifying commits (one of the three G-0380 facts) —
// a single hop back, never further (per the design, a slow day
// doesn't cascade into a week-old digest). "Today"/"yesterday" are UTC
// calendar days, matching the convention BuildStatus's own Date field
// uses (now.UTC().Format("2006-01-02")).
func buildDayDigest(ctx context.Context, root string, tr *tree.Tree, now time.Time) (ActivityDigest, error) {
	nowUTC := now.UTC()
	todayStr := nowUTC.Format("2006-01-02")
	yesterdayStr := nowUTC.AddDate(0, 0, -1).Format("2006-01-02")

	// The --since bound is an I/O-narrowing pre-filter only (mirrors
	// ReadRecentActivity's --grep pre-filter doc comment) — a 3-day
	// margin absorbs any committer-vs-author-date skew git's --since
	// evaluates against; the exact UTC-day match happens in Go below,
	// so a loose bound here cannot produce a wrong answer, only extra
	// rows to filter out.
	bound := nowUTC.AddDate(0, 0, -3).Format("2006-01-02") + "T00:00:00Z"
	commits, err := readDigestCommits(ctx, root, "--since="+bound)
	if err != nil {
		//coverage:ignore bound is derived from a valid time.Time via
		// time.Format, so this is never a malformed argument in
		// practice — see readDigestCommits' own note on its
		// non-ExitError fallback.
		return ActivityDigest{}, err
	}

	todayDigest := digestFromCommits(commits, tr, todayStr, "Today's work")
	if !isEmptyDigest(todayDigest) {
		return todayDigest, nil
	}
	return digestFromCommits(commits, tr, yesterdayStr, "Yesterday's work"), nil
}

// buildReleaseDigest returns the digest scoped to every commit since
// the latest v*-shaped release tag (exclusive), or the full history
// when no such tag exists yet.
func buildReleaseDigest(ctx context.Context, root string, tr *tree.Tree) (ActivityDigest, error) {
	tag := latestReleaseTag(ctx, root)
	label := "Since project start"
	var commits []digestCommit
	var err error
	if tag == "" {
		commits, err = readDigestCommits(ctx, root)
	} else {
		label = fmt.Sprintf("Since last release (%s)", tag)
		commits, err = readDigestCommits(ctx, root, tag+"..HEAD")
	}
	if err != nil {
		//coverage:ignore tag is either "" (the branch above, no range
		// arg) or a real tag name `git describe` itself reported —
		// git ref names can't contain "..", so `tag+"..HEAD"` is
		// always a syntactically valid range.
		return ActivityDigest{}, err
	}
	d := bucketDigestCommits(commits, tr)
	d.RangeLabel = label
	return d, nil
}

// digestFromCommits narrows commits to those whose author-date's UTC
// calendar day equals dayStr, then buckets the result and labels it
// rangeLabel. A commit whose author-date fails to parse is skipped
// defensively — %aI always emits a valid ISO-8601 timestamp for a
// real commit, so this is not expected to fire in practice.
func digestFromCommits(commits []digestCommit, tr *tree.Tree, dayStr, rangeLabel string) ActivityDigest {
	var scoped []digestCommit
	for _, c := range commits {
		t, err := time.Parse(time.RFC3339, c.AuthorDate)
		if err != nil {
			continue
		}
		if t.UTC().Format("2006-01-02") == dayStr {
			scoped = append(scoped, c)
		}
	}
	d := bucketDigestCommits(scoped, tr)
	d.RangeLabel = rangeLabel
	return d
}

// bucketDigestCommits classifies commits into the three G-0380 digest
// facts: gaps opened (aiwf-verb: add + Gap kind), gaps closed
// (aiwf-verb: promote + Gap kind + a non-"open" aiwf-to — a promote
// that (re)opens a gap, if that's ever legal, does not count as
// closing it), and ADRs created (aiwf-verb: add + ADR kind, annotated
// with the ADR's CURRENT tree status). A composite id (e.g. an AC
// promote's `M-NNN/AC-N`) resolves to its parent's kind via
// entity.KindFromID's own composite handling, so it falls through as
// "milestone" and is silently outside this function's scope, same as
// any other verb/kind combination G-0380's digest doesn't track.
func bucketDigestCommits(commits []digestCommit, tr *tree.Tree) ActivityDigest {
	var d ActivityDigest
	for _, c := range commits {
		kind, ok := entity.KindFromID(c.EntityID)
		if !ok {
			continue
		}
		switch {
		case c.Verb == "add" && kind == entity.KindGap:
			d.GapsOpened = append(d.GapsOpened, digestEntry(tr, c.EntityID, ""))
		case c.Verb == "promote" && kind == entity.KindGap && c.To != "" && c.To != entity.StatusOpen:
			d.GapsClosed = append(d.GapsClosed, digestEntry(tr, c.EntityID, c.To))
		case c.Verb == "add" && kind == entity.KindADR:
			d.ADRsCreated = append(d.ADRsCreated, digestEntry(tr, c.EntityID, currentADRStatus(tr, c.EntityID)))
		}
	}
	return d
}

// digestEntry builds one DigestEntry for id, resolving its title from
// tr — best-effort; left empty when tr can't resolve id (e.g. a
// cross-branch entity not present in this tree). status is
// caller-supplied since its meaning varies by bucket (see DigestEntry).
func digestEntry(tr *tree.Tree, id, status string) DigestEntry {
	entry := DigestEntry{ID: entity.Canonicalize(id), Status: status}
	if e := tr.ByID(id); e != nil {
		entry.Title = e.Title
	}
	return entry
}

// currentADRStatus reads an ADR's live status from the loaded tree.
// Per G-0380's design, "ADRs created" shows the CURRENT status, not
// anything a commit's trailers carry (an add commit has no aiwf-to).
// Empty when tr can't resolve id.
func currentADRStatus(tr *tree.Tree, id string) string {
	if e := tr.ByID(id); e != nil {
		return e.Status
	}
	return ""
}

// isEmptyDigest reports whether every bucket in d is empty — the
// signal buildDayDigest uses to decide whether today's digest falls
// back to yesterday's.
func isEmptyDigest(d ActivityDigest) bool {
	return len(d.GapsOpened) == 0 && len(d.GapsClosed) == 0 && len(d.ADRsCreated) == 0
}

// statusEpicArea is the area accessor the grouping helper uses; an epic
// carries its effective area (populated in BuildStatus).
func statusEpicArea(e StatusEpic) string { return e.Area }

// writeStatusEpicsText renders a section's epics, grouped per area when
// members is non-empty (E-0043, M-0175/AC-2), else flat — today's output
// (AC-6). In grouped mode each area is a bold subheading; a declared area
// with no epics is suppressed and the untagged complement is always shown
// (AC-5, via areagroup.Partition). emptyMsg is the flat-mode "(none)"-style
// line shown when the whole section is empty.
func writeStatusEpicsText(b *strings.Builder, epics []StatusEpic, members []string, defaultLabel string, termWidth int, colorEnabled bool, emptyMsg string) {
	if len(members) == 0 {
		if len(epics) == 0 {
			b.WriteString("  " + emptyMsg + "\n")
			return
		}
		for _, e := range epics {
			WriteStatusEpicText(b, e, termWidth)
		}
		return
	}
	for _, g := range areagroup.Partition(epics, statusEpicArea, members, defaultLabel) {
		// "▸ " marks the line as an area heading independently of color, so
		// the grouping survives piped / no-color output (the status palette
		// treats glyphs as content, not style — G-0080); the epic rows below
		// carry their own "E-NNNN … [status]" shape.
		b.WriteString("  " + render.Bold("▸ "+g.Label, colorEnabled) + "\n")
		if len(g.Items) == 0 {
			b.WriteString("    (none)\n")
			continue
		}
		for _, e := range g.Items {
			WriteStatusEpicText(b, e, termWidth)
		}
	}
}

// WriteStatusEpicText writes one epic plus its milestones in the
// terminal-friendly shape shared by the In flight and Roadmap sections.
// termWidth caps title widths so long titles don't wrap into the next
// row's column-zero (the G-0080 visual-scan bug); 0 disables truncation.
//
// Milestone rows lead with a glyph from the G-0080 palette so every
// row carries a visible state marker — the in-progress and done
// glyphs (→ ✓) have always been present; this function also emits ○
// for draft and ✗ for cancelled so the four-glyph palette is uniformly
// applied across all milestone states.
func WriteStatusEpicText(b *strings.Builder, e StatusEpic, termWidth int) {
	epicPrefix := fmt.Sprintf("  %s — ", e.ID)
	epicTail := fmt.Sprintf("    [%s]", e.Status)
	epicTitle := TruncStatusTitle(e.Title, termWidth, epicPrefix, epicTail)
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
		if progress := RenderACProgress(m.ACs); progress != "" {
			suffix = "    · " + progress
		}
		if m.TDD != "" {
			suffix += "    · tdd: " + m.TDD
		}
		if m.WorktreeDivergence != nil {
			suffix += fmt.Sprintf("    · %s on %s", m.WorktreeDivergence.Status, m.WorktreeDivergence.Label)
		}
		msPrefix := fmt.Sprintf("    %s%s — ", marker, m.ID)
		msTail := fmt.Sprintf("    [%s]%s", m.Status, suffix)
		msTitle := TruncStatusTitle(m.Title, termWidth, msPrefix, msTail)
		fmt.Fprintf(b, "%s%s%s\n", msPrefix, msTitle, msTail)
	}
}

// TruncStatusTitle caps title to fit termWidth given the non-title
// prefix/tail of the line. Returns title unchanged when termWidth is 0
// (no TTY / --no-trunc) or when the available room would fall below
// the per-G-0080 minimum useful column width. The minimum (10 runes)
// is the same floor renderListRowsText uses — see list.MinTitleColumnRunes.
func TruncStatusTitle(title string, termWidth int, prefix, tail string) string {
	if termWidth <= 0 {
		return title
	}
	other := utf8.RuneCountInString(prefix) + utf8.RuneCountInString(tail)
	avail := termWidth - other
	if avail < list.MinTitleColumnRunes {
		return title
	}
	return render.Truncate(title, avail)
}

// RenderStatusText writes the human-readable status report to w. The
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
func RenderStatusText(w io.Writer, r *StatusReport, termWidth int, colorEnabled bool) error {
	var b strings.Builder
	fmt.Fprintf(&b, "aiwf status — %s\n\n", r.Date)

	b.WriteString(render.Bold("In flight", colorEnabled) + "\n")
	writeStatusEpicsText(&b, r.InFlightEpics, r.AreaMembers, r.AreaDefault, termWidth, colorEnabled, "(no active epics)")

	// Worktrees: one-line-per-worktree short view, placed directly
	// under "In flight" since both sections answer "what's being
	// worked on now." Shown only when ≥2 worktrees exist (single-
	// worktree projects don't need it). G-0122 option 1.
	if len(r.Worktrees) >= 2 {
		b.WriteByte('\n')
		b.WriteString(render.Bold("Worktrees", colorEnabled) + "\n")
		renderWorktreeShortLines(&b, r.Worktrees, termWidth, colorEnabled)
		b.WriteString(render.Dim("  for the full per-worktree view: aiwf status --worktrees", colorEnabled))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')

	b.WriteString(render.Bold("Roadmap", colorEnabled) + "\n")
	writeStatusEpicsText(&b, r.PlannedEpics, r.AreaMembers, r.AreaDefault, termWidth, colorEnabled, "(nothing planned)")
	b.WriteByte('\n')

	b.WriteString(render.Bold("Open decisions", colorEnabled) + "\n")
	if len(r.OpenDecisions) == 0 {
		b.WriteString("  (none)\n")
	}
	for _, d := range r.OpenDecisions {
		prefix := fmt.Sprintf("  %s — ", d.ID)
		tail := fmt.Sprintf("    [%s]", d.Status)
		title := TruncStatusTitle(d.Title, termWidth, prefix, tail)
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
		title := TruncStatusTitle(g.Title, termWidth, prefix, tail)
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

// RenderStatusMarkdown writes the status report as a self-contained
// markdown document, with mermaid `flowchart` blocks for in-flight and
// roadmap epics. The output renders unchanged in any markdown viewer
// that supports mermaid (GitHub web, VSCode, Obsidian, glow + mermaid
// extension, etc.). Plain markdown — no HTML, no JS.
func RenderStatusMarkdown(w io.Writer, r *StatusReport) error {
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

	writeActivityDigestMarkdown(&b, r.TodayDigest)
	writeActivityDigestMarkdown(&b, r.ReleaseDigest)

	b.WriteString("## In flight\n\n")
	writeStatusEpicsMarkdown(&b, r.InFlightEpics, r.AreaMembers, r.AreaDefault, "_(no active epics)_")

	b.WriteString("## Roadmap\n\n")
	writeStatusEpicsMarkdown(&b, r.PlannedEpics, r.AreaMembers, r.AreaDefault, "_(nothing planned)_")

	b.WriteString("## Open decisions\n\n")
	if len(r.OpenDecisions) == 0 {
		b.WriteString("_(none)_\n\n")
	} else {
		b.WriteString("| ID | Kind | Title | Status |\n")
		b.WriteString("|----|------|-------|--------|\n")
		for _, d := range r.OpenDecisions {
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
				d.ID, d.Kind, MdEscape(d.Title), d.Status)
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
				g.ID, MdEscape(g.Title), g.DiscoveredIn)
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
				ww.Code, ww.EntityID, ww.Path, MdEscape(ww.Message))
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
				date, ev.Actor, ev.Verb, MdEscape(ev.Detail))
		}
		b.WriteByte('\n')
	}

	_, err := io.WriteString(w, b.String())
	return err
}

// writeActivityDigestMarkdown renders one G-0380 activity-digest
// section: an h2 header carrying d's own RangeLabel ("Today's work" /
// "Yesterday's work" / "Since last release (vX.Y.Z)" / "Since project
// start" — computed once in BuildActivityDigests so the renderer
// never re-derives it), followed by the three digest facts as
// bold-labelled bullet sub-lists.
func writeActivityDigestMarkdown(b *strings.Builder, d ActivityDigest) {
	fmt.Fprintf(b, "## %s\n\n", d.RangeLabel)
	writeDigestBucketMarkdown(b, "Gaps opened", d.GapsOpened)
	writeDigestBucketMarkdown(b, "Gaps closed", d.GapsClosed)
	writeDigestBucketMarkdown(b, "ADRs created", d.ADRsCreated)
}

// writeDigestBucketMarkdown renders one digest fact as a bold label
// plus a bullet per entry ("- ID — Title _(status)_"); an empty
// bucket renders the file's usual "_(none)_" placeholder, matching
// writeStatusEpicsMarkdown's empty-state convention. Title and status
// are each omitted from the bullet when absent (a best-effort-
// unresolved title, or a bucket with no status concept), so a
// same-tree entry with a real title never renders a dangling "— ").
func writeDigestBucketMarkdown(b *strings.Builder, label string, entries []DigestEntry) {
	fmt.Fprintf(b, "**%s**\n\n", label)
	if len(entries) == 0 {
		b.WriteString("_(none)_\n\n")
		return
	}
	for _, e := range entries {
		line := "- " + e.ID
		if e.Title != "" {
			line += " — " + MdEscape(e.Title)
		}
		if e.Status != "" {
			line += fmt.Sprintf(" _(%s)_", e.Status)
		}
		b.WriteString(line + "\n")
	}
	b.WriteByte('\n')
}

// writeStatusEpicsMarkdown renders a section's epics, grouped per area
// when members is non-empty (E-0043, M-0175/AC-2), else flat — today's
// output (AC-6). Grouped mode emits a bold area label before each group
// (keeping the h2→h3 epic hierarchy intact); a declared area with no epics
// is suppressed and the complement is always shown (AC-5). emptyMsg is the
// flat-mode placeholder shown when the section has no epics.
func writeStatusEpicsMarkdown(b *strings.Builder, epics []StatusEpic, members []string, defaultLabel, emptyMsg string) {
	if len(members) == 0 {
		if len(epics) == 0 {
			b.WriteString(emptyMsg + "\n\n")
		}
		for _, e := range epics {
			WriteStatusEpicMarkdown(b, e)
		}
		return
	}
	for _, g := range areagroup.Partition(epics, statusEpicArea, members, defaultLabel) {
		fmt.Fprintf(b, "**%s**\n\n", MdEscape(g.Label))
		if len(g.Items) == 0 {
			b.WriteString("_(none)_\n\n")
			continue
		}
		for _, e := range g.Items {
			WriteStatusEpicMarkdown(b, e)
		}
	}
}

// WriteStatusEpicMarkdown writes one epic — header, milestone list, and
// a mermaid `flowchart LR` keyed by milestone status — into b. Empty
// milestone lists render an explicit "(no milestones)" line so the
// section stays visually balanced and the diagram is omitted (mermaid
// barfs on a flowchart with one node and no edges).
func WriteStatusEpicMarkdown(b *strings.Builder, e StatusEpic) {
	fmt.Fprintf(b, "### %s — %s _(%s)_\n\n", e.ID, MdEscape(e.Title), e.Status)
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
		if progress := RenderACProgress(m.ACs); progress != "" {
			suffix = " — " + progress
		}
		if m.TDD != "" {
			suffix += " — tdd: " + m.TDD
		}
		if m.WorktreeDivergence != nil {
			suffix += fmt.Sprintf(" — %s on %s", m.WorktreeDivergence.Status, m.WorktreeDivergence.Label)
		}
		fmt.Fprintf(b, "- %s**%s** — %s _(%s)_%s\n", marker, m.ID, MdEscape(m.Title), m.Status, suffix)
	}
	b.WriteByte('\n')

	b.WriteString("```mermaid\nflowchart LR\n")
	fmt.Fprintf(b, "  %s[\"%s<br/>%s\"]:::epic_%s\n",
		mermaidID(e.ID), e.ID, MdEscape(e.Title), e.Status)
	for _, m := range e.Milestones {
		// Append "(M/T)" badge to the mermaid label when the milestone
		// has any in-scope ACs. Cancelled-only milestones get no badge
		// (the design's "all cancelled" case isn't useful at a glance).
		acBadge := ""
		if m.ACs != nil && m.ACs.InScope > 0 {
			acBadge = fmt.Sprintf(" (%d/%d)", m.ACs.Met, m.ACs.InScope)
		}
		fmt.Fprintf(b, "  %s[\"%s%s<br/>%s\"]:::ms_%s\n",
			mermaidID(m.ID), m.ID, acBadge, MdEscape(m.Title), m.Status)
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

// MdEscape escapes the four characters that break a markdown table row
// or a mermaid label: pipe, backtick, the bracket pair (mermaid uses
// them for node syntax), and the double-quote (mermaid uses it to
// delimit labels). Newlines are stripped so a single field stays on
// one row. Conservative; not a full markdown sanitizer.
func MdEscape(s string) string {
	r := strings.NewReplacer(
		"|", "\\|",
		"`", "\\`",
		"\"", "'",
		"\n", " ",
		"\r", " ",
	)
	return r.Replace(s)
}
