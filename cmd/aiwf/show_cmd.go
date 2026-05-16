package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/tree"
)

// readEntityBody reads the entity file at root/relPath and returns the
// body bytes (the prose after the closing `---`). Errors are
// swallowed — `aiwf show` already emits findings for unreadable /
// malformed entities via the load-error finding; surfacing the same
// problem on the body field would double-count. Empty body or missing
// file produces nil.
//
// Entity.Path is repo-relative (the loader normalizes it that way) so
// callers must join with root before hitting the filesystem; doing
// the join in this helper keeps each caller from re-deriving it.
func readEntityBody(root, relPath string) []byte {
	if relPath == "" {
		return nil
	}
	abs := relPath
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(root, relPath)
	}
	content, err := os.ReadFile(abs)
	if err != nil {
		return nil
	}
	_, body, ok := entity.Split(content)
	if !ok {
		return nil
	}
	return body
}

// newShowCmd builds `aiwf show <id>`. Aggregates per-entity state from
// the existing data sources — frontmatter (entity), git log (history),
// aiwf check (findings) — into one human-readable view (or one JSON
// envelope when --format=json). No new state; pure projection.
//
// For composite ids (M-NNN/AC-N), renders just the AC's slice of the
// parent milestone plus its history.
func newShowCmd() *cobra.Command {
	var (
		root         string
		format       string
		pretty       bool
		historyLimit int
	)
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Aggregate per-entity view: frontmatter, ACs, history, findings, referenced_by",
		Example: `  # Aggregate view of an epic
  aiwf show E-01

  # JSON envelope (carries body + per-AC descriptions on milestones)
  aiwf show M-007 --format=json --pretty

  # Composite id: just the AC slice of its parent milestone
  aiwf show M-007/AC-1`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(runShowCmd(args[0], root, format, pretty, historyLimit))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "indent JSON output (only with --format=json)")
	cmd.Flags().IntVar(&historyLimit, "history", 10, "max recent history events to render (0 = none, -1 = all)")
	registerFormatCompletion(cmd)
	cmd.ValidArgsFunction = completeEntityIDArg("", 0)
	return cmd
}

func runShowCmd(id, root, format string, pretty bool, historyLimit int) int {
	if format != "text" && format != "json" {
		fmt.Fprintf(os.Stderr, "aiwf show: --format must be text or json, got %q\n", format)
		return cliutil.ExitUsage
	}

	rootDir, err := resolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf show: %v\n", err)
		return cliutil.ExitUsage
	}

	ctx := context.Background()
	tr, loadErrs, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf show: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}

	view, ok := buildShowView(ctx, rootDir, tr, loadErrs, id, historyLimit)
	if !ok {
		fmt.Fprintf(os.Stderr, "aiwf show: %s not found\n", id)
		return cliutil.ExitUsage
	}

	switch format {
	case "text":
		renderShowText(view)
	case "json":
		env := render.Envelope{
			Tool:    "aiwf",
			Version: Version,
			Status:  "ok",
			Result:  view,
			Metadata: map[string]any{
				"root": rootDir,
				"id":   id,
			},
		}
		if err := render.JSON(os.Stdout, env, pretty); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf show: %v\n", err)
			return cliutil.ExitInternal
		}
	}
	return cliutil.ExitOK
}

// ShowView is the aggregated per-entity state. Exported for the JSON
// envelope. Field-set varies by what kind of id was queried; absent
// fields render as empty / omitted in JSON via omitempty.
//
// ReferencedBy is the inversion of the reference graph — every entity
// id that names this one as a target. Always emitted in JSON (zero-
// value `[]`) so downstream consumers never have to check for field
// presence; populated from tree.Tree.ReverseRefs at view-build time.
// For composite ids (M-NNN/AC-N), this lists referrers of the AC
// specifically; the parent milestone's referrers are not rolled in
// (use `aiwf show M-NNN` for that).
type ShowView struct {
	ID           string            `json:"id"`
	Kind         string            `json:"kind"`
	Title        string            `json:"title"`
	Status       string            `json:"status"`
	Path         string            `json:"path,omitempty"`
	Parent       string            `json:"parent,omitempty"`
	TDD          string            `json:"tdd,omitempty"`
	ACs          []ShowAC          `json:"acs,omitempty"`
	Body         map[string]string `json:"body,omitempty"`
	History      []HistoryEvent    `json:"history,omitempty"`
	Findings     []check.Finding   `json:"findings,omitempty"`
	ReferencedBy []string          `json:"referenced_by"`
	Scopes       []ScopeView       `json:"scopes,omitempty"`

	// Archived is true when the resolved entity's path lives under a
	// per-kind `archive/` subdirectory per ADR-0004. JSON shape uses
	// `omitempty` so active envelopes don't carry the field at all —
	// downstream tooling treats presence as the indicator. Text
	// rendering appends ` · archived` to the header line (see
	// renderShowText). M-0087/AC-5.
	Archived bool `json:"archived,omitempty"`

	// Composite-id-only fields (when querying M-NNN/AC-N): the AC's
	// own state, populated instead of (not in addition to) the
	// milestone's full ACs slice.
	AC       *ShowAC `json:"ac,omitempty"`
	ParentID string  `json:"parent_id,omitempty"`
}

// ShowAC is one AC's view inside a milestone show. Description carries
// the prose under the matching `### AC-N — <title>` heading in the
// milestone body, trimmed of surrounding whitespace; empty when the
// milestone body has no body section for this AC (e.g. seeded purely
// via frontmatter).
type ShowAC struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	TDDPhase    string `json:"tdd_phase,omitempty"`
	Description string `json:"description,omitempty"`
}

// buildShowView assembles the view for id; ok=false when no entity
// (or AC) matches. Composite ids resolve via the parent milestone's
// ACs slice.
func buildShowView(ctx context.Context, root string, t *tree.Tree, loadErrs []tree.LoadError, id string, historyLimit int) (ShowView, bool) {
	if entity.IsCompositeID(id) {
		return buildCompositeShowView(ctx, root, t, loadErrs, id, historyLimit)
	}
	e := t.ByID(id)
	if e == nil {
		return ShowView{}, false
	}
	body := readEntityBody(root, e.Path)
	// Emit canonical ids per AC-3 in M-081 — display surfaces are
	// uniform-width regardless of on-disk filename.
	view := ShowView{
		ID:           entity.Canonicalize(e.ID),
		Kind:         string(e.Kind),
		Title:        e.Title,
		Status:       e.Status,
		Path:         e.Path,
		Parent:       entity.Canonicalize(e.Parent),
		TDD:          e.TDD,
		Body:         entity.ParseBodySections(body),
		ReferencedBy: nonNilStrings(t.ReferencedBy(id)),
		Archived:     entity.IsArchivedPath(e.Path),
	}
	var acDesc map[string]string
	if e.Kind == entity.KindMilestone && len(e.ACs) > 0 {
		acDesc = entity.ParseACSections(body)
	}
	for _, ac := range e.ACs {
		view.ACs = append(view.ACs, ShowAC{
			ID:          ac.ID,
			Title:       ac.Title,
			Status:      ac.Status,
			TDDPhase:    ac.TDDPhase,
			Description: acDesc[ac.ID],
		})
	}

	events, err := readHistory(ctx, root, id)
	if err == nil {
		view.History = limitEvents(events, historyLimit)
	}
	if scopes, err := loadEntityScopeViews(ctx, root, id); err == nil {
		view.Scopes = scopes
	}

	allFindings := check.Run(t, loadErrs)
	view.Findings = filterFindingsByID(allFindings, id, e)

	return view, true
}

// nonNilStrings returns the slice unchanged when non-nil, or an empty
// (non-nil) slice when nil. Used to keep ReferencedBy as `[]` in JSON
// output instead of `null`, so downstream consumers never have to
// check for field absence vs. empty list.
func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// buildCompositeShowView handles `aiwf show M-NNN/AC-N`. Returns
// ok=false when the parent or AC doesn't exist.
func buildCompositeShowView(ctx context.Context, root string, t *tree.Tree, loadErrs []tree.LoadError, id string, historyLimit int) (ShowView, bool) {
	parentID, subID, _ := entity.ParseCompositeID(id)
	parent := t.ByID(parentID)
	if parent == nil {
		return ShowView{}, false
	}
	var found *entity.AcceptanceCriterion
	for i := range parent.ACs {
		if parent.ACs[i].ID == subID {
			found = &parent.ACs[i]
			break
		}
	}
	if found == nil {
		return ShowView{}, false
	}
	desc := entity.ParseACSections(readEntityBody(root, parent.Path))[found.ID]
	// Emit canonical ids per AC-3 in M-081.
	view := ShowView{
		ID:       entity.Canonicalize(id),
		Kind:     "ac",
		Title:    found.Title,
		Status:   found.Status,
		Path:     parent.Path,
		ParentID: entity.Canonicalize(parentID),
		AC: &ShowAC{
			ID:          found.ID,
			Title:       found.Title,
			Status:      found.Status,
			TDDPhase:    found.TDDPhase,
			Description: desc,
		},
		ReferencedBy: nonNilStrings(t.ReferencedBy(id)),
		// Composite show inherits archived state from its parent
		// milestone — an AC under an archived milestone reads as
		// archived too. M-0087/AC-5.
		Archived: entity.IsArchivedPath(parent.Path),
	}

	events, err := readHistory(ctx, root, id)
	if err == nil {
		view.History = limitEvents(events, historyLimit)
	}
	if scopes, err := loadEntityScopeViews(ctx, root, id); err == nil {
		view.Scopes = scopes
	}

	allFindings := check.Run(t, loadErrs)
	view.Findings = filterFindingsByID(allFindings, id, parent)

	return view, true
}

// limitEvents trims the history slice. negative limit returns all;
// zero returns nil; positive returns the most recent N (events come
// oldest-first from readHistory, so we slice from the tail).
func limitEvents(events []HistoryEvent, limit int) []HistoryEvent {
	switch {
	case limit < 0:
		return events
	case limit == 0:
		return nil
	case len(events) <= limit:
		return events
	default:
		return events[len(events)-limit:]
	}
}

// filterFindingsByID keeps only findings that scope to the queried
// id. For top-level entities, that's findings whose EntityID equals
// the id OR whose Path matches the entity's path. For composite ids,
// findings whose EntityID equals the composite id.
func filterFindingsByID(all []check.Finding, id string, parent *entity.Entity) []check.Finding {
	canon := entity.Canonicalize(id)
	var out []check.Finding
	for i := range all {
		f := all[i]
		if entity.Canonicalize(f.EntityID) == canon {
			out = append(out, f)
			continue
		}
		if !entity.IsCompositeID(id) && parent != nil && f.Path == parent.Path && f.EntityID == "" {
			// Some checks fire without an entity id (e.g., load-error);
			// scope by path when we can.
			out = append(out, f)
		}
	}
	return out
}

// renderShowText writes the human-readable view to stdout. Layout
// matches the design's example: header line with id + title + status,
// indented attribute block (parent, tdd), an ACs block, a recent-
// history block, and a findings block.
func renderShowText(v ShowView) {
	// archivedMarker is the terse one-word suffix appended to the
	// header line for archived entities. Per ADR-0004 §"Display
	// surfaces": "render output indicates archived state visibly."
	// Kept to a single word so the header still reads as one scan
	// rather than a multi-row badge block. M-0087/AC-5.
	archivedMarker := ""
	if v.Archived {
		archivedMarker = " · archived"
	}
	if v.AC != nil {
		// Composite-id view.
		fmt.Printf("%s · %q · status: %s · phase: %s%s\n",
			v.ID, v.AC.Title, v.AC.Status, displayPhase(v.AC.TDDPhase), archivedMarker)
		fmt.Printf("  parent: %s\n", v.ParentID)
	} else {
		// Top-level view.
		header := fmt.Sprintf("%s · %s · status: %s", v.ID, v.Title, v.Status)
		if v.TDD != "" {
			header += " · tdd: " + v.TDD
		}
		header += archivedMarker
		fmt.Println(header)
		if v.Parent != "" {
			fmt.Printf("  parent: %s\n", v.Parent)
		}
		if len(v.ACs) > 0 {
			fmt.Println()
			fmt.Println("  ACs:")
			for i := range v.ACs {
				ac := v.ACs[i]
				fmt.Printf("    %s [%s]   · phase: %-9s · %q\n",
					ac.ID, ac.Status, displayPhase(ac.TDDPhase), ac.Title)
			}
		}
	}
	if len(v.ReferencedBy) > 0 {
		fmt.Println()
		fmt.Printf("  Referenced by (%d):\n", len(v.ReferencedBy))
		for _, ref := range v.ReferencedBy {
			fmt.Printf("    %s\n", ref)
		}
	}
	if len(v.Scopes) > 0 {
		fmt.Println()
		fmt.Printf("  Scopes (%d):\n", len(v.Scopes))
		for i := range v.Scopes {
			s := v.Scopes[i]
			ended := ""
			if s.EndedAt != "" {
				ended = "  ended " + s.EndedAt[:10]
			}
			fmt.Printf("    %s  %s → %s  state: %-7s  opened %s%s  events: %d\n",
				shortHash(s.AuthSHA), s.Principal, s.Agent, s.State,
				dateOnly(s.Opened), ended, s.EventCount)
		}
	}
	if len(v.History) > 0 {
		fmt.Println()
		fmt.Printf("  Recent history (%d):\n", len(v.History))
		for i := range v.History {
			e := v.History[i]
			detail := e.Detail
			if e.Force != "" {
				detail += " [forced]"
			}
			fmt.Printf("    %s  %-10s  %-12s  %s\n",
				e.Date[:10], e.Verb, renderTo(e.To), detail)
		}
	}
	fmt.Println()
	if len(v.Findings) == 0 {
		fmt.Println("  Findings: (none)")
	} else {
		fmt.Printf("  Findings (%d):\n", len(v.Findings))
		for i := range v.Findings {
			f := v.Findings[i]
			subcode := ""
			if f.Subcode != "" {
				subcode = "/" + f.Subcode
			}
			fmt.Printf("    %s%s [%s]: %s\n", f.Code, subcode, f.Severity, f.Message)
		}
	}
}

// displayPhase formats a TDD phase for show output. Empty (absent)
// renders as "-" so the column reads cleanly.
func displayPhase(phase string) string {
	if phase == "" {
		return "-"
	}
	return phase
}

// dateOnly returns the calendar-day prefix of an ISO-8601 timestamp,
// or the input unchanged when shorter than 10 chars. Used by the
// scopes block where the time-of-day is noise.
func dateOnly(s string) string {
	if len(s) < 10 {
		return s
	}
	return s[:10]
}
