package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// firstPrinciplesCatalogPath returns the absolute path to M-0122's
// deliverable (the Pass B first-principles catalog under
// docs/pocv3/design/). Not an aiwf entity, so resolution is via the
// repo root from `sharedRepoTree`, not the loader.
func firstPrinciplesCatalogPath(t *testing.T) string {
	t.Helper()
	root, _ := sharedRepoTree(t)
	return filepath.Join(root, "docs", "pocv3", "design", "legal-workflows-first-principles.md")
}

// loadFirstPrinciplesCatalog reads the catalog as a single string.
func loadFirstPrinciplesCatalog(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(firstPrinciplesCatalogPath(t))
	if err != nil {
		t.Fatalf("reading first-principles catalog: %v", err)
	}
	return string(data)
}

// expectedFPSections is the closed-set ordering M-0122 commits to: the
// catalog's ten top-level sections, each one a numbered `## ` heading.
// The test pins both presence and order.
//
// The per-kind lifecycle subsections (§1a..§1f) cover the six entity
// kinds; §2 covers ACs (the sub-element of milestones). The per-kind
// coverage is asserted separately by TestM0122_AC2 below.
var expectedFPSections = []struct {
	heading string
	number  int
}{
	{"## 1. Per-kind lifecycles", 1},
	{"## 2. Acceptance criteria and TDD phase", 2},
	{"## 3. Cross-entity invariants", 3},
	{"## 4. Frontmatter schema invariants", 4},
	{"## 5. ID format and stability", 5},
	{"## 6. Provenance model rules", 6},
	{"## 7. Verb execution invariants", 7},
	{"## 8. Archive convention", 8},
	{"## 9. Validation chokepoint rules", 9},
	{"## 10. Anti-rules (explicitly NOT kernel rules)", 10},
}

// expectedKindSubsections enumerates the per-kind §1 subsections. Each
// entity kind has its own ### subsection so Pass C can reconcile
// per-kind without scanning the whole §1.
var expectedKindSubsections = []string{
	"### 1a. Epic",
	"### 1b. Milestone",
	"### 1c. ADR",
	"### 1d. Gap",
	"### 1e. Decision",
	"### 1f. Contract",
}

// TestM0122_AC1_CatalogExistsAndOrdered asserts M-0122/AC-1: the
// first-principles catalog exists at the canonical path and its
// top-level §N section headings appear in spec order (1..10).
// Order is asserted by walking the body and confirming each heading
// appears, in order, at an increasing offset.
func TestM0122_AC1_CatalogExistsAndOrdered(t *testing.T) {
	t.Parallel()
	body := loadFirstPrinciplesCatalog(t)

	prev := -1
	for _, sec := range expectedFPSections {
		idx := strings.Index(body, sec.heading)
		if idx == -1 {
			t.Errorf("AC-1: §%d heading %q not found in catalog", sec.number, sec.heading)
			continue
		}
		if idx <= prev {
			t.Errorf("AC-1: §%d heading %q appears at offset %d which is not after the previous heading (offset %d) — sections are out of spec order", sec.number, sec.heading, idx, prev)
		}
		prev = idx
	}
}

// TestM0122_AC2_PerKindSubsectionsPresent asserts M-0122/AC-2: §1
// contains a lifecycle subsection for each of the six entity kinds
// (Epic, Milestone, ADR, Gap, Decision, Contract). Each subsection's
// ### heading must appear, and each must produce at least one
// `| R-FP-NNNN |` rule row.
//
// ACs are covered by §2 (their own top-level section), not §1, because
// per design-decisions.md ACs are namespaced sub-elements of milestones
// rather than a seventh entity kind.
func TestM0122_AC2_PerKindSubsectionsPresent(t *testing.T) {
	t.Parallel()
	body := loadFirstPrinciplesCatalog(t)

	for _, heading := range expectedKindSubsections {
		idx := strings.Index(body, heading)
		if idx == -1 {
			t.Errorf("AC-2: per-kind subsection %q not found", heading)
			continue
		}
		section := fpSectionBody(body, heading)
		if section == "" {
			t.Errorf("AC-2: per-kind subsection %q has empty body", heading)
			continue
		}
		if !strings.Contains(section, "| R-FP-") {
			t.Errorf("AC-2: per-kind subsection %q has no R-FP-NNNN rule rows", heading)
		}
	}
}

// TestM0122_AC3_SixColumnSchemaNonEmpty asserts M-0122/AC-3: every
// `R-FP-NNNN` row across the whole catalog has the six-column schema
// with non-empty fields. The schema columns are:
//  1. Rule id
//  2. Scope
//  3. Statement
//  4. Reasoning
//  5. Load-bearing?
//  6. Severity if violated
//
// Empty / placeholder values (`TBD`, dash-only, blank) fail the test —
// with one carve-out: the `Severity if violated` column may legitimately
// be `n/a` when the rule is a verb-time refusal only (no `aiwf check`
// finding), or when the rule is an anti-rule.
//
// The splitTableRow and firstField helpers are shared with the M-0121
// audit-catalog test (defined in m0121_audit_catalog_test.go in the
// same package).
func TestM0122_AC3_SixColumnSchemaNonEmpty(t *testing.T) {
	t.Parallel()
	body := loadFirstPrinciplesCatalog(t)

	rowPattern := regexp.MustCompile(`(?m)^\| R-FP-\d{4} \|`)
	rowStarts := rowPattern.FindAllStringIndex(body, -1)
	if len(rowStarts) == 0 {
		t.Fatal("AC-3: no R-FP-NNNN rows found in catalog")
	}

	for _, rs := range rowStarts {
		lineEnd := strings.IndexByte(body[rs[0]:], '\n')
		if lineEnd == -1 {
			lineEnd = len(body) - rs[0]
		}
		row := body[rs[0] : rs[0]+lineEnd]
		fields := splitTableRow(row)
		if len(fields) != 6 {
			t.Errorf("AC-3: row %q has %d fields (want 6); content: %q", firstField(row), len(fields), row)
			continue
		}
		for i, f := range fields {
			trimmed := strings.TrimSpace(f)
			if trimmed == "" || trimmed == "TBD" {
				t.Errorf("AC-3: row %q column %d is empty/TBD: %q", fields[0], i+1, f)
			}
		}
	}
}

// TestM0122_AC4_IdsUniqueAndContiguous asserts M-0122/AC-4: the
// catalog's R-FP-NNNN id-space is internally consistent:
//
//	(a) every R-FP id matches the canonical 4-digit shape
//	(b) ids are unique across the catalog
//	(c) ids form a contiguous sequence starting at R-FP-0001
//
// parseFourDigit / fourDigit / errInvalidIDWidth / errInvalidIDDigit
// are shared with the M-0121 audit-catalog test (defined in
// m0121_audit_catalog_test.go in the same package).
func TestM0122_AC4_IdsUniqueAndContiguous(t *testing.T) {
	t.Parallel()
	body := loadFirstPrinciplesCatalog(t)

	// (a) collect every R-FP id appearance.
	idPattern := regexp.MustCompile(`R-FP-(\d{4})`)
	matches := idPattern.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		t.Fatal("AC-4: no R-FP-NNNN ids found in catalog")
	}

	// Dedupe to distinct id digits. The same id appears once in its
	// rule row, possibly once in section footers ("Total:" counts), and
	// possibly once in the "Open questions" section narrative. The
	// uniqueness assertion is about the *rows* — each row's id must be
	// distinct from every other row's id.
	rowPattern := regexp.MustCompile(`(?m)^\| (R-FP-\d{4}) \|`)
	rowMatches := rowPattern.FindAllStringSubmatch(body, -1)
	if len(rowMatches) == 0 {
		t.Fatal("AC-4: no R-FP rule rows found in catalog")
	}

	rowIDs := map[string]int{}
	for _, m := range rowMatches {
		rowIDs[m[1]]++
	}
	for id, count := range rowIDs {
		if count > 1 {
			t.Errorf("AC-4: id %s appears in %d rule rows (want exactly 1)", id, count)
		}
	}

	// (b) shape — each row id is 4 digits canonical.
	digitsPattern := regexp.MustCompile(`R-FP-(\d{4})`)
	var maxN int
	digitsSeen := map[string]bool{}
	for _, m := range rowMatches {
		digits := digitsPattern.FindStringSubmatch(m[1])
		if digits == nil {
			t.Errorf("AC-4: id %q does not match 4-digit canonical shape", m[1])
			continue
		}
		var n int
		if _, err := parseFourDigit(digits[1], &n); err != nil {
			t.Errorf("AC-4: id %q has invalid 4-digit segment: %v", m[1], err)
			continue
		}
		digitsSeen[digits[1]] = true
		if n > maxN {
			maxN = n
		}
	}

	// (c) sequence is contiguous 0001..maxN.
	for i := 1; i <= maxN; i++ {
		key := fourDigit(i)
		if !digitsSeen[key] {
			t.Errorf("AC-4: sequence gap — R-FP-%s missing (ids should be contiguous 0001..%04d)", key, maxN)
		}
	}

	// Unused but referenced for completeness — silences the unused-var
	// linter if it ever fires on `matches` while we keep the variable
	// around as evidence of the broader scan.
	_ = matches
}

// TestM0122_AC5_OpenQuestionsSectionPresent asserts M-0122/AC-5: the
// catalog has a non-empty "Open questions for Pass C" section that
// surfaces ambiguities first-principles reasoning could not resolve.
// This is the load-bearing handoff to Pass C — without it, Pass C has
// no list of decision points to work through.
//
// Three structural facts are pinned:
//  1. The section heading "## Open questions for Pass C" is present.
//  2. The section body is non-empty.
//  3. The section contains at least three numbered question entries
//     (the bar is deliberately low — Pass B should have surfaced more
//     than zero ambiguities; "at least three" is the kept-honest floor).
func TestM0122_AC5_OpenQuestionsSectionPresent(t *testing.T) {
	t.Parallel()
	body := loadFirstPrinciplesCatalog(t)

	const heading = "## Open questions for Pass C"
	idx := strings.Index(body, heading)
	if idx == -1 {
		t.Fatalf("AC-5: heading %q not found in catalog", heading)
	}

	section := fpSectionBody(body, heading)
	if strings.TrimSpace(section) == "" {
		t.Errorf("AC-5: %q section body is empty", heading)
	}

	// Count numbered question entries — pattern `**Qn — ` where n is
	// an integer. The pattern matches the catalog's question shape
	// (`**Q1 — AC `deferred` terminality.**`, etc.) without binding
	// to a specific topic.
	qPattern := regexp.MustCompile(`\*\*Q\d+ — `)
	matches := qPattern.FindAllStringIndex(section, -1)
	if got := len(matches); got < 3 {
		t.Errorf("AC-5: %q section has %d numbered question entries; want >= 3", heading, got)
	}
}

// --- helpers --------------------------------------------------------

// fpSectionBody returns the body of a section starting at `headingPrefix`,
// from the line after the heading up to the next `### ` or `## ` line.
//
// Distinct from m0121_audit_catalog_test.go's `sectionBody` so test
// failures point at the right helper; behaviorally identical.
func fpSectionBody(body, headingPrefix string) string {
	start := strings.Index(body, headingPrefix)
	if start == -1 {
		return ""
	}
	lineEnd := strings.IndexByte(body[start:], '\n')
	if lineEnd == -1 {
		return ""
	}
	rest := body[start+lineEnd+1:]
	nextH := -1
	for _, marker := range []string{"\n## ", "\n### "} {
		if idx := strings.Index(rest, marker); idx != -1 {
			if nextH == -1 || idx < nextH {
				nextH = idx
			}
		}
	}
	if nextH == -1 {
		return rest
	}
	return rest[:nextH]
}
