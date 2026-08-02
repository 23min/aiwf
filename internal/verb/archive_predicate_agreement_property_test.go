package verb_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// This file carries M-0286/AC-5: the claim that the decline predicate and
// the rewrite predicate never count different entities.
//
// The other four criteria each name one arrangement where the two
// disagreed. This one asserts no fifth exists, so its evidence is a
// property over constructed tree states rather than a reproduction of any
// particular one. It deliberately asserts nothing structural — that one
// function calls another is an implementation fact a refactor may change
// while the behaviour rots underneath it, which is how a ledger recording
// names rather than behaviour failed earlier in this epic.
//
// Enumeration is exhaustive over a small grammar rather than randomized,
// following the house style already set by the entity-FSM property tests:
// where the input space is small enough to cover completely, covering it
// completely is strictly stronger than sampling it.
//
// The grammar varies the referrer, not the candidate. A dimension for the
// candidate's own body being mid-edit was measured and dropped: it
// doubled the runtime and detected nothing these arrangements do not
// already detect, and the candidate's own dirty file is pinned directly
// by TestArchive_DirtyTargetItself_SkipsIt. A dimension that costs time
// and catches nothing makes the suite slower to run and no likelier to
// fail when the behaviour breaks.

// workingState is what the operator has done to the referrer since it was
// committed. Each is a shape the sweep must reach a verdict about.
type workingState string

const (
	referrerClean    workingState = "clean"
	referrerDropLink workingState = "drop-link" // mid-edit, the committed link removed
	referrerAddLink  workingState = "add-link"  // mid-edit, a link added since the commit
	referrerCorrupt  workingState = "corrupt"   // frontmatter momentarily unparseable
	referrerDeleted  workingState = "deleted"   // gone from disk, still in the record
)

// arrangement is one constructed tree state.
type arrangement struct {
	committedLink bool // the referrer's committed body links into the target's move
	working       workingState
	archived      bool // the referrer lives under archive/
	occupyDest    bool // something already sits at the target's archive destination
}

func (a arrangement) name() string {
	return fmt.Sprintf("link=%t/%s/archived=%t/occupied=%t",
		a.committedLink, a.working, a.archived, a.occupyDest)
}

// arrangements enumerates every coherent combination. A drop-link state
// needs a committed link to drop, and an add-link state needs the commit
// not to carry one already; the rest apply either way.
func arrangements() []arrangement {
	var out []arrangement
	for _, committedLink := range []bool{true, false} {
		states := []workingState{referrerClean, referrerCorrupt, referrerDeleted}
		if committedLink {
			states = append(states, referrerDropLink)
		} else {
			states = append(states, referrerAddLink)
		}
		for _, ws := range states {
			for _, archived := range []bool{true, false} {
				for _, occupy := range []bool{true, false} {
					out = append(out, arrangement{
						committedLink: committedLink,
						working:       ws,
						archived:      archived,
						occupyDest:    occupy,
					})
				}
			}
		}
	}
	return out
}

// buildArrangement materializes a into a fresh repo and returns the
// runner plus the target's active path.
//
// Every call that needs a loaded tree happens before the working copy is
// disturbed: a corrupted or deleted referrer makes the runner's tree()
// helper fatal, which would be a fixture failure rather than a finding.
func buildArrangement(t *testing.T, a arrangement) (r *runner, targetPath string) {
	t.Helper()
	r = newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Target gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	target := r.tree().ByID("G-0001")
	if target == nil {
		t.Fatal("G-0001 missing from the fixture tree")
	}
	targetPath = filepath.ToSlash(target.Path)

	body := bornCompleteFixtureBody(entity.KindGap)
	if a.committedLink {
		body = []byte("## What's missing\n\nSee [the target](" + targetPath + ") for context.\n\n" +
			"## Why it matters\n\nFixture prose.\n")
	}
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Referrer gap", testActor,
		verb.AddOptions{BodyOverride: body}))

	if a.archived {
		// Sweep the referrer into archive/ while the target is still active,
		// so its committed link keeps pointing at the target's active path.
		r.must(verb.Cancel(r.ctx, r.tree(), "G-0002", testActor, "fixture", false))
		r.must(verb.Archive(r.ctx, r.root, testActor, ""))
	}

	// The target becomes the sweep candidate.
	r.must(verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "fixture", false))

	referrer := r.tree().ByID("G-0002")
	if referrer == nil {
		t.Fatal("G-0002 missing from the fixture tree")
	}
	referrerPath := filepath.ToSlash(referrer.Path)

	// From here on the working copy may stop loading; no tree() calls.
	switch a.working {
	case referrerClean:
	case referrerDropLink:
		dirtyEntity(t, r, "G-0002", "See [the target]("+targetPath+") for context.",
			"Draft rewording, link not yet re-added.")
	case referrerAddLink:
		dirtyEntity(t, r, "G-0002",
			"## Why it matters\n\nFixture prose for test setup; not the subject under test.",
			"## Why it matters\n\nSee [the target]("+targetPath+") for context.")
	case referrerCorrupt:
		corruptFrontmatter(t, r, referrerPath)
	case referrerDeleted:
		if err := os.Remove(filepath.Join(r.root, referrerPath)); err != nil {
			t.Fatalf("removing %s: %v", referrerPath, err)
		}
	}

	if a.occupyDest {
		occupyPath(t, r, archiveDestinationOf(t, targetPath))
	}
	return r, targetPath
}

// TestArchive_DeclineAndRewriteAgree_Property is M-0286/AC-5.
//
// Every disagreement between the two predicates ends the same way: a move
// nothing declined carries a write against a file the commit-side guard
// then refuses, taking the whole verb down with it. So "the two
// predicates count the same entities" and "every plan the sweep offers is
// a plan it can land" are the same claim, and the second is observable
// without reaching inside either predicate.
func TestArchive_DeclineAndRewriteAgree_Property(t *testing.T) {
	t.Parallel()
	for _, a := range arrangements() {
		t.Run(a.name(), func(t *testing.T) {
			t.Parallel()
			r, _ := buildArrangement(t, a)

			res, err := verb.Archive(r.ctx, r.root, testActor, "")
			if err != nil {
				t.Fatalf("Archive errored on a tree it should have a verdict about: %v", err)
			}
			if res.Plan == nil {
				return // converged: nothing promised, nothing owed
			}
			if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
				t.Fatalf("the sweep offered a plan it cannot land: %v\n"+
					"A move survived the decline while the rewrite pass emitted a write the guard "+
					"refuses — the two predicates counted different entities.\nReport:\n%s",
					applyErr, skipReport(res))
			}
		})
	}
}

// TestArchive_DeclineIsAttributable_Property is the criterion's "only if"
// direction: a candidate is declined *only* when a file its verdict rests
// on is mid-edit. Two things have to hold of every path the sweep blames,
// and they fail independently.
//
// It must genuinely differ from the record — blaming a clean file means
// the decline counts an entity nothing else does. And, unless it is a
// destination the sweep lands on, it must be a file some sweep could
// rewrite, which excludes anything under archive/: planArchiveRewrites
// skips archived entities under ADR-0004's forget-by-default rule, so an
// archived body can lose no link however mid-edit it is. A blamed path
// that is dirty but archived is dirty and irrelevant, which the first
// check alone would wave through.
//
// The destination carve-out is not a softening. A move's destination sits
// under archive/ by construction, and occupied destinations are exactly
// what the sweep must blame; the rule being checked is about referrers.
func TestArchive_DeclineIsAttributable_Property(t *testing.T) {
	t.Parallel()
	for _, a := range arrangements() {
		t.Run(a.name(), func(t *testing.T) {
			t.Parallel()
			r, targetPath := buildArrangement(t, a)
			destination := archiveDestinationOf(t, targetPath)

			res, err := verb.Archive(r.ctx, r.root, testActor, "")
			if err != nil {
				t.Fatalf("Archive: %v", err)
			}
			dirty := divergentFromRecord(t, r.root)
			for _, blamed := range blamedPaths(skipReport(res)) {
				if !dirty[blamed] {
					t.Errorf("the sweep declined a candidate and blamed %q, which matches the record; "+
						"nothing about that file is undecidable.\ngit reports dirty: %v\nReport:\n%s",
						blamed, sortedKeys(dirty), skipReport(res))
				}
				if entity.IsArchivedPath(blamed) && blamed != destination {
					t.Errorf("the sweep declined a candidate and blamed the archived file %q, which is "+
						"not a destination it lands on; no sweep rewrites an archived body, so its "+
						"state cannot decide any move.\nReport:\n%s", blamed, skipReport(res))
				}
			}
		})
	}
}

// entityLink matches a markdown link destination pointing at an entity
// file, which is the only link shape a sweep rewrites.
var entityLink = regexp.MustCompile(`\]\(([^)]*\.md)\)`)

// TestArchive_NoDanglingLinkAfterSweep_Property is the half of M-0286/AC-5
// that an applicability property cannot reach.
//
// A disagreement where one predicate counts an entity the other does not
// shows up as a refused plan — but a disagreement where *neither* counts
// it is silent. That is the shape AC-1 names: a referrer the loader
// dropped is invisible to the decline and to the rewrite pass alike, so
// the move lands, no write is emitted, nothing is refused, and the record
// keeps a link to a path nothing occupies.
//
// Nothing repairs it afterwards. IsArchivedPath excludes the archived
// target from every later scan and `aiwf check` reports no error, so the
// only place this is observable is here, immediately after the sweep.
//
// Scope is the active tree. An archived body is deliberately not
// maintained — ADR-0004's forget-by-default rule is why the rewrite pass
// skips archived entities at all, and AC-2 rests on that same exclusion —
// so a stale link inside archive/ is the documented cost of the rule, not
// a disagreement between predicates.
func TestArchive_NoDanglingLinkAfterSweep_Property(t *testing.T) {
	t.Parallel()
	for _, a := range arrangements() {
		t.Run(a.name(), func(t *testing.T) {
			t.Parallel()
			r, _ := buildArrangement(t, a)

			res, err := verb.Archive(r.ctx, r.root, testActor, "")
			if err != nil {
				t.Fatalf("Archive: %v", err)
			}
			if res.Plan == nil {
				return
			}
			if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
				return // the applicability property owns this failure
			}
			for _, rel := range committedEntityFiles(t, r.root) {
				if entity.IsArchivedPath(rel) {
					continue
				}
				raw, readErr := exec.Command("git", "-C", r.root, "show", "HEAD:"+rel).Output()
				if readErr != nil {
					t.Fatalf("reading %s at HEAD: %v", rel, readErr)
				}
				for _, dest := range entityLink.FindAllStringSubmatch(string(raw), -1) {
					if linkResolves(r.root, rel, dest[1]) {
						continue
					}
					t.Errorf("after the sweep, %s links to %q at HEAD and nothing occupies that path.\n"+
						"The move landed without its link rewrite: neither predicate counted the "+
						"referrer, so nothing declined the move and nothing rewrote the link. "+
						"No later run repairs this — the archived target leaves every subsequent "+
						"scan.", rel, dest[1])
				}
			}
		})
	}
}

// linkResolves reports whether dest, as written inside the file at
// linkingPath, names a file that exists. Both spellings a body may use
// are accepted: repo-relative, and relative to the linking file's own
// directory.
func linkResolves(root, linkingPath, dest string) bool {
	if _, err := os.Stat(filepath.Join(root, dest)); err == nil {
		return true
	}
	rel := filepath.Join(filepath.Dir(filepath.Join(root, linkingPath)), dest)
	_, err := os.Stat(rel)
	return err == nil
}

// committedEntityFiles lists the entity files HEAD records.
func committedEntityFiles(t *testing.T, root string) []string {
	t.Helper()
	out, err := exec.Command("git", "-C", root, "ls-tree", "--full-tree", "-r", "--name-only", "HEAD").Output()
	if err != nil {
		t.Fatalf("git ls-tree: %v", err)
	}
	var files []string
	for _, p := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if p == "" {
			continue
		}
		if _, ok := entity.PathKind(p); ok {
			files = append(files, p)
		}
	}
	return files
}

// blamedPaths pulls the file paths out of the sweep's skip report. The
// report renders them after "uncommitted changes in ", comma-separated.
func blamedPaths(report string) []string {
	var out []string
	for _, line := range strings.Split(report, "\n") {
		_, rest, found := strings.Cut(line, "uncommitted changes in ")
		if !found {
			continue
		}
		rest = strings.TrimSuffix(strings.TrimSpace(rest), ")")
		for _, p := range strings.Split(rest, ",") {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}

// divergentFromRecord returns the paths git reports as differing from the
// index or HEAD — modified, deleted, or never recorded alike.
func divergentFromRecord(t *testing.T, root string) map[string]bool {
	t.Helper()
	out, err := exec.Command("git", "-C", root, "status", "--porcelain", "-z", "--untracked-files=all").Output()
	if err != nil {
		t.Fatalf("git status: %v", err)
	}
	dirty := map[string]bool{}
	for _, rec := range strings.Split(strings.TrimRight(string(out), "\x00"), "\x00") {
		if len(rec) < 4 {
			continue
		}
		dirty[strings.TrimSpace(rec[3:])] = true
	}
	return dirty
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
