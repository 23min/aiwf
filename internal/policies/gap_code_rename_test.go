package policies

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// oldGapCode is the pre-M-0142 finding code, assembled from two fragments
// so this very test file does not contain the literal it forbids — the
// absence walk below scans every file under internal/ including this one,
// so a verbatim occurrence here would be a false positive. The new code
// (gap-addressed-has-resolver) is written contiguously elsewhere; only the
// old literal is forbidden, and it is not a substring of the new one.
const oldGapCode = "gap-resolved" + "-has-resolver"

const newGapCode = "gap-addressed-has-resolver"

// TestM0142_AC1_Decision is M-0142/AC-1: the rename is governed by an
// accepted decision (D-0012) that resolves via the loader, carries its
// named sections with non-empty prose, and records both code strings plus
// the JSON-surface downstream-consumer caveat in its Resolution section.
//
// Per CLAUDE.md *Testing* §"Substring assertions are not structural
// assertions", the caveat literals are asserted inside the extracted
// `## Resolution` section, not flat over the whole file — a caveat phrase
// floating in Context would not satisfy the AC.
func TestM0142_AC1_Decision(t *testing.T) {
	t.Parallel()
	root, tr := sharedRepoTree(t)

	e := tr.ByID("D-0012")
	if e == nil {
		t.Fatal("AC-1: D-0012 not found in tree (active or archive)")
	}
	if e.Status != "accepted" {
		t.Errorf("AC-1: D-0012 status = %q, want accepted", e.Status)
	}

	data, err := os.ReadFile(filepath.Join(root, e.Path))
	if err != nil {
		t.Fatalf("reading D-0012 at %s: %v", e.Path, err)
	}
	body := string(data)

	// Named sections present with non-empty prose.
	for _, name := range []string{"Context", "Resolution", "Consequences"} {
		section := extractMarkdownSection(body, 2, name)
		if section == "" {
			t.Errorf("AC-1: D-0012 must have a `## %s` section", name)
			continue
		}
		if !hasNonEmptyProse(section) {
			t.Errorf("AC-1: D-0012 `## %s` section is empty / placeholder only", name)
		}
	}

	// The Resolution section records the concrete rename (both code strings)
	// and the downstream-consumer caveat over the --format=json surface.
	resolution := extractMarkdownSection(body, 2, "Resolution")
	for _, lit := range []string{oldGapCode, newGapCode} {
		if !strings.Contains(resolution, lit) {
			t.Errorf("AC-1: `## Resolution` must name the code string %q", lit)
		}
	}
	lower := strings.ToLower(resolution)
	for _, caveat := range []string{"json", "breaking", "downstream"} {
		if !strings.Contains(lower, caveat) {
			t.Errorf("AC-1: `## Resolution` must convey the %q caveat", caveat)
		}
	}
}

// TestM0142_AC2_OldGapCodeFullyRenamed is M-0142/AC-2's absence chokepoint:
// the retired literal appears nowhere in non-archive internal/ source —
// impl, spec, hint, embedded skills, tests, fixtures, and goldens. It walks
// the whole internal/ tree (skipping archive/ subtrees per ADR-0004's
// forget-by-default) and fails CI naming every file that reintroduces the
// old code. Paired with TestGapAddressedHasResolver (the behavioral half in
// internal/check), this pins both "fires under the new name" and "the old
// name is gone".
func TestM0142_AC2_OldGapCodeFullyRenamed(t *testing.T) {
	t.Parallel()
	internalDir := filepath.Join(repoRoot(t), "internal")

	var offenders []string
	err := filepath.WalkDir(internalDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Historical entities under archive/ keep their original
			// code references (ADR-0004 forget-by-default); the rename
			// does not rewrite them.
			if d.Name() == "archive" {
				return filepath.SkipDir
			}
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(data), oldGapCode) {
			rel, relErr := filepath.Rel(repoRoot(t), path)
			if relErr != nil {
				rel = path
			}
			offenders = append(offenders, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", internalDir, err)
	}
	if len(offenders) > 0 {
		t.Errorf("AC-2: retired finding code %q must not appear in non-archive internal/ source; found in:\n  %s",
			oldGapCode, strings.Join(offenders, "\n  "))
	}
}

// gapAddressedHintKeyPattern matches the hint-table key for the renamed
// code in internal/check/hint.go — used by the AC-2 spec note's sibling
// AC-3 guarantee (PolicyFindingCodesHaveHints) as a readable cross-check
// that the new key is present, independent of the policy's AST walk.
var gapAddressedHintKeyPattern = regexp.MustCompile(`(?m)^\s*"` + regexp.QuoteMeta(newGapCode) + `"\s*:`)

// TestM0142_AC3_HintKeyPresent is a readable companion to AC-3's
// PolicyFindingCodesHaveHints guarantee: it asserts the renamed code has a
// concrete key in the hint table source. PolicyFindingCodesHaveHints is the
// load-bearing chokepoint (it fails if any emitted Code: literal lacks a
// hint); this test makes the new-key presence explicit and legible.
func TestM0142_AC3_HintKeyPresent(t *testing.T) {
	t.Parallel()
	hintPath := filepath.Join(repoRoot(t), "internal", "check", "hint.go")
	data, err := os.ReadFile(hintPath)
	if err != nil {
		t.Fatalf("reading %s: %v", hintPath, err)
	}
	if !gapAddressedHintKeyPattern.Match(data) {
		t.Errorf("AC-3: hint.go hintTable must carry a %q key", newGapCode)
	}
}
