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
// reader treats as current truth. docs/adr/ is excluded per the file
// header; docs/archive/, docs/research/ and docs/explorations/ are
// not normative.
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
			t.Fatalf("AC-4: normative doc root %s is missing — this scan would assert nothing: %v", sub, err)
		}
		walkAndAssertAbsent(t, dir, root)
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
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if loc := retiredVerbToken.FindIndex(raw); loc != nil {
			rel, _ := filepath.Rel(root, path)
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
