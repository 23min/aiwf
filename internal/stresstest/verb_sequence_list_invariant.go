package stresstest

import (
	"context"
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// verb_sequence_list_invariant.go — M-0250/AC-3: after every walk
// step, checkListInvariant cross-checks `aiwf list --archived`'s real
// subprocess output against an independently-derived ground truth
// (tree.Load, walked directly). This deliberately never imports
// internal/cli/list — calling its own BuildListRows/BuildListCounts
// to derive the expected value would validate `list` against itself,
// making the check vacuous (per this milestone's own Constraints).

// listRow is the subset of `aiwf list --format=json`'s per-entity
// result fields this invariant compares. Mirrors
// internal/cli/list.ListSummary's field set structurally (same JSON
// shape on the wire) without importing that package.
type listRow struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
	Title  string `json:"title"`
	Parent string `json:"parent"`
	Path   string `json:"path"`
}

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
	for _, r := range gotRows {
		got[r.ID] = r
	}
	want := make(map[string]listRow, len(wantEntities))
	for _, e := range wantEntities {
		want[entity.Canonicalize(e.ID)] = listRow{
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
		if diff := diffListRow(w, g); diff != "" {
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
