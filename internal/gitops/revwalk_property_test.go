package gitops

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"testing/quick"

	"github.com/google/go-cmp/cmp"
)

// revwalk_property_test.go — G-0372 Fix 1 metamorphic property test.
//
// The example-based tests (TestBulkRevwalk_MergeCommit here, plus
// TestBatchedWalker_OctopusMerge and
// TestWalkStatusChanges_MergeWithBothParentsDiffering in internal/check)
// each pin ONE hand-picked repo shape. This file instead generates many
// randomized commit-graph shapes via testing/quick (the same generative
// framework internal/areagroup/partition_property_test.go and
// internal/entity/transition_property_test.go's sibling packages use — no
// new dependency) and checks the general claim that motivated dropping -m:
// for ANY history, a non-merge commit's CommitRecord is identical whether
// -m is requested or not, and a merge commit's CommitRecord carries zero
// Paths without -m (production behavior) but retains a real per-parent
// fan-out with -m (the pre-G-0372 oracle, reached via the unexported
// bulkRevwalk(..., []string{"-m"}, ...) — see its doc comment in
// revwalk.go). This is the same "cached ∪ incremental == fresh" style of
// set-equality validation the design initiative
// (docs/initiatives/check-performance-incremental-revwalk-cache.md) used to
// validate the larger (out-of-scope) cache proposal, applied here to the
// smaller, already-shipped -m removal.
//
// Determinism: fixed-seed *rand.Rand, so a green run is reproducible and any
// counterexample is stable (no wall-clock dependence, per the repo's test
// discipline). Each trial drives real git subprocesses, so MaxCount is much
// lower than a pure in-memory property (mergePropertyMaxCount vs.
// partition_property_test.go's 2000) — still a large multiple of the 3
// hand-picked example shapes it complements.

const (
	maxMergeBranches      = 3
	maxCommitsPerBranch   = 2
	mergePropertyMaxCount = 15
)

var mergePropFiles = []string{"a.txt", "b.txt", "c.txt"}

// mergeShapeInput is one generated commit-graph shape: how many feature
// branches fork from the root commit, how many commits each contributes,
// whether each renames a.txt partway through, and a pool of content seeds
// used to fabricate distinct file bodies and merge-resolution content
// deterministically — a second, unseeded randomness source would break
// reproducibility.
type mergeShapeInput struct {
	numBranches      int
	commitsPerBranch []int
	renamesA         []bool
	contentSeeds     []int
}

// Generate implements testing/quick.Generator.
func (mergeShapeInput) Generate(r *rand.Rand, _ int) reflect.Value {
	n := 1 + r.Intn(maxMergeBranches)
	commits := make([]int, n)
	renames := make([]bool, n)
	for i := range commits {
		commits[i] = 1 + r.Intn(maxCommitsPerBranch)
		renames[i] = r.Intn(2) == 0
	}
	seeds := make([]int, 64) // generous fixed pool; unused entries are harmless
	for i := range seeds {
		seeds[i] = r.Intn(1_000_000)
	}
	return reflect.ValueOf(mergeShapeInput{numBranches: n, commitsPerBranch: commits, renamesA: renames, contentSeeds: seeds})
}

// TestBulkRevwalk_DropM_Property is the metamorphic property test described
// above.
func TestBulkRevwalk_DropM_Property(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	var note string
	property := func(in mergeShapeInput) bool {
		seedIdx := 0
		nextSeed := func() int {
			s := in.contentSeeds[seedIdx%len(in.contentSeeds)]
			seedIdx++
			return s
		}

		root := t.TempDir()
		buildRandomMergeRepo(t, ctx, root, in, nextSeed)

		withoutM := collectRecordsGrouped(t, ctx, root, nil)
		withM := collectRecordsGrouped(t, ctx, root, []string{"-m"})

		if diff := cmp.Diff(sortedMapKeys(withoutM), sortedMapKeys(withM)); diff != "" {
			note = "commit SHA sets differ between -m and no -m (-withoutM +withM):\n" + diff
			return false
		}

		for sha, noMRecs := range withoutM {
			if len(noMRecs) != 1 {
				note = fmt.Sprintf("sha %s: expected exactly 1 record without -m, got %d", sha, len(noMRecs))
				return false
			}
			noM := noMRecs[0]
			isMerge := len(noM.Parents) > 1

			if isMerge {
				if len(noM.Paths) != 0 {
					note = fmt.Sprintf("merge sha %s: Paths without -m = %+v, want empty", sha, noM.Paths)
					return false
				}
				continue
			}

			mRecs := withM[sha]
			if len(mRecs) != 1 {
				note = fmt.Sprintf("non-merge sha %s: expected exactly 1 record with -m, got %d", sha, len(mRecs))
				return false
			}
			if diff := cmp.Diff(sortedPathTouches(mRecs[0].Paths), sortedPathTouches(noM.Paths)); diff != "" {
				note = fmt.Sprintf("non-merge sha %s: Paths differ between -m and no -m (-withM +withoutM):\n%s", sha, diff)
				return false
			}
		}
		return true
	}
	cfg := &quick.Config{MaxCount: mergePropertyMaxCount, Rand: rand.New(rand.NewSource(20372))}
	if err := quick.Check(property, cfg); err != nil {
		t.Errorf("BulkRevwalk -m removal property: %s\n%v", note, err)
	}
}

// buildRandomMergeRepo constructs the randomized-but-deterministic commit
// graph in.numBranches feature branches, each forked from a shared root
// commit, each with commitsPerBranch[i] commits touching a random file from
// mergePropFiles, optionally renaming a.txt partway through; main itself
// advances once more; then every feature branch is merged back into main
// one at a time via a --no-ff merge whose resolution is FORCED to overwrite
// a.txt (guaranteeing a real diff at every merge commit, so the property
// exercises something rather than a no-op merge).
func buildRandomMergeRepo(t *testing.T, ctx context.Context, root string, in mergeShapeInput, nextSeed func() int) {
	t.Helper()
	if err := Init(ctx, root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	writeFileAndCommit(t, ctx, root, "a.txt", "root\n", "root commit")

	branches := make([]string, 0, in.numBranches)
	for i := 0; i < in.numBranches; i++ {
		name := fmt.Sprintf("feature-%d", i)
		if err := run(ctx, root, "checkout", "-q", "-b", name); err != nil {
			t.Fatalf("checkout -b %s: %v", name, err)
		}
		for j := 0; j < in.commitsPerBranch[i]; j++ {
			path := mergePropFiles[nextSeed()%len(mergePropFiles)]
			writeFileAndCommit(t, ctx, root, path, randomFileContent(nextSeed()), fmt.Sprintf("%s commit %d on %s", path, j, name))
		}
		if in.renamesA[i] {
			newName := fmt.Sprintf("a-renamed-%d.txt", i)
			if err := Mv(ctx, root, "a.txt", newName); err == nil {
				if err := Commit(ctx, root, "rename a.txt on "+name, "", nil); err != nil {
					t.Fatalf("commit rename on %s: %v", name, err)
				}
			}
			// a.txt may already be absent on this branch (an earlier
			// commit in the same loop could, in principle, have removed
			// it); Mv failing is tolerated rather than fatal since the
			// rename is a bonus exercise of the walker's rename-chain
			// path, not the property's core claim.
		}
		branches = append(branches, name)
		if err := run(ctx, root, "checkout", "-q", "main"); err != nil {
			t.Fatalf("checkout main: %v", err)
		}
	}

	// Advance main itself so every merge has a real, distinguishing
	// first-parent history rather than merging straight onto the root.
	writeFileAndCommit(t, ctx, root, "b.txt", randomFileContent(nextSeed()), "main advances")

	for _, name := range branches {
		mergeWithForcedResolution(t, ctx, root, name, nextSeed())
	}
}

// mergeWithForcedResolution merges branch into the current HEAD (main) via
// --no-ff --no-commit, then unconditionally overwrites a.txt with new
// content before completing the commit — guaranteeing the resulting merge
// commit's tree differs from at least one parent's a.txt blob regardless of
// whether git's own auto-merge would have conflicted. Mirrors the
// gitMergeWithResolution helper in
// internal/check/fsm_history_consistent_test.go (unavoidable duplication
// across the package boundary — that helper is unexported in a different
// package).
func mergeWithForcedResolution(t *testing.T, ctx context.Context, root, branch string, seed int) {
	t.Helper()
	// Tolerate a non-zero exit: --no-ff --no-commit "fails" (leaves the
	// index conflicted) exactly when both sides touched the same file,
	// which is expected and handled by the forced overwrite below either
	// way.
	_ = run(ctx, root, "merge", "--no-ff", "--no-commit", branch)
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte(randomFileContent(seed)+"-merged\n"), 0o644); err != nil {
		t.Fatalf("write a.txt merge resolution: %v", err)
	}
	if err := run(ctx, root, "add", "-A"); err != nil {
		t.Fatalf("add -A after merge attempt with %s: %v", branch, err)
	}
	if err := run(ctx, root, "commit", "-q", "-m", "merge "+branch); err != nil {
		t.Fatalf("commit merge %s: %v", branch, err)
	}
}

func randomFileContent(seed int) string {
	return fmt.Sprintf("v%d\n", seed)
}

func writeFileAndCommit(t *testing.T, ctx context.Context, root, path, content, subj string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, path), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if err := Add(ctx, root, path); err != nil {
		t.Fatalf("add %s: %v", path, err)
	}
	if err := Commit(ctx, root, subj, "", nil); err != nil {
		t.Fatalf("commit %s: %v", subj, err)
	}
}

// collectRecordsGrouped runs bulkRevwalk with extraArgs and groups the
// resulting records by Commit SHA — a non-merge commit maps to exactly one
// record; a merge commit maps to one record without -m (this is the claim
// under test) or len(Parents) records with -m (the pre-fix oracle).
func collectRecordsGrouped(t *testing.T, ctx context.Context, root string, extraArgs []string) map[string][]CommitRecord {
	t.Helper()
	out := map[string][]CommitRecord{}
	err := bulkRevwalk(ctx, root, extraArgs, func(rec CommitRecord) error {
		out[rec.Commit] = append(out[rec.Commit], rec)
		return nil
	})
	if err != nil {
		t.Fatalf("bulkRevwalk(extraArgs=%v): %v", extraArgs, err)
	}
	return out
}

func sortedMapKeys(m map[string][]CommitRecord) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// sortedPathTouches returns a copy of paths sorted by (Path, SrcPath,
// Status) — comparing sorted copies rather than raw slices removes any
// dependency on git's output order being guaranteed identical between the
// -m and no-m invocations; the property under test is set equality, not
// order.
func sortedPathTouches(paths []PathTouch) []PathTouch {
	out := append([]PathTouch(nil), paths...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		if out[i].SrcPath != out[j].SrcPath {
			return out[i].SrcPath < out[j].SrcPath
		}
		return out[i].Status < out[j].Status
	})
	return out
}
