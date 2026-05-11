package htmlrender

import (
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// defaultResolver builds page data straight from the in-memory tree —
// no git access, no scopes, no history. Used when the caller passes
// a nil PageDataResolver and as a baseline fixture for the package's
// own tests. Production callers (cmd/aiwf) supply a richer resolver
// that walks `git log`, `aiwf history`, and the scope FSM.
type defaultResolver struct {
	tree *tree.Tree
}

// IndexData implements PageDataResolver. Walks every epic and counts
// AC met / total per milestone, rolling them up to the epic line.
//
// Active-default: archived epics (whose path is under
// work/epics/archive/) are filtered out; the full set is reachable
// via epics-all.html. M-0087/AC-6.
//
// KindIndexLinks populates the home page's "Browse by kind" nav
// block — one entry per kind that participates in the per-kind
// active+all page family.
func (r defaultResolver) IndexData() (*IndexData, error) {
	sidebar := r.sidebar("", "")
	sidebar.IsCurrentIndex = true
	out := &IndexData{Title: "Overview", Sidebar: sidebar}
	for _, e := range sortedByID(r.tree.ByKind(entity.KindEpic)) {
		if entity.IsArchivedPath(e.Path) {
			continue
		}
		canonEpic := entity.Canonicalize(e.ID)
		summary := EpicSummary{
			ID:       canonEpic,
			Title:    e.Title,
			Status:   e.Status,
			FileName: idToFileName(e.ID),
		}
		for _, m := range sortedByID(r.tree.ByKind(entity.KindMilestone)) {
			if entity.Canonicalize(m.Parent) != canonEpic {
				continue
			}
			summary.MilestoneCount++
			met, total := acMetTotal(m.ACs)
			summary.ACMet += met
			summary.ACTotal += total
		}
		out.Epics = append(out.Epics, summary)
	}
	out.KindIndexLinks = r.buildKindIndexLinks()
	return out, nil
}

// buildKindIndexLinks enumerates the per-kind nav entries displayed
// on the home page's "Browse by kind" block. Counts are split into
// active and archived so the user sees at a glance how much of the
// kind is reachable from the active-default page vs. only from the
// all-set page. M-0087/AC-6.
func (r defaultResolver) buildKindIndexLinks() []KindIndexLink {
	links := []KindIndexLink{
		{Kind: "gaps", FileName: "gaps.html", AllFileName: "gaps-all.html"},
		{Kind: "decisions", FileName: "decisions.html", AllFileName: "decisions-all.html"},
		{Kind: "adrs", FileName: "adrs.html", AllFileName: "adrs-all.html"},
		{Kind: "contracts", FileName: "contracts.html", AllFileName: "contracts-all.html"},
	}
	for i := range links {
		k, _ := kindPluralToKind(links[i].Kind)
		for _, e := range r.tree.ByKind(k) {
			if entity.IsArchivedPath(e.Path) {
				links[i].ArchivedCount++
			} else {
				links[i].ActiveCount++
			}
		}
	}
	return links
}

// StatusData implements PageDataResolver. The default resolver
// (used by the htmlrender package's own tests) returns nil so the
// status page is skipped — production callers (cmd/aiwf) override
// this with a resolver that walks git for the recent activity and
// reuses the existing buildStatus() helper.
func (r defaultResolver) StatusData() (*StatusData, error) {
	return nil, nil
}

// sidebar builds the SidebarData payload for the page being
// rendered. activeEpicID names the epic to mark active (the page's
// own id when rendering an epic, the parent for a milestone page,
// "" for the index). activeMilestoneID names the current milestone
// (when rendering one) so the link can carry aria-current="page".
func (r defaultResolver) sidebar(activeEpicID, activeMilestoneID string) SidebarData {
	canonActiveEpic := entity.Canonicalize(activeEpicID)
	canonActiveMilestone := entity.Canonicalize(activeMilestoneID)
	var s SidebarData
	for _, e := range sortedByID(r.tree.ByKind(entity.KindEpic)) {
		canonEpic := entity.Canonicalize(e.ID)
		entry := SidebarEpic{
			ID:        canonEpic,
			Title:     e.Title,
			FileName:  idToFileName(e.ID),
			IsActive:  canonEpic == canonActiveEpic,
			IsCurrent: canonEpic == canonActiveEpic && activeMilestoneID == "",
		}
		for _, m := range sortedByID(r.tree.ByKind(entity.KindMilestone)) {
			if entity.Canonicalize(m.Parent) != canonEpic {
				continue
			}
			canonM := entity.Canonicalize(m.ID)
			entry.Milestones = append(entry.Milestones, SidebarMilestone{
				ID:        canonM,
				Title:     m.Title,
				FileName:  idToFileName(m.ID),
				IsCurrent: canonM == canonActiveMilestone,
			})
		}
		s.Epics = append(s.Epics, entry)
	}
	// M-0100/AC-1: GapCount surfaces the non-archived gap count in
	// the sidebar's top section. Path-based archived determination
	// (per ADR-0004) keeps this independent of frontmatter status.
	for _, g := range r.tree.ByKind(entity.KindGap) {
		if !entity.IsArchivedPath(g.Path) {
			s.GapCount++
		}
	}
	return s
}

// EpicData implements PageDataResolver. No history / linked-entities
// resolution — the cmd-side resolver handles those; this default is
// the minimum shape templates can render.
func (r defaultResolver) EpicData(id string) (*EpicData, error) {
	e := r.tree.ByID(id)
	if e == nil || e.Kind != entity.KindEpic {
		return nil, nil
	}
	canonEpic := entity.Canonicalize(e.ID)
	data := &EpicData{
		Epic: &EntityRef{
			ID:       canonEpic,
			Title:    e.Title,
			Status:   e.Status,
			Kind:     string(e.Kind),
			Path:     e.Path,
			FileName: idToFileName(e.ID),
		},
		Sidebar: r.sidebar(e.ID, ""),
	}
	for _, m := range sortedByID(r.tree.ByKind(entity.KindMilestone)) {
		if entity.Canonicalize(m.Parent) != canonEpic {
			continue
		}
		canonM := entity.Canonicalize(m.ID)
		met, total := acMetTotal(m.ACs)
		data.Milestones = append(data.Milestones, MilestoneSummary{
			ID:       canonM,
			Title:    m.Title,
			Status:   m.Status,
			TDD:      m.TDD,
			FileName: idToFileName(m.ID),
			ACMet:    met,
			ACTotal:  total,
		})
		data.ACMet += met
		data.ACTotal += total
		for _, dep := range m.DependsOn {
			data.DependencyDAG = append(data.DependencyDAG, DependencyEdge{
				From: canonM,
				To:   entity.Canonicalize(dep),
			})
		}
	}
	return data, nil
}

// EntityData implements PageDataResolver for gap, ADR, decision,
// and contract pages. The default resolver returns frontmatter +
// sidebar only — Sections (body markdown) requires reading the
// entity file from disk, which the cmd-side resolver handles. The
// page still renders (G35: no more 404s) showing id/title/status,
// just without the body prose; cmd/aiwf supplies the richer view.
//
// Returns nil when id resolves to a kind that has its own dedicated
// template (epic, milestone) so the renderer doesn't double-emit.
func (r defaultResolver) EntityData(id string) (*EntityData, error) {
	e := r.tree.ByID(id)
	if e == nil {
		return nil, nil
	}
	switch e.Kind {
	case entity.KindEpic, entity.KindMilestone:
		return nil, nil
	case entity.KindGap, entity.KindADR, entity.KindDecision, entity.KindContract:
		// fall through
	default:
		return nil, nil
	}
	return &EntityData{
		Entity: &EntityRef{
			ID:       entity.Canonicalize(e.ID),
			Title:    e.Title,
			Status:   e.Status,
			Kind:     string(e.Kind),
			Path:     e.Path,
			FileName: idToFileName(e.ID),
			Archived: entity.IsArchivedPath(e.Path),
		},
		Sidebar: r.sidebar("", ""),
	}, nil
}

// KindIndexData implements PageDataResolver. Builds the per-kind
// listing payload for the active-default (`<kind>.html`) and all-set
// (`<kind>-all.html`) pages. Returns nil for unrecognized kind slugs
// so the renderer skips emission cleanly.
//
// Filtering: when includeArchived is false, entries whose path lives
// under `<kind>/archive/` are excluded. Sorting is by canonical id.
//
// M-0087/AC-6 + AC-7.
func (r defaultResolver) KindIndexData(kind string, includeArchived bool) (*KindIndexData, error) {
	resolved, ok := kindPluralToKind(kind)
	if !ok {
		return nil, nil
	}
	title := titleForKindIndex(kind, includeArchived)
	data := &KindIndexData{
		Sidebar:         r.sidebar("", ""),
		Title:           title,
		Kind:            kind,
		IncludeArchived: includeArchived,
		ActiveFileName:  kind + ".html",
		AllFileName:     kind + "-all.html",
	}
	for _, e := range sortedByID(r.tree.ByKind(resolved)) {
		isArchived := entity.IsArchivedPath(e.Path)
		if isArchived && !includeArchived {
			continue
		}
		data.Entries = append(data.Entries, KindIndexEntry{
			ID:       entity.Canonicalize(e.ID),
			Title:    e.Title,
			Status:   e.Status,
			FileName: idToFileName(e.ID),
			Archived: isArchived,
		})
	}
	return data, nil
}

// kindPluralToKind maps the URL-facing plural slug back to the
// closed-set Kind value. Single source of truth for the active+all
// page family — the slug also drives the filename and the
// kind-index nav link.
func kindPluralToKind(plural string) (entity.Kind, bool) {
	switch plural {
	case "epics":
		return entity.KindEpic, true
	case "gaps":
		return entity.KindGap, true
	case "decisions":
		return entity.KindDecision, true
	case "adrs":
		return entity.KindADR, true
	case "contracts":
		return entity.KindContract, true
	}
	return "", false
}

// titleForKindIndex returns the page title for a per-kind index.
// Format: "Gaps" / "All gaps" so the title plus the page's filename
// together make the active/all distinction unambiguous.
func titleForKindIndex(plural string, includeArchived bool) string {
	if includeArchived {
		return "All " + plural
	}
	// Capitalize the first ASCII byte; the closed plural set is
	// known-ASCII so no Unicode upper-case handling is needed.
	title := plural
	if title != "" && title[0] >= 'a' && title[0] <= 'z' {
		title = string(rune(title[0])-32) + title[1:]
	}
	return title
}

// MilestoneData implements PageDataResolver. No body / history /
// scopes — those need git access (cmd-side resolver). The default
// surface is enough to render the Overview and Manifest tabs from
// frontmatter alone.
func (r defaultResolver) MilestoneData(id string) (*MilestoneData, error) {
	m := r.tree.ByID(id)
	if m == nil || m.Kind != entity.KindMilestone {
		return nil, nil
	}
	met, total := acMetTotal(m.ACs)
	data := &MilestoneData{
		Milestone: &EntityRef{
			ID:       entity.Canonicalize(m.ID),
			Title:    m.Title,
			Status:   m.Status,
			Kind:     string(m.Kind),
			Path:     m.Path,
			FileName: idToFileName(m.ID),
			TDD:      m.TDD,
		},
		ACMet:   met,
		ACTotal: total,
		Sidebar: r.sidebar(m.Parent, m.ID),
	}
	if m.Parent != "" {
		if parent := r.tree.ByID(m.Parent); parent != nil {
			data.ParentEpic = &EntityRef{
				ID:       entity.Canonicalize(parent.ID),
				Title:    parent.Title,
				Status:   parent.Status,
				Kind:     string(parent.Kind),
				FileName: idToFileName(parent.ID),
			}
		}
	}
	for _, ac := range m.ACs {
		data.ACs = append(data.ACs, ACDetail{
			ID:       ac.ID,
			Title:    ac.Title,
			Status:   ac.Status,
			TDDPhase: ac.TDDPhase,
			Anchor:   ACAnchor(ac.ID),
		})
	}
	return data, nil
}

// acMetTotal returns the (met, total-cancelled) counts for a slice
// of ACs. The progress metric on every page is "met / (total -
// cancelled)" — cancelled ACs don't count toward the denominator,
// matching what `aiwf show` already enforces in its renderer.
func acMetTotal(acs []entity.AcceptanceCriterion) (met, total int) {
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
