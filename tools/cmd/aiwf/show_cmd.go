package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/render"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

// runShow handles `aiwf show <id>`. Aggregates per-entity state from
// the existing data sources — frontmatter (entity), git log (history),
// aiwf check (findings) — into one human-readable view (or one JSON
// envelope when --format=json). No new state; pure projection.
//
// For composite ids (M-NNN/AC-N), renders just the AC's slice of the
// parent milestone plus its history.
func runShow(args []string) int {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	root := fs.String("root", "", "consumer repo root")
	format := fs.String("format", "text", "output format: text or json")
	pretty := fs.Bool("pretty", false, "indent JSON output (only with --format=json)")
	historyLimit := fs.Int("history", 10, "max recent history events to render (0 = none, -1 = all)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(reorderFlagsFirst(args, []string{"root", "format", "history"}, []string{"pretty"})); err != nil {
		return exitUsage
	}
	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(os.Stderr, "aiwf show: usage: aiwf show <id-or-composite-id>")
		return exitUsage
	}
	id := rest[0]
	if *format != "text" && *format != "json" {
		fmt.Fprintf(os.Stderr, "aiwf show: --format must be text or json, got %q\n", *format)
		return exitUsage
	}

	rootDir, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf show: %v\n", err)
		return exitUsage
	}

	ctx := context.Background()
	tr, loadErrs, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf show: loading tree: %v\n", err)
		return exitInternal
	}

	view, ok := buildShowView(ctx, rootDir, tr, loadErrs, id, *historyLimit)
	if !ok {
		fmt.Fprintf(os.Stderr, "aiwf show: %s not found\n", id)
		return exitUsage
	}

	switch *format {
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
		if err := render.JSON(os.Stdout, env, *pretty); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf show: %v\n", err)
			return exitInternal
		}
	}
	return exitOK
}

// ShowView is the aggregated per-entity state. Exported for the JSON
// envelope. Field-set varies by what kind of id was queried; absent
// fields render as empty / omitted in JSON via omitempty.
type ShowView struct {
	ID       string          `json:"id"`
	Kind     string          `json:"kind"`
	Title    string          `json:"title"`
	Status   string          `json:"status"`
	Path     string          `json:"path,omitempty"`
	Parent   string          `json:"parent,omitempty"`
	TDD      string          `json:"tdd,omitempty"`
	ACs      []ShowAC        `json:"acs,omitempty"`
	History  []HistoryEvent  `json:"history,omitempty"`
	Findings []check.Finding `json:"findings,omitempty"`

	// Composite-id-only fields (when querying M-NNN/AC-N): the AC's
	// own state, populated instead of (not in addition to) the
	// milestone's full ACs slice.
	AC       *ShowAC `json:"ac,omitempty"`
	ParentID string  `json:"parent_id,omitempty"`
}

// ShowAC is one AC's view inside a milestone show.
type ShowAC struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	TDDPhase string `json:"tdd_phase,omitempty"`
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
	view := ShowView{
		ID:     e.ID,
		Kind:   string(e.Kind),
		Title:  e.Title,
		Status: e.Status,
		Path:   e.Path,
		Parent: e.Parent,
		TDD:    e.TDD,
	}
	for _, ac := range e.ACs {
		view.ACs = append(view.ACs, ShowAC{ID: ac.ID, Title: ac.Title, Status: ac.Status, TDDPhase: ac.TDDPhase})
	}

	events, err := readHistory(ctx, root, id)
	if err == nil {
		view.History = limitEvents(events, historyLimit)
	}

	allFindings := check.Run(t, loadErrs)
	view.Findings = filterFindingsByID(allFindings, id, e)

	return view, true
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
	view := ShowView{
		ID:       id,
		Kind:     "ac",
		Title:    found.Title,
		Status:   found.Status,
		Path:     parent.Path,
		ParentID: parentID,
		AC:       &ShowAC{ID: found.ID, Title: found.Title, Status: found.Status, TDDPhase: found.TDDPhase},
	}

	events, err := readHistory(ctx, root, id)
	if err == nil {
		view.History = limitEvents(events, historyLimit)
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
	var out []check.Finding
	for i := range all {
		f := all[i]
		if f.EntityID == id {
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
	if v.AC != nil {
		// Composite-id view.
		fmt.Printf("%s · %q · status: %s · phase: %s\n",
			v.ID, v.AC.Title, v.AC.Status, displayPhase(v.AC.TDDPhase))
		fmt.Printf("  parent: %s\n", v.ParentID)
	} else {
		// Top-level view.
		header := fmt.Sprintf("%s · %s · status: %s", v.ID, v.Title, v.Status)
		if v.TDD != "" {
			header += " · tdd: " + v.TDD
		}
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
