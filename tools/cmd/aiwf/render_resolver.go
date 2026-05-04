package main

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/config"
	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/htmlrender"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

// renderResolver implements htmlrender.PageDataResolver against a
// loaded planning tree plus git access. It pulls per-entity body
// sections from disk, walks `aiwf history` per entity for the
// commits / build / tests / provenance tabs, and reads the scope
// FSM for the scopes table.
//
// One resolver instance per render — internal caches live for the
// duration of the call. History results are cached per id so the
// epic page (recent activity) and milestone page (commits + build
// + tests + provenance) don't read git twice for the same entity.
type renderResolver struct {
	ctx          context.Context
	root         string
	tree         *tree.Tree
	cfg          *config.Config
	historyCache map[string][]HistoryEvent
	findings     []check.Finding // pre-computed once per render
}

// newRenderResolver builds a resolver bound to a single render
// invocation. cfg may be nil (the consumer's aiwf.yaml might be
// missing); the resolver treats nil as "default settings."
func newRenderResolver(ctx context.Context, root string, tr *tree.Tree, cfg *config.Config, findings []check.Finding) *renderResolver {
	return &renderResolver{
		ctx:          ctx,
		root:         root,
		tree:         tr,
		cfg:          cfg,
		historyCache: map[string][]HistoryEvent{},
		findings:     findings,
	}
}

// IndexData implements htmlrender.PageDataResolver.
func (r *renderResolver) IndexData() (*htmlrender.IndexData, error) {
	sidebar := r.sidebar("", "")
	sidebar.IsCurrentIndex = true
	out := &htmlrender.IndexData{Title: "Overview", Sidebar: sidebar}
	for _, e := range sortedEntitiesByID(r.tree.ByKind(entity.KindEpic)) {
		summary := htmlrender.EpicSummary{
			ID:       e.ID,
			Title:    e.Title,
			Status:   e.Status,
			FileName: idToHTMLFile(e.ID),
		}
		for _, m := range r.milestonesUnder(e.ID) {
			summary.MilestoneCount++
			met, total := acRollup(m.ACs)
			summary.ACMet += met
			summary.ACTotal += total
		}
		summary.LastActivity = r.lastActivityFor(e.ID)
		out.Epics = append(out.Epics, summary)
	}
	for i := range r.findings {
		switch r.findings[i].Severity {
		case check.SeverityError:
			out.FindingCounts.Errors++
		case check.SeverityWarning:
			out.FindingCounts.Warnings++
		}
	}
	return out, nil
}

// EpicData implements htmlrender.PageDataResolver.
func (r *renderResolver) EpicData(id string) (*htmlrender.EpicData, error) {
	e := r.tree.ByID(id)
	if e == nil || e.Kind != entity.KindEpic {
		return nil, nil
	}
	body := r.bodyForEntity(e.Path)
	data := &htmlrender.EpicData{
		Epic:    r.entityRef(e),
		Body:    entity.ParseBodySections(body),
		Sidebar: r.sidebar(e.ID, ""),
	}
	for _, m := range r.milestonesUnder(e.ID) {
		met, total := acRollup(m.ACs)
		data.Milestones = append(data.Milestones, htmlrender.MilestoneSummary{
			ID:           m.ID,
			Title:        m.Title,
			Status:       m.Status,
			TDD:          m.TDD,
			FileName:     idToHTMLFile(m.ID),
			ACMet:        met,
			ACTotal:      total,
			LastActivity: r.lastActivityFor(m.ID),
		})
		data.ACMet += met
		data.ACTotal += total
		for _, dep := range m.DependsOn {
			data.DependencyDAG = append(data.DependencyDAG, htmlrender.DependencyEdge{From: m.ID, To: dep})
		}
	}
	data.LinkedEntities = r.linkedEntitiesFor(e)
	data.History = r.historyRows(e.ID, 10)
	return data, nil
}

// EntityData implements htmlrender.PageDataResolver for the four
// kinds with no specialized template (gap, ADR, decision, contract).
// Reads the body from disk and parses sections in document order so
// the page reads as a recognizable rendering of the source markdown.
// G35 fix.
func (r *renderResolver) EntityData(id string) (*htmlrender.EntityData, error) {
	e := r.tree.ByID(id)
	if e == nil {
		return nil, nil
	}
	switch e.Kind {
	case entity.KindGap, entity.KindADR, entity.KindDecision, entity.KindContract:
		// fall through
	default:
		return nil, nil
	}
	body := r.bodyForEntity(e.Path)
	var sections []htmlrender.BodySectionView
	for _, s := range entity.ParseBodySectionsOrdered(body) {
		sections = append(sections, htmlrender.BodySectionView{
			Slug:    s.Slug,
			Heading: s.Heading,
			Content: s.Content,
		})
	}
	data := &htmlrender.EntityData{
		Entity:         r.entityRef(e),
		Sections:       sections,
		LinkedEntities: r.linkedEntitiesFor(e),
		History:        r.historyRows(e.ID, 10),
		Sidebar:        r.sidebar("", ""),
	}
	return data, nil
}

// MilestoneData implements htmlrender.PageDataResolver.
func (r *renderResolver) MilestoneData(id string) (*htmlrender.MilestoneData, error) {
	m := r.tree.ByID(id)
	if m == nil || m.Kind != entity.KindMilestone {
		return nil, nil
	}
	body := r.bodyForEntity(m.Path)
	met, total := acRollup(m.ACs)
	data := &htmlrender.MilestoneData{
		Milestone: r.entityRef(m),
		Body:      entity.ParseBodySections(body),
		ACMet:     met,
		ACTotal:   total,
		Sidebar:   r.sidebar(m.Parent, m.ID),
	}
	if m.Parent != "" {
		if parent := r.tree.ByID(m.Parent); parent != nil {
			data.ParentEpic = r.entityRef(parent)
		}
	}
	acDescriptions := entity.ParseACSections(body)
	for _, ac := range m.ACs {
		composite := m.ID + "/" + ac.ID
		acHistory := r.history(composite)
		detail := htmlrender.ACDetail{
			ID:          ac.ID,
			Title:       ac.Title,
			Status:      ac.Status,
			TDDPhase:    ac.TDDPhase,
			Description: acDescriptions[ac.ID],
			Anchor:      htmlrender.ACAnchor(ac.ID),
			Phases:      phaseEventsFromHistory(acHistory),
			Tests:       firstTestsTrailer(acHistory),
		}
		data.ACs = append(data.ACs, detail)
	}
	data.Commits = r.historyRows(m.ID, 50)
	data.Provenance = r.provenanceFor(m)
	data.LinkedEntities = r.linkedEntitiesFor(m)
	for i := range data.LinkedEntities {
		if data.LinkedEntities[i].Kind == string(entity.KindDecision) {
			data.LinkedDecisions = append(data.LinkedDecisions, data.LinkedEntities[i])
		}
	}
	if r.cfg != nil {
		data.TestsPolicy.Strict = r.cfg.TDD.RequireTestMetrics
	}
	return data, nil
}

// sidebar builds the SidebarData for the page being rendered.
// activeEpicID names the epic ancestor to mark active (the page
// itself for an epic page, the parent for a milestone page, "" for
// the index / status). activeMilestoneID, when set, marks one
// milestone link as the current page. currentStatus marks the
// "Project status" sidebar link as the current page.
//
// HasStatus is always true on the cmd-side resolver — the status
// page is part of the standard render. The default resolver
// (htmlrender package tests) leaves it false to skip the page.
//
// The walk uses the same sortedByID helper as the index/epic page
// rollups, so sidebar order is the canonical id order across every
// page. No git access — the sidebar is a pure projection of the
// frontmatter tree.
func (r *renderResolver) sidebar(activeEpicID, activeMilestoneID string) htmlrender.SidebarData {
	return r.sidebarWithStatus(activeEpicID, activeMilestoneID, false)
}

func (r *renderResolver) sidebarWithStatus(activeEpicID, activeMilestoneID string, currentStatus bool) htmlrender.SidebarData {
	s := htmlrender.SidebarData{
		HasStatus:       true,
		IsCurrentStatus: currentStatus,
	}
	for _, e := range sortedEntitiesByID(r.tree.ByKind(entity.KindEpic)) {
		entry := htmlrender.SidebarEpic{
			ID:        e.ID,
			Title:     e.Title,
			FileName:  idToHTMLFile(e.ID),
			IsActive:  e.ID == activeEpicID,
			IsCurrent: e.ID == activeEpicID && activeMilestoneID == "",
		}
		for _, m := range r.milestonesUnder(e.ID) {
			entry.Milestones = append(entry.Milestones, htmlrender.SidebarMilestone{
				ID:        m.ID,
				Title:     m.Title,
				FileName:  idToHTMLFile(m.ID),
				IsCurrent: m.ID == activeMilestoneID,
			})
		}
		s.Epics = append(s.Epics, entry)
	}
	return s
}

// StatusData implements htmlrender.PageDataResolver. Reuses the
// existing buildStatus() + readRecentActivity() helpers (which
// power the `aiwf status` verb) and projects the result into the
// renderer-facing types.
func (r *renderResolver) StatusData() (*htmlrender.StatusData, error) {
	report := buildStatus(r.tree, nil)
	if recent, err := readRecentActivity(r.ctx, r.root, recentActivityLimit); err == nil {
		report.RecentActivity = recent
	}
	out := &htmlrender.StatusData{
		Sidebar:     r.sidebarWithStatus("", "", true),
		GeneratedAt: report.Date,
		Health: htmlrender.StatusHealth{
			Entities: report.Health.Entities,
			Errors:   report.Health.Errors,
			Warnings: report.Health.Warnings,
		},
	}
	for _, e := range report.InFlightEpics {
		ev := htmlrender.StatusEpicView{
			ID:       e.ID,
			Title:    e.Title,
			Status:   e.Status,
			FileName: idToHTMLFile(e.ID),
		}
		for _, m := range e.Milestones {
			mv := htmlrender.StatusMilestoneView{
				ID:       m.ID,
				Title:    m.Title,
				Status:   m.Status,
				FileName: idToHTMLFile(m.ID),
				TDD:      m.TDD,
			}
			if m.ACs != nil {
				mv.ACMet = m.ACs.Met
				mv.ACTotal = m.ACs.InScope
				mv.OpenACs = m.ACs.Open
			}
			ev.Milestones = append(ev.Milestones, mv)
		}
		out.InFlightEpics = append(out.InFlightEpics, ev)
	}
	for _, d := range report.OpenDecisions {
		out.OpenDecisions = append(out.OpenDecisions, htmlrender.StatusEntityLink{
			ID:       d.ID,
			Title:    d.Title,
			Status:   d.Status,
			FileName: idToHTMLFile(d.ID),
		})
	}
	for _, g := range report.OpenGaps {
		// The aiwf status report's gap struct doesn't carry a
		// Status field (gaps are listed only when open); leave
		// the renderer-facing Status empty so the template can
		// suppress the pill.
		out.OpenGaps = append(out.OpenGaps, htmlrender.StatusGapView{
			ID:           g.ID,
			Title:        g.Title,
			FileName:     idToHTMLFile(g.ID),
			DiscoveredIn: g.DiscoveredIn,
		})
	}
	for _, w := range report.Warnings {
		out.Warnings = append(out.Warnings, htmlrender.StatusFinding{
			Code:     w.Code,
			EntityID: w.EntityID,
			Path:     w.Path,
			Message:  w.Message,
		})
	}
	for i := range report.RecentActivity {
		out.RecentActivity = append(out.RecentActivity, historyEventToRow(&report.RecentActivity[i]))
	}
	return out, nil
}

// entityRef builds the minimal renderer-facing struct from an
// entity.Entity. FileName is canonical (idToHTMLFile == htmlrender's
// own resolver, kept in sync via the same scheme).
func (r *renderResolver) entityRef(e *entity.Entity) *htmlrender.EntityRef {
	return &htmlrender.EntityRef{
		ID:       e.ID,
		Title:    e.Title,
		Status:   e.Status,
		Path:     e.Path,
		Kind:     string(e.Kind),
		TDD:      e.TDD,
		FileName: idToHTMLFile(e.ID),
	}
}

// milestonesUnder returns every milestone whose Parent == epicID,
// sorted by id.
func (r *renderResolver) milestonesUnder(epicID string) []*entity.Entity {
	var out []*entity.Entity
	for _, m := range r.tree.ByKind(entity.KindMilestone) {
		if m.Parent == epicID {
			out = append(out, m)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// history returns events for id, cached per resolver instance.
func (r *renderResolver) history(id string) []HistoryEvent {
	if events, ok := r.historyCache[id]; ok {
		return events
	}
	events, err := readHistory(r.ctx, r.root, id)
	if err != nil {
		// Best-effort: a history-walk error degrades the page (empty
		// Commits / Build / Tests tabs) but doesn't fail the render.
		events = nil
	}
	r.historyCache[id] = events
	return events
}

// historyRows materializes the renderer-facing rows from cached
// HistoryEvents. limit clips to the most recent N.
func (r *renderResolver) historyRows(id string, limit int) []htmlrender.HistoryRow {
	events := r.history(id)
	if len(events) == 0 {
		return nil
	}
	if limit > 0 && len(events) > limit {
		events = events[len(events)-limit:]
	}
	out := make([]htmlrender.HistoryRow, 0, len(events))
	for i := range events {
		out = append(out, historyEventToRow(&events[i]))
	}
	return out
}

// lastActivityFor returns the YYYY-MM-DD of the most recent aiwf
// trailer commit on id, or "" when none.
func (r *renderResolver) lastActivityFor(id string) string {
	events := r.history(id)
	if len(events) == 0 {
		return ""
	}
	last := events[len(events)-1].Date
	if len(last) >= 10 {
		return last[:10]
	}
	return last
}

// linkedEntitiesFor returns ADRs / decisions / gaps / contracts
// that reference the entity, plus the entity's own outbound
// references that aren't already milestones-of-the-epic. Sorted by
// id; deduplicated.
func (r *renderResolver) linkedEntitiesFor(e *entity.Entity) []htmlrender.LinkedEntity {
	seen := map[string]struct{}{}
	var out []htmlrender.LinkedEntity
	add := func(id, dir string) {
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		other := r.tree.ByID(id)
		if other == nil {
			return
		}
		// Skip same-epic milestones in the linked list — they show
		// up in their own table.
		if other.Kind == entity.KindMilestone && other.Parent == e.ID {
			return
		}
		out = append(out, htmlrender.LinkedEntity{
			ID:        other.ID,
			Title:     other.Title,
			Status:    other.Status,
			Kind:      string(other.Kind),
			FileName:  idToHTMLFile(other.ID),
			Direction: dir,
		})
	}
	// Forward references from this entity's frontmatter.
	for _, ref := range entity.ForwardRefs(e) {
		add(ref.Target, "forward")
	}
	// Reverse references — every entity that names this one.
	for _, ref := range r.tree.ReferencedBy(e.ID) {
		add(ref, "reverse")
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// provenanceFor builds the Provenance tab payload for a milestone:
// scope-table + chronological timeline.
func (r *renderResolver) provenanceFor(m *entity.Entity) htmlrender.ProvenanceData {
	var data htmlrender.ProvenanceData
	scopes, err := loadEntityScopeViews(r.ctx, r.root, m.ID)
	if err == nil {
		for _, s := range scopes {
			data.Scopes = append(data.Scopes, htmlrender.ScopeRow{
				AuthSHA:    shortSHA(s.AuthSHA, 8),
				FullSHA:    s.AuthSHA,
				Agent:      s.Agent,
				Principal:  s.Principal,
				Opened:     dateOnlyOrEmpty(s.Opened),
				EndedAt:    dateOnlyOrEmpty(s.EndedAt),
				State:      s.State,
				EventCount: s.EventCount,
			})
		}
	}
	timelineEvents := r.history(m.ID)
	for i := range timelineEvents {
		data.Timeline = append(data.Timeline, historyEventToRow(&timelineEvents[i]))
	}
	return data
}

// bodyForEntity reads the body bytes of the entity at path; nil on
// any IO / parse failure (the renderer treats absent body sections
// as "skip that block" via the templates' `with` guards).
func (r *renderResolver) bodyForEntity(relPath string) []byte {
	if relPath == "" {
		return nil
	}
	abs := relPath
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(r.root, relPath)
	}
	return readEntityBody(r.root, abs)
}

// historyEventToRow maps a cmd-side HistoryEvent to the renderer's
// HistoryRow. Pulled out so the epic and milestone pages share one
// transformation; if the cmd-side struct gains a field, only this
// function changes. Takes a pointer so range loops over []HistoryEvent
// can avoid the per-iteration value copy (the struct is large
// enough that gocritic flags it).
func historyEventToRow(e *HistoryEvent) htmlrender.HistoryRow {
	row := htmlrender.HistoryRow{
		Date:         dateOnlyOrEmpty(e.Date),
		Commit:       e.Commit,
		Actor:        e.Actor,
		Principal:    e.Principal,
		OnBehalfOf:   e.OnBehalfOf,
		Verb:         e.Verb,
		Detail:       e.Detail,
		To:           e.To,
		Force:        e.Force != "",
		ForceReason:  e.Force,
		AuditOnly:    e.AuditOnly != "",
		AuditReason:  e.AuditOnly,
		AuthorizedBy: e.AuthorizedBy,
		Scope:        e.Scope,
		ScopeEnds:    append([]string(nil), e.ScopeEnds...),
		Reason:       e.Reason,
	}
	if e.Tests != nil {
		row.Tests = &htmlrender.TestMetricsView{
			Pass:  e.Tests.Pass,
			Fail:  e.Tests.Fail,
			Skip:  e.Tests.Skip,
			Total: e.Tests.TotalOrDerive(),
		}
	}
	return row
}

// phaseEventsFromHistory walks an AC's history and projects each
// `aiwf-to: <phase>` event into a PhaseEvent for the Build tab.
// Both status promotions and phase promotions write aiwf-to:; the
// kernel doesn't distinguish them at the trailer level. We filter
// to the closed set of recognized TDD phases (red / green /
// refactor / done) — anything else (open, met, deferred,
// cancelled) is a status event and goes to the Commits tab, not
// the Build tab.
func phaseEventsFromHistory(events []HistoryEvent) []htmlrender.PhaseEvent {
	var out []htmlrender.PhaseEvent
	for i := range events {
		e := &events[i]
		if e.To == "" || e.Verb != "promote" {
			continue
		}
		if !entity.IsAllowedTDDPhase(e.To) {
			continue
		}
		row := htmlrender.PhaseEvent{
			Date:   dateOnlyOrEmpty(e.Date),
			Phase:  e.To,
			Forced: e.Force != "",
			Reason: e.Reason,
		}
		if e.Tests != nil {
			row.Tests = &htmlrender.TestMetricsView{
				Pass:  e.Tests.Pass,
				Fail:  e.Tests.Fail,
				Skip:  e.Tests.Skip,
				Total: e.Tests.TotalOrDerive(),
			}
		}
		out = append(out, row)
	}
	return out
}

// firstTestsTrailer returns the first commit's parsed metrics in
// `aiwf history` order that carries a Tests pointer. Per the I3
// plan §4 aggregation rule: rebase- and amend-stable, since it
// derives from the iterator order rather than wall-clock time.
func firstTestsTrailer(events []HistoryEvent) *htmlrender.TestMetricsView {
	for i := range events {
		e := &events[i]
		if e.Tests != nil {
			return &htmlrender.TestMetricsView{
				Pass:  e.Tests.Pass,
				Fail:  e.Tests.Fail,
				Skip:  e.Tests.Skip,
				Total: e.Tests.TotalOrDerive(),
			}
		}
	}
	return nil
}

// idToHTMLFile mirrors htmlrender.idToFileName (which is
// unexported) for cmd-side ref construction. Keep in sync with
// htmlrender/paths.go.
func idToHTMLFile(id string) string {
	if id == "" {
		return "index.html"
	}
	return id + ".html"
}

// sortedEntitiesByID returns a copy of entities sorted by id.
// Mirrors htmlrender.sortedByID.
func sortedEntitiesByID(in []*entity.Entity) []*entity.Entity {
	out := append([]*entity.Entity(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// acRollup mirrors htmlrender's defaultResolver acMetTotal so the
// cmd-side resolver doesn't import unexported helpers from the
// htmlrender package.
func acRollup(acs []entity.AcceptanceCriterion) (met, total int) {
	for _, ac := range acs {
		if ac.Status == entity.StatusCancelled {
			continue
		}
		total++
		if ac.Status == entity.StatusMet {
			met++
		}
	}
	return met, total
}

// shortSHA returns the first n characters of sha when it's longer
// than n; otherwise returns sha unchanged. Used for the scopes
// table where 8-char short forms are easier to scan.
func shortSHA(sha string, n int) string {
	if len(sha) > n {
		return sha[:n]
	}
	return sha
}

// dateOnlyOrEmpty returns the YYYY-MM-DD prefix of an ISO timestamp,
// or "" when the input is shorter than 10 chars (i.e., absent).
func dateOnlyOrEmpty(s string) string {
	if len(s) < 10 {
		return ""
	}
	return s[:10]
}

// errorf is a tiny adapter matching the package's error-wrapping
// convention without pulling fmt into more files than necessary.
//
// Reserved for future use when the resolver gains validation that
// surfaces as wrapped errors. Unused today.
var _ = errorf

func errorf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

// strings import guard — keeps gofumpt happy while strings stays
// reserved for trailing-slash-style normalization the resolver
// will need when incremental --scope rendering lands in step 4's
// follow-up.
var _ = strings.TrimSpace
