package verb_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/verb"
)

// corruptFrontmatter breaks the YAML of the entity at rel in the working
// copy, leaving HEAD's copy intact. This is what an entity looks like
// part-way through a hand edit: the loader cannot parse it, so it drops
// out of the loaded tree while the record still carries it.
//
// The premise is asserted here rather than assumed, via a load that
// tolerates the very errors the runner's own tree() helper fatals on.
func corruptFrontmatter(t *testing.T, r *runner, rel string) {
	t.Helper()
	abs := filepath.Join(r.root, rel)
	raw, err := os.ReadFile(abs) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("reading %s: %v", rel, err)
	}
	// An unterminated flow sequence — valid mid-keystroke, invalid YAML.
	patched := strings.Replace(string(raw), "title:", "title: [unclosed", 1)
	if patched == string(raw) {
		t.Fatalf("fixture did not corrupt %s: no title: line in\n%s", rel, raw)
	}
	if err := os.WriteFile(abs, []byte(patched), 0o600); err != nil {
		t.Fatalf("writing %s: %v", rel, err)
	}
	tr, _, loadErr := tree.Load(r.ctx, r.root)
	if loadErr != nil {
		t.Fatalf("tree.Load after corrupting %s: %v", rel, loadErr)
	}
	for _, e := range tr.Entities {
		if filepath.ToSlash(e.Path) == rel {
			t.Fatalf("premise failed: %s still appears in the loaded tree after its frontmatter was corrupted; "+
				"this fixture cannot reproduce a referrer the tree has dropped", rel)
		}
	}
}

// TestArchive_ReferrerUnparseableInWorkingCopy_DeclinesTheMove pins
// M-0286/AC-1.
//
// The sweep enumerates its referrers from the loaded tree, so an entity
// the loader dropped is examined by nothing: not by the decline, which
// never sees its committed link, and not by the rewrite pass, which never
// emits an op for it. The move lands and HEAD keeps a link to a path
// nothing occupies.
//
// The damage does not heal. IsArchivedPath excludes the archived target
// from every later scan, so no re-run reaches the stale link and
// `aiwf check` reports no error against it. Restoring the referrer
// changes nothing — which is why the verdict has to be reached before the
// move, not after.
func TestArchive_ReferrerUnparseableInWorkingCopy_DeclinesTheMove(t *testing.T) {
	t.Parallel()
	r, targetPath, linkerPath := linkingGapRunner(t)

	// The ordinary bless rhythm: the operator is mid-edit in the referrer,
	// and its frontmatter is momentarily invalid.
	corruptFrontmatter(t, r, linkerPath)

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	for _, src := range archiveMovesFor(res) {
		if src == targetPath {
			t.Errorf("archive planned to sweep %s while its referrer %s is unparseable in the working copy; "+
				"the referrer is absent from the loaded tree, so no rewrite op is emitted and "+
				"HEAD keeps a link to a path the move vacates", targetPath, linkerPath)
		}
	}
	report := skipReport(res)
	if !strings.Contains(report, targetPath) && !strings.Contains(report, "G-0001") {
		t.Errorf("the declined move is not reported; the operator sees no reason for the omission:\n%s", report)
	}
}

// TestArchive_ReferrerHandRenamed_DeclinesTheMove covers the third route
// the criterion names. The entity still parses, so the loaded tree
// carries it — at its new path. The record carries the link at the old
// one, which no working-tree walk reaches.
func TestArchive_ReferrerHandRenamed_DeclinesTheMove(t *testing.T) {
	t.Parallel()
	r, targetPath, linkerPath := linkingGapRunner(t)

	renamed := strings.Replace(linkerPath, "G-0002-linking-gap.md", "G-0002-renamed-by-hand.md", 1)
	if renamed == linkerPath {
		t.Fatalf("fixture assumption broken: %q has no recognizable slug to rename", linkerPath)
	}
	if err := os.Rename(filepath.Join(r.root, linkerPath), filepath.Join(r.root, renamed)); err != nil {
		t.Fatalf("renaming %s: %v", linkerPath, err)
	}

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	for _, src := range archiveMovesFor(res) {
		if src == targetPath {
			t.Errorf("archive planned to sweep %s while its referrer moved from %s to %s outside a verb; "+
				"the record's copy of the link lives at the old path, which no working-tree walk reaches",
				targetPath, linkerPath, renamed)
		}
	}
}

// TestArchive_UnparseableReferrer_LeavesUnrelatedCandidatesSweeping is
// the property that makes declining the right answer rather than
// refusing. Widening the candidate set to the record risks the opposite
// failure — one unparseable file stalling every move — so the per-candidate
// scope is pinned on this route too, not only on the routes that predate it.
func TestArchive_UnparseableReferrer_LeavesUnrelatedCandidatesSweeping(t *testing.T) {
	t.Parallel()
	r, targetPath, linkerPath := linkingGapRunner(t)

	// A third gap, terminal, that nothing links to.
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Independent gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	r.must(verb.Cancel(r.ctx, r.tree(), "G-0003", testActor, "fixture", false))
	independent := r.tree().ByID("G-0003")
	if independent == nil {
		t.Fatal("G-0003 missing from the fixture tree")
	}
	independentPath := filepath.ToSlash(independent.Path)

	corruptFrontmatter(t, r, linkerPath)

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	moves := archiveMovesFor(res)
	var sweptIndependent bool
	for _, src := range moves {
		if src == targetPath {
			t.Errorf("archive planned to sweep %s despite its unparseable referrer", targetPath)
		}
		if src == independentPath {
			sweptIndependent = true
		}
	}
	if !sweptIndependent {
		t.Errorf("%s links to nothing and is terminal, yet the sweep skipped it; "+
			"one unparseable file has become a whole-verb refusal, which is the behaviour "+
			"the per-candidate decline exists to replace. Planned moves: %v", independentPath, moves)
	}
}

// TestArchive_MidEditNonEntityFile_DoesNotDeclineTheMove bounds what
// widening the candidate set to the record may claim. The record carries
// every committed file, not only entity files, and plenty of them quote
// an entity path in passing — a design doc, a test fixture, a comment.
//
// Those are not referrers. Nothing rewrites them when a sweep lands, so a
// mid-edit one cannot lose a link, and declining a move for it would let
// an unrelated draft stall the sweep — reintroducing through the record
// exactly the whole-verb refusal the per-candidate decline replaced.
// entity.PathKind is what draws the line, and this is what holds it there.
func TestArchive_MidEditNonEntityFile_DoesNotDeclineTheMove(t *testing.T) {
	t.Parallel()
	r, targetPath, _ := linkingGapRunner(t)

	// A committed, non-entity document that quotes the target's path.
	noteRel := filepath.Join("docs", "design-notes.md")
	if err := os.MkdirAll(filepath.Join(r.root, "docs"), 0o750); err != nil {
		t.Fatal(err)
	}
	committed := "# Notes\n\nBackground: see [the target](" + targetPath + ").\n"
	if err := os.WriteFile(filepath.Join(r.root, noteRel), []byte(committed), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", r.root, "add", noteRel).Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", r.root, "commit", "-m", "seed a non-entity referrer").Run(); err != nil {
		t.Fatal(err)
	}
	// Now mid-edit, exactly as a referring entity would be.
	if err := os.WriteFile(filepath.Join(r.root, noteRel), []byte("# Notes\n\nDraft rewording.\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	var swept bool
	for _, src := range archiveMovesFor(res) {
		if src == targetPath {
			swept = true
		}
	}
	if !swept {
		t.Errorf("archive declined to sweep %s because the mid-edit non-entity file %s quotes its path; "+
			"no sweep rewrites that file, so it can lose no link, and blocking on it makes any "+
			"uncommitted document a whole-verb refusal. Report:\n%s", targetPath, noteRel, skipReport(res))
	}
}

// TestArchive_ReferrerDeletedFromDisk_DeclinesTheMove covers the same
// enumeration hole reached by a different route: the referrer's file is
// gone from the working copy while the record still carries it and its
// link. A path the loaded tree never lists cannot be compared against the
// record, whatever removed it from the tree.
func TestArchive_ReferrerDeletedFromDisk_DeclinesTheMove(t *testing.T) {
	t.Parallel()
	r, targetPath, linkerPath := linkingGapRunner(t)

	if err := os.Remove(filepath.Join(r.root, linkerPath)); err != nil {
		t.Fatalf("removing %s: %v", linkerPath, err)
	}

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	for _, src := range archiveMovesFor(res) {
		if src == targetPath {
			t.Errorf("archive planned to sweep %s while its referrer %s is missing from disk; "+
				"the record still carries the link, so the move strands it at HEAD",
				targetPath, linkerPath)
		}
	}
	report := skipReport(res)
	if !strings.Contains(report, targetPath) && !strings.Contains(report, "G-0001") {
		t.Errorf("the declined move is not reported; the operator sees no reason for the omission:\n%s", report)
	}
}
