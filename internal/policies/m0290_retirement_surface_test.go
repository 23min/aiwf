package policies

// M-0290/AC-4: no surface an operator reads to learn what aiwf can do
// still offers the retired width-migration verb.
//
// Scope is deliberate. Shipped surfaces materialize into consumer
// repos, so a line there hands a consumer a command their binary
// rejects. Normative docs are kept in lockstep with the code by the
// documentation-hierarchy contract, so a line there is simply false.
//
// Three tiers are out of scope, each for its own reason. ADRs are
// dated decision records — superseded, never rewritten — and the
// retirement's own ADR is the mechanism that records which of their
// clauses lapsed; editing them would falsify what was decided. The
// archival tier is a frozen snapshot by the same convention. And
// CHANGELOG.md is append-only, where the verb's arrival and removal
// both belong.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// retiredVerbToken matches the retired verb's name as a word, so a
// longer identifier that merely contains it does not register.
var retiredVerbToken = regexp.MustCompile(`\brewidth\b`)

// m0290DocumentaryMentionAllowlist carries the doc-tier paths whose
// subject IS the retired verb's lingering citations, keyed to the
// reason. The ban targets a document that offers the verb to an
// operator; a dated inventory reporting where others still offer it
// has the opposite effect, and cannot state its finding without
// naming the token — its evidence is verbatim quotation of entity
// bodies and of one real commit subject. m0127's allowlist draws the
// same line for the relocated-path ban, down to an entry reading
// "quotes an actual git commit message verbatim".
//
// Keep this list short. An entry is warranted only when the file
// documents the retirement; a file that merely instructs a reader to
// run the verb is the defect this test exists to catch.
var m0290DocumentaryMentionAllowlist = map[string]string{
	"docs/initiatives/entity-truth-audit.md":       "dated drift inventory whose finding is that live records still cite the retired verb",
	"docs/initiatives/gap-truth-audit.md":          "dated drift inventory naming the verb as the offender the open gaps it enumerates still cite in the present tense",
	"docs/initiatives/gap-truth-audit-evidence.md": "the same inventory's evidence file, carrying the quoted gap bodies, the measured output of the failing command, and prescriptions to delete the citation",
}

// TestM0290_AC4_NoShippedSurfaceOffersTheRetiredVerb walks every file
// that `aiwf init` / `aiwf update` materializes into a consumer repo.
func TestM0290_AC4_NoShippedSurfaceOffersTheRetiredVerb(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	shipped := []string{
		filepath.Join("internal", "skills", "embedded"),
		filepath.Join("internal", "skills", "embedded-rituals"),
		filepath.Join("internal", "skills", "embedded-guidance"),
		filepath.Join("internal", "skills", "embedded-statusline"),
		filepath.Join("internal", "skills", "embedded-hooks"),
	}
	for _, sub := range shipped {
		dir := filepath.Join(root, sub)
		if _, err := os.Stat(dir); err != nil {
			t.Fatalf("AC-4: shipped-surface root %s is missing — this scan would assert nothing: %v", sub, err)
		}
		walkAndAssertAbsent(t, dir, root)
	}
}

// TestM0290_AC4_NoNormativeDocOffersTheRetiredVerb covers the docs a
// reader treats as current truth — the Normative tier as CLAUDE.md's
// documentation hierarchy defines it, minus docs/adr/ per the file
// header. docs/archive/, docs/research/ and docs/explorations/ are not
// normative and are out of scope.
//
// docs/initiatives/ is Forward-looking rather than Normative, but is
// scanned anyway: an initiative proposing that aiwf advise an operator
// to run a retired verb is the same defect wherever it is tiered.
func TestM0290_AC4_NoNormativeDocOffersTheRetiredVerb(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	claudeMd := filepath.Join(root, "CLAUDE.md")
	raw, err := os.ReadFile(claudeMd)
	if err != nil {
		t.Fatalf("AC-4: reading CLAUDE.md: %v", err)
	}
	if retiredVerbToken.Match(raw) {
		t.Errorf("AC-4: CLAUDE.md still names the retired verb; the stable-ids commitment must state the "+
			"canonical-width rule without pointing at a migration verb that no longer exists (%s)", claudeMd)
	}

	for _, sub := range []string{
		filepath.Join("docs", "design"),
		filepath.Join("docs", "initiatives"),
	} {
		dir := filepath.Join(root, sub)
		if _, err := os.Stat(dir); err != nil {
			t.Fatalf("AC-4: doc root %s is missing — this scan would assert nothing: %v", sub, err)
		}
		walkAndAssertAbsent(t, dir, root)
	}
	// The Normative tier's standalone files, which no directory walk
	// above reaches.
	for _, rel := range []string{
		filepath.Join("docs", "architecture.md"),
		filepath.Join("docs", "overview.md"),
		filepath.Join("docs", "workflows.md"),
		filepath.Join("docs", "skill-author-guide.md"),
	} {
		raw, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("AC-4: normative doc %s is missing — this scan would assert nothing: %v", rel, err)
		}
		if loc := retiredVerbToken.FindIndex(raw); loc != nil {
			t.Errorf("AC-4: %s still names the retired width-migration verb;\n  line: %s", rel, lineAround(raw, loc[0]))
		}
	}
}

// TestM0290_AC4_AllowlistKeysAreLive keeps the allowlist from outliving
// what it exempts. The walk that consumes it skips any directory named
// "archive", which is where the archival convention moves a realized
// initiative, so an entry for an archived file stops matching anything
// and goes on reading like live coverage.
//
// Existence is too weak a test on its own: a document reworded free of
// the token leaves an entry that still resolves. A key earns its place
// only while the file it names both exists and still carries the token.
func TestM0290_AC4_AllowlistKeysAreLive(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	if len(m0290DocumentaryMentionAllowlist) == 0 {
		t.Fatal("AC-4: the allowlist is empty — delete it rather than leaving an unused exemption door open")
	}
	for rel, why := range m0290DocumentaryMentionAllowlist {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Errorf("AC-4: the allowlist names %s (%q), which cannot be read — drop the entry rather than "+
				"leaving the exemption standing: %v", rel, why, err)
			continue
		}
		if !retiredVerbToken.Match(raw) {
			t.Errorf("AC-4: the allowlist names %s (%q), which no longer names the retired verb — drop the "+
				"entry rather than leaving the exemption standing", rel, why)
		}
	}
}

// walkAndAssertAbsent reports every *.md under dir whose text names
// the retired verb, skipping the archival tier wherever it is nested.
func walkAndAssertAbsent(t *testing.T, dir, root string) {
	t.Helper()
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "archive" {
				return filepath.SkipDir
			}
			return nil
		}
		if ext := strings.ToLower(filepath.Ext(path)); ext != ".md" && ext != ".sh" {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if _, ok := m0290DocumentaryMentionAllowlist[filepath.ToSlash(rel)]; ok {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if loc := retiredVerbToken.FindIndex(raw); loc != nil {
			t.Errorf("AC-4: %s still names the retired width-migration verb at byte %d;\n"+
				"  line: %s", rel, loc[0], lineAround(raw, loc[0]))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("AC-4: walking %s: %v", dir, err)
	}
}

// lineAround returns the single line containing off, trimmed, so the
// failure names the offending sentence rather than a byte offset.
func lineAround(raw []byte, off int) string {
	start := strings.LastIndexByte(string(raw[:off]), '\n') + 1
	end := strings.IndexByte(string(raw[off:]), '\n')
	if end < 0 {
		end = len(raw)
	} else {
		end += off
	}
	return strings.TrimSpace(string(raw[start:end]))
}
