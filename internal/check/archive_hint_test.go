package check

// archive_hint_test.go — the two archive findings name their remedy as
// a runnable command.
//
// `terminal-entity-not-archived` and `archive-sweep-pending` are both
// cleared by the same two-step sweep, and the hint is the surface that
// says so: it is what `aiwf check` prints beside the finding and what
// JSON consumers read. So both invocation forms appear backticked
// rather than named in prose, pinned at each surface the value passes
// through — HintFor, and the Hint field applyHints fills.

import (
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
	hint := HintFor(CodeTerminalEntityNotArchived, "")
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
	hint := HintFor(CodeArchiveSweepPending, "")
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
		{CodeTerminalEntityNotArchived},
		{CodeArchiveSweepPending},
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
