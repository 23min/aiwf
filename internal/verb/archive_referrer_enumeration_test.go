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

// TestArchive_UnparseableADRReferrer_DeclinesTheMove holds the referrer
// scan to every entity kind, not the one the other tests happen to use.
//
// A referrer is any entity whose body links into the move; nothing about
// the criterion is gap-specific. But `entity.PathKind` is the sole
// classifier separating referrers from ordinary files, and a narrowing of
// it — to one kind, or to one directory prefix — is invisible to a suite
// whose referrers are all gaps. Measured: restricting the classifier to
// gaps leaves every other test in this package green while an ADR
// referrer strands the dangling link AC-1 exists to prevent.
//
// One arrangement on a second kind buys that, where a kind dimension
// across the property grammar would double its runtime for the same
// protection.
func TestArchive_UnparseableADRReferrer_DeclinesTheMove(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Target gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	target := r.tree().ByID("G-0001")
	if target == nil {
		t.Fatal("G-0001 missing from the fixture tree")
	}
	targetPath := filepath.ToSlash(target.Path)

	r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "Adopt the target's approach", testActor,
		verb.AddOptions{BodyOverride: []byte(
			"## Context\n\nSee [the target](" + targetPath + ") for background.\n\n" +
				"## Decision\n\nFixture prose.\n\n## Consequences\n\nFixture prose.\n",
		)}))
	adr := r.tree().ByID("ADR-0001")
	if adr == nil {
		t.Fatal("ADR-0001 missing from the fixture tree")
	}
	adrPath := filepath.ToSlash(adr.Path)
	r.must(verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "fixture", false))

	corruptFrontmatter(t, r, adrPath)

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	for _, src := range archiveMovesFor(res) {
		if src == targetPath {
			t.Errorf("archive planned to sweep %s while its ADR referrer %s is unparseable; "+
				"the referrer scan is reaching gaps only, so HEAD keeps a link to a path "+
				"the move vacates", targetPath, adrPath)
		}
	}
}

// TestArchive_DirectoryAtRecordedEntityPath_DoesNotFailTheSweep pins the
// one shape that reaches the comparison but cannot be compared.
//
// Enumerating candidates from the record admits paths the working tree no
// longer holds as files. DivergentPaths refuses a directory outright, and
// that refusal is a verb-level error: it names no candidate, offers no
// remedy, and takes down every unrelated move with it — which is the
// whole-verb refusal this milestone's own constraint forbids. The path is
// divergent, so it is reported as such and judged per candidate like any
// other.
func TestArchive_DirectoryAtRecordedEntityPath_DoesNotFailTheSweep(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Target gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Other gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	r.must(verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "fixture", false))

	target := r.tree().ByID("G-0001")
	other := r.tree().ByID("G-0002")
	if target == nil || other == nil {
		t.Fatal("fixture entities missing from the tree")
	}
	targetPath := filepath.ToSlash(target.Path)
	otherPath := filepath.ToSlash(other.Path)

	// The record carries a file here; the working tree now holds a directory.
	abs := filepath.Join(r.root, otherPath)
	if err := os.Remove(abs); err != nil {
		t.Fatalf("removing %s: %v", otherPath, err)
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		t.Fatalf("creating a directory at %s: %v", otherPath, err)
	}

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive failed the whole sweep over one uncomparable path %s: %v\n"+
			"A path the record holds as a file and the tree holds as a directory is a "+
			"divergence to report, not a verb-level error", otherPath, err)
	}
	var planned bool
	for _, src := range archiveMovesFor(res) {
		if src == targetPath {
			planned = true
		}
	}
	if !planned {
		t.Errorf("%s links to nothing and is terminal, yet the sweep skipped it because %s "+
			"is a directory; one uncomparable path must cost at most its own candidate.\nReport:\n%s",
			targetPath, otherPath, skipReport(res))
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
