package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPolicy_M0282ADRWriteScopeDecisions is the live assertion: the
// repo's own ADR-0038 has the shape M-0282's criteria require.
func TestPolicy_M0282ADRWriteScopeDecisions(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyM0282ADRWriteScopeDecisions)
}

// completeFixtureBody is a minimal ADR body satisfying every assertion.
// Each firing case below removes or hollows exactly one piece, so a
// failure names which criterion lost its evidence. The prose is
// deliberately terse: this policy asserts placement and presence, so a
// fixture that reads like a real decision would imply a reach the policy
// does not have.
const completeFixtureBody = `## Context

Fixture.

## Decision

### Seam

Two seams, both inside the verb layer.

### Path scope

Every path the plan touches, and every path prefixed by one.

### Field scope

Whole-file at both seams.

### Verdict

Divergence refuses.

### Escape hatch

None.

## Consequences

The guard fires during the review-before-commit window by design.
`

// writeFixtureADR materializes a temp root carrying one ADR entity with
// the given body and status. The filename is built by concatenation
// rather than as a single literal so it does not trip
// PolicyNoHardcodedEntityPaths, which flags entity-slug string literals
// passed to filepath.Join.
func writeFixtureADR(t *testing.T, body, status string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "docs", "adr")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "---\nid: " + adrWriteScopeID + "\ntitle: Fixture\nstatus: " + status + "\n---\n" + body
	if err := os.WriteFile(filepath.Join(dir, adrWriteScopeID+"-fixture.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return root
}

func TestPolicyM0282_FiringFixtures(t *testing.T) {
	t.Parallel()

	// hollow replaces a subsection's prose with blank space, exercising
	// the present-but-empty arm distinctly from the absent-heading arm.
	hollow := func(prose string) string {
		return strings.Replace(completeFixtureBody, prose, "", 1)
	}

	cases := []struct {
		name       string
		body       string
		status     string
		wantDetail string
	}{
		{
			name:       "AC-1 seam subsection absent",
			body:       strings.Replace(completeFixtureBody, "### Seam", "### Something else", 1),
			wantDetail: "`### Seam` subsection",
		},
		{
			name:       "AC-1 seam subsection present but empty",
			body:       hollow("Two seams, both inside the verb layer."),
			wantDetail: "`### Seam` subsection",
		},
		{
			name:       "AC-2 path scope subsection absent",
			body:       strings.Replace(completeFixtureBody, "### Path scope", "### Musings", 1),
			wantDetail: "`### Path scope` subsection",
		},
		{
			name:       "AC-5 field scope subsection absent",
			body:       strings.Replace(completeFixtureBody, "### Field scope", "### Musings", 1),
			wantDetail: "`### Field scope` subsection",
		},
		{
			name:       "AC-3 verdict subsection absent",
			body:       strings.Replace(completeFixtureBody, "### Verdict", "### Musings", 1),
			wantDetail: "`### Verdict` subsection",
		},
		{
			name:       "AC-4 escape hatch subsection absent",
			body:       strings.Replace(completeFixtureBody, "### Escape hatch", "### Trapdoor", 1),
			wantDetail: "`### Escape hatch` subsection",
		},
		{
			name:       "a decision heading outside the Decision section does not count",
			body:       strings.Replace(completeFixtureBody, "## Decision", "## Prologue", 1),
			wantDetail: "`### Seam` subsection",
		},
		{
			name:       "consequences absent",
			body:       strings.Split(completeFixtureBody, "## Consequences")[0],
			wantDetail: "`## Consequences` is missing or empty",
		},
		{
			name:       "consequences present but empty",
			body:       hollow("The guard fires during the review-before-commit window by design."),
			wantDetail: "`## Consequences` is missing or empty",
		},
		{
			name:       "a rejected ADR does not satisfy the criteria",
			body:       completeFixtureBody,
			status:     "rejected",
			wantDetail: `status is "rejected"`,
		},
		{
			name:       "a superseded ADR does not satisfy the criteria",
			body:       completeFixtureBody,
			status:     "superseded",
			wantDetail: `status is "superseded"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status := tc.status
			if status == "" {
				status = "accepted"
			}
			vs, err := PolicyM0282ADRWriteScopeDecisions(writeFixtureADR(t, tc.body, status))
			if err != nil {
				t.Fatalf("policy returned error: %v", err)
			}
			var matched bool
			for _, v := range vs {
				if strings.Contains(v.Detail, tc.wantDetail) {
					matched = true
				}
			}
			if !matched {
				t.Errorf("no violation contains %q; got:\n%s", tc.wantDetail, joinDetails(vs))
			}
		})
	}
}

// TestPolicyM0282_CompleteFixturePasses pins the negative control: the
// fixture every firing case mutates is itself clean, so each case above
// fails for the reason it names rather than for a defect baked into the
// fixture.
func TestPolicyM0282_CompleteFixturePasses(t *testing.T) {
	t.Parallel()
	vs, err := PolicyM0282ADRWriteScopeDecisions(writeFixtureADR(t, completeFixtureBody, "accepted"))
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 0 {
		t.Errorf("complete fixture should produce no violations, got:\n%s", joinDetails(vs))
	}
}

// TestPolicyM0282_MissingADRFires covers the loader-miss arm: no
// ADR-0038 in the tree at all.
func TestPolicyM0282_MissingADRFires(t *testing.T) {
	t.Parallel()
	vs, err := PolicyM0282ADRWriteScopeDecisions(t.TempDir())
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 1 || !strings.Contains(vs[0].Detail, "not found") {
		t.Errorf("expected a not-found violation, got:\n%s", joinDetails(vs))
	}
}

// TestPolicyM0282_DoesNotClaimToDetectADecision records the policy's
// deliberate limit as a test rather than only as a comment: a document
// that defers every question has the required shape and passes. The
// judgement that a real decision was made is a review responsibility,
// and a future tightening that tries to mechanise it should have to
// change this test on purpose.
func TestPolicyM0282_DoesNotClaimToDetectADecision(t *testing.T) {
	t.Parallel()
	deferred := `## Context

Fixture.

## Decision

### Seam

Which seam we adopt is deferred.

### Path scope

Which paths are compared is deferred.

### Field scope

Which parts of each file are compared is deferred.

### Verdict

Whether divergence refuses or warns is deferred.

### Escape hatch

Whether any override exists is deferred.

## Consequences

Deferred with the decisions above.
`
	vs, err := PolicyM0282ADRWriteScopeDecisions(writeFixtureADR(t, deferred, "accepted"))
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 0 {
		t.Errorf("this policy asserts shape only; a deferring document is expected to pass, got:\n%s", joinDetails(vs))
	}
}

func joinDetails(vs []Violation) string {
	var b strings.Builder
	for _, v := range vs {
		b.WriteString("  - ")
		b.WriteString(v.Detail)
		b.WriteString("\n")
	}
	return b.String()
}
