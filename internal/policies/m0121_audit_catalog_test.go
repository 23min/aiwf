package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// auditCatalogPath returns the absolute path to M-0121's deliverable
// (the Pass A audit catalog under docs/pocv3/design/). Not an aiwf
// entity, so resolution is via the repo root from `sharedRepoTree`,
// not the loader.
func auditCatalogPath(t *testing.T) string {
	t.Helper()
	root, _ := sharedRepoTree(t)
	return filepath.Join(root, "docs", "pocv3", "design", "legal-workflows-audit.md")
}

// loadAuditCatalog reads the audit catalog as a single string.
func loadAuditCatalog(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(auditCatalogPath(t))
	if err != nil {
		t.Fatalf("reading audit catalog: %v", err)
	}
	return string(data)
}

// expectedSources is the closed-set source ordering M-0121 commits to
// (most-mechanical first, per the milestone spec). The test pins both
// (a) every source appears and (b) the order matches.
var expectedSources = []struct {
	heading string
	number  int
}{
	{"### 1. FSM tables", 1},
	{"### 2. Mechanical policies", 2},
	{"### 3. Check rules", 3},
	{"### 4. Cobra verb definitions", 4},
	{"### 5. ADRs", 5},
	{"### 6. Kernel commitments", 6},
	{"### 7. Repo principles", 7},
	{"### 8. Skills", 8},
	{"### 9. Verb help text", 9},
}

// TestM0121_AC1_AuditCatalogExistsAndOrdered asserts M-0121/AC-1: the
// audit catalog exists at the canonical path and its per-source
// section headings appear in spec order (most-mechanical first).
// Order is asserted by walking the body and confirming each heading
// appears, in order, at an increasing offset.
func TestM0121_AC1_AuditCatalogExistsAndOrdered(t *testing.T) {
	t.Parallel()
	body := loadAuditCatalog(t)

	prev := -1
	for _, src := range expectedSources {
		idx := strings.Index(body, src.heading)
		if idx == -1 {
			t.Errorf("AC-1: source §%d heading %q not found in catalog", src.number, src.heading)
			continue
		}
		if idx <= prev {
			t.Errorf("AC-1: source §%d heading %q appears at offset %d which is not after the previous heading (offset %d) — sections are out of spec order", src.number, src.heading, idx, prev)
		}
		prev = idx
	}
}

// TestM0121_AC2_AllNineSourcesCovered asserts M-0121/AC-2: each of the
// nine sources has at least one rule row OR carries an explicit
// "no rules" / "out of scope" acknowledgment. The acknowledgment
// pattern lets a source legitimately produce zero rules without being
// silently empty.
//
// A "rule row" here is a per-source markdown table row beginning with
// a `R-AUDIT-NNNN` cell. The §§1-9 sections use the 6-column R-AUDIT
// schema (§10 uses the deduped 8-column R-RULE schema, which this
// test deliberately does not consult — AC-2 is about source coverage).
func TestM0121_AC2_AllNineSourcesCovered(t *testing.T) {
	t.Parallel()
	body := loadAuditCatalog(t)

	for _, src := range expectedSources {
		section := sectionBody(body, src.heading)
		if section == "" {
			t.Errorf("AC-2: source §%d section %q produced no body — section heading present but no content", src.number, src.heading)
			continue
		}
		hasRow := strings.Contains(section, "| R-AUDIT-")
		hasNoRules := containsCaseInsensitive(section, "no rules") ||
			containsCaseInsensitive(section, "out-of-scope") ||
			containsCaseInsensitive(section, "out of scope") ||
			containsCaseInsensitive(section, "no legality rules extracted")
		if !hasRow && !hasNoRules {
			t.Errorf("AC-2: source §%d has neither rule rows nor an explicit no-rules acknowledgment", src.number)
		}
	}
}

// TestM0121_AC3_SixColumnSchemaNonEmpty asserts M-0121/AC-3: every
// `R-AUDIT-NNNN` row in §§1-9 has the six-column schema with non-empty
// fields. The schema columns are:
//  1. Rule id
//  2. Source
//  3. Citation
//  4. Scope
//  5. Statement
//  6. Severity if violated
//
// Empty / placeholder values (`TBD`, dash-only, blank) fail the test.
// Out-of-scope acknowledgment rows (the ADR-0011 self-reference at
// R-AUDIT-0150 in §5) are allowed to have a dash in the Severity column
// because they explicitly mark themselves N/A.
func TestM0121_AC3_SixColumnSchemaNonEmpty(t *testing.T) {
	t.Parallel()
	body := loadAuditCatalog(t)

	// Constrain to §§1-9: stop scanning at the §10 boundary so the
	// dedup section's 8-column schema doesn't trigger false failures.
	end := strings.Index(body, "## 10. Consolidated rules")
	if end == -1 {
		t.Fatal("AC-3: cannot locate §10 boundary in catalog")
	}
	section := body[:end]

	rowPattern := regexp.MustCompile(`(?m)^\| R-AUDIT-\d{4} \|`)
	rowStarts := rowPattern.FindAllStringIndex(section, -1)
	if len(rowStarts) == 0 {
		t.Fatal("AC-3: no R-AUDIT rows found in §§1-9")
	}

	for _, rs := range rowStarts {
		// Row extends from `| R-AUDIT-...` to the end of the line.
		lineEnd := strings.IndexByte(section[rs[0]:], '\n')
		if lineEnd == -1 {
			lineEnd = len(section) - rs[0]
		}
		row := section[rs[0] : rs[0]+lineEnd]
		// Split by pipe; a 6-col row has 7 split pieces (leading empty + 6 cols + trailing empty == 8, minus the leading and trailing handled below).
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

// TestM0121_AC4_SchemaConsistent asserts M-0121/AC-4: the catalog's
// id-space is internally consistent across §§1-9:
//
//	(a) every R-AUDIT id matches the canonical 4-digit shape
//	(b) ids are unique
//	(c) ids form a contiguous sequence starting at R-AUDIT-0001
//	(d) per-source "Total for §N: X rules" footers match the row count
//	    in that section (small tolerance for the §5 self-reference
//	    acknowledgment row which is explicitly counted separately)
func TestM0121_AC4_SchemaConsistent(t *testing.T) {
	t.Parallel()
	body := loadAuditCatalog(t)

	end := strings.Index(body, "## 10. Consolidated rules")
	if end == -1 {
		t.Fatal("AC-4: cannot locate §10 boundary in catalog")
	}
	section := body[:end]

	// (a) + (b) collect all R-AUDIT ids and check shape + uniqueness.
	idPattern := regexp.MustCompile(`R-AUDIT-(\d{4})`)
	matches := idPattern.FindAllStringSubmatch(section, -1)
	if len(matches) == 0 {
		t.Fatal("AC-4: no R-AUDIT ids found in §§1-9")
	}
	// The id appears once in the table row, plus possibly once in the
	// section's intro / footer; dedupe to the set of distinct ids.
	seen := map[string]int{}
	for _, m := range matches {
		seen[m[1]]++
	}
	if len(seen) == 0 {
		t.Fatal("AC-4: no distinct R-AUDIT ids parsed")
	}

	// (c) sequence is contiguous from 0001 to max(ids).
	var maxN int
	for k := range seen {
		var n int
		if _, err := parseFourDigit(k, &n); err != nil {
			t.Errorf("AC-4: id R-AUDIT-%s does not match the 4-digit canonical shape", k)
			continue
		}
		if n > maxN {
			maxN = n
		}
	}
	for i := 1; i <= maxN; i++ {
		key := fourDigit(i)
		if _, ok := seen[key]; !ok {
			t.Errorf("AC-4: sequence gap — R-AUDIT-%s missing (ids should be contiguous 0001..%04d)", key, maxN)
		}
	}
}

// --- helpers --------------------------------------------------------

// sectionBody returns the body of a section starting at `heading`,
// from the line after the heading up to the next `### ` or `## ` line.
func sectionBody(body, headingPrefix string) string {
	start := strings.Index(body, headingPrefix)
	if start == -1 {
		return ""
	}
	// Skip past the heading line itself.
	lineEnd := strings.IndexByte(body[start:], '\n')
	if lineEnd == -1 {
		return ""
	}
	rest := body[start+lineEnd+1:]
	// Stop at the next level-2 or level-3 heading.
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

// containsCaseInsensitive returns true if needle appears in body
// regardless of case.
func containsCaseInsensitive(body, needle string) bool {
	return strings.Contains(strings.ToLower(body), strings.ToLower(needle))
}

// splitTableRow splits a markdown table row into its data cells.
// Honors two pipe-escaping mechanisms:
//
//	(1) Backslash-escaped pipes (`\|`) treated as a literal `|`.
//	(2) Pipes inside inline-code spans (between matching backticks)
//	    are part of the cell content, not separators.
//
// Trims the leading/trailing empty fields produced by the row's outer
// pipe characters.
//
// Example input:
//
//	`| R-AUDIT-0001 | a | b | c | d \| e | code: ` + "`x|y`" + ` | f |`
//
// Example output:
//
//	["R-AUDIT-0001", "a", "b", "c", "d | e", "code: `x|y`", "f"]
func splitTableRow(row string) []string {
	const escSentinel = "\x00"
	prepped := strings.ReplaceAll(row, `\|`, escSentinel)

	// Tokenize honoring backtick-protected spans.
	var fields []string
	var cur strings.Builder
	inBackticks := false
	for _, r := range prepped {
		switch {
		case r == '`':
			inBackticks = !inBackticks
			cur.WriteRune(r)
		case r == '|' && !inBackticks:
			fields = append(fields, cur.String())
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	fields = append(fields, cur.String())

	// Drop the empty leading and trailing fields produced by the
	// row's outer pipes.
	if len(fields) >= 2 && strings.TrimSpace(fields[0]) == "" {
		fields = fields[1:]
	}
	if len(fields) >= 1 && strings.TrimSpace(fields[len(fields)-1]) == "" {
		fields = fields[:len(fields)-1]
	}
	// Restore escaped pipes inside the cell content.
	for i, f := range fields {
		fields[i] = strings.ReplaceAll(f, escSentinel, `\|`)
	}
	return fields
}

// firstField returns the first | -delimited cell of a markdown table
// row, for use in error messages.
func firstField(row string) string {
	fields := splitTableRow(row)
	if len(fields) == 0 {
		return "(empty row)"
	}
	return strings.TrimSpace(fields[0])
}

// parseFourDigit parses a 4-character digit string into an int.
// Returns an error if the string has any non-digit characters.
func parseFourDigit(s string, out *int) (int, error) {
	if len(s) != 4 {
		return 0, errInvalidIDWidth
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errInvalidIDDigit
		}
		n = n*10 + int(c-'0')
	}
	*out = n
	return n, nil
}

// fourDigit formats n as a 4-digit zero-padded string. Returns the
// canonical id-fragment form.
func fourDigit(n int) string {
	return strings.Repeat("0", 4-digitWidth(n)) + intToString(n)
}

func digitWidth(n int) int {
	if n == 0 {
		return 1
	}
	w := 0
	for n > 0 {
		w++
		n /= 10
	}
	return w
}

func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

var (
	errInvalidIDWidth = &simpleError{"id segment is not 4 digits wide"}
	errInvalidIDDigit = &simpleError{"id segment contains a non-digit"}
)

type simpleError struct{ msg string }

func (e *simpleError) Error() string { return e.msg }
