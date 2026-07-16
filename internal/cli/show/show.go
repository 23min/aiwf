// Package show implements the `aiwf show` verb (per-verb subpackage of M-0116;
// includes the show-scopes helpers moved from show_scopes.go).
package show

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/history"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/logger"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/trunk"
	"github.com/23min/aiwf/internal/version"
)

// AreaMissLine is the one-line text rendered when `aiwf show <id>
// --area <name>` filters the named entity out because its effective area
// differs (E-0043, M-0174/AC-3). `show` is single-entity, so the
// predicate hides the one entity (like an empty list) rather than listing
// — the line tells the operator the entity exists but lives elsewhere.
// An untagged entity (actual == "") gets its own wording.
func AreaMissLine(id, actual, requested string) string {
	if actual == "" {
		return fmt.Sprintf("%s is untagged; not in area %q", id, requested)
	}
	return fmt.Sprintf("%s is in area %q, not %q", id, actual, requested)
}

// ReadEntityBody reads the entity file at root/relPath and returns the
// body bytes (the prose after the closing `---`). Errors are
// swallowed — `aiwf show` already emits findings for unreadable /
// malformed entities via the load-error finding; surfacing the same
// problem on the body field would double-count. Empty body or missing
// file produces nil.
//
// Entity.Path is repo-relative (the loader normalizes it that way) so
// callers must join with root before hitting the filesystem; doing
// the join in this helper keeps each caller from re-deriving it.
func ReadEntityBody(root, relPath string) []byte {
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

// NewCmd builds `aiwf show <id>`. Aggregates per-entity state from
// the existing data sources — frontmatter (entity), git log (history),
// aiwf check (findings) — into one human-readable view (or one JSON
// envelope when --format=json). No new state; pure projection.
//
// For composite ids (M-NNN/AC-N), renders just the AC's slice of the
// parent milestone plus its history.
func NewCmd(correlationID string) *cobra.Command {
	var (
		root         string
		format       string
		pretty       bool
		historyLimit int
		area         string
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
			return cliutil.WrapExitCode(Run(args[0], root, format, area, pretty, historyLimit, correlationID))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	cmd.Flags().StringVar(&area, "area", "", "show the entity only when its effective area equals this workstream tag (E-0043)")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "indent JSON output (only with --format=json)")
	cmd.Flags().IntVar(&historyLimit, "history", 10, "max recent history events to render (0 = none, -1 = all)")
	cliutil.RegisterFormatCompletion(cmd)
	_ = cmd.RegisterFlagCompletionFunc("area", cliutil.CompleteAreaFlag())
	cmd.ValidArgsFunction = cliutil.CompleteEntityIDArg("", 0)
	return cmd
}

// Run executes `aiwf show`. Returns one of the cliutil.Exit* codes.
func Run(id, root, format, area string, pretty bool, historyLimit int, correlationID string) (code int) {
	if format != "text" && format != "json" {
		cliutil.Errorf("aiwf show: --format must be text or json, got %q\n", format)
		return cliutil.ExitUsage
	}

	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil { //coverage:ignore cliutil.ResolveRoot only fails on missing aiwf.yaml + non-existent --root path
		cliutil.Errorf("aiwf show: %v\n", err)
		return cliutil.ExitUsage
	}

	ctx := context.Background()

	// M-0249: diagnostic-logging wiring, mirroring cancel.Run's own
	// M-0238/AC-5 pattern — with one difference: show is a pure read
	// with no --actor flag and no commit, so actor resolution is
	// best-effort only. A missing git identity must never fail a read
	// verb that never needed one before; ADR-0017's own principle
	// ("diagnostic logging must never affect a verb's own behavior or
	// exit code") governs here even though ResolveLogger's own
	// fallback only covers the logger's own resolve/open failures, not
	// this actor lookup.
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
		diagLog = logger.WithVerb(diagLog, "show", id, actorStr, runID)
	}
	defer func() { cliutil.EmitVerbOutcome(diagLog, "verb", code, "") }()

	// Advisory note when --area names an undeclared value (M-0174/AC-5).
	if note := cliutil.UndeclaredAreaNote(rootDir, area); note != "" {
		cliutil.Errorln(note)
	}

	tr, loadErrs, err := tree.Load(ctx, rootDir)
	if err != nil { //coverage:ignore tree.Load errors only on filesystem IO failure (e.g. a permission fault) or context cancellation; malformed entities surface as load findings, not an error here.
		cliutil.Errorf("aiwf show: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}

	view, ok := BuildShowView(ctx, rootDir, tr, loadErrs, id, historyLimit)
	if !ok {
		message := fmt.Sprintf("%s not found", id)
		switch format {
		case "text":
			cliutil.Errorf("aiwf show: %s\n", message)
		case "json":
			env := render.Envelope{
				Tool:     "aiwf",
				Version:  version.Current().Version,
				Status:   "error",
				Error:    &render.EnvelopeError{Message: message},
				Metadata: map[string]any{"root": rootDir, "id": id},
			}
			if err := render.JSON(os.Stdout, env, pretty); err != nil { //coverage:ignore render.JSON only errors on a stdout write failure (not portably triggerable in test); mirrors this verb's other json render branches
				cliutil.Errorf("aiwf show: %v\n", err)
				return cliutil.ExitInternal
			}
		}
		return cliutil.ExitUsage
	}

	// Predicate filter (M-0174/AC-3): `show` names one entity, so --area
	// hides it (like an empty list) when its effective area differs,
	// rather than listing. ResolvedAreaByID handles composite AC ids
	// (roll up to the parent epic). Exit 0 — the entity exists, it is
	// just out of the requested workstream.
	if area != "" {
		actual := tr.ResolvedAreaByID(id)
		if actual != area {
			switch format {
			case "text":
				cliutil.Println(AreaMissLine(view.ID, actual, area))
			case "json":
				env := render.Envelope{
					Tool:    "aiwf",
					Version: version.Current().Version,
					Status:  "ok",
					Result:  nil,
					Metadata: map[string]any{
						"root":         rootDir,
						"id":           id,
						"filtered_out": true,
						"area":         area,
						"actual_area":  actual,
					},
				}
				if err := render.JSON(os.Stdout, env, pretty); err != nil { //coverage:ignore render.JSON only errors on a stdout write failure (not portably triggerable in test); mirrors this verb's other json render branches
					cliutil.Errorf("aiwf show: %v\n", err)
					return cliutil.ExitInternal
				}
			}
			return cliutil.ExitOK
		}
	}

	switch format {
	case "text":
		renderShowText(view)
	case "json":
		env := render.Envelope{
			Tool:    "aiwf",
			Version: version.Current().Version,
			Status:  "ok",
			Result:  view,
			Metadata: map[string]any{
				"root": rootDir,
				"id":   id,
			},
		}
		if err := render.JSON(os.Stdout, env, pretty); err != nil { //coverage:ignore render.JSON only errors on a stdout write failure (not portably triggerable in test); mirrors this verb's other json render branches
			cliutil.Errorf("aiwf show: %v\n", err)
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
	ID           string                 `json:"id"`
	Kind         string                 `json:"kind"`
	Title        string                 `json:"title"`
	Status       string                 `json:"status"`
	Path         string                 `json:"path,omitempty"`
	Parent       string                 `json:"parent,omitempty"`
	TDD          string                 `json:"tdd,omitempty"`
	ACs          []ShowAC               `json:"acs,omitempty"`
	Body         map[string]string      `json:"body,omitempty"`
	History      []history.HistoryEvent `json:"history,omitempty"`
	Findings     []check.Finding        `json:"findings,omitempty"`
	ReferencedBy []string               `json:"referenced_by"`
	Scopes       []ScopeView            `json:"scopes,omitempty"`

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

	// CrossBranch is non-nil when id missed the local working tree but
	// resolved live from another local or remote-tracking ref (M-0260,
	// ADR-0030's read-side extension point). Nil for a locally-resolved
	// entity — the JSON envelope omits the field entirely so a
	// downstream consumer can treat its absence as "this is ordinary
	// local state."
	CrossBranch *CrossBranchView `json:"cross_branch,omitempty"`
}

// CrossBranchView carries the read-side resolution state for an id
// resolved from another ref (M-0260/AC-1/AC-2/AC-3). Refs always lists
// every candidate ref the id is known on — one entry when Collision is
// false, two or more when true.
type CrossBranchView struct {
	// Ref is the ref content was read from. Empty when Collision is
	// true — no single ref was chosen.
	Ref string `json:"ref,omitempty"`
	// Collision is true when the candidate refs carry divergent
	// content (M-0259/AC-3). Title/Status/Body/ACs are left empty on
	// the ShowView in that case — aiwf show declines to pick a side
	// (M-0260/AC-3) rather than render one ref's content as canonical.
	Collision bool     `json:"collision"`
	Refs      []string `json:"refs"`
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

// BuildShowView assembles the view for id; ok=false when no entity
// (or AC) matches. Composite ids resolve via the parent milestone's
// ACs slice.
func BuildShowView(ctx context.Context, root string, t *tree.Tree, loadErrs []tree.LoadError, id string, historyLimit int) (ShowView, bool) {
	if entity.IsCompositeID(id) {
		return BuildCompositeShowView(ctx, root, t, loadErrs, id, historyLimit)
	}
	e := t.ByID(id)
	if e == nil {
		return buildCrossBranchShowView(ctx, root, t, id)
	}
	body := ReadEntityBody(root, e.Path)
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

	events, err := history.ReadHistory(ctx, root, id)
	if err == nil {
		view.History = limitEvents(events, historyLimit)
	}
	if scopes, err := LoadEntityScopeViews(ctx, root, id); err == nil {
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

// buildCrossBranchShowView resolves id from another local or
// remote-tracking ref when it misses the local working tree (M-0260/
// AC-1, ADR-0030's read-side extension point). Scoped to id alone —
// it scans refs live only on this local-tree miss, never eagerly, so
// the common case (the entity resolves locally) pays no extra
// subprocess cost (the milestone's own risk-mitigation constraint).
//
// A collision (divergent content across two or more refs, M-0259/
// AC-3's shape) declines to pick a side: the returned view carries
// only identity (id, kind) plus the candidate refs — no title,
// status, or body, since those are exactly the content in dispute
// (M-0260/AC-3). A single-answer hit reads its content live via
// gitops.BlobReader and renders it labeled as cross-branch (AC-2).
//
// Returns ok=false when id is unknown everywhere (no local entity, no
// cross-branch hit) or when the git-side resolution can't complete
// (best-effort, mirroring trunk.LocalRefHits/DetectCollisions'
// degrade-to-nothing contract) — both read as an ordinary "not found"
// to the caller.
//
// History/Scopes/Findings are deliberately left empty on this view: a
// cross-branch entity's git log and check findings are scoped to this
// branch's own history/tree, neither of which meaningfully covers an
// entity that hasn't merged into this branch yet.
func buildCrossBranchShowView(ctx context.Context, root string, t *tree.Tree, id string) (ShowView, bool) {
	if root == "" {
		// Defense-in-depth, mirroring crossBranchListRows' own guard:
		// exec.Cmd treats an empty Dir as "inherit the caller's cwd,"
		// which would otherwise run these git subprocesses against
		// whatever directory happens to be the caller's actual working
		// directory. Every production call site resolves root via
		// cliutil.ResolveRoot first, so this never fires today — kept
		// so a future bare-root caller degrades safely instead of
		// scanning an unintended repo.
		return ShowView{}, false
	}
	canon := entity.Canonicalize(id)
	all := append(trunk.LocalRefHits(ctx, root), trunk.RemoteRefHits(ctx, root)...)
	var hits []trunk.RefHit
	for _, h := range all {
		if entity.Canonicalize(h.ID) == canon {
			hits = append(hits, h)
		}
	}
	if len(hits) == 0 {
		return ShowView{}, false
	}
	refs := trunk.DistinctRefs(hits)

	if trunk.DetectCollisions(ctx, root, hits)[canon] {
		return ShowView{
			ID:           canon,
			Kind:         string(hits[0].Kind),
			CrossBranch:  &CrossBranchView{Collision: true, Refs: refs},
			ReferencedBy: nonNilStrings(t.ReferencedBy(id)),
		}, true
	}

	hit := hits[0]
	br, err := gitops.NewBlobReader(ctx, root)
	if err != nil { //coverage:ignore by the time hits is non-empty, LocalRefHits/RemoteRefHits already confirmed root is a real repo (gitops.IsRepo); NewBlobReader failing here needs the repo to break between that scan and this construction — not reproducible against a healthy subprocess, same class as gitops.NewBlobReader's own internal coverage:ignore branches
		return ShowView{}, false
	}
	defer func() { _ = br.Close() }()
	content, err := br.Read(hit.Ref, hit.Path)
	if err != nil { //coverage:ignore hit.Path was just confirmed present at hit.Ref by the LsTreePaths scan that produced this RefHit; a subsequent blob read at the same ref:path failing needs the object store to change mid-request (repack/gc race), not reproducible in a unit test
		return ShowView{}, false
	}
	resolved, err := entity.Parse(hit.Path, content)
	if err != nil {
		return ShowView{}, false
	}
	resolved.Kind = hit.Kind
	_, body, _ := entity.Split(content)

	view := ShowView{
		ID:           entity.Canonicalize(resolved.ID),
		Kind:         string(resolved.Kind),
		Title:        resolved.Title,
		Status:       resolved.Status,
		Path:         resolved.Path,
		Parent:       entity.Canonicalize(resolved.Parent),
		TDD:          resolved.TDD,
		Body:         entity.ParseBodySections(body),
		ReferencedBy: nonNilStrings(t.ReferencedBy(id)),
		CrossBranch:  &CrossBranchView{Ref: hit.Ref, Refs: refs},
	}
	var acDesc map[string]string
	if resolved.Kind == entity.KindMilestone && len(resolved.ACs) > 0 {
		acDesc = entity.ParseACSections(body)
	}
	for _, ac := range resolved.ACs {
		view.ACs = append(view.ACs, ShowAC{
			ID:          ac.ID,
			Title:       ac.Title,
			Status:      ac.Status,
			TDDPhase:    ac.TDDPhase,
			Description: acDesc[ac.ID],
		})
	}
	return view, true
}

// BuildCompositeShowView handles `aiwf show M-NNN/AC-N`. Returns
// ok=false when the parent or AC doesn't exist.
func BuildCompositeShowView(ctx context.Context, root string, t *tree.Tree, loadErrs []tree.LoadError, id string, historyLimit int) (ShowView, bool) {
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
	desc := entity.ParseACSections(ReadEntityBody(root, parent.Path))[found.ID]
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

	events, err := history.ReadHistory(ctx, root, id)
	if err == nil {
		view.History = limitEvents(events, historyLimit)
	}
	if scopes, err := LoadEntityScopeViews(ctx, root, id); err == nil {
		view.Scopes = scopes
	}

	allFindings := check.Run(t, loadErrs)
	view.Findings = filterFindingsByID(allFindings, id, parent)

	return view, true
}

// limitEvents trims the history slice. negative limit returns all;
// zero returns nil; positive returns the most recent N (events come
// oldest-first from readHistory, so we slice from the tail).
func limitEvents(events []history.HistoryEvent, limit int) []history.HistoryEvent {
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
	switch {
	case v.AC != nil:
		// Composite-id view.
		cliutil.Printf("%s · %q · status: %s · phase: %s%s\n",
			v.ID, v.AC.Title, v.AC.Status, displayPhase(v.AC.TDDPhase), archivedMarker)
		cliutil.Printf("  parent: %s\n", v.ParentID)
	case v.CrossBranch != nil && v.CrossBranch.Collision:
		// Cross-branch collision (M-0260/AC-3): decline to render
		// title/status/body — that content is exactly what's in
		// dispute — and name every candidate ref instead.
		cliutil.Printf("%s · cross-branch collision · known on: %s\n",
			v.ID, strings.Join(v.CrossBranch.Refs, ", "))
		cliutil.Println("  content diverges across refs — declining to pick one; resolve by merging or reconciling")
	default:
		// Top-level view.
		header := fmt.Sprintf("%s · %s · status: %s", v.ID, v.Title, v.Status)
		if v.TDD != "" {
			header += " · tdd: " + v.TDD
		}
		header += archivedMarker
		if v.CrossBranch != nil {
			header += fmt.Sprintf(" · cross-branch (ref: %s)", v.CrossBranch.Ref)
		}
		cliutil.Println(header)
		if v.Parent != "" {
			cliutil.Printf("  parent: %s\n", v.Parent)
		}
		if len(v.ACs) > 0 {
			cliutil.Println()
			cliutil.Println("  ACs:")
			for i := range v.ACs {
				ac := v.ACs[i]
				cliutil.Printf("    %s [%s]   · phase: %-9s · %q\n",
					ac.ID, ac.Status, displayPhase(ac.TDDPhase), ac.Title)
			}
		}
	}
	if len(v.ReferencedBy) > 0 {
		cliutil.Println()
		cliutil.Printf("  Referenced by (%d):\n", len(v.ReferencedBy))
		for _, ref := range v.ReferencedBy {
			cliutil.Printf("    %s\n", ref)
		}
	}
	if len(v.Scopes) > 0 {
		cliutil.Println()
		cliutil.Printf("  Scopes (%d):\n", len(v.Scopes))
		for i := range v.Scopes {
			s := v.Scopes[i]
			ended := ""
			if s.EndedAt != "" {
				ended = "  ended " + s.EndedAt[:10]
			}
			cliutil.Printf("    %s  %s → %s  state: %-7s  opened %s%s  events: %d\n",
				history.ShortHash(s.AuthSHA), s.Principal, s.Agent, s.State,
				dateOnly(s.Opened), ended, s.EventCount)
		}
	}
	if len(v.History) > 0 {
		cliutil.Println()
		cliutil.Printf("  Recent history (%d):\n", len(v.History))
		for i := range v.History {
			e := v.History[i]
			detail := e.Detail
			if e.Force != "" {
				detail += " [forced]"
			}
			cliutil.Printf("    %s  %-10s  %-12s  %s\n",
				e.Date[:10], e.Verb, history.RenderTo(e.To), detail)
		}
	}
	cliutil.Println()
	if len(v.Findings) == 0 {
		cliutil.Println("  Findings: (none)")
	} else {
		cliutil.Printf("  Findings (%d):\n", len(v.Findings))
		for i := range v.Findings {
			f := v.Findings[i]
			subcode := ""
			if f.Subcode != "" {
				subcode = "/" + f.Subcode
			}
			cliutil.Printf("    %s%s [%s]: %s\n", f.Code, subcode, f.Severity, f.Message)
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
