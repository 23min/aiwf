package verb

// archive_test.go — M-0085 unit tests for the verb-side helpers in
// archive.go. Dispatcher-level behavior (single-commit invariant,
// trailer shape, idempotence, per-kind storage layout) lives in
// `cmd/aiwf/archive_cmd_test.go`. These tests cover:
//
//   - Pure helpers (archiveTargetForEpic, archiveTargetForContract,
//     archiveTargetForFlatFile, pluralize, isKnownKind).
//   - Error paths in computeArchiveMoves (unknown-kind filter).
//   - The milestone-filter no-op path (a kindFilter of "milestone"
//     produces zero moves because milestones don't archive
//     independently per ADR-0004).
//   - The empty-tree no-op path of Archive.
//
// Per CLAUDE.md "Test untested code paths before declaring code
// paths 'done'": every branch in archive.go is exercised either
// here or via the dispatcher tests in cmd/aiwf.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

func TestArchiveTargetForEpic(t *testing.T) {
	t.Parallel()
	got := archiveTargetForEpic("work/epics/E-0010-foo-bar")
	want := "work/epics/archive/E-0010-foo-bar"
	if got != want {
		t.Errorf("archiveTargetForEpic(...) = %q, want %q", got, want)
	}
}

func TestArchiveTargetForContract(t *testing.T) {
	t.Parallel()
	got := archiveTargetForContract("work/contracts/C-0010-some-api")
	want := "work/contracts/archive/C-0010-some-api"
	if got != want {
		t.Errorf("archiveTargetForContract(...) = %q, want %q", got, want)
	}
}

func TestArchiveTargetForFlatFile(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, in string
		kind     entity.Kind
		want     string
	}{
		{"gap", "work/gaps/G-0010-some-gap.md", entity.KindGap, "work/gaps/archive/G-0010-some-gap.md"},
		{"decision", "work/decisions/D-0010-some-decision.md", entity.KindDecision, "work/decisions/archive/D-0010-some-decision.md"},
		{"adr", "docs/adr/ADR-0010-some-adr.md", entity.KindADR, "docs/adr/archive/ADR-0010-some-adr.md"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := archiveTargetForFlatFile(tc.in, tc.kind)
			if got != tc.want {
				t.Errorf("archiveTargetForFlatFile(%q, %s) = %q, want %q", tc.in, tc.kind, got, tc.want)
			}
		})
	}
}

func TestPluralize(t *testing.T) {
	t.Parallel()
	cases := []struct {
		n                int
		sing, plur, want string
	}{
		{1, "y", "ies", "y"},
		{0, "y", "ies", "ies"},
		{2, "y", "ies", "ies"},
		{1, "entity", "entities", "entity"},
		{17, "entity", "entities", "entities"},
	}
	for _, tc := range cases {
		got := pluralize(tc.n, tc.sing, tc.plur)
		if got != tc.want {
			t.Errorf("pluralize(%d, %q, %q) = %q, want %q", tc.n, tc.sing, tc.plur, got, tc.want)
		}
	}
}

func TestIsKnownKind(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want bool
	}{
		{"epic", true},
		{"milestone", true},
		{"adr", true},
		{"gap", true},
		{"decision", true},
		{"contract", true},
		{"finding", false}, // proposed seventh kind, not yet in the closed set
		{"", false},
		{"EPIC", false}, // case-sensitive
	}
	for _, tc := range cases {
		got := isKnownKind(tc.in)
		if got != tc.want {
			t.Errorf("isKnownKind(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestComputeArchiveMoves_UnknownKindFilter pins the error branch in
// computeArchiveMoves: a kindFilter that doesn't match any of the six
// kinds returns a wrapped error naming the bad input.
func TestComputeArchiveMoves_UnknownKindFilter(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{}
	moves, _, err := computeArchiveMoves(tr, "widget")
	if err == nil {
		t.Fatal("computeArchiveMoves with unknown kind returned no error")
	}
	if moves != nil {
		t.Errorf("moves = %v, want nil on error path", moves)
	}
	// The error message names the closed set so a typo gets actionable
	// remediation.
	for _, kindName := range []string{"epic", "milestone", "adr", "gap", "decision", "contract"} {
		if !strings.Contains(err.Error(), kindName) {
			t.Errorf("error message does not enumerate kind %q (caller cannot self-correct):\n  %v", kindName, err)
		}
	}
}

// TestComputeArchiveMoves_MilestoneFilterNoOp exercises the
// "kindFilter=milestone returns no moves" branch. Per ADR-0004
// milestones don't archive independently — they ride with their
// parent epic. A user who explicitly asks --kind milestone gets a
// truthful no-op, not an error.
func TestComputeArchiveMoves_MilestoneFilterNoOp(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{
				ID:     "M-0020",
				Kind:   entity.KindMilestone,
				Status: entity.StatusDone,
				Path:   "work/epics/E-0010-foo/M-0020-some-milestone.md",
			},
		},
	}
	moves, _, err := computeArchiveMoves(tr, "milestone")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(moves) != 0 {
		t.Errorf("--kind milestone produced %d moves; want 0 (milestones don't archive independently per ADR-0004)", len(moves))
	}
}

// TestComputeArchiveMoves_EpicWithMultipleMilestones_OneMove pins
// the deduplication: an epic with three milestones inside produces
// exactly ONE OpMove (the dir rename). The milestones don't generate
// their own moves; they ride with the epic dir rename.
func TestComputeArchiveMoves_EpicWithMultipleMilestones_OneMove(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{
				ID:     "E-0010",
				Kind:   entity.KindEpic,
				Status: entity.StatusDone,
				Path:   "work/epics/E-0010-foo/epic.md",
			},
			{
				ID:     "M-0020",
				Kind:   entity.KindMilestone,
				Status: entity.StatusDone,
				Path:   "work/epics/E-0010-foo/M-0020-a.md",
			},
			{
				ID:     "M-0021",
				Kind:   entity.KindMilestone,
				Status: entity.StatusDone,
				Path:   "work/epics/E-0010-foo/M-0021-b.md",
			},
			{
				ID:     "M-0022",
				Kind:   entity.KindMilestone,
				Status: entity.StatusDone,
				Path:   "work/epics/E-0010-foo/M-0022-c.md",
			},
		},
	}
	moves, _, err := computeArchiveMoves(tr, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(moves) != 1 {
		t.Errorf("expected 1 move (the epic dir rename); got %d:\n  %+v", len(moves), moves)
	}
	if moves[0].kind != entity.KindEpic {
		t.Errorf("move[0].kind = %s, want epic", moves[0].kind)
	}
	if moves[0].from != "work/epics/E-0010-foo" {
		t.Errorf("move[0].from = %q, want %q", moves[0].from, "work/epics/E-0010-foo")
	}
	if moves[0].to != "work/epics/archive/E-0010-foo" {
		t.Errorf("move[0].to = %q, want %q", moves[0].to, "work/epics/archive/E-0010-foo")
	}
}

// TestComputeArchiveMoves_DirShapeKindsDeduplicate pins the
// epicDirSeen / contractDirSeen guards: if the same dir surfaces
// twice in tr.Entities (defensive against future loader changes),
// the verb emits exactly one move per dir. The synthesized fixture
// here is pathologically duplicated; production trees never load
// two epic.md or contract.md records with the same parent dir, but
// the guard exists to keep the move set deterministic regardless.
func TestComputeArchiveMoves_DirShapeKindsDeduplicate(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{ID: "E-0010", Kind: entity.KindEpic, Status: entity.StatusDone, Path: "work/epics/E-0010-foo/epic.md"},
			{ID: "E-0010", Kind: entity.KindEpic, Status: entity.StatusDone, Path: "work/epics/E-0010-foo/epic.md"},
			{ID: "C-0010", Kind: entity.KindContract, Status: entity.StatusRetired, Path: "work/contracts/C-0010-bar/contract.md"},
			{ID: "C-0010", Kind: entity.KindContract, Status: entity.StatusRetired, Path: "work/contracts/C-0010-bar/contract.md"},
		},
	}
	moves, _, err := computeArchiveMoves(tr, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(moves) != 2 {
		t.Errorf("expected 2 moves (one per dir, dedup); got %d:\n  %+v", len(moves), moves)
	}
}

// TestComputeArchiveMoves_AlreadyArchivedSkipped pins the skip-archive
// branch in computeArchiveMoves: an entity whose path is already
// under archive/ is left alone, regardless of status. This is the
// idempotence-load-bearing branch.
func TestComputeArchiveMoves_AlreadyArchivedSkipped(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{
				ID:     "G-0010",
				Kind:   entity.KindGap,
				Status: entity.StatusAddressed,
				Path:   "work/gaps/archive/G-0010-already-swept.md",
			},
		},
	}
	moves, _, err := computeArchiveMoves(tr, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(moves) != 0 {
		t.Errorf("expected 0 moves (entity is already archived); got %d", len(moves))
	}
}

// TestComputeArchiveMoves_NonTerminalSkipped pins the
// "non-terminal status -> skip" branch for each entity-kind arm of
// the switch. Active-status entities never produce moves.
func TestComputeArchiveMoves_NonTerminalSkipped(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{ID: "E-0010", Kind: entity.KindEpic, Status: entity.StatusActive, Path: "work/epics/E-0010-x/epic.md"},
			{ID: "G-0010", Kind: entity.KindGap, Status: entity.StatusOpen, Path: "work/gaps/G-0010-x.md"},
			{ID: "D-0010", Kind: entity.KindDecision, Status: entity.StatusProposed, Path: "work/decisions/D-0010-x.md"},
			{ID: "C-0010", Kind: entity.KindContract, Status: entity.StatusProposed, Path: "work/contracts/C-0010-x/contract.md"},
			{ID: "ADR-0010", Kind: entity.KindADR, Status: entity.StatusProposed, Path: "docs/adr/ADR-0010-x.md"},
		},
	}
	moves, _, err := computeArchiveMoves(tr, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(moves) != 0 {
		t.Errorf("expected 0 moves (all active); got %d:\n  %+v", len(moves), moves)
	}
}

// TestNonTerminalEpicChildren_FiltersByParent (G-0394) pins the parent
// filter shared by Cancel, Promote's epic-done guard, and Archive: a
// milestone belonging to a DIFFERENT epic must never appear in the
// result, even when it is itself non-terminal.
func TestNonTerminalEpicChildren_FiltersByParent(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{ID: "M-0020", Kind: entity.KindMilestone, Status: entity.StatusDraft, Parent: "E-0010"},
			{ID: "M-0030", Kind: entity.KindMilestone, Status: entity.StatusDraft, Parent: "E-0099"},
		},
	}
	got := nonTerminalEpicChildren(tr, "E-0010")
	if len(got) != 1 || got[0] != "M-0020" {
		t.Errorf("nonTerminalEpicChildren(tr, E-0010) = %v; want [M-0020] (M-0030 belongs to E-0099)", got)
	}
}

// TestComputeArchiveMoves_EpicWithNonTerminalChild_Skipped (G-0394,
// Direction B): a done epic whose milestone is still non-terminal
// must not produce a move — Archive independently declines to strand
// the child in archive/ alongside its terminal parent, regardless of
// how the epic reached done (the promote-time guard is the primary
// chokepoint; this is the unconditional defense-in-depth backstop for
// the --force / raw-edit cases that bypass it).
func TestComputeArchiveMoves_EpicWithNonTerminalChild_Skipped(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{ID: "E-0010", Kind: entity.KindEpic, Status: entity.StatusDone, Path: "work/epics/E-0010-foo/epic.md"},
			{ID: "M-0020", Kind: entity.KindMilestone, Status: entity.StatusDraft, Parent: "E-0010", Path: "work/epics/E-0010-foo/M-0020-a.md"},
		},
	}
	moves, skipped, err := computeArchiveMoves(tr, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(moves) != 0 {
		t.Errorf("expected 0 moves (epic has a non-terminal child); got %d:\n  %+v", len(moves), moves)
	}
	if len(skipped) != 1 || skipped[0].epic != "E-0010" {
		t.Fatalf("skipped = %+v; want exactly one skip naming E-0010", skipped)
	}
	if len(skipped[0].children) != 1 || skipped[0].children[0] != "M-0020" {
		t.Errorf("skipped[0].children = %v; want [M-0020]", skipped[0].children)
	}
}

// TestComputeArchiveMoves_EpicWithMixedChildren_SkippedEvenIfSomeDone
// (G-0394): one done sibling milestone must not mask a second,
// non-terminal sibling — the epic is still skipped.
func TestComputeArchiveMoves_EpicWithMixedChildren_SkippedEvenIfSomeDone(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{ID: "E-0010", Kind: entity.KindEpic, Status: entity.StatusDone, Path: "work/epics/E-0010-foo/epic.md"},
			{ID: "M-0020", Kind: entity.KindMilestone, Status: entity.StatusDone, Parent: "E-0010", Path: "work/epics/E-0010-foo/M-0020-a.md"},
			{ID: "M-0021", Kind: entity.KindMilestone, Status: entity.StatusInProgress, Parent: "E-0010", Path: "work/epics/E-0010-foo/M-0021-b.md"},
		},
	}
	moves, skipped, err := computeArchiveMoves(tr, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(moves) != 0 {
		t.Errorf("expected 0 moves (M-0021 is still in_progress); got %d:\n  %+v", len(moves), moves)
	}
	if len(skipped) != 1 || skipped[0].epic != "E-0010" {
		t.Fatalf("skipped = %+v; want exactly one skip naming E-0010", skipped)
	}
	// The done sibling M-0020 must not appear among the offending
	// children — only the still-in_progress M-0021 does.
	if len(skipped[0].children) != 1 || skipped[0].children[0] != "M-0021" {
		t.Errorf("skipped[0].children = %v; want [M-0021] (M-0020 is done, not offending)", skipped[0].children)
	}
}

// TestComputeArchiveMoves_EpicAllTerminalChildren_NotSkipped (G-0394):
// characterization against a "refuse everything" regression — a done
// epic whose only child milestone is also done still produces its
// move exactly as before this guard existed.
func TestComputeArchiveMoves_EpicAllTerminalChildren_NotSkipped(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{ID: "E-0010", Kind: entity.KindEpic, Status: entity.StatusDone, Path: "work/epics/E-0010-foo/epic.md"},
			{ID: "M-0020", Kind: entity.KindMilestone, Status: entity.StatusDone, Parent: "E-0010", Path: "work/epics/E-0010-foo/M-0020-a.md"},
		},
	}
	moves, skipped, err := computeArchiveMoves(tr, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(moves) != 1 {
		t.Errorf("expected 1 move (epic's only child is done); got %d:\n  %+v", len(moves), moves)
	}
	if len(skipped) != 0 {
		t.Errorf("expected 0 skipped epics; got %+v", skipped)
	}
}

// TestArchiveCommitBody_ListsSkippedEpics (G-0394): the commit body
// surfaces which epics were skipped and why, since the multi-entity
// sweep otherwise silently drops them from the moves list with no
// operator-visible trace.
func TestArchiveCommitBody_ListsSkippedEpics(t *testing.T) {
	t.Parallel()
	skipped := []archiveSkip{
		{epic: "E-0010", children: []string{"M-0020", "M-0021"}},
	}
	body := archiveCommitBody(nil, skipped)
	if !strings.Contains(body, "E-0010") {
		t.Errorf("commit body does not name skipped epic E-0010:\n%s", body)
	}
	if !strings.Contains(body, "M-0020") || !strings.Contains(body, "M-0021") {
		t.Errorf("commit body does not list offending children M-0020/M-0021:\n%s", body)
	}
}

// TestArchiveCommitBody_ListsBothMovesAndSkipped (G-0394) pins the
// separator branch: when both a real move and a skipped epic are
// present in the same sweep, the skipped section follows a blank-line
// separator after the moves section rather than colliding with it.
func TestArchiveCommitBody_ListsBothMovesAndSkipped(t *testing.T) {
	t.Parallel()
	moves := []archiveMove{{kind: entity.KindGap, id: "G-0001"}}
	skipped := []archiveSkip{{epic: "E-0010", children: []string{"M-0020"}}}
	body := archiveCommitBody(moves, skipped)
	if !strings.Contains(body, "G-0001") {
		t.Errorf("commit body missing moved id G-0001:\n%s", body)
	}
	if !strings.Contains(body, "E-0010") || !strings.Contains(body, "M-0020") {
		t.Errorf("commit body missing skipped epic/child:\n%s", body)
	}
	idxAffected := strings.Index(body, "Affected ids:")
	idxSkipped := strings.Index(body, "Skipped")
	if idxAffected < 0 || idxSkipped < 0 || idxAffected > idxSkipped {
		t.Errorf("expected 'Affected ids:' section before 'Skipped' section:\n%s", body)
	}
	if !strings.Contains(body, "\n\nSkipped") {
		t.Errorf("expected a blank-line separator before the 'Skipped' section, not a collision with the moves section:\n%s", body)
	}
}

// TestArchive_NoOpResultOnConvergedTree pins the verb's NoOp branch:
// when planArchive returns nil (nothing to sweep), Archive returns
// a Result with NoOp=true and a human-readable message.
func TestArchive_NoOpResultOnConvergedTree(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// An empty tempdir has no entities to load; planArchive returns
	// (nil, nil) and Archive's NoOp branch fires.
	res, err := Archive(context.Background(), root, "human/test", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("nil Result on no-op path")
	}
	if !res.NoOp {
		t.Errorf("expected NoOp=true on converged tree; got Plan=%+v", res.Plan)
	}
	if res.NoOpMessage == "" {
		t.Error("NoOpMessage should be non-empty so the dispatcher has something to print")
	}
}

// TestArchive_NoOpMessageMentionsSkippedEpics (G-0394, Direction B):
// when every eligible epic is skipped for a non-terminal child (so
// moves is empty), Archive's NoOp message must not claim the tree is
// "converged" — it must name the skipped epic and its offending
// child, so the operator isn't misled into thinking nothing needs
// attention.
func TestArchive_NoOpMessageMentionsSkippedEpics(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWrite := func(rel, body string) {
		t.Helper()
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	mustWrite("work/epics/E-0010-foo/epic.md", "---\nid: E-0010\ntitle: foo\nstatus: done\n---\n")
	mustWrite("work/epics/E-0010-foo/M-0020-a.md", "---\nid: M-0020\ntitle: a\nstatus: draft\nparent: E-0010\n---\n")

	res, err := Archive(context.Background(), root, "human/test", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.NoOp {
		t.Fatalf("expected NoOp=true (nothing to sweep); got Plan=%+v", res.Plan)
	}
	if !strings.Contains(res.NoOpMessage, "E-0010") || !strings.Contains(res.NoOpMessage, "M-0020") {
		t.Errorf("NoOpMessage = %q; want it to name skipped epic E-0010 and offending child M-0020", res.NoOpMessage)
	}
	if strings.Contains(res.NoOpMessage, "converged") {
		t.Errorf("NoOpMessage = %q; must not claim the tree is converged when an epic was skipped", res.NoOpMessage)
	}
}

// TestPlanArchive_SortsBySameKindThenFrom exercises the sort.Slice
// comparator's "same kind, compare by from" branch (the secondary
// sort key). Without two moves of the same kind, the comparator's
// from-comparison line goes uncovered.
func TestPlanArchive_SortsBySameKindThenFrom(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// Stage three terminal-status gaps (same kind), in non-alphabetical
	// id order, so the sort comparator must fire and order them.
	mustWrite := func(rel, body string) {
		t.Helper()
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	mustWrite("work/gaps/G-0030-zebra.md", "---\nid: G-0030\ntitle: z\nstatus: addressed\n---\n")
	mustWrite("work/gaps/G-0010-alpha.md", "---\nid: G-0010\ntitle: a\nstatus: addressed\n---\n")
	mustWrite("work/gaps/G-0020-mango.md", "---\nid: G-0020\ntitle: m\nstatus: addressed\n---\n")

	plan, _, err := planArchive(context.Background(), root, "")
	if err != nil {
		t.Fatalf("planArchive: %v", err)
	}
	if plan == nil {
		t.Fatal("plan is nil; expected three gap moves")
	}
	// Extract the from-paths in op order; they must be sorted.
	var froms []string
	for _, op := range plan.Ops {
		if op.Type == OpMove {
			froms = append(froms, op.Path)
		}
	}
	if len(froms) != 3 {
		t.Fatalf("expected 3 moves; got %d (%v)", len(froms), froms)
	}
	for i := 1; i < len(froms); i++ {
		if froms[i-1] > froms[i] {
			t.Errorf("moves not sorted by from-path: %s came before %s", froms[i-1], froms[i])
		}
	}
}

// TestArchiveCommitSubject_Determinism pins the per-kind iteration
// order: the subject's per-kind summary follows entity.AllKinds()
// order regardless of how the moves slice was built. Determinism is
// load-bearing for human-diffable commit messages.
func TestArchiveCommitSubject_Determinism(t *testing.T) {
	t.Parallel()
	moves := []archiveMove{
		{kind: entity.KindADR, id: "ADR-0001"},
		{kind: entity.KindEpic, id: "E-0001"},
		{kind: entity.KindGap, id: "G-0001"},
		{kind: entity.KindContract, id: "C-0001"},
		{kind: entity.KindDecision, id: "D-0001"},
	}
	got := archiveCommitSubject(moves)
	// entity.AllKinds() order is: epic, milestone, adr, gap, decision, contract.
	// Subject should list them in that order, ignoring milestone (zero count).
	wantOrder := []string{"epic", "adr", "gap", "decision", "contract"}
	last := -1
	for _, kind := range wantOrder {
		idx := strings.Index(got, kind)
		if idx < 0 {
			t.Errorf("subject does not name kind %q:\n  %s", kind, got)
			continue
		}
		if idx <= last {
			t.Errorf("kind %q appears at idx=%d, after a later kind (idx=%d) — order broken:\n  %s", kind, idx, last, got)
		}
		last = idx
	}
}

// TestArchiveCommitBody_DeterministicAndCompliant pins the body
// shape: the body cites ADR-0004, lists per-kind counts in
// entity.AllKinds() order, and lists affected ids alphabetically
// within each kind. ADR-0004 §"`aiwf archive` verb": "the commit
// message body lists affected ids and per-kind counts."
func TestArchiveCommitBody_DeterministicAndCompliant(t *testing.T) {
	t.Parallel()
	moves := []archiveMove{
		{kind: entity.KindGap, id: "G-0017"},
		{kind: entity.KindGap, id: "G-0010"},
		{kind: entity.KindGap, id: "G-0014"},
		{kind: entity.KindEpic, id: "E-0005"},
		{kind: entity.KindEpic, id: "E-0001"},
	}
	body := archiveCommitBody(moves, nil)
	if !strings.Contains(body, "ADR-0004") {
		t.Errorf("commit body should cite ADR-0004:\n%s", body)
	}
	// Per-kind counts in AllKinds order: epic before gap.
	idxEpic := strings.Index(body, "epic")
	idxGap := strings.Index(body, "gap")
	if idxEpic < 0 || idxGap < 0 {
		t.Fatalf("body missing per-kind summary lines:\n%s", body)
	}
	if idxEpic > idxGap {
		t.Errorf("epic should appear before gap in per-kind summary; got body:\n%s", body)
	}
	// Affected ids alphabetical within each kind. The gap section
	// should list G-0010 before G-0014 before G-0017.
	idxG10 := strings.Index(body, "G-0010")
	idxG14 := strings.Index(body, "G-0014")
	idxG17 := strings.Index(body, "G-0017")
	if idxG10 < 0 || idxG14 < 0 || idxG17 < 0 {
		t.Fatalf("body missing one of G-0010/G-0014/G-0017:\n%s", body)
	}
	if idxG10 >= idxG14 || idxG14 >= idxG17 {
		t.Errorf("gap ids should be lexicographic; got order G-0010@%d G-0014@%d G-0017@%d in body:\n%s",
			idxG10, idxG14, idxG17, body)
	}
}
