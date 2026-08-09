package policies

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestPolicySeverityPassComposition_LiveTree is the standing assertion:
// this repo's own tree composes every severity pass in one seam and
// routes every check.Run call site through it.
func TestPolicySeverityPassComposition_LiveTree(t *testing.T) {
	t.Parallel()
	vs, err := PolicySeverityPassComposition(repoRoot(t))
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 0 {
		for _, v := range vs {
			t.Errorf("%s:%d %s", v.File, v.Line, v.Detail)
		}
	}
}

// severityScaffold returns a minimal tree that satisfies both anchors —
// four exported passes and a seam calling all of them — so a row can
// introduce exactly one defect and read the finding it produces.
func severityScaffold() map[string]string {
	return map[string]string{
		"internal/check/apply.go": "package check\n\n" +
			"type Finding struct{}\n\n" +
			"func ApplyTDDStrict(findings []Finding, strict bool) {}\n" +
			"func Run(t *T) []Finding { return nil }\n",
		"internal/severity/severity.go": "package severity\n\n" +
			"func Apply(findings []check.Finding) { check.ApplyTDDStrict(findings, true) }\n",
	}
}

// TestPolicySeverityPassComposition_Fires walks each way the seam can
// be defeated, on its own synthetic tree. Every row is a positive
// control for one Violation construction site.
func TestPolicySeverityPassComposition_Fires(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		mutate   func(map[string]string)
		wantPart string
	}{
		{
			name: "a call site that never applies the policy",
			mutate: func(files map[string]string) {
				files["internal/cli/probe/probe.go"] = "package probe\n\n" +
					"func Report() { _ = check.Run(nil) }\n"
			},
			wantPart: "never severity.Apply",
		},
		{
			name: "a call site handed a policy literal rather than the consumer's",
			mutate: func(files map[string]string) {
				files["internal/cli/probe/probe.go"] = "package probe\n\n" +
					"func Report(t *tree.Tree) {\n" +
					"\tfs := check.Run(t)\n\tseverity.Apply(fs, severity.Policy{}, t)\n}\n"
			},
			wantPart: "does not come from severity.Load",
		},
		{
			name: "a policy local assigned from something else",
			mutate: func(files map[string]string) {
				files["internal/cli/probe/probe.go"] = "package probe\n\n" +
					"func Report(t *tree.Tree) {\n\tp := severity.Policy{TDDStrict: true}\n" +
					"\tfs := check.Run(t)\n\tseverity.Apply(fs, p, t)\n}\n"
			},
			wantPart: "does not come from severity.Load",
		},
		{
			name: "a pass the seam does not compose",
			mutate: func(files map[string]string) {
				files["internal/check/apply.go"] += "\nfunc ApplyDocsStrict(findings []Finding, strict bool) {}\n"
			},
			wantPart: "severity.Apply never calls",
		},
		{
			name: "the seam anchor moved away",
			mutate: func(files map[string]string) {
				delete(files, "internal/severity/severity.go")
			},
			wantPart: "is not declared under internal/severity",
		},
		{
			name: "the passes moved away",
			mutate: func(files map[string]string) {
				files["internal/check/apply.go"] = "package check\n\ntype Finding struct{}\n"
			},
			wantPart: "no exported Apply* function",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			files := severityScaffold()
			tc.mutate(files)
			root := t.TempDir()
			for rel, content := range files {
				mustWrite(t, filepath.Join(root, rel), content)
			}
			vs, err := PolicySeverityPassComposition(root)
			if err != nil {
				t.Fatalf("policy returned error: %v", err)
			}
			if !hasPolicyViolation(vs, "severity-pass-composition") {
				t.Fatalf("policy did not fire; got %d violations: %+v", len(vs), vs)
			}
			if !violationDetailContains(vs, tc.wantPart) {
				t.Errorf("no violation mentioning %q; got %+v", tc.wantPart, vs)
			}
		})
	}
}

// TestPolicySeverityPassComposition_AcceptsAWiredCallSite pins the
// negative control the firing rows cannot: the scaffold plus a call site
// that does apply the policy is clean. Without it, a policy that fired
// on every tree would pass every row above.
func TestPolicySeverityPassComposition_AcceptsAWiredCallSite(t *testing.T) {
	t.Parallel()
	files := severityScaffold()
	files["internal/cli/probe/probe.go"] = "package probe\n\n" +
		"func Report(root string, t *tree.Tree) {\n" +
		"\tfs := check.Run(t)\n\tseverity.Apply(fs, severity.Load(root), t)\n}\n"
	root := t.TempDir()
	for rel, content := range files {
		mustWrite(t, filepath.Join(root, rel), content)
	}
	vs, err := PolicySeverityPassComposition(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 0 {
		t.Errorf("policy fired on a correctly-wired tree: %+v", vs)
	}
}

// TestPolicySeverityPassComposition_OnlyFindingSlicesArePasses pins the
// discrimination that decides what the seam is obliged to call. "A pass"
// is a shape, not a naming convention: an exported Apply* that takes no
// findings — or takes something other than a []Finding — mutates no
// severities, and demanding the seam call it would turn every unrelated
// exported Apply into a false violation.
func TestPolicySeverityPassComposition_OnlyFindingSlicesArePasses(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		decl string
	}{
		{"no parameters at all", "func ApplyDefaults() {}\n"},
		{"a non-slice first parameter", "func ApplyHint(f *Finding) {}\n"},
		{"a fixed-size array rather than a slice", "func ApplyPair(f [2]Finding) {}\n"},
		{"a slice of something else", "func ApplyPaths(p []string) {}\n"},
		{"unexported, whatever it takes", "func applyHints(f []Finding) {}\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			files := severityScaffold()
			files["internal/check/apply.go"] += "\n" + tc.decl
			root := t.TempDir()
			for rel, content := range files {
				mustWrite(t, filepath.Join(root, rel), content)
			}
			vs, err := PolicySeverityPassComposition(root)
			if err != nil {
				t.Fatalf("policy returned error: %v", err)
			}
			if len(vs) != 0 {
				t.Errorf("%s was counted as a severity pass the seam must call: %+v", tc.decl, vs)
			}
		})
	}
}

// violationDetailContains reports whether any violation's Detail carries
// substr.
func violationDetailContains(vs []Violation, substr string) bool {
	for _, v := range vs {
		if strings.Contains(v.Detail, substr) {
			return true
		}
	}
	return false
}
