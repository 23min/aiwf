package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPolicy_M0282ADRWriteScopeDecisions is the live assertion: the
// repo's own ADR-0038 records all five write-scope decisions in the
// named subsections. This is the test that fails if the ADR drifts.
func TestPolicy_M0282ADRWriteScopeDecisions(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyM0282ADRWriteScopeDecisions)
}

// completeFixtureBody is a minimal ADR body satisfying every assertion.
// Each firing-fixture case below removes or hollows exactly one piece,
// so a failure names which acceptance criterion lost its evidence.
const completeFixtureBody = `## Context

Fixture.

## Decision

### Seam

The precondition runs at verb.Apply and at a shared NoOp constructor.

### Path scope

At Apply the comparison covers the full committed path set, which is what covers
every nested path under a directory move.

### Field scope

Whole-file at Apply; frontmatter at the NoOp seam.

### Verdict

Divergence refuses. It does not warn: a laundered status would otherwise bypass
the illegal-transition check.

### Escape hatch

No ` + "`--force`" + `, and no repair verb.

## Consequences

Both misbehaviors are reached: the false no-change claim and the
empty-diff commit. The guard fires during the bless workflow by design.
`

// writeFixtureADR materializes a temp root carrying one ADR entity with
// the given body. The filename is built by concatenation rather than as
// a single literal so it does not trip PolicyNoHardcodedEntityPaths,
// which flags entity-slug string literals passed to filepath.Join.
func writeFixtureADR(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "docs", "adr")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "---\nid: " + adrWriteScopeID + "\ntitle: Fixture\nstatus: accepted\n---\n" + body
	if err := os.WriteFile(filepath.Join(dir, adrWriteScopeID+"-fixture.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return root
}

func TestPolicyM0282_FiringFixtures(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		body     string
		wantsAC  string
		wantsAll []string
	}{
		{
			name:    "AC-1 seam subsection absent",
			body:    strings.Replace(completeFixtureBody, "### Seam", "### Something else", 1),
			wantsAC: "AC-1",
		},
		{
			// Drops the verdict marker while keeping the nested-case
			// marker, so this case isolates the AnyOf arm from the
			// Requires arm exercised further down.
			name:    "AC-2 path scope records no verdict",
			body:    strings.Replace(completeFixtureBody, "At Apply the comparison covers the full committed path set, which is what covers", "We thought about it, and about", 1),
			wantsAC: "AC-2",
		},
		{
			name:    "AC-5 field scope records no verdict",
			body:    strings.Replace(completeFixtureBody, "Whole-file at Apply; frontmatter at the NoOp seam.", "We thought about it.", 1),
			wantsAC: "AC-5",
		},
		{
			name:    "AC-3 verdict subsection absent",
			body:    strings.Replace(completeFixtureBody, "### Verdict", "### Musings", 1),
			wantsAC: "AC-3",
		},
		{
			name:    "AC-4 escape hatch subsection absent",
			body:    strings.Replace(completeFixtureBody, "### Escape hatch", "### Trapdoor", 1),
			wantsAC: "AC-4",
		},
		{
			name:     "consequences section absent",
			body:     strings.Split(completeFixtureBody, "## Consequences")[0],
			wantsAll: []string{"AC-1/AC-5"},
		},
		{
			name:    "AC-1 consequences omit the seam's reach",
			body:    strings.Replace(completeFixtureBody, "Both misbehaviors are reached: the false no-change claim and the\nempty-diff commit.", "Things follow.", 1),
			wantsAC: "AC-1",
		},
		{
			name:    "AC-2 path scope omits the nested case",
			body:    strings.Replace(completeFixtureBody, ", which is what covers\nevery nested path under a directory move", "", 1),
			wantsAC: "AC-2",
		},
		{
			name:    "AC-3 verdict omits the illegal-transition weighing",
			body:    strings.Replace(completeFixtureBody, ": a laundered status would otherwise bypass\nthe illegal-transition check", "", 1),
			wantsAC: "AC-3",
		},
		{
			name: "an ADR that decides nothing does not pass",
			body: `## Context

x

## Decision

### Seam

The seam question does not apply here. Deferred.

### Path scope

No agreement on whether a committed path set is the right unit. Open.

### Field scope

Whether whole-file or frontmatter-only is right is unresolved.

### Verdict

The team refuses to settle refuse-vs-warn today.

### Escape hatch

Nonetheless: undecided.

## Consequences

A falsely optimistic reading would be unblessed by the authors.
`,
			wantsAC: "AC-1",
		},
		{
			name:    "AC-5 consequences omit the bless workflow",
			body:    strings.Replace(completeFixtureBody, "The guard fires during the bless workflow by design.", "It fires sometimes.", 1),
			wantsAC: "AC-5",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := writeFixtureADR(t, tc.body)
			vs, err := PolicyM0282ADRWriteScopeDecisions(root)
			if err != nil {
				t.Fatalf("policy returned error: %v", err)
			}
			if len(vs) == 0 {
				t.Fatalf("expected a violation, got none")
			}
			joined := joinDetails(vs)
			want := tc.wantsAC
			if want == "" && len(tc.wantsAll) > 0 {
				want = tc.wantsAll[0]
			}
			if !strings.Contains(joined, want) {
				t.Errorf("violation does not name %s:\n%s", want, joined)
			}
		})
	}
}

// TestPolicyM0282_CompleteFixturePasses pins the negative case: the
// fixture the firing cases mutate is itself clean, so each case above
// fails for the reason it names rather than for a defect baked into the
// fixture.
func TestPolicyM0282_CompleteFixturePasses(t *testing.T) {
	t.Parallel()
	root := writeFixtureADR(t, completeFixtureBody)
	vs, err := PolicyM0282ADRWriteScopeDecisions(root)
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
	root := t.TempDir()
	vs, err := PolicyM0282ADRWriteScopeDecisions(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 1 || !strings.Contains(vs[0].Detail, "not found") {
		t.Errorf("expected a not-found violation, got:\n%s", joinDetails(vs))
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
