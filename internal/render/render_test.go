package render

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/check"
)

func TestStatusFor(t *testing.T) {
	t.Parallel()
	if got := StatusFor(nil); got != "ok" {
		t.Errorf("StatusFor(nil) = %q, want ok", got)
	}
	if got := StatusFor([]check.Finding{{Severity: check.SeverityError}}); got != "findings" {
		t.Errorf("StatusFor(non-empty) = %q, want findings", got)
	}
}

func TestText_Empty(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := Text(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "no findings") {
		t.Errorf("output: %q", buf.String())
	}
}

func TestText_PathLineSeverityCodeMessageHint(t *testing.T) {
	t.Parallel()
	findings := []check.Finding{
		{
			Code:     "refs-resolve",
			Severity: check.SeverityError,
			Subcode:  "unresolved",
			Message:  `milestone field "parent" references unknown id "E-0099"`,
			Path:     "work/epics/E-01-foo/M-007.md",
			Line:     5,
			EntityID: "M-0007",
			Hint:     "check the spelling, or remove the reference if the target was deleted",
		},
		{
			Code:     "titles-nonempty",
			Severity: check.SeverityWarning,
			Message:  "title is empty or whitespace-only",
			Path:     "work/epics/E-01-foo/epic.md",
			Line:     3,
			Hint:     "set a non-empty `title:` in the frontmatter",
		},
	}
	var buf bytes.Buffer
	if err := Text(&buf, findings); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	for _, want := range []string{
		`work/epics/E-01-foo/M-007.md:5: error refs-resolve/unresolved: milestone field "parent" references unknown id "E-0099" — hint: check the spelling, or remove the reference if the target was deleted`,
		"work/epics/E-01-foo/epic.md:3: warning titles-nonempty: title is empty or whitespace-only — hint: set a non-empty `title:` in the frontmatter",
		"2 findings (1 errors, 1 warnings)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, got)
		}
	}
}

// TestText_PathWithoutLine: a finding with a path but no resolved line
// (e.g., a load error whose file failed to parse) should still render
// path-prefixed but without the :line suffix.
func TestText_PathWithoutLine(t *testing.T) {
	t.Parallel()
	findings := []check.Finding{{
		Code:     "load-error",
		Severity: check.SeverityError,
		Message:  "yaml: line 2: malformed",
		Path:     "work/epics/E-01-foo/epic.md",
	}}
	var buf bytes.Buffer
	if err := Text(&buf, findings); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	want := "work/epics/E-01-foo/epic.md: error load-error: yaml: line 2: malformed"
	if !strings.Contains(got, want) {
		t.Errorf("output missing %q:\n%s", want, got)
	}
}

func TestText_NoPath(t *testing.T) {
	t.Parallel()
	findings := []check.Finding{{
		Code:     "load-error",
		Severity: check.SeverityError,
		Message:  "could not list directory",
	}}
	var buf bytes.Buffer
	if err := Text(&buf, findings); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(buf.String(), "error load-error: could not list directory") {
		t.Errorf("got %q", buf.String())
	}
}

// TestText_HintOmittedWhenEmpty: a finding without a Hint produces no
// trailing "— hint: ..." suffix. The renderer is responsible for
// degrading gracefully when checks haven't been hint-annotated.
func TestText_HintOmittedWhenEmpty(t *testing.T) {
	t.Parallel()
	findings := []check.Finding{{
		Code:     "ids-unique",
		Severity: check.SeverityError,
		Message:  `id "M-0001" is also used by other.md`,
		Path:     "work/epics/dup.md",
		Line:     2,
	}}
	var buf bytes.Buffer
	if err := Text(&buf, findings); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "— hint:") {
		t.Errorf("expected no hint suffix; got:\n%s", buf.String())
	}
}

// TestTextSummary_WarningsCollapsedByCode (M-0089 AC-1):
// each warning-severity code appears once in summary form, not per
// instance. Two distinct codes with multiple instances each must
// produce two summary lines, sorted by count descending with
// alphabetic tie-break (pinned in the milestone spec's *Constraints*).
//
// Per CLAUDE.md "Substring assertions are not structural assertions"
// (AC-8 of the same milestone): the assertion parses the rendered
// output line-by-line and extracts the leading code token from each
// summary line via the documented format `<code> (warning) × N — <msg>`.
// A flat substring grep would fire even if the code drifted into the
// hint or the footer.
func TestTextSummary_WarningsCollapsedByCode(t *testing.T) {
	t.Parallel()
	// terminal-entity-not-archived × 3 (highest count), titles-nonempty × 2.
	// Constructed in the order Run() would emit (sorted alphabetically by
	// code, then path) so the "first message" rule is unambiguous.
	findings := []check.Finding{
		{Code: "terminal-entity-not-archived", Severity: check.SeverityWarning, Message: "entity ADR-0001 has terminal status \"superseded\" but file is still in the active tree", Path: "docs/adr/ADR-0001.md", Line: 4, Hint: "run `aiwf archive --dry-run`"},
		{Code: "terminal-entity-not-archived", Severity: check.SeverityWarning, Message: "entity ADR-0002 has terminal status \"superseded\" but file is still in the active tree", Path: "docs/adr/ADR-0002.md", Line: 4, Hint: "run `aiwf archive --dry-run`"},
		{Code: "terminal-entity-not-archived", Severity: check.SeverityWarning, Message: "entity G-0001 has terminal status \"addressed\" but file is still in the active tree", Path: "work/gaps/G-0001.md", Line: 4, Hint: "run `aiwf archive --dry-run`"},
		{Code: "titles-nonempty", Severity: check.SeverityWarning, Message: "title is empty or whitespace-only (epic.md)", Path: "work/epics/E-0099/epic.md", Line: 3, Hint: "set a non-empty `title:`"},
		{Code: "titles-nonempty", Severity: check.SeverityWarning, Message: "title is empty or whitespace-only (M-0099.md)", Path: "work/epics/E-0099/M-0099.md", Line: 3, Hint: "set a non-empty `title:`"},
	}
	var buf bytes.Buffer
	if err := TextSummary(&buf, findings); err != nil {
		t.Fatalf("TextSummary: %v", err)
	}

	got := buf.String()
	summaryLines := extractSummaryLines(t, got)

	// AC-1 structural: exactly two summary lines, one per code.
	if len(summaryLines) != 2 {
		t.Fatalf("want 2 summary lines, got %d:\n%s", len(summaryLines), got)
	}

	// AC-1 ordering: count desc, alphabetic tie-break.
	// terminal-entity-not-archived (3) comes before titles-nonempty (2).
	want := []summaryLine{
		{code: "terminal-entity-not-archived", severity: "warning", count: 3, sample: "entity ADR-0001 has terminal status \"superseded\" but file is still in the active tree"},
		{code: "titles-nonempty", severity: "warning", count: 2, sample: "title is empty or whitespace-only (epic.md)"},
	}
	if diff := cmp.Diff(want, summaryLines, cmp.AllowUnexported(summaryLine{})); diff != "" {
		t.Errorf("summary lines mismatch (-want +got):\n%s\nfull output:\n%s", diff, got)
	}

	// AC-1 negative: the per-leaf line must NOT appear in default mode
	// for warnings. Asserted structurally — the per-leaf rendering
	// always starts with a path prefix followed by ": warning <code>".
	for _, perLeaf := range []string{
		"docs/adr/ADR-0001.md:4: warning terminal-entity-not-archived",
		"work/gaps/G-0001.md:4: warning terminal-entity-not-archived",
		"work/epics/E-0099/epic.md:3: warning titles-nonempty",
	} {
		if strings.Contains(got, perLeaf) {
			t.Errorf("default mode leaked per-leaf warning line %q:\n%s", perLeaf, got)
		}
	}

	// Footer line still names the total count (errors + warnings).
	if !strings.Contains(got, "5 findings (0 errors, 5 warnings)") {
		t.Errorf("footer line missing or wrong; got:\n%s", got)
	}
}

// TestTextSummary_ErrorsPrintPerInstance (M-0089 AC-2):
// errors are per-instance-actionable; in default-summary mode they
// must still print one line per finding, not collapsed into a
// summary. A fixture mixing 3 error instances of one code with
// 2 warning instances of another code produces 3 error lines + 1
// warning summary line.
//
// Structural assertion (per AC-8): the per-instance lines are
// identified by leading `<path>:<line>: error <code>` shape, not by
// substring grep on the path or code alone.
func TestTextSummary_ErrorsPrintPerInstance(t *testing.T) {
	t.Parallel()
	findings := []check.Finding{
		{Code: "refs-resolve", Subcode: "unresolved", Severity: check.SeverityError, Message: `milestone field "parent" references unknown id "E-0099"`, Path: "work/epics/E-0001/M-0001.md", Line: 5, Hint: "check spelling"},
		{Code: "refs-resolve", Subcode: "unresolved", Severity: check.SeverityError, Message: `milestone field "parent" references unknown id "E-0077"`, Path: "work/epics/E-0001/M-0002.md", Line: 5, Hint: "check spelling"},
		{Code: "refs-resolve", Subcode: "unresolved", Severity: check.SeverityError, Message: `milestone field "parent" references unknown id "E-0066"`, Path: "work/epics/E-0001/M-0003.md", Line: 5, Hint: "check spelling"},
		{Code: "titles-nonempty", Severity: check.SeverityWarning, Message: "title is empty (a)", Path: "work/epics/E-0001/M-0004.md", Line: 3, Hint: "set title"},
		{Code: "titles-nonempty", Severity: check.SeverityWarning, Message: "title is empty (b)", Path: "work/epics/E-0001/M-0005.md", Line: 3, Hint: "set title"},
	}
	var buf bytes.Buffer
	if err := TextSummary(&buf, findings); err != nil {
		t.Fatalf("TextSummary: %v", err)
	}
	got := buf.String()

	// AC-2: error rows must appear per-instance. Parse them by their
	// `<path>:<line>: error <code>[/<subcode>]: <msg>` shape; a flat
	// substring grep on "refs-resolve" would fire on a summary line too.
	errorLineRE := regexp.MustCompile(`^(\S+):(\d+): error (\S+): (.+)$`)
	var errorLines []string
	for _, ln := range strings.Split(got, "\n") {
		if errorLineRE.MatchString(ln) {
			errorLines = append(errorLines, ln)
		}
	}
	if len(errorLines) != 3 {
		t.Errorf("want 3 per-instance error lines, got %d:\n%s", len(errorLines), got)
	}

	// AC-2 negative: no error-severity summary line. The summary parser
	// flags both "error" and "warning" forms; here we expect zero.
	summaries := extractSummaryLines(t, got)
	for _, s := range summaries {
		if s.severity == "error" {
			t.Errorf("default mode collapsed an error into summary form: %+v\n%s", s, got)
		}
	}

	// Sanity: the warning code still summarizes.
	if len(summaries) != 1 || summaries[0].code != "titles-nonempty" || summaries[0].count != 2 {
		t.Errorf("expected one warning summary (titles-nonempty × 2); got %+v\n%s", summaries, got)
	}

	if !strings.Contains(got, "5 findings (3 errors, 2 warnings)") {
		t.Errorf("footer missing or wrong: %s", got)
	}
}

// TestTextSummary_Empty (M-0089 AC-1 edge case):
// the empty-findings shortcut shared with Text is preserved — an
// empty input produces the same "ok — no findings" banner regardless
// of which entry point the caller chose.
func TestTextSummary_Empty(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := TextSummary(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "no findings") {
		t.Errorf("output: %q", buf.String())
	}
}

// TestTextSummary_TieBreakAlphabeticByCode (M-0089 *Constraints*):
// when two codes have equal count, the order is alphabetic by code.
// This is the deterministic tie-break that keeps golden files stable.
func TestTextSummary_TieBreakAlphabeticByCode(t *testing.T) {
	t.Parallel()
	findings := []check.Finding{
		// zeta-code appears first in input order but sorts second
		// alphabetically; alpha-code must therefore come first.
		{Code: "zeta-code", Severity: check.SeverityWarning, Message: "z one"},
		{Code: "zeta-code", Severity: check.SeverityWarning, Message: "z two"},
		{Code: "alpha-code", Severity: check.SeverityWarning, Message: "a one"},
		{Code: "alpha-code", Severity: check.SeverityWarning, Message: "a two"},
	}
	var buf bytes.Buffer
	if err := TextSummary(&buf, findings); err != nil {
		t.Fatal(err)
	}
	got := extractSummaryLines(t, buf.String())
	if len(got) != 2 {
		t.Fatalf("want 2 summary lines, got %d: %+v", len(got), got)
	}
	if got[0].code != "alpha-code" || got[1].code != "zeta-code" {
		t.Errorf("tie-break ordering wrong; want [alpha-code, zeta-code], got [%s, %s]", got[0].code, got[1].code)
	}
}

// failWriter is an io.Writer that returns an error on the Nth Write
// call (Nth starting at 1). Used by the branch-coverage tests below
// to drive the defensive error returns inside TextSummary and
// renderPerInstance.
type failWriter struct {
	failOn int // 1-based call index where Write should error
	n      int // running counter of Write calls
}

func (fw *failWriter) Write(p []byte) (int, error) {
	fw.n++
	if fw.n == fw.failOn {
		return 0, errFailWriter
	}
	return len(p), nil
}

var errFailWriter = newFailWriterError()

func newFailWriterError() error { return errFailWriterValue{} }

type errFailWriterValue struct{}

func (errFailWriterValue) Error() string { return "failWriter: simulated write failure" }

// TestTextSummary_WriteErrorBubblesUp covers the defensive error
// paths in TextSummary that fire when the io.Writer fails — once for
// the per-instance render (the errors slice) and once for the
// summary-line writes. A failing writer surfaces both branches in
// one test. Branch-coverage rule per CLAUDE.md "Test untested code
// paths before declaring code paths 'done'": every reachable
// conditional branch is exercised.
func TestTextSummary_WriteErrorBubblesUp(t *testing.T) {
	t.Parallel()
	findings := []check.Finding{
		{Code: "boom", Severity: check.SeverityError, Message: "an error", Path: "a.md", Line: 1},
		{Code: "fizz", Severity: check.SeverityWarning, Message: "a warning"},
	}

	// Failure on the first write (the per-instance error line —
	// drives the renderPerInstance return-err branch).
	if err := TextSummary(&failWriter{failOn: 1}, findings); err == nil {
		t.Error("expected error from failing writer (per-instance branch); got nil")
	}

	// Failure on the second write (the summary-line branch — the
	// first write succeeds for the error line, the second fails on
	// the warning summary).
	if err := TextSummary(&failWriter{failOn: 2}, findings); err == nil {
		t.Error("expected error from failing writer (summary-line branch); got nil")
	}

	// Failure on the third write (the footer-line branch — both
	// the error and summary writes succeed, the footer Fprintf fails).
	if err := TextSummary(&failWriter{failOn: 3}, findings); err == nil {
		t.Error("expected error from failing writer (footer branch); got nil")
	}

	// Empty-findings branch's error path.
	if err := TextSummary(&failWriter{failOn: 1}, nil); err == nil {
		t.Error("expected error from failing writer on empty findings; got nil")
	}
}

// TestText_WriteErrorBubblesUp mirrors the failure coverage for the
// verbose path (Text). Same branch shape, same defensive returns,
// but Text shares the renderPerInstance helper so the failure
// surfaces through it too.
func TestText_WriteErrorBubblesUp(t *testing.T) {
	t.Parallel()
	findings := []check.Finding{
		{Code: "boom", Severity: check.SeverityError, Message: "an error", Path: "a.md", Line: 1},
	}
	// Failure on the per-instance write.
	if err := Text(&failWriter{failOn: 1}, findings); err == nil {
		t.Error("expected error from failing writer (Text per-instance branch); got nil")
	}
	// Failure on the footer write.
	if err := Text(&failWriter{failOn: 2}, findings); err == nil {
		t.Error("expected error from failing writer (Text footer branch); got nil")
	}
	// Failure on the empty-findings banner.
	if err := Text(&failWriter{failOn: 1}, nil); err == nil {
		t.Error("expected error from failing writer (Text empty-findings branch); got nil")
	}
}

// TestRenderPerInstance_WriteErrorPerShape covers the three case-arms
// inside renderPerInstance: path+line (load-error), path-only
// (file-shape load error), and default (no path). Each arm has its
// own Fprintf with its own return-err branch; a failure on the first
// write of a single-finding slice surfaces the matching arm's branch.
func TestRenderPerInstance_WriteErrorPerShape(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		finding check.Finding
	}{
		{"path-and-line", check.Finding{Code: "c", Severity: check.SeverityError, Message: "m", Path: "p.md", Line: 1}},
		{"path-only", check.Finding{Code: "c", Severity: check.SeverityError, Message: "m", Path: "p.md"}},
		{"no-path", check.Finding{Code: "c", Severity: check.SeverityError, Message: "m"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := renderPerInstance(&failWriter{failOn: 1}, []check.Finding{tc.finding})
			if err == nil {
				t.Errorf("expected error from failing writer (%s branch); got nil", tc.name)
			}
		})
	}
}

// TestTextSummary_SampleMessageIsFirstFinding (M-0089 *Constraints*):
// the representative message in each summary line is the Message
// field of the first finding in the input slice with that code,
// verbatim. No template substitution, no truncation.
func TestTextSummary_SampleMessageIsFirstFinding(t *testing.T) {
	t.Parallel()
	findings := []check.Finding{
		{Code: "same-code", Severity: check.SeverityWarning, Message: "first instance message", Path: "a.md", Line: 1},
		{Code: "same-code", Severity: check.SeverityWarning, Message: "second instance message", Path: "b.md", Line: 1},
		{Code: "same-code", Severity: check.SeverityWarning, Message: "third instance message", Path: "c.md", Line: 1},
	}
	var buf bytes.Buffer
	if err := TextSummary(&buf, findings); err != nil {
		t.Fatal(err)
	}
	got := extractSummaryLines(t, buf.String())
	if len(got) != 1 {
		t.Fatalf("want 1 summary line, got %d", len(got))
	}
	if got[0].sample != "first instance message" {
		t.Errorf("sample message = %q, want %q (first finding wins)", got[0].sample, "first instance message")
	}
}

// summaryLine is the structured shape extracted from a single summary
// line by extractSummaryLines below. Used by the AC-1, AC-2, and AC-8
// tests to assert structural correctness rather than substring presence.
type summaryLine struct {
	code     string
	severity string
	count    int
	sample   string
}

// extractSummaryLines parses the per-code summary lines from rendered
// text output. The expected format is:
//
//	<code> (<severity>) × N — <message>
//
// Lines that don't match the format are skipped (e.g. per-instance
// error lines, the blank line, the footer). Returns the parsed slice
// in source order.
//
// AC-8 chokepoint: tests assert on the parsed structure, not on flat
// substring matches in the raw output.
func extractSummaryLines(t *testing.T, out string) []summaryLine {
	t.Helper()
	// Anchored at start-of-line so a per-instance error line cannot
	// accidentally satisfy the pattern via its embedded code token.
	// Severity is captured so the parser distinguishes the warning
	// summary form from any future error summary form.
	re := regexp.MustCompile(`^(\S+) \((warning|error)\) × (\d+) — (.+)$`)
	var lines []summaryLine
	for _, ln := range strings.Split(out, "\n") {
		m := re.FindStringSubmatch(ln)
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[3])
		if err != nil {
			t.Fatalf("parsing count from %q: %v", ln, err)
		}
		lines = append(lines, summaryLine{code: m[1], severity: m[2], count: n, sample: m[4]})
	}
	return lines
}

func TestJSON_RoundTrip(t *testing.T) {
	t.Parallel()
	env := Envelope{
		Tool:    "aiwf",
		Version: "0.1.0",
		Status:  "findings",
		Findings: []check.Finding{
			{Code: "ids-unique", Severity: check.SeverityError, Message: "dup"},
		},
		Metadata: map[string]any{"count": float64(1)},
	}
	var buf bytes.Buffer
	if err := JSON(&buf, env, false); err != nil {
		t.Fatal(err)
	}
	var got Envelope
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if diff := cmp.Diff(env, got); diff != "" {
		t.Errorf("round-trip mismatch (-want +got):\n%s", diff)
	}
}

func TestJSON_PrettyIndents(t *testing.T) {
	t.Parallel()
	var compact, pretty bytes.Buffer
	env := Envelope{Tool: "aiwf", Version: "dev", Status: "ok"}
	if err := JSON(&compact, env, false); err != nil {
		t.Fatal(err)
	}
	if err := JSON(&pretty, env, true); err != nil {
		t.Fatal(err)
	}
	if pretty.Len() <= compact.Len() {
		t.Errorf("pretty output (%d bytes) should be longer than compact (%d bytes)", pretty.Len(), compact.Len())
	}
}

func TestJSON_NilFindingsBecomesEmptyArray(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	env := Envelope{Tool: "aiwf", Version: "dev", Status: "ok", Findings: nil}
	if err := JSON(&buf, env, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"findings":[]`) {
		t.Errorf("expected findings:[] in output, got %q", buf.String())
	}
}
