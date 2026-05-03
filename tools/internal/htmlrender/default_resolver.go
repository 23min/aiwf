package htmlrender

import (
	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
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
	out := &IndexData{Title: "Governance", Sidebar: r.sidebar("", "")}
	for _, e := range sortedByID(r.tree.ByKind(entity.KindEpic)) {
		summary := EpicSummary{
			ID:       e.ID,
			Title:    e.Title,
			Status:   e.Status,
			FileName: idToFileName(e.ID),
		}
		for _, m := range sortedByID(r.tree.ByKind(entity.KindMilestone)) {
			if m.Parent != e.ID {
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

// sidebar builds the SidebarData payload for the page being
// rendered. activeEpicID names the epic to mark active (the page's
// own id when rendering an epic, the parent for a milestone page,
// "" for the index). activeMilestoneID names the current milestone
// (when rendering one) so the link can carry aria-current="page".
func (r defaultResolver) sidebar(activeEpicID, activeMilestoneID string) SidebarData {
	var s SidebarData
	for _, e := range sortedByID(r.tree.ByKind(entity.KindEpic)) {
		entry := SidebarEpic{
			ID:        e.ID,
			Title:     e.Title,
			FileName:  idToFileName(e.ID),
			IsActive:  e.ID == activeEpicID,
			IsCurrent: e.ID == activeEpicID && activeMilestoneID == "",
		}
		for _, m := range sortedByID(r.tree.ByKind(entity.KindMilestone)) {
			if m.Parent != e.ID {
				continue
			}
			entry.Milestones = append(entry.Milestones, SidebarMilestone{
				ID:        m.ID,
				Title:     m.Title,
				FileName:  idToFileName(m.ID),
				IsCurrent: m.ID == activeMilestoneID,
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
	data := &EpicData{
		Epic: &EntityRef{
			ID:       e.ID,
			Title:    e.Title,
			Status:   e.Status,
			Kind:     string(e.Kind),
			Path:     e.Path,
			FileName: idToFileName(e.ID),
		},
		Sidebar: r.sidebar(e.ID, ""),
	}
	for _, m := range sortedByID(r.tree.ByKind(entity.KindMilestone)) {
		if m.Parent != e.ID {
			continue
		}
		met, total := acMetTotal(m.ACs)
		data.Milestones = append(data.Milestones, MilestoneSummary{
			ID:       m.ID,
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
			data.DependencyDAG = append(data.DependencyDAG, DependencyEdge{From: m.ID, To: dep})
		}
	}
	return data, nil
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
			ID:       m.ID,
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
				ID:       parent.ID,
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
