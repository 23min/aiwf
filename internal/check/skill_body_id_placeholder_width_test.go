package check

// M-0287 AC-2: a shipped surface's placeholders teach id shape to whatever
// reads them. A narrow placeholder models a width no allocator emits, and the
// planning rituals — where the population concentrates — are exactly the
// surfaces an assistant consults before it writes anything.
//
// The canonical arm matters as much as the narrow one: a rule that fires on
// every letter-N form would leave no way to write a placeholder at all.

import (
	"fmt"
	"strings"
	"testing"
)

// The two shapes this rule rejects are distinguishable only by the defect the
// message names — they share a code, a severity, and a remediation. These
// substrings are therefore the assertion surface for classification, and a
// test that checks only "something fired" cannot tell the classes apart.
const (
	realIDDefect      = "cites real entity id"
	placeholderDefect = "non-canonical placeholder"
)

// kindPrefixes is every kind prefix the id grammar admits. Driving the table
// from this list rather than a hand-picked sample is what makes "every kind
// prefix" true as the grammar changes rather than true when it was written.
var kindPrefixes = []string{"E", "M", "G", "D", "C", "ADR"}

func TestScanSkillBodyID_PlaceholderWidth_Bare(t *testing.T) {
	t.Parallel()
	for _, k := range kindPrefixes {
		for _, width := range []string{"N", "NN", "NNN"} {
			tok := k + "-" + width
			t.Run("narrow/"+tok, func(t *testing.T) {
				t.Parallel()
				body := fmt.Sprintf("# Title\n\nAllocate the next %s id.\n", tok)
				got := ScanSkillBodyID([]byte(body), "shipped.md")
				if len(got) != 1 {
					t.Fatalf("want 1 finding for narrow placeholder %q, got %d: %+v", tok, len(got), got)
				}
				if got[0].Severity != SeverityWarning {
					t.Errorf("severity = %q, want %q (the sweep is outstanding)", got[0].Severity, SeverityWarning)
				}
				if got[0].Line != 3 {
					t.Errorf("line = %d, want 3", got[0].Line)
				}
				if !strings.Contains(got[0].Message, placeholderDefect) {
					t.Errorf("message %q does not name a placeholder defect", got[0].Message)
				}
			})
		}
		tok := k + "-NNNN"
		t.Run("canonical/"+tok, func(t *testing.T) {
			t.Parallel()
			body := fmt.Sprintf("# Title\n\nAllocate the next %s id.\n", tok)
			if got := ScanSkillBodyID([]byte(body), "shipped.md"); len(got) != 0 {
				t.Fatalf("canonical placeholder %q must be silent, got %d: %+v", tok, len(got), got)
			}
		})
	}
}

// TestScanSkillBodyID_PlaceholderWidth_Composite covers the composite form,
// whose parent segment carries the width while the AC segment is a single N at
// any width. Only the milestone kind takes a composite id.
func TestScanSkillBodyID_PlaceholderWidth_Composite(t *testing.T) {
	t.Parallel()
	cases := []struct {
		tok       string
		wantFires bool
	}{
		{"M-N/AC-N", true},
		{"M-NN/AC-N", true},
		{"M-NNN/AC-N", true},
		{"M-NNNN/AC-N", false},
	}
	for _, tc := range cases {
		t.Run(tc.tok, func(t *testing.T) {
			t.Parallel()
			body := fmt.Sprintf("# Title\n\nAddress it as %s in prose.\n", tc.tok)
			got := ScanSkillBodyID([]byte(body), "shipped.md")
			if tc.wantFires && len(got) != 1 {
				t.Fatalf("want 1 finding for %q, got %d: %+v", tc.tok, len(got), got)
			}
			if tc.wantFires && !strings.Contains(got[0].Message, placeholderDefect) {
				t.Errorf("message %q does not name a placeholder defect", got[0].Message)
			}
			if !tc.wantFires && len(got) != 0 {
				t.Fatalf("canonical composite %q must be silent, got %d: %+v", tc.tok, len(got), got)
			}
		})
	}
}

// TestScanSkillBodyID_PlaceholderWidth_InCodeConstruct pins that width is
// judged wherever the token sits. The narrow population concentrates in command
// examples, so a width rule that stopped at prose would miss most of it.
func TestScanSkillBodyID_PlaceholderWidth_InCodeConstruct(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
	}{
		{"inline code span", "# Title\n\nRun `aiwf show E-NN` to inspect.\n"},
		{"fenced block", "# Title\n\n```bash\naiwf promote M-NNN active\n```\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ScanSkillBodyID([]byte(tc.body), "shipped.md")
			if len(got) != 1 {
				t.Fatalf("want 1 finding, got %d: %+v\nbody:\n%s", len(got), got, tc.body)
			}
			if got[0].Severity != SeverityWarning {
				t.Errorf("severity = %q, want %q", got[0].Severity, SeverityWarning)
			}
			if !strings.Contains(got[0].Message, placeholderDefect) {
				t.Errorf("message %q does not name a placeholder defect", got[0].Message)
			}
		})
	}
}

// TestScanSkillBodyID_PlaceholderWidth_RealIDsUnaffected guards the boundary
// between the two shapes this rule rejects. A narrow NUMERIC id is a real id at
// a legacy width, not a placeholder — read tolerance keeps it resolving, and
// it must keep classifying as a real-id citation rather than a width defect.
func TestScanSkillBodyID_PlaceholderWidth_RealIDsUnaffected(t *testing.T) {
	t.Parallel()
	for _, tok := range []string{"E-01", "M-001", "ADR-0004"} {
		t.Run(tok, func(t *testing.T) {
			t.Parallel()
			body := fmt.Sprintf("# Title\n\nThis supersedes %s entirely.\n", tok)
			got := ScanSkillBodyID([]byte(body), "shipped.md")
			if len(got) != 1 {
				t.Fatalf("want 1 finding for real id %q, got %d: %+v", tok, len(got), got)
			}
			if !strings.Contains(got[0].Message, realIDDefect) {
				t.Errorf("message %q classifies %q as a placeholder defect; a narrow numeric id is a real id at a legacy width", got[0].Message, tok)
			}
			if strings.Contains(got[0].Message, placeholderDefect) {
				t.Errorf("message %q misclassifies real id %q as a placeholder defect", got[0].Message, tok)
			}
		})
	}
}
