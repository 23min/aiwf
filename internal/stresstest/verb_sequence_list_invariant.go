package stresstest

import (
	"context"
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/cli/list"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// verb_sequence_list_invariant.go — M-0250/AC-3: after every walk
// step, checkListInvariant cross-checks `aiwf list --archived`'s real
// subprocess output against an independently-derived ground truth
// (tree.Load, walked directly). This deliberately never calls
// internal/cli/list's own BuildListRows/BuildListCounts to derive the
// expected value — that would validate `list` against itself, making
// the check vacuous (per this milestone's own Constraints). Reusing
// just the row *type* below carries none of that risk: it's a plain
// data shape, not the row-building logic under test.

// listRow is list.ListSummary aliased, not hand-duplicated: a field
// added to ListSummary tomorrow shows up here automatically, so this
// invariant can't silently drift out of sync with what `aiwf list`
// actually emits on the wire.
type listRow = list.ListSummary

// checkListInvariant runs `aiwf list --archived` against dir and
// compares its row set to ground truth loaded directly from disk via
// tree.Load, returning one violation per divergence. label identifies
// the walk step that triggered this check, so a failure is
// reproducible without re-running the whole walk.
func checkListInvariant(aiwfBin, dir, label string) ([]Violation, error) {
	listEnv, err := runAiwfListJSON(aiwfBin, dir)
	if err != nil { //coverage:ignore defensive: same launch-failure class pinned at its source by TestVerbSequenceScenario_RealBinary_RunErrorsWhenBinaryMissing
		return nil, fmt.Errorf("running aiwf list --archived: %w", err)
	}
	if listEnv.Status != "ok" { //coverage:ignore defensive: `aiwf list --archived` (no --kind/--status/--area) has no validation branch that can refuse given a well-formed binary and a repo this scenario itself is actively driving; kept as a guard against a future list.Run refusal mode, not a currently-reachable path
		return []Violation{{Message: fmt.Sprintf(
			"%s: aiwf list --archived unexpectedly refused (status=%s, error=%+v)", label, listEnv.Status, listEnv.Error)}}, nil
	}

	tr, _, err := tree.Load(context.Background(), dir)
	if err != nil { //coverage:ignore defensive: tree.Load against a repo this scenario itself just created and is still driving has no realistic failure mode
		return nil, fmt.Errorf("loading ground truth tree: %w", err)
	}

	return classifyListInvariant(label, listEnv.Result, tr.Entities), nil
}

// classifyListInvariant is checkListInvariant's pure comparison core,
// factored out so it's directly unit-testable against fabricated
// inputs. wantEntities is ground truth (from tree.Load); gotRows is
// `aiwf list --archived`'s real output.
func classifyListInvariant(label string, gotRows []listRow, wantEntities []*entity.Entity) []Violation {
	got := make(map[string]listRow, len(gotRows))
	for i := range gotRows {
		got[gotRows[i].ID] = gotRows[i]
	}
	want := make(map[string]*listRow, len(wantEntities))
	for _, e := range wantEntities {
		want[entity.Canonicalize(e.ID)] = &listRow{
			ID:     entity.Canonicalize(e.ID),
			Kind:   string(e.Kind),
			Status: e.Status,
			Title:  e.Title,
			Parent: entity.Canonicalize(e.Parent),
			Path:   e.Path,
		}
	}

	var violations []Violation
	for id, w := range want {
		g, ok := got[id]
		if !ok {
			violations = append(violations, Violation{Message: fmt.Sprintf(
				"%s: ground truth has %s but aiwf list --archived did not show it", label, id)})
			continue
		}
		if diff := diffListRow(*w, g); diff != "" {
			violations = append(violations, Violation{Message: fmt.Sprintf(
				"%s: %s diverges from ground truth: %s", label, id, diff)})
		}
	}
	for id := range got {
		if _, ok := want[id]; !ok {
			violations = append(violations, Violation{Message: fmt.Sprintf(
				"%s: aiwf list --archived shows %s but ground truth does not have it", label, id)})
		}
	}
	return violations
}

// diffListRow returns a human-readable description of every field
// where want and got disagree (want is ground truth), or "" if they
// match exactly. ID is not compared — the caller already matched want
// and got by id.
func diffListRow(want, got listRow) string {
	var diffs []string
	if want.Kind != got.Kind {
		diffs = append(diffs, fmt.Sprintf("kind: want %q, got %q", want.Kind, got.Kind))
	}
	if want.Status != got.Status {
		diffs = append(diffs, fmt.Sprintf("status: want %q, got %q", want.Status, got.Status))
	}
	if want.Title != got.Title {
		diffs = append(diffs, fmt.Sprintf("title: want %q, got %q", want.Title, got.Title))
	}
	if want.Parent != got.Parent {
		diffs = append(diffs, fmt.Sprintf("parent: want %q, got %q", want.Parent, got.Parent))
	}
	if want.Path != got.Path {
		diffs = append(diffs, fmt.Sprintf("path: want %q, got %q", want.Path, got.Path))
	}
	return strings.Join(diffs, "; ")
}
