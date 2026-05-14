package check

// archive_hint_test.go — M-0085 AC-8: hint-text regression pin and
// SKILL.md table-cell polish.
//
// Per the M-0086 wrap log, the SKILL.md table cells for the two
// finding codes (`terminal-entity-not-archived`, `archive-sweep-pending`)
// dropped backticked references to `aiwf archive --apply` to satisfy
// the skill-coverage policy (which fails CI on backticked references
// to non-existent verbs). With M-0085 landing the verb, the backticks
// are restored. This file pins both surfaces:
//
//   - The hint-table regression (HintFor returns the backticked form
//     the user reads in `aiwf check` output).
//   - The SKILL.md table-cell polish: each row's "Fix:" cell now
//     contains the backticked verb form, not the prose-name fallback.
//
// Per CLAUDE.md "Substring assertions are not structural assertions":
// the SKILL.md test scopes the substring search to the table row for
// the named finding code, not a flat-grep over the whole file. A
// stray backticked mention elsewhere in the document would not count.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHintFor_TerminalEntityNotArchived_ContainsBacktickedVerb pins
// the M-0085 AC-8 polish at the structured-Finding surface: the hint
// returned by HintFor for `terminal-entity-not-archived` references
// the verb in backticks (`aiwf archive --apply`), not in prose. A
// future drop of backticks (or a pivot to prose phrasing) fails this
// test before it ships to consumer output.
func TestHintFor_TerminalEntityNotArchived_ContainsBacktickedVerb(t *testing.T) {
	t.Parallel()
	hint := HintFor("terminal-entity-not-archived", "")
	if hint == "" {
		t.Fatal("HintFor(\"terminal-entity-not-archived\") returned empty string")
	}
	// Both invocation forms should appear backticked: the dry-run
	// preview and the apply-to-commit step.
	for _, want := range []string{"`aiwf archive --dry-run`", "`aiwf archive --apply`"} {
		if !strings.Contains(hint, want) {
			t.Errorf("hint for terminal-entity-not-archived does not contain %s\n  hint: %q", want, hint)
		}
	}
}

// TestHintFor_ArchiveSweepPending_ContainsBacktickedVerb is the
// per-tree aggregate counterpart. Same structural shape as the leaf
// rule above.
func TestHintFor_ArchiveSweepPending_ContainsBacktickedVerb(t *testing.T) {
	t.Parallel()
	hint := HintFor("archive-sweep-pending", "")
	if hint == "" {
		t.Fatal("HintFor(\"archive-sweep-pending\") returned empty string")
	}
	for _, want := range []string{"`aiwf archive --dry-run`", "`aiwf archive --apply`"} {
		if !strings.Contains(hint, want) {
			t.Errorf("hint for archive-sweep-pending does not contain %s\n  hint: %q", want, hint)
		}
	}
}

// TestApplyHints_ArchiveFindings_CarryBacktickedHint pins the
// structured-Finding-value surface the prompt names: a Finding
// constructed with one of the two M-0086 codes, after applyHints,
// carries the backticked hint string in its Hint field. This is
// what JSON consumers and rendered text both read.
func TestApplyHints_ArchiveFindings_CarryBacktickedHint(t *testing.T) {
	t.Parallel()
	cases := []struct {
		code string
	}{
		{"terminal-entity-not-archived"},
		{"archive-sweep-pending"},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			t.Parallel()
			findings := []Finding{
				{Code: tc.code, Severity: SeverityWarning, Message: "test fixture"},
			}
			applyHints(findings)
			if findings[0].Hint == "" {
				t.Fatalf("applyHints left Hint empty for code %s", tc.code)
			}
			if !strings.Contains(findings[0].Hint, "`aiwf archive --apply`") {
				t.Errorf("Finding(code=%s).Hint does not contain `aiwf archive --apply`:\n  %q", tc.code, findings[0].Hint)
			}
		})
	}
}

// TestSkillCheckSkillMd_ArchiveTableRowsBacktickedVerb is the
// SKILL.md polish surface: the aiwf-check SKILL.md's table row for
// each of the two finding codes contains a backticked
// `aiwf archive --apply` reference in the row's Fix cell, not a
// prose name like "the archive sweep verb (M-0085)".
//
// Per CLAUDE.md "Substring assertions are not structural assertions":
// we scope the substring match to the row whose first column literally
// names the finding code, so a stray reference elsewhere in the
// document doesn't satisfy the assertion.
func TestSkillCheckSkillMd_ArchiveTableRowsBacktickedVerb(t *testing.T) {
	t.Parallel()
	// Locate the SKILL.md by walking up to the kernel root, then
	// dropping into the embedded path.
	skillPath := findSkillMd(t, "aiwf-check")
	body, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read %s: %v", skillPath, err)
	}
	cases := []struct {
		findingCode string
	}{
		{"terminal-entity-not-archived"},
		{"archive-sweep-pending"},
	}
	for _, tc := range cases {
		t.Run(tc.findingCode, func(t *testing.T) {
			t.Parallel()
			row := findTableRowForCode(string(body), tc.findingCode)
			if row == "" {
				t.Fatalf("no table row whose first column names `%s` in %s", tc.findingCode, skillPath)
			}
			if !strings.Contains(row, "`aiwf archive --apply`") {
				t.Errorf("table row for `%s` does not name `aiwf archive --apply` in backticks (M-0085 AC-8 polish):\n  row: %s", tc.findingCode, strings.TrimSpace(row))
			}
		})
	}
}

// findSkillMd locates `internal/skills/embedded/<dir>/SKILL.md` from
// the test working directory. Walks upward until a `go.mod` is found
// and joins the canonical embedded path. Mirrors the lookup the
// AC-7 binary-integration test uses.
func findSkillMd(t *testing.T, skillDir string) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return filepath.Join(dir, "internal", "skills", "embedded", skillDir, "SKILL.md")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not find go.mod walking up from test cwd")
	return "" //coverage:ignore unreachable: t.Fatalf above terminates
}

// findTableRowForCode returns the markdown-table row whose first cell
// literally backticks the named finding code. Empty when no such row
// exists. The match is scoped to the row, not flat over the file —
// per CLAUDE.md "Substring assertions are not structural assertions",
// a single literal occurring elsewhere in the document doesn't count.
//
// Markdown table grammar: rows are lines that start with `|` (after
// leading whitespace). A row's first cell is the substring between
// the first `|` and the second `|`. We extract that, trim, and check
// for the backticked code form `\`<code>\“.
func findTableRowForCode(body, code string) string {
	want := "`" + code + "`"
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}
		// Find the second `|` to delimit the first cell.
		rest := trimmed[1:]
		end := strings.Index(rest, "|")
		if end < 0 {
			continue
		}
		firstCell := strings.TrimSpace(rest[:end])
		if firstCell == want {
			return line
		}
	}
	return ""
}
