package render

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/ai-workflow-v2/internal/check"
)

func TestStatusFor(t *testing.T) {
	if got := StatusFor(nil); got != "ok" {
		t.Errorf("StatusFor(nil) = %q, want ok", got)
	}
	if got := StatusFor([]check.Finding{{Severity: check.SeverityError}}); got != "findings" {
		t.Errorf("StatusFor(non-empty) = %q, want findings", got)
	}
}

func TestText_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := Text(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "no findings") {
		t.Errorf("output: %q", buf.String())
	}
}

func TestText_PathLineSeverityCodeMessageHint(t *testing.T) {
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

func TestJSON_RoundTrip(t *testing.T) {
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
	var buf bytes.Buffer
	env := Envelope{Tool: "aiwf", Version: "dev", Status: "ok", Findings: nil}
	if err := JSON(&buf, env, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"findings":[]`) {
		t.Errorf("expected findings:[] in output, got %q", buf.String())
	}
}
