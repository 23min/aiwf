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
func (r defaultResolver) IndexData() (*IndexData, error) {
	sidebar := r.sidebar("", "")
	sidebar.IsCurrentIndex = true
	out := &IndexData{Title: "Overview", Sidebar: sidebar}
	for _, e := range sortedByID(r.tree.ByKind(entity.KindEpic)) {
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
	return out, nil
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
		},
		Sidebar: r.sidebar("", ""),
	}, nil
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
