package htmlrender

// IndexData is the input to the index template. Epics is sorted by id;
// each row carries the AC met-rollup so the index can show the
// `met / (total - cancelled)` column without reaching back into the
// tree.
type IndexData struct {
	Title         string
	Epics         []EpicSummary
	FindingCounts FindingCounts
	LastActivity  string // ISO date of most recent commit on aiwf-* trailer set; empty in pre-aiwf repos
	Sidebar       SidebarData
}

// SidebarData is the left-nav payload every page receives. Epics is
// sorted by id; each carries its child milestones (also sorted) so
// the renderer can emit a <details> per epic with milestones nested
// inside. The current page's ancestors carry IsActive=true so the
// template emits <details open> on the right epic and aria-
// current="page" on the right link.
type SidebarData struct {
	Epics []SidebarEpic
}

// SidebarEpic is one epic row in the sidebar. IsActive is true when
// this epic is the page's own (epic page) or its parent (milestone
// page).
type SidebarEpic struct {
	ID         string
	Title      string
	FileName   string
	IsActive   bool
	IsCurrent  bool // true on the epic page itself (drives aria-current)
	Milestones []SidebarMilestone
}

// SidebarMilestone is one milestone row inside a SidebarEpic.
// IsCurrent is true when this milestone is the page being rendered.
type SidebarMilestone struct {
	ID        string
	Title     string
	FileName  string
	IsCurrent bool
}

// EpicSummary is one row on the index page. ACMet is rolled up across
// every milestone under the epic; ACTotal is the sum of non-cancelled
// AC counts.
type EpicSummary struct {
	ID             string
	Title          string
	Status         string
	FileName       string
	MilestoneCount int
	ACMet          int
	ACTotal        int // total - cancelled
	LastActivity   string
}

// FindingCounts feeds the "findings rollup" line on the index page.
// Errors and Warnings are the counts emitted by `aiwf check`; the
// renderer does not run check itself, the caller passes them in.
type FindingCounts struct {
	Errors   int
	Warnings int
}

// EpicData is the input to the epic template. Milestones is sorted by
// id; LinkedEntities is the union of forward and reverse references
// (deduplicated, sorted), with each entry naming kind/id/title and
// the file the link should point at.
type EpicData struct {
	Epic           *EntityRef
	Body           map[string]string // section slug → prose, per entity.ParseBodySections
	Milestones     []MilestoneSummary
	DependencyDAG  []DependencyEdge
	LinkedEntities []LinkedEntity
	History        []HistoryRow
	ACMet          int
	ACTotal        int
	Sidebar        SidebarData
}

// EntityRef is the minimal data the templates need about the page's
// own entity: id, title, status, plus `Path` (repo-relative file
// path) and the rendered file's name. Decouples templates from the
// internal entity.Entity struct so future schema changes don't
// rewrite templates.
type EntityRef struct {
	ID       string
	Title    string
	Status   string
	Path     string
	FileName string
	Kind     string
	TDD      string
}

// MilestoneSummary is one milestone row on an epic page.
type MilestoneSummary struct {
	ID, Title, Status, FileName string
	ACMet, ACTotal              int
	TDD                         string
	LastActivity                string
}

// DependencyEdge captures a single `depends_on` edge between two
// milestones inside an epic. From and To are short ids (M-NNN).
type DependencyEdge struct {
	From string
	To   string
}

// LinkedEntity is one row in an epic / milestone "Linked entities"
// block. Direction is "forward" (this entity references Target) or
// "reverse" (Target references this entity).
type LinkedEntity struct {
	ID        string
	Title     string
	Status    string
	Kind      string
	FileName  string
	Direction string
}

// MilestoneData is the input to the milestone template. The six
// tabs (overview, manifest, build, tests, commits, provenance) read
// from this single struct; the templates branch internally on what
// to show in each tab.
//
// LinkedDecisions / LinkedEntities are split so the Overview tab
// can render the decisions block conditionally (an empty list
// suppresses the heading entirely). LinkedEntities is the union
// shown on a separate "Linked entities" block when populated;
// LinkedDecisions is the kind-filtered subset surfaced in the
// Overview tab per I3 plan §3.3.
type MilestoneData struct {
	Milestone       *EntityRef
	ParentEpic      *EntityRef
	Body            map[string]string
	ACs             []ACDetail
	Commits         []HistoryRow
	Provenance      ProvenanceData
	LinkedEntities  []LinkedEntity
	LinkedDecisions []LinkedEntity
	TestsPolicy     TestsPolicy
	ACMet           int
	ACTotal         int
	Sidebar         SidebarData
}

// ACDetail is one AC's view inside the milestone Manifest tab.
// Description is the body prose under the `### AC-N — <title>`
// heading; Phases is the per-AC TDD timeline assembled from the AC's
// history (one row per phase transition); Tests is the latest test
// metrics for the AC (per-AC iterator authority).
type ACDetail struct {
	ID          string
	Title       string
	Status      string
	TDDPhase    string
	Description string
	Phases      []PhaseEvent
	Tests       *TestMetricsView
	Anchor      string // ac-N — pre-derived so templates don't call ACAnchor() per row
}

// PhaseEvent is one TDD-phase transition for an AC. Date is ISO
// (YYYY-MM-DD); Phase is the to-state ("red"/"green"/"refactor"/
// "done"); Forced when this transition was --force; Tests when the
// commit carried an aiwf-tests trailer.
type PhaseEvent struct {
	Date   string
	Phase  string
	Forced bool
	Reason string
	Tests  *TestMetricsView
}

// TestMetricsView is the renderer-facing view of a single
// aiwf-tests trailer. Total is computed by the caller (caller knows
// whether the on-wire trailer recorded total= explicitly).
type TestMetricsView struct {
	Pass  int
	Fail  int
	Skip  int
	Total int
}

// HistoryRow is one event in the Commits tab and in epic-page
// "Recent activity" sections. Verb is the trailer value; Detail is
// the commit subject; Force, AuditOnly, OnBehalfOf, Scope, ScopeEnds
// surface the I2.5 provenance shape; Tests is the parsed metrics
// when present.
type HistoryRow struct {
	Date         string
	Commit       string
	Actor        string
	Principal    string
	OnBehalfOf   string
	Verb         string
	Detail       string
	To           string
	Force        bool
	ForceReason  string
	AuditOnly    bool
	AuditReason  string
	AuthorizedBy string
	Scope        string
	ScopeEnds    []string
	Reason       string
	Tests        *TestMetricsView
}

// ProvenanceData is the milestone Provenance tab payload — scopes
// table on top, full event timeline below. The renderer groups the
// timeline by scope at template time.
type ProvenanceData struct {
	Scopes   []ScopeRow
	Timeline []HistoryRow
}

// ScopeRow is one row in the Provenance tab's scopes table.
type ScopeRow struct {
	AuthSHA    string // 8-char short form for table display
	FullSHA    string // full SHA for the show_authorization-style toggle
	Agent      string
	Principal  string
	Opened     string // YYYY-MM-DD
	EndedAt    string // YYYY-MM-DD; empty when state != ended
	State      string // active|paused|ended
	EventCount int
}

// TestsPolicy controls the milestone Tests tab's `strict` /
// `advisory` badge. Strict is true when aiwf.yaml.tdd.
// require_test_metrics is true.
type TestsPolicy struct {
	Strict bool
}
