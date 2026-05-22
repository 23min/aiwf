package status

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/tree"
)

// WorktreeView is the per-worktree row in `aiwf status --worktrees`
// output. One per `git worktree list --porcelain` entry.
//
// G-0122.
type WorktreeView struct {
	Path           string    `json:"path"`
	Branch         string    `json:"branch"`
	HeadTime       time.Time `json:"head_time,omitempty"`        // author-date of the HEAD commit on this worktree's branch (G-0122 age display)
	CreatedTime    time.Time `json:"created_time,omitempty"`     // author-date of the first ahead-of-trunk commit on this branch (worktree creation proxy)
	LastEntityTime time.Time `json:"last_entity_time,omitempty"` // author-date of the most recent aiwf-verb-trailered commit on this branch
	Dirty          bool      `json:"dirty,omitempty"`            // true when `git status --porcelain` reports any uncommitted changes (G-0122 option A)
	DriverEntityID string    `json:"driver_entity_id,omitempty"`
	DriverKind     string    `json:"driver_kind,omitempty"` // "epic" / "milestone" / "gap"
	DriverStatus   string    `json:"driver_status,omitempty"`
	DriverTitle    string    `json:"driver_title,omitempty"`
	Stale          bool      `json:"stale,omitempty"`
	// AheadOfTrunk is the count of commits on this worktree's branch
	// that are ahead of main. Zero on git failure or when the branch
	// is fully merged into trunk. Used by the stale-rendering arm to
	// distinguish wrap-pending (driver terminal + ahead > 0; merge
	// before removal) from safe-to-remove (driver terminal + ahead = 0;
	// commits are on trunk so the worktree can be removed). G-0153.
	AheadOfTrunk int `json:"ahead_of_trunk,omitempty"`
	// Populated only when DriverKind == "epic": milestones under this
	// epic + gaps the epic (or its milestones) closes + gaps the epic
	// (or its milestones) surfaced.
	EpicMilestones   []EpicChildRow `json:"epic_milestones,omitempty"`
	EpicClosesGaps   []EpicChildRow `json:"epic_closes_gaps,omitempty"`
	EpicSurfacedGaps []EpicChildRow `json:"epic_surfaced_gaps,omitempty"`
	// Milestone-driver breadcrumb + AC enumeration + related rows.
	ParentEpicID     string         `json:"parent_epic_id,omitempty"`
	ParentEpicTitle  string         `json:"parent_epic_title,omitempty"`
	ParentEpicStatus string         `json:"parent_epic_status,omitempty"`
	ACs              []ACRow        `json:"acs,omitempty"`
	DependsOn        []EpicChildRow `json:"depends_on,omitempty"`
	SurfacedGaps     []EpicChildRow `json:"surfaced_gaps,omitempty"` // gaps with discovered_in == driver milestone
	// OtherInFlight is populated only on the main-checkout worktree —
	// in-flight entities that no worktree drives, either on a branch
	// that has no worktree or directly on trunk (no dedicated branch).
	OtherInFlight []OtherInFlightRow `json:"other_in_flight,omitempty"`
}

// EpicChildRow is one row under an epic-driver worktree's expanded
// listing: a milestone or a gap the epic owns / closes.
type EpicChildRow struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	DrivenByPath string `json:"driven_by_path,omitempty"` // worktree path driving this child, if any (omit when self-driven)
}

// OtherInFlightRow is one in-flight entity that no worktree is driving
// — either a dedicated branch exists but is not checked out anywhere
// (Branch != ""), or no dedicated branch exists at all and work happens
// directly on trunk (Branch == ""). Rendered under the main-checkout
// worktree's section per G-0122 user feedback "work might be on a
// branch but not on a worktree".
type OtherInFlightRow struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Status     string    `json:"status"`
	Branch     string    `json:"branch,omitempty"`      // empty = no dedicated branch (work on trunk)
	BranchTime time.Time `json:"branch_time,omitempty"` // HEAD commit time on the branch (zero when no branch)
}

// ACRow is one acceptance-criterion row under a milestone-driver
// worktree. Status is the AC's status (open / met / cancelled /
// deferred); TDDPhase is the optional phase (red / green / refactor /
// done) when the parent milestone is `tdd: required` / `advisory`.
type ACRow struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	TDDPhase string `json:"tdd_phase,omitempty"`
}

// BuildWorktreeViews enumerates the repo's worktrees, correlates each
// to an entity via the hybrid cascade (scope-defining events →
// trailer-recency → branch-name parsing), and returns one WorktreeView
// per worktree.
//
// Status fields on each row (driver, parent epic, depends_on, ACs,
// surfaced gaps, epic-expansion children) are resolved from the
// *worktree's own* loaded tree — entity files as they exist on that
// worktree's branch, not on main. The main-checkout worktree's path
// equals rootDir so it reuses the passed-in tr without re-loading.
// If a worktree-side load fails the worktree falls back to tr; the
// renderer degrades to "main-tree status" rather than dropping the
// worktree section.
//
// rootDir is the consumer repo root (any worktree's path resolves the
// shared git dir). tr is the loaded entity tree for the main checkout.
// ctx scopes the git subprocess calls.
func BuildWorktreeViews(ctx context.Context, rootDir string, tr *tree.Tree) ([]WorktreeView, error) {
	worktrees, err := gitops.ListWorktrees(ctx, rootDir)
	if err != nil {
		return nil, fmt.Errorf("listing worktrees: %w", err)
	}
	views := make([]WorktreeView, 0, len(worktrees))
	for _, wt := range worktrees {
		v := WorktreeView{Path: wt.Path, Branch: wt.Branch}
		wtTree := worktreeTree(ctx, wt.Path, rootDir, tr)
		// HEAD time: author-date of the most recent commit on this
		// worktree's branch. Best-effort — git failure leaves
		// HeadTime zero and the renderer omits the line. Use the
		// worktree's path so detached-HEAD worktrees still resolve
		// (querying by branch name would fail there).
		if t, err := worktreeHeadTime(ctx, wt.Path); err == nil {
			v.HeadTime = t
		}
		// Dirty check: any uncommitted changes (staged, unstaged, or
		// untracked) in this worktree. Best-effort — git failure
		// leaves Dirty as false so a transient failure doesn't make
		// the operator chase a phantom dirty flag.
		v.Dirty = worktreeIsDirty(ctx, wt.Path)
		// Creation proxy: author-date of the first ahead-of-trunk
		// commit on the branch (when the worktree's work started
		// diverging from main). Best-effort — empty for branches
		// without ahead-of-trunk commits (fresh worktree, main itself,
		// or detached HEAD).
		if !v.HeadTime.IsZero() && wt.Branch != "" && wt.Branch != "main" {
			if t, err := branchFirstAheadCommitTime(ctx, rootDir, wt.Branch); err == nil {
				v.CreatedTime = t
			}
			// Ahead-of-trunk count: drives the stale-arm's
			// wrap-pending-vs-safe-to-remove decision when the driver
			// is terminal. Best-effort; zero on any git failure. G-0153.
			v.AheadOfTrunk = branchAheadOfTrunkCount(ctx, rootDir, wt.Branch)
			// Last entity touch: most recent commit on the branch
			// with an aiwf-verb trailer. Best-effort; zero when no
			// aiwf-verb commits exist on the branch.
			if t, err := branchLastEntityCommitTime(ctx, rootDir, wt.Branch); err == nil {
				v.LastEntityTime = t
			}
		}
		if wt.Branch == "" {
			// Detached HEAD — no correlation possible.
			views = append(views, v)
			continue
		}
		driverID := correlateBranchToEntity(ctx, rootDir, wt.Branch)
		if driverID == "" {
			views = append(views, v)
			continue
		}
		e := wtTree.ByID(driverID)
		if e == nil {
			// Correlated to an id that's not in the tree (renamed, archived
			// beyond the current load, or otherwise unresolvable). Treat as
			// no-driver — operator sees the worktree without a misleading
			// label.
			views = append(views, v)
			continue
		}
		v.DriverEntityID = e.ID
		v.DriverKind = string(e.Kind)
		v.DriverStatus = e.Status
		v.DriverTitle = e.Title
		v.Stale = isTerminalStatus(e.Kind, e.Status)
		switch e.Kind {
		case entity.KindEpic:
			v.EpicMilestones, v.EpicClosesGaps, v.EpicSurfacedGaps = epicExpansion(wtTree, e.ID, worktrees)
		case entity.KindMilestone:
			if parent := wtTree.ByID(e.Parent); parent != nil {
				v.ParentEpicID = parent.ID
				v.ParentEpicTitle = parent.Title
				v.ParentEpicStatus = parent.Status
			}
			for _, ac := range e.ACs {
				v.ACs = append(v.ACs, ACRow{ID: ac.ID, Title: ac.Title, Status: ac.Status, TDDPhase: ac.TDDPhase})
			}
			// depends_on enumeration with resolved title/status.
			// Try the worktree tree first (the operator cares about
			// state on their branch); fall back to the main tree when
			// the dep isn't present locally (typical case: the dep
			// milestone was added on main after this worktree branched,
			// or the dep lives on another worktree that may never merge,
			// so the main tree is the only safe public reference).
			for _, depID := range e.DependsOn {
				row := EpicChildRow{ID: depID, Title: "(unknown)", Status: "?"}
				dep := wtTree.ByID(depID)
				if dep == nil && wtTree != tr {
					dep = tr.ByID(depID)
				}
				if dep != nil {
					row.ID = dep.ID
					row.Title = dep.Title
					row.Status = dep.Status
				}
				v.DependsOn = append(v.DependsOn, row)
			}
			// Surfaced gaps: every gap whose discovered_in references
			// this milestone. Both narrow and canonical id widths
			// resolve via entity.Canonicalize.
			driverCanonical := entity.Canonicalize(e.ID)
			for _, g := range wtTree.ByKind(entity.KindGap) {
				if g.DiscoveredIn == "" {
					continue
				}
				if entity.Canonicalize(g.DiscoveredIn) == driverCanonical {
					v.SurfacedGaps = append(v.SurfacedGaps, EpicChildRow{ID: g.ID, Title: g.Title, Status: g.Status})
				}
			}
			sort.SliceStable(v.SurfacedGaps, func(i, j int) bool { return v.SurfacedGaps[i].ID < v.SurfacedGaps[j].ID })
		default:
			// Gaps, decisions, ADRs, contracts: driver row is enough,
			// no kind-specific expansion. Branch-name parsing won't
			// produce ADR/decision/contract drivers today (only epic /
			// milestone / gap shapes are recognized), so this branch
			// is effectively the gap-driver case in practice.
		}
		views = append(views, v)
	}
	// G-0122 user-feedback extension: enumerate in-flight entities
	// not driven by any worktree, attach to the main-checkout
	// worktree so the operator sees "what's in flight but loose"
	// alongside the trunk row.
	worktreeDriverIDs := map[string]bool{}
	for i := range views {
		if views[i].DriverEntityID != "" {
			worktreeDriverIDs[entity.Canonicalize(views[i].DriverEntityID)] = true
		}
	}
	branches := listBranchesWithAge(ctx, rootDir)
	other := buildOtherInFlight(tr, worktreeDriverIDs, branches)
	if len(other) > 0 {
		for i := range views {
			if views[i].Branch == "main" {
				views[i].OtherInFlight = other
				break
			}
		}
	}
	return views, nil
}

// worktreeTree returns the loaded entity tree for the worktree at path.
// The main checkout's path equals rootDir so the passed-in main tree is
// reused without a duplicate disk walk. For non-main worktrees, the
// tree is loaded from the worktree's path so per-row status reflects
// the worktree's branch — what the operator has actually committed
// there — rather than the main-tree's stale view.
//
// Best-effort: on load failure (missing work/ dir, malformed
// frontmatter, etc.) the worktree falls back to the main tree. The
// load's findings/loadErrs are discarded — the main tree's load already
// surfaced repo-level errors to the operator via BuildStatus, and a
// per-worktree degradation should not block the wider --worktrees
// output.
func worktreeTree(ctx context.Context, path, rootDir string, mainTree *tree.Tree) *tree.Tree {
	if path == "" || path == rootDir {
		return mainTree
	}
	loaded, _, err := tree.Load(ctx, path)
	if err != nil {
		return mainTree
	}
	return loaded
}

// correlateBranchToEntity correlates a worktree's branch to the entity
// it drives. Per G-0154, branch-name parsing is the *primary* signal
// when the branch follows a ritual shape (epic/E-NNN-…,
// milestone/M-NNN-…, patch/[Gg]-NNN-…); the operator named it
// deliberately and that intent must beat any trailer-derived
// inference. The trailer cascade is the fallback for non-ritual
// branches where the entity scope must be inferred from work history:
//
//  1. **Branch-name parse** (G-0154): if `branchEntityPattern` matches,
//     return that entity. Ritual branch shapes are deliberate operator
//     intent; honoring them avoids the post-merge mislabeling where a
//     child milestone's promote-trailer (pulled onto the epic branch by
//     a merge) would otherwise beat the epic's branch name.
//  2. **Scope-defining events** (G-0122): walk `git log main..<branch>`
//     for `aiwf-verb:` trailers (authorize, promote-to-active/
//     in_progress, --phase promotes). Single-entity match wins;
//     multi-entity prefers the most-recent active-state event.
//  3. **Trailer recency** (G-0122): the most recent `aiwf-entity:`
//     trailer on any ahead-of-trunk commit.
//  4. Return "" when none of the above resolve.
//
// "main" is the trunk reference; if it doesn't exist (a repo whose
// trunk is named differently, or a fresh worktree before main exists),
// the function returns "" and the renderer treats the worktree as
// uncorrelated — the operator sees it under "Trunk / no in-flight
// scope," not as a false positive.
func correlateBranchToEntity(ctx context.Context, rootDir, branch string) string {
	if branch == "main" {
		// The trunk itself never drives a specific entity by definition.
		// Skip the git-log walk; branch-parse on "main" would also yield
		// nothing.
		return ""
	}
	// G-0154: ritual branch names are the operator's deliberate
	// declaration of intent. Honor them ahead of trailer inference so
	// an `epic/E-NNN-...` worktree that has just merged its child
	// milestones is still labeled as driving E-NNN.
	if id := parseEntityFromBranch(branch); id != "" {
		return id
	}
	return correlateFromTrailerEvents(branchAiwfEvents(ctx, rootDir, branch))
}

// correlateFromTrailerEvents derives an entity id from the trailer
// cascade alone — the fallback path of correlateBranchToEntity for
// non-ritual branches where the branch name carries no entity.
// Factored out as a pure function so the cascade ordering can be
// unit-tested without shelling to git. G-0154.
func correlateFromTrailerEvents(events []branchAiwfEventRecord) string {
	if id := scopeDefiningEntity(events); id != "" {
		return id
	}
	return mostRecentEntity(events)
}

// branchAiwfEventRecord is one ahead-of-trunk commit with the trailers
// the correlator cares about. Newest-first order (git log default).
type branchAiwfEventRecord struct {
	Verb   string // aiwf-verb: value (e.g. "promote", "authorize", "add")
	Entity string // aiwf-entity: value (e.g. "M-0123", "E-0033", "G-0146")
	To     string // aiwf-to: value (e.g. "in_progress", "active", "green")
}

// branchAiwfEvents returns the aiwf-verb / aiwf-entity / aiwf-to
// trailers from every commit on `branch` ahead of `main`. Newest first.
// Empty when `branch == main` or when there's no merge-base (separate
// histories). git invocation failure returns empty (treat as "no
// ahead-of-trunk events", not a fatal error — the worktree view
// degrades gracefully to branch-name parsing).
func branchAiwfEvents(ctx context.Context, rootDir, branch string) []branchAiwfEventRecord {
	const sep = "\x1f"
	const recSep = "\x1e\n"
	cmd := exec.CommandContext(
		ctx, "git",
		"log",
		"main.."+branch,
		"--pretty=tformat:"+
			"%(trailers:key=aiwf-verb,valueonly=true,unfold=true)"+sep+
			"%(trailers:key=aiwf-entity,valueonly=true,unfold=true)"+sep+
			"%(trailers:key=aiwf-to,valueonly=true,unfold=true)\x1e",
	)
	cmd.Dir = rootDir
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var events []branchAiwfEventRecord
	for _, rec := range strings.Split(string(out), recSep) {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, sep, 3)
		if len(parts) < 3 {
			continue
		}
		entityField := strings.TrimSpace(parts[1])
		if entityField == "" {
			continue
		}
		events = append(events, branchAiwfEventRecord{
			Verb:   strings.TrimSpace(parts[0]),
			Entity: entityField,
			To:     strings.TrimSpace(parts[2]),
		})
	}
	return events
}

// scopeDefiningEntity walks events (newest first) and returns the
// driver entity if any commit carries a scope-defining trailer
// combination. Returns "" when no scope event found.
//
// Scope-defining patterns:
//   - aiwf-verb: authorize  (any entity)
//   - aiwf-verb: promote + aiwf-to: active        (epic activation)
//   - aiwf-verb: promote + aiwf-to: in_progress   (milestone driver)
//   - aiwf-verb: promote + aiwf-to: red/green/refactor/done  (phase work)
//     For composite ids like M-NNN/AC-N, the parent milestone is the
//     driver — strip "/AC-N" before returning.
//
// Multi-entity disambiguation: when multiple entities have scope
// events, prefer the most recent active-state event (any of authorize /
// active / in_progress / red / green / refactor — non-`done`).
func scopeDefiningEntity(events []branchAiwfEventRecord) string {
	var firstActive, firstAny string
	activeStates := map[string]bool{
		entity.StatusActive:     true,
		entity.StatusInProgress: true,
		entity.TDDPhaseRed:      true,
		entity.TDDPhaseGreen:    true,
		entity.TDDPhaseRefactor: true,
	}
	for _, e := range events {
		id := parentEntity(e.Entity)
		isScope := e.Verb == "authorize" ||
			(e.Verb == "promote" && (activeStates[e.To] || e.To == entity.StatusDone))
		if !isScope {
			continue
		}
		if firstAny == "" {
			firstAny = id
		}
		if firstActive == "" && (e.Verb == "authorize" || activeStates[e.To]) {
			firstActive = id
		}
	}
	if firstActive != "" {
		return firstActive
	}
	return firstAny
}

// mostRecentEntity returns the entity from the newest aiwf-entity
// trailer in events. Returns "" when events is empty.
func mostRecentEntity(events []branchAiwfEventRecord) string {
	if len(events) == 0 {
		return ""
	}
	return parentEntity(events[0].Entity)
}

// parentEntity strips a composite-id suffix (`/AC-N`) so AC-level
// events resolve to the parent milestone's id.
func parentEntity(id string) string {
	if i := strings.Index(id, "/"); i >= 0 {
		return id[:i]
	}
	return id
}

// branchEntityPattern matches the conventional ritual-branch prefixes:
//
//	epic/E-NNNN-<slug>          → E-NNNN
//	milestone/M-NNNN-<slug>     → M-NNNN
//	patch/g-NNNN-<slug>         → G-NNNN (case-insensitive id segment)
//
// Other shapes (fix/*, chore/*, patch/<topic-without-id>) yield "".
var branchEntityPattern = regexp.MustCompile(`^(?:epic|milestone|patch)/([EeMmGg]-\d+)(?:-|$)`)

// parseEntityFromBranch tries to derive an entity id from the branch
// name when the hybrid cascade's git-log walk found nothing. Honors
// the conventional `epic/E-NNNN-...`, `milestone/M-NNNN-...`,
// `patch/g-NNNN-...` shapes. Returns "" on no match.
func parseEntityFromBranch(branch string) string {
	m := branchEntityPattern.FindStringSubmatch(branch)
	if m == nil {
		return ""
	}
	id := strings.ToUpper(m[1])
	return id
}

// isTerminalStatus reports whether the kind's status is a terminal
// state (done / cancelled / wontfix / rejected / addressed / retired /
// superseded). Mirrors entity.IsTerminalStatus when present; falls back
// to a closed-set check here so the worktree view doesn't pull in a
// package-level dependency for a narrowly-scoped check.
func isTerminalStatus(kind entity.Kind, status string) bool {
	switch status {
	case entity.StatusDone,
		entity.StatusCancelled,
		entity.StatusWontfix,
		entity.StatusRejected,
		entity.StatusAddressed,
		entity.StatusRetired,
		entity.StatusSuperseded,
		entity.StatusDeprecated:
		return true
	}
	return false
}

// epicExpansion returns the milestone and gap children for an epic
// driver row. Milestones come from the tree's milestone-by-parent
// index; gaps come from any gap whose `addressed_by:` references the
// epic. Each child row carries the driving worktree path when another
// worktree is driving the same milestone (so a glance shows whether
// the milestone is in-flight elsewhere).
//
// epicID is the epic's id (canonical or narrow form — Canonicalize
// handles either). worktrees is the full ListWorktrees set so the
// cross-reference can name the other worktree.
func epicExpansion(tr *tree.Tree, epicID string, worktrees []gitops.Worktree) (milestones, closesGaps, surfacedGaps []EpicChildRow) {
	canonical := entity.Canonicalize(epicID)
	// Collect milestone children and a canonical-id set for the
	// surfaced-gaps query below.
	milestoneIDs := map[string]bool{}
	for _, m := range tr.ByKind(entity.KindMilestone) {
		if entity.Canonicalize(m.Parent) != canonical {
			continue
		}
		milestoneIDs[entity.Canonicalize(m.ID)] = true
		row := EpicChildRow{ID: m.ID, Title: m.Title, Status: m.Status}
		for _, wt := range worktrees {
			if wt.Branch == "" {
				continue
			}
			otherID := parseEntityFromBranch(wt.Branch)
			if otherID == m.ID && wt.Path != "" {
				row.DrivenByPath = wt.Path
				break
			}
		}
		milestones = append(milestones, row)
	}
	sort.SliceStable(milestones, func(i, j int) bool { return milestones[i].ID < milestones[j].ID })
	for _, g := range tr.ByKind(entity.KindGap) {
		// "Closes" — gap.addressed_by references this epic directly,
		// or any of its child milestones (so wrap-time closures via
		// `aiwf promote G-NNN addressed --by M-...` surface under the
		// epic-driver worktree view too).
		closed := false
		for _, ref := range g.AddressedBy {
			if entity.Canonicalize(ref) == canonical || milestoneIDs[entity.Canonicalize(ref)] {
				closed = true
				break
			}
		}
		if closed {
			closesGaps = append(closesGaps, EpicChildRow{ID: g.ID, Title: g.Title, Status: g.Status})
			continue
		}
		// "Surfaced" — gap.discovered_in points at this epic or any
		// of its milestones. Surface only when not already covered by
		// the closes-set above (a closed gap that was also surfaced
		// belongs under Closes, not duplicated).
		if g.DiscoveredIn == "" {
			continue
		}
		di := entity.Canonicalize(g.DiscoveredIn)
		if di == canonical || milestoneIDs[di] {
			surfacedGaps = append(surfacedGaps, EpicChildRow{ID: g.ID, Title: g.Title, Status: g.Status})
		}
	}
	sort.SliceStable(closesGaps, func(i, j int) bool { return closesGaps[i].ID < closesGaps[j].ID })
	sort.SliceStable(surfacedGaps, func(i, j int) bool { return surfacedGaps[i].ID < surfacedGaps[j].ID })
	return milestones, closesGaps, surfacedGaps
}

// RenderWorktreeViews writes one section per worktree to w. Each
// section carries the worktree path as a header, the branch on its own
// line prefixed with the bold ⎇ glyph, the driver entity row, and any
// kind-specific expansion (milestones+gaps under an epic, parent-epic
// breadcrumb + ACs under a milestone). Stale and trunk worktrees use
// the same shape with a one-line marker line — no separate top-level
// grouping.
//
// G-0122.
func RenderWorktreeViews(w io.Writer, views []WorktreeView, colorEnabled bool) error {
	if len(views) == 0 {
		_, err := fmt.Fprintln(w, "No worktrees found.")
		return err
	}
	// Two-line header: title + timestamp. Timestamp anchors the
	// relative ages in the body (every "Xh ago" reads against this
	// moment) and provides breathing room before the dense per-
	// worktree sections start. G-0122 user feedback.
	if _, err := fmt.Fprintln(w, render.Bold("In-flight work across worktrees", colorEnabled)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, render.Dim("generated "+time.Now().Format("2006-01-02 15:04"), colorEnabled)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	// Order: in-flight first, then trunk (no scope), then stale at the
	// end — the operator's eye lands on active work, the cleanup
	// candidates sink. Within each group, original ListWorktrees order
	// (git's administrative order) is preserved.
	ordered := orderWorktreesByActivity(views)
	for i := range ordered {
		if i > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		if err := renderWorktreeSection(w, ordered[i], colorEnabled); err != nil {
			return err
		}
	}
	return nil
}

// orderWorktreesByActivity sorts worktree views: in-flight (has a
// non-terminal driver entity) first, then trunk (no driver), then
// stale (terminal driver). Returns a slice of pointers into the
// original views (no copies).
func orderWorktreesByActivity(views []WorktreeView) []*WorktreeView {
	out := make([]*WorktreeView, 0, len(views))
	for i := range views {
		if views[i].DriverEntityID != "" && !views[i].Stale {
			out = append(out, &views[i])
		}
	}
	for i := range views {
		if views[i].DriverEntityID == "" {
			out = append(out, &views[i])
		}
	}
	for i := range views {
		if views[i].Stale {
			out = append(out, &views[i])
		}
	}
	return out
}

const branchGlyph = "⎇"

func renderWorktreeSection(w io.Writer, v *WorktreeView, colorEnabled bool) error {
	if _, err := fmt.Fprintf(w, "%s %s\n", render.Bold("Worktree:", colorEnabled), render.Bold(v.Path, colorEnabled)); err != nil {
		return err
	}
	// Branch + last-commit + dirty on one line (G-0122 user feedback
	// B + option A): consolidates secondary context, dimmed so the
	// eye lands on driver content. "last commit" labels HEAD age
	// honestly; "dirty" flags uncommitted working-tree changes so
	// the operator sees both "when did I commit" and "have I done
	// work since."
	secondaryLine := fmt.Sprintf("  %s %s", render.Bold(branchGlyph, colorEnabled), branchLabel(v.Branch))
	if !v.HeadTime.IsZero() {
		rel := relativeAge(time.Since(v.HeadTime))
		secondaryLine += render.Dim("  •  last commit "+rel, colorEnabled)
	}
	if v.Dirty {
		// "dirty" is colored (yellow) rather than dimmed so it stands
		// out — the operator wants to know about uncommitted work.
		secondaryLine += render.Dim("  •  ", colorEnabled) + render.StatusColor("dirty", entity.StatusInProgress, colorEnabled)
	}
	if _, err := fmt.Fprintln(w, secondaryLine); err != nil {
		return err
	}
	// Optional third metadata line: worktree creation (first ahead-
	// of-trunk commit) + last entity touch (last aiwf-verb-trailered
	// commit). Each metric is suppressed when it would be uselessly
	// redundant (created ≈ last update, or last entity ≈ last update
	// within an hour). G-0122 user-feedback extension.
	parts := worktreeMetadataLine(v, time.Now())
	if parts != "" {
		if _, err := fmt.Fprintln(w, render.Dim("  "+parts, colorEnabled)); err != nil {
			return err
		}
	}
	if v.DriverEntityID == "" {
		if len(v.OtherInFlight) > 0 {
			return renderOtherInFlight(w, v.OtherInFlight, time.Now(), colorEnabled)
		}
		_, err := fmt.Fprintln(w, render.Dim("  No in-flight scope (trunk)", colorEnabled))
		return err
	}
	// Stale worktrees branch three ways (G-0153):
	//   - cancelled / rejected / wontfix → "abandoned"; safe to remove
	//   - positive terminal + branch ahead of trunk → "wrap pending";
	//     the operator must merge before removing. Full body context
	//     (parent epic, ACs, depends_on, surfaced gaps) is preserved
	//     because the wrap step is still in front of them.
	//   - positive terminal + branch fully merged → "safe to remove"
	if v.Stale {
		return renderStaleSection(w, v, colorEnabled)
	}
	switch v.DriverKind {
	case string(entity.KindEpic):
		// Epic-driver: the epic itself is the driver row, then
		// milestones + closes-gaps + surfaced-gaps lists below.
		if _, err := fmt.Fprintf(w, "  %s — %s %s\n",
			render.Bold(v.DriverEntityID, colorEnabled),
			v.DriverTitle,
			render.StatusColor("["+v.DriverStatus+"]", v.DriverStatus, colorEnabled)); err != nil {
			return err
		}
		return renderEpicExpansion(w, v, colorEnabled)
	case string(entity.KindMilestone):
		// Milestone-driver: parent epic line first (G-0122 user
		// feedback: epics before milestones), then driven milestone
		// with `→ (driven)` marker, then depends_on / ACs /
		// surfaced-gaps lists. renderMilestoneDriver owns the whole
		// block including both the epic and milestone rows.
		return renderMilestoneDriver(w, v, colorEnabled)
	default:
		// Gap / decision / ADR / contract drivers: just the driver
		// row, no kind-specific expansion in this G-0122 patch.
		_, err := fmt.Fprintf(w, "  %s — %s %s\n",
			render.Bold(v.DriverEntityID, colorEnabled),
			v.DriverTitle,
			render.StatusColor("["+v.DriverStatus+"]", v.DriverStatus, colorEnabled))
		return err
	}
}

// renderStaleSection handles the three terminal-driver cases per G-0153.
// The shape of each case is driven by two questions:
//
//  1. Was the driver positively terminal (done/addressed/etc.) or
//     negatively terminal (cancelled/rejected/wontfix)? Negative
//     terminal is "abandoned" — the work isn't landing, so the
//     worktree can be cleaned up regardless of ahead-of-trunk count.
//  2. For positive terminal: are the branch's commits on trunk yet
//     (AheadOfTrunk == 0) or still local (> 0)? "Still local" means
//     the wrap step is pending — removing the worktree would drop
//     the working tree before the merge, so the cleanup hint is
//     inverted ("merge first" not "remove now").
//
// In all three cases the parent-epic breadcrumb is restored vs the
// prior implementation: terminal driver status doesn't change the
// fact that the worktree belongs under E-NNNN, and the active parent
// epic is exactly the context the operator wants while reading a
// done-but-not-yet-merged milestone ("right, this is the wrap step
// in E-NNNN").
func renderStaleSection(w io.Writer, v *WorktreeView, colorEnabled bool) error {
	cancelledFlavor := v.DriverStatus == entity.StatusCancelled ||
		v.DriverStatus == entity.StatusRejected ||
		v.DriverStatus == entity.StatusWontfix

	// Wrap-pending: positively terminal driver with unmerged commits.
	// Preserve the full body layout (parent epic, ACs, depends_on,
	// surfaced gaps) by delegating to the kind-specific in-flight
	// renderer, then append a WRAP PENDING marker that names the
	// pending step explicitly. No `git worktree remove` suggestion —
	// running it now would drop the wrap-step working tree.
	if !cancelledFlavor && v.AheadOfTrunk > 0 {
		switch v.DriverKind {
		case string(entity.KindEpic):
			if _, err := fmt.Fprintf(w, "  %s — %s %s\n",
				render.Bold(v.DriverEntityID, colorEnabled),
				v.DriverTitle,
				render.StatusColor("["+v.DriverStatus+"]", v.DriverStatus, colorEnabled)); err != nil {
				return err
			}
			if err := renderEpicExpansion(w, v, colorEnabled); err != nil {
				return err
			}
		case string(entity.KindMilestone):
			if err := renderMilestoneDriver(w, v, colorEnabled); err != nil {
				return err
			}
		default:
			if _, err := fmt.Fprintf(w, "  %s — %s %s\n",
				render.Bold(v.DriverEntityID, colorEnabled),
				v.DriverTitle,
				render.StatusColor("["+v.DriverStatus+"]", v.DriverStatus, colorEnabled)); err != nil {
				return err
			}
		}
		commitsWord := "commits"
		if v.AheadOfTrunk == 1 {
			commitsWord = "commit"
		}
		_, err := fmt.Fprintf(w, "  %s — driver %s but branch ahead of trunk by %d %s; merge to trunk before removing\n",
			render.Bold("WRAP PENDING", colorEnabled),
			v.DriverStatus,
			v.AheadOfTrunk,
			commitsWord)
		return err
	}

	// Compact rendering for safe-to-remove and abandoned cases.
	// Parent-epic breadcrumb comes first (when applicable) so the
	// operator sees the structural context before the cleanup hint.
	if v.DriverKind == string(entity.KindMilestone) && v.ParentEpicID != "" {
		if _, err := fmt.Fprintf(w, "  %s — %s %s\n",
			render.Bold(v.ParentEpicID, colorEnabled),
			v.ParentEpicTitle,
			render.StatusColor("["+v.ParentEpicStatus+"]", v.ParentEpicStatus, colorEnabled)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "  %s — %s %s\n",
		v.DriverEntityID,
		v.DriverTitle,
		render.StatusColor("["+v.DriverStatus+"]", v.DriverStatus, colorEnabled)); err != nil {
		return err
	}
	if cancelledFlavor {
		_, err := fmt.Fprintf(w, "  %s — driver was %s; cleanup: %s\n",
			render.Bold("ABANDONED", colorEnabled),
			v.DriverStatus,
			render.Dim("git worktree remove "+v.Path, colorEnabled))
		return err
	}
	_, err := fmt.Fprintf(w, "  %s — driver %s and branch merged to trunk; cleanup: %s\n",
		render.Bold("SAFE TO REMOVE", colorEnabled),
		v.DriverStatus,
		render.Dim("git worktree remove "+v.Path, colorEnabled))
	return err
}

func renderEpicExpansion(w io.Writer, v *WorktreeView, colorEnabled bool) error {
	if len(v.EpicMilestones) > 0 {
		if _, err := fmt.Fprintln(w, "  Milestones:"); err != nil {
			return err
		}
		// Active (in_progress) milestones first, then drafts, then
		// done — so the eye lands on what's running.
		ordered := orderMilestonesByActivity(v.EpicMilestones)
		for _, m := range ordered {
			if err := renderChildRow(w, "    ", m.Status, m.ID, m.Title, colorEnabled); err != nil {
				return err
			}
			if m.DrivenByPath != "" && m.DrivenByPath != v.Path {
				if _, err := fmt.Fprintln(w, render.Dim("        (driven by "+m.DrivenByPath+")", colorEnabled)); err != nil {
					return err
				}
			}
		}
	}
	if len(v.EpicClosesGaps) > 0 {
		if _, err := fmt.Fprintln(w, "  Closes gaps:"); err != nil {
			return err
		}
		for _, g := range v.EpicClosesGaps {
			if err := renderChildRow(w, "    ", g.Status, g.ID, g.Title, colorEnabled); err != nil {
				return err
			}
		}
	}
	if len(v.EpicSurfacedGaps) > 0 {
		if _, err := fmt.Fprintln(w, "  Surfaced gaps:"); err != nil {
			return err
		}
		for _, g := range v.EpicSurfacedGaps {
			if err := renderChildRow(w, "    ", g.Status, g.ID, g.Title, colorEnabled); err != nil {
				return err
			}
		}
	}
	return nil
}

// renderChildRow writes a "<indent><glyph> <ID> — <title> [<status>]"
// line with glyph + status badge colored per StatusColor. Used by
// every nested child-row site (epic milestones, gaps, depends_on,
// surfaced gaps, other-in-flight) so the coloring stays consistent.
func renderChildRow(w io.Writer, indent, status, id, title string, colorEnabled bool) error {
	glyph := render.StatusColor(render.StatusGlyph(status), status, colorEnabled)
	badge := render.StatusColor("["+status+"]", status, colorEnabled)
	_, err := fmt.Fprintf(w, "%s%s %s — %s %s\n", indent, glyph, id, title, badge)
	return err
}

// renderMilestoneDriver renders a milestone-driver worktree's body:
// the parent epic line first (per user feedback: epics before
// milestones, hierarchy clear), then the driven milestone marked with
// `→`, then its depends_on / ACs / surfaced-gaps lists nested under.
func renderMilestoneDriver(w io.Writer, v *WorktreeView, colorEnabled bool) error {
	if v.ParentEpicID != "" {
		if _, err := fmt.Fprintf(w, "  %s — %s %s\n",
			render.Bold(v.ParentEpicID, colorEnabled),
			v.ParentEpicTitle,
			render.StatusColor("["+v.ParentEpicStatus+"]", v.ParentEpicStatus, colorEnabled)); err != nil {
			return err
		}
	}
	// The driver line was already printed by renderWorktreeSection as
	// `  <ID> — <title> [status]`. Re-print it here with `→ (driven)`
	// marker to keep the hierarchy visually clean. Bold the id +
	// status badge so the driver visually pops.
	if _, err := fmt.Fprintf(w, "  %s %s — %s %s  %s\n",
		render.StatusColor("→", v.DriverStatus, colorEnabled),
		render.Bold(v.DriverEntityID, colorEnabled),
		v.DriverTitle,
		render.StatusColor("["+v.DriverStatus+"]", v.DriverStatus, colorEnabled),
		render.Dim("(driven)", colorEnabled)); err != nil {
		return err
	}
	if len(v.DependsOn) > 0 {
		if _, err := fmt.Fprintln(w, "    depends on:"); err != nil {
			return err
		}
		for _, d := range v.DependsOn {
			if err := renderChildRow(w, "      ", d.Status, d.ID, d.Title, colorEnabled); err != nil {
				return err
			}
		}
	}
	if len(v.ACs) > 0 {
		if _, err := fmt.Fprintln(w, "    ACs:"); err != nil {
			return err
		}
		for _, ac := range v.ACs {
			tail := "[" + ac.Status
			if ac.TDDPhase != "" {
				tail += ", " + ac.TDDPhase
			}
			tail += "]"
			glyph := render.StatusColor(render.StatusGlyph(ac.Status), ac.Status, colorEnabled)
			badge := render.StatusColor(tail, ac.Status, colorEnabled)
			if _, err := fmt.Fprintf(w, "      %s %s — %s %s\n", glyph, ac.ID, ac.Title, badge); err != nil {
				return err
			}
		}
	}
	if len(v.SurfacedGaps) > 0 {
		if _, err := fmt.Fprintln(w, "    Surfaced gaps:"); err != nil {
			return err
		}
		for _, g := range v.SurfacedGaps {
			if err := renderChildRow(w, "      ", g.Status, g.ID, g.Title, colorEnabled); err != nil {
				return err
			}
		}
	}
	return nil
}

// orderMilestonesByActivity sorts in_progress milestones first, then
// non-terminal (draft / proposed), then terminal (done / cancelled).
// Within each group, original order (id-sorted from epicExpansion) is
// preserved.
func orderMilestonesByActivity(rows []EpicChildRow) []EpicChildRow {
	out := make([]EpicChildRow, 0, len(rows))
	for _, r := range rows {
		if r.Status == entity.StatusInProgress {
			out = append(out, r)
		}
	}
	for _, r := range rows {
		if r.Status != entity.StatusInProgress && !isTerminalStatus("", r.Status) {
			out = append(out, r)
		}
	}
	for _, r := range rows {
		if isTerminalStatus("", r.Status) {
			out = append(out, r)
		}
	}
	return out
}

// renderOtherInFlight writes the "Other in-flight" sub-section under
// the trunk worktree: one row per in-flight entity that's not driven
// by any worktree. Each row has its driver line; rows with a branch
// get an indented "branch: <name> (no worktree, <age>)" continuation,
// rows without a branch get "(no branch, on trunk)".
//
// G-0122.
func renderOtherInFlight(w io.Writer, rows []OtherInFlightRow, now time.Time, colorEnabled bool) error {
	if _, err := fmt.Fprintln(w, "  Other in-flight:"); err != nil {
		return err
	}
	for _, r := range rows {
		if err := renderChildRow(w, "    ", r.Status, r.ID, r.Title, colorEnabled); err != nil {
			return err
		}
		var sub string
		if r.Branch != "" {
			if r.BranchTime.IsZero() {
				sub = fmt.Sprintf("        branch: %s (no worktree)", r.Branch)
			} else {
				sub = fmt.Sprintf("        branch: %s (no worktree, %s)", r.Branch, relativeAge(now.Sub(r.BranchTime)))
			}
		} else {
			sub = "        (no branch, on trunk)"
		}
		if _, err := fmt.Fprintln(w, render.Dim(sub, colorEnabled)); err != nil {
			return err
		}
	}
	return nil
}

// worktreeMetadataLine returns the optional third metadata line
// content for renderWorktreeSection: a `•`-separated list of
// "created <age>" + "last entity <age>" entries, suppressed when
// either metric would be redundant with `last update` from the
// branch+age line above.
//
// Suppression rules (G-0122): each metric is shown only when its
// rendered "Xh ago" / "X days ago" string differs from the head-age
// string already shown on the line above. Same-display = collapse;
// any meaningful difference = surface. Captures the operator-facing
// "would the eye see new information?" criterion exactly.
//
// Returns "" when no metric survives the suppression.
func worktreeMetadataLine(v *WorktreeView, now time.Time) string {
	headLabel := ""
	if !v.HeadTime.IsZero() {
		headLabel = relativeAge(now.Sub(v.HeadTime))
	}
	var parts []string
	if !v.CreatedTime.IsZero() {
		createdLabel := relativeAge(now.Sub(v.CreatedTime))
		if createdLabel != headLabel {
			parts = append(parts, "created "+createdLabel)
		}
	}
	if !v.LastEntityTime.IsZero() {
		entityLabel := relativeAge(now.Sub(v.LastEntityTime))
		if entityLabel != headLabel {
			parts = append(parts, "last entity touch "+entityLabel)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "  •  ")
}

// renderWorktreeShortLines writes one line per worktree to b, used by
// the default `aiwf status` text output's Worktrees section. Format
// matches the G-0122 short-view design:
//
//	→  G-0122 — Worktree-aware view of in-flight work [open]  •  5h ago  •  dirty
//	→  M-0123 — Pass C reconcile to canonical Go spec table [draft]  •  1 day ago
//	/workspaces/aiwf  •  main  •  trunk (no in-flight scope)
//
// Driver worktrees get the entity-row shape (glyph + ID + title + status badge
// + age + dirty indicator). Trunk worktrees and worktrees without an entity
// correlation get a compact path • branch • marker line.
//
// G-0122.
func renderWorktreeShortLines(b *strings.Builder, views []WorktreeView, termWidth int, colorEnabled bool) {
	now := time.Now()
	ordered := orderWorktreesByActivity(views)
	for _, v := range ordered {
		if v.DriverEntityID != "" {
			renderWorktreeShortDriver(b, v, now, termWidth, colorEnabled)
			continue
		}
		renderWorktreeShortTrunk(b, v, now, colorEnabled)
	}
}

func renderWorktreeShortDriver(b *strings.Builder, v *WorktreeView, now time.Time, termWidth int, colorEnabled bool) {
	glyph := render.StatusColor(render.StatusGlyph(v.DriverStatus), v.DriverStatus, colorEnabled)
	badge := render.StatusColor("["+v.DriverStatus+"]", v.DriverStatus, colorEnabled)
	id := render.Bold(v.DriverEntityID, colorEnabled)
	prefix := fmt.Sprintf("  %s %s — ", glyph, id)
	var tail string
	switch {
	case v.Stale:
		tail = render.Dim("  •  ", colorEnabled) + render.StatusColor("STALE", "cancelled", colorEnabled)
	default:
		var ageParts []string
		if !v.HeadTime.IsZero() {
			ageParts = append(ageParts, relativeAge(now.Sub(v.HeadTime)))
		}
		dirty := ""
		if v.Dirty {
			dirty = render.Dim("  •  ", colorEnabled) + render.StatusColor("dirty", entity.StatusInProgress, colorEnabled)
		}
		if len(ageParts) > 0 {
			tail = render.Dim("  •  "+strings.Join(ageParts, "  •  "), colorEnabled) + dirty
		} else {
			tail = dirty
		}
	}
	title := TruncStatusTitle(v.DriverTitle+" "+stripAnsi(badge), termWidth, prefix, "  "+stripAnsi(tail))
	// The truncation budget had to include the badge so the row fits;
	// but the printed row uses the colored badge separately to keep
	// the styling. Reassemble: prefix + truncated-title (without
	// trailing badge text) + badge + tail.
	titleOnly := strings.TrimSuffix(title, " "+stripAnsi(badge))
	if titleOnly == title {
		// truncation didn't reach the badge segment — title fits as-is
		// without the badge suffix on it. Use as-is.
		titleOnly = title
	}
	fmt.Fprintf(b, "%s%s %s%s\n", prefix, titleOnly, badge, tail)
}

func renderWorktreeShortTrunk(b *strings.Builder, v *WorktreeView, now time.Time, colorEnabled bool) {
	parts := []string{v.Path, branchLabel(v.Branch)}
	if v.Branch == "main" {
		parts = append(parts, "trunk (no in-flight scope)")
	} else {
		parts = append(parts, "no driver entity")
	}
	if !v.HeadTime.IsZero() {
		parts = append(parts, "last commit "+relativeAge(now.Sub(v.HeadTime)))
	}
	if v.Dirty {
		parts = append(parts, render.StatusColor("dirty", entity.StatusInProgress, colorEnabled))
	}
	fmt.Fprintf(b, "  %s\n", render.Dim(strings.Join(parts, "  •  "), colorEnabled))
}

// stripAnsi removes SGR sequences from s. Used by the truncation logic
// so the budget measures the rendered-character width, not the escape-
// inflated byte count. Naive but sufficient for the bold/dim/SGR
// sequences this package emits.
func stripAnsi(s string) string {
	if !strings.ContainsRune(s, '\x1b') {
		return s
	}
	var out strings.Builder
	skip := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			skip = true
		case skip && r == 'm':
			skip = false
		case skip:
			// inside CSI sequence, drop
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}

// branchLabel renders the branch field, handling detached HEAD with
// an explicit label so the output never produces an empty value.
func branchLabel(branch string) string {
	if branch == "" {
		return "(detached HEAD)"
	}
	return branch
}

// worktreeHeadTime returns the author-date of the HEAD commit on the
// worktree at path. ISO 8601 ("strict ISO" %aI format) is parsed via
// time.RFC3339. Best-effort: any git failure or parse failure returns
// a zero time.Time + the error, and the caller treats zero-time as
// "no age info" (renderer omits the line).
//
// G-0122 age display.
func worktreeHeadTime(ctx context.Context, path string) (time.Time, error) {
	cmd := exec.CommandContext(ctx, "git", "log", "-1", "--format=%aI", "HEAD")
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, strings.TrimSpace(string(out)))
}

// worktreeIsDirty returns true when `git status --porcelain` reports
// any output for the worktree at path — i.e., any staged, unstaged,
// or untracked changes. False on success-and-clean OR on any git
// failure (best-effort; a transient git error should not surface a
// phantom dirty flag).
//
// G-0122 option A: separates "last commit time" from "have I made
// changes since." Honest signal for active in-flight worktrees where
// HEAD is from a past commit but the operator is currently editing.
func worktreeIsDirty(ctx context.Context, path string) bool {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

// branchFirstAheadCommitTime returns the author-date of the first
// ahead-of-trunk commit on `branch` — a proxy for "when this worktree
// started diverging from main." Empty / err when branch has no ahead
// commits or when git log fails.
//
// Uses `main..<branch>` to scope to ahead-of-trunk commits; `--reverse`
// + `-1` returns the oldest of those (the divergence point's first
// step). G-0122 user-feedback extension.
func branchFirstAheadCommitTime(ctx context.Context, rootDir, branch string) (time.Time, error) {
	cmd := exec.CommandContext(ctx, "git", "log", "--reverse", "-1", "--format=%aI", "main.."+branch)
	cmd.Dir = rootDir
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}, err
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, s)
}

// branchLastEntityCommitTime returns the author-date of the most
// recent commit on `branch` (ahead of main) that carries an
// `aiwf-verb:` trailer — the last meaningful entity-touching change.
// Empty when no such commit exists. G-0122 user-feedback extension.
func branchLastEntityCommitTime(ctx context.Context, rootDir, branch string) (time.Time, error) {
	cmd := exec.CommandContext(ctx, "git", "log", "-1", "--format=%aI",
		"--grep", "^aiwf-verb:", "-E", "main.."+branch)
	cmd.Dir = rootDir
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}, err
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, s)
}

// branchAheadOfTrunkCount returns the count of commits on `branch`
// that are ahead of main — i.e. the number of unmerged commits this
// worktree carries. Zero on any git failure or when the branch is
// fully merged into trunk (or doesn't exist).
//
// Used by the stale-rendering arm to distinguish wrap-pending (count
// > 0; the operator must merge before removing) from safe-to-remove
// (count == 0; commits live on trunk so removal is non-destructive).
// G-0153.
func branchAheadOfTrunkCount(ctx context.Context, rootDir, branch string) int {
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--count", "main.."+branch)
	cmd.Dir = rootDir
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0
	}
	return n
}

// branchAge is one local branch's name + its HEAD commit time. Used
// by the "Other in-flight" surfacing to render branch+age info for
// entities that have a branch but no worktree.
//
// G-0122.
type branchAge struct {
	Name string
	Time time.Time
}

// listBranchesWithAge enumerates every local branch via git
// for-each-ref refs/heads/, returning each branch's short name and
// committer-date. Best-effort: a git failure returns nil (the caller
// treats "no branches found" the same as "branches list unavailable").
func listBranchesWithAge(ctx context.Context, rootDir string) []branchAge {
	cmd := exec.CommandContext(
		ctx, "git",
		"for-each-ref",
		"refs/heads/",
		"--format=%(refname:short)\x1f%(committerdate:strict-iso)",
	)
	cmd.Dir = rootDir
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var branches []branchAge
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		name, dateStr, ok := strings.Cut(line, "\x1f")
		if !ok {
			continue
		}
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(dateStr))
		if err != nil {
			// Some git versions emit "strict-iso" with a space instead
			// of a 'T' between date and time; fall back to the
			// permissive parse.
			t, err = time.Parse("2006-01-02 15:04:05 -0700", strings.TrimSpace(dateStr))
			if err != nil {
				continue
			}
		}
		branches = append(branches, branchAge{Name: name, Time: t})
	}
	return branches
}

// buildOtherInFlight builds the "Other in-flight" list for the trunk
// worktree's section: every in-flight entity (active epic or
// in_progress milestone) that no worktree is driving. Each row carries
// the entity's branch + age when a matching branch exists, or
// empty/zero when work is on trunk directly.
//
// worktreeDriverIDs is the canonical-id set of entities already shown
// under worktree sections. branches is the full local-branch list
// from listBranchesWithAge.
//
// G-0122 user-feedback extension.
func buildOtherInFlight(tr *tree.Tree, worktreeDriverIDs map[string]bool, branches []branchAge) []OtherInFlightRow {
	// Index branches by the entity id their name parses to (only
	// ritual-shaped branches contribute — fix/, chore/, etc. are
	// not entity-bearing).
	branchByEntity := map[string]branchAge{}
	for _, b := range branches {
		if id := parseEntityFromBranch(b.Name); id != "" {
			branchByEntity[entity.Canonicalize(id)] = b
		}
	}
	var rows []OtherInFlightRow
	collect := func(e *entity.Entity) {
		canonical := entity.Canonicalize(e.ID)
		if worktreeDriverIDs[canonical] {
			return // already shown under a worktree section
		}
		row := OtherInFlightRow{ID: e.ID, Title: e.Title, Status: e.Status}
		if b, ok := branchByEntity[canonical]; ok {
			row.Branch = b.Name
			row.BranchTime = b.Time
		}
		rows = append(rows, row)
	}
	for _, e := range tr.FilterByKindStatuses(entity.KindEpic, entity.StatusActive) {
		collect(e)
	}
	for _, e := range tr.FilterByKindStatuses(entity.KindMilestone, entity.StatusInProgress) {
		collect(e)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows
}

// renderAge formats a HEAD-time as "YYYY-MM-DD HH:MM (N <unit> ago)"
// using local time and a coarse relative-time suffix. Returns "" when
// t is the zero value (no age info available — renderer skips the
// line).
//
// Relative grain: "just now" (<1m), "Xm ago" (<1h), "Xh ago" (<1d),
// "X day(s) ago" (<30d), "X month(s) ago" (<365d), "X year(s) ago".
func renderAge(t, now time.Time) string {
	if t.IsZero() {
		return ""
	}
	local := t.Local()
	delta := now.Sub(t)
	if delta < 0 {
		// Clock skew — render absolute timestamp without a relative
		// suffix rather than printing a nonsensical "negative time".
		return local.Format("2006-01-02 15:04")
	}
	return fmt.Sprintf("%s (%s)", local.Format("2006-01-02 15:04"), relativeAge(delta))
}

// relativeAge returns just the relative-time suffix ("Xm ago" /
// "X days ago" / etc.) without the absolute timestamp. Used by the
// consolidated branch+age line per G-0122 user feedback (B).
func relativeAge(delta time.Duration) string {
	if delta < 0 {
		return "future"
	}
	switch {
	case delta < time.Minute:
		return "just now"
	case delta < time.Hour:
		return fmt.Sprintf("%dm ago", int(delta.Minutes()))
	case delta < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(delta.Hours()))
	case delta < 30*24*time.Hour:
		return pluralizeAgo(int(delta.Hours()/24), "day")
	case delta < 365*24*time.Hour:
		return pluralizeAgo(int(delta.Hours()/24/30), "month")
	default:
		return pluralizeAgo(int(delta.Hours()/24/365), "year")
	}
}

func pluralizeAgo(n int, unit string) string {
	if n == 1 {
		return "1 " + unit + " ago"
	}
	return fmt.Sprintf("%d %ss ago", n, unit)
}
