package severity

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// escalatableCodes is every finding code some aiwf.yaml knob raises to
// error severity, grouped by the pass that raises it. It is a
// hand-maintained per-code regression net, not a mechanical mirror of
// the passes: a pass the seam forgets to call is caught by
// PolicySeverityPassComposition, which derives the set from the
// package. What this list buys is the narrower claim that each code the
// passes cover is still covered — asserting over all of them rather
// than a sample, so a pass that silently drops one code shows up.
var escalatableCodes = []string{
	// tdd.strict
	check.CodeEntityBodyEmpty,
	check.CodeMilestoneTDDUndeclared,
	// areas.required
	check.CodeAreaUnknown,
	check.CodeAreaDeadGlob,
	check.CodeAreaOverlap,
	check.CodeAreaUnslotted,
	check.CodeAreaCoverageRootMissing,
	check.CodeAreaCoverageNoPaths,
	// docs.strict
	check.CodeDocIDWidth,
	check.CodeDocIDSlug,
	// archive.sweep_threshold
	check.CodeArchiveSweepPending,
}

// warningsFor returns one warning finding per code in codes.
func warningsFor(codes []string) []check.Finding {
	out := make([]check.Finding, 0, len(codes))
	for _, code := range codes {
		out = append(out, check.Finding{
			Code:     code,
			Severity: check.SeverityWarning,
			Message:  code + " fixture message",
		})
	}
	return out
}

// severityByCode indexes findings for per-code assertions.
func severityByCode(fs []check.Finding) map[string]check.Severity {
	out := make(map[string]check.Severity, len(fs))
	for i := range fs {
		out[fs[i].Code] = fs[i].Severity
	}
	return out
}

// sweptTree returns a tree carrying n terminal, non-archived gaps —
// the shape CountPendingSweep counts, so ApplyArchiveSweepThreshold has
// a real count to compare against its ceiling.
func sweptTree(n int) *tree.Tree {
	t := &tree.Tree{}
	for i := 0; i < n; i++ {
		t.Entities = append(t.Entities, &entity.Entity{
			Kind:   entity.KindGap,
			ID:     "G-0001",
			Status: "wontfix",
			Path:   "work/gaps/G-0001-fixture.md",
		})
	}
	return t
}

// TestApply_EscalatesEveryConfiguredPass is the seam's central claim:
// one call applies every aiwf.yaml severity pass, so a call site that
// routes through it cannot carry a subset.
func TestApply_EscalatesEveryConfiguredPass(t *testing.T) {
	t.Parallel()
	findings := warningsFor(escalatableCodes)
	Apply(findings, Policy{
		TDDStrict:           true,
		AreasRequired:       true,
		DocsStrict:          true,
		ArchiveThreshold:    1,
		ArchiveThresholdSet: true,
	}, sweptTree(2))

	got := severityByCode(findings)
	for _, code := range escalatableCodes {
		if got[code] != check.SeverityError {
			t.Errorf("%s: severity = %q, want %q — the pass covering this code did not reach the seam",
				code, got[code], check.SeverityError)
		}
	}
}

// TestApply_ZeroPolicyEscalatesNothing pins the other direction: with
// every knob at its default, the seam is inert. A seam that escalated
// unconditionally would pass the test above while making every
// consumer repo stricter than it asked to be.
func TestApply_ZeroPolicyEscalatesNothing(t *testing.T) {
	t.Parallel()
	findings := warningsFor(escalatableCodes)
	Apply(findings, Policy{}, sweptTree(2))

	for _, f := range findings {
		if f.Severity != check.SeverityWarning {
			t.Errorf("%s: severity = %q under a zero Policy, want %q",
				f.Code, f.Severity, check.SeverityWarning)
		}
	}
}

// TestApply_ArchiveThresholdReadsTheTreesPendingCount pins the one
// argument the seam derives rather than carries: the sweep count comes
// from the tree it is handed, so a caller cannot pass a count that
// disagrees with the findings it is escalating. Below the ceiling the
// aggregate stays advisory.
func TestApply_ArchiveThresholdReadsTheTreesPendingCount(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		pending int
		want    check.Severity
	}{
		{"at the ceiling stays advisory", 2, check.SeverityWarning},
		{"past the ceiling blocks", 3, check.SeverityError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			findings := warningsFor([]string{check.CodeArchiveSweepPending})
			Apply(findings, Policy{ArchiveThreshold: 2, ArchiveThresholdSet: true}, sweptTree(tc.pending))
			if findings[0].Severity != tc.want {
				t.Errorf("%d pending against a ceiling of 2: severity = %q, want %q",
					tc.pending, findings[0].Severity, tc.want)
			}
		})
	}
}

// TestLoad_ReadsEachKnobIntoItsOwnField pins the config→Policy mapping
// at the seam's other end. The mapping is the half a shared Apply
// cannot protect on its own: a knob read into the wrong field, or not
// read at all, reaches every call site equally wrong.
//
// One knob per case, each asserting the whole Policy. A fixture that
// turns everything on cannot see a misroute — every permutation of the
// three booleans produces the same all-true Policy — so the isolation
// is the assertion, not a stylistic choice. Each case also pins that
// the knobs it leaves unset stay at their zero value, which is what
// catches `archive.sweep_threshold` reading as set on a config that
// never declares an archive block.
func TestLoad_ReadsEachKnobIntoItsOwnField(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		yaml string
		want Policy
	}{
		{
			name: "tdd.strict alone",
			yaml: "tdd:\n  strict: true\n",
			want: Policy{TDDStrict: true},
		},
		{
			name: "areas.required alone",
			yaml: "areas:\n  members:\n    - name: kernel\n  required: true\n",
			want: Policy{AreasRequired: true},
		},
		{
			name: "docs.strict alone",
			yaml: "docs:\n  strict: true\n",
			want: Policy{DocsStrict: true},
		},
		{
			name: "archive.sweep_threshold alone",
			yaml: "archive:\n  sweep_threshold: 7\n",
			want: Policy{ArchiveThreshold: 7, ArchiveThresholdSet: true},
		},
		{
			// A zero ceiling is a real declared ceiling — every pending
			// sweep exceeds it — and is the value an unset knob would be
			// confused with if the bool were dropped.
			name: "archive.sweep_threshold: 0 is declared, not absent",
			yaml: "archive:\n  sweep_threshold: 0\n",
			want: Policy{ArchiveThreshold: 0, ArchiveThresholdSet: true},
		},
		{
			name: "a config declaring no archive block leaves the ceiling unset",
			yaml: "tdd:\n  strict: true\ndocs:\n  strict: true\n",
			want: Policy{TDDStrict: true, DocsStrict: true},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeConfig(t, root, tc.yaml)
			if got := Load(root); got != tc.want {
				t.Errorf("Load = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestLoad_AbsentOrUnreadableConfigIsTheZeroPolicy pins the tolerance
// the read surfaces depend on: a repo with no aiwf.yaml, and one whose
// aiwf.yaml does not parse, both escalate nothing rather than failing
// the verb or the render that called the seam.
func TestLoad_AbsentOrUnreadableConfigIsTheZeroPolicy(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		yaml string // "" writes no file at all
	}{
		{"no aiwf.yaml at all", ""},
		{"malformed aiwf.yaml", "tdd: [this is not a mapping\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if tc.yaml != "" {
				writeConfig(t, root, tc.yaml)
			}
			if got := Load(root); got != (Policy{}) {
				t.Errorf("Load = %+v, want the zero Policy", got)
			}
		})
	}
}

// TestLoad_EmptyRootDoesNotReadTheProcessWorkingDirectory pins the
// guard that keeps an unrooted tree from escalating against a stranger's
// configuration. config.Load resolves a relative "aiwf.yaml" against
// the process working directory, so without the guard a caller holding
// a tree whose Root was never set would read whichever repo the
// operator happens to be standing in — and silently, since a policy is
// indistinguishable from the one the tree deserved.
func TestLoad_EmptyRootDoesNotReadTheProcessWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "tdd:\n  strict: true\n")
	t.Chdir(root)

	if got := Load(""); got != (Policy{}) {
		t.Errorf("Load(\"\") = %+v, want the zero Policy — it read the process working directory's aiwf.yaml", got)
	}
	// Control: the same config through a real root does escalate, so the
	// assertion above is about the empty root and not a broken fixture.
	if got := Load(root); !got.TDDStrict {
		t.Errorf("Load(%q) = %+v, want TDDStrict — the fixture config is not being read at all", root, got)
	}
}

// TestFrom_SetsEveryPolicyField closes the half neither Apply nor
// PolicySeverityPassComposition can see. Apply calling
// check.ApplyXStrict(findings, p.XStrict) forces a new field to exist,
// and the policy forces the pass to be composed — but a field declared
// and never assigned in From stays zero, escalates nothing on every
// surface at once, and satisfies both. A hand-written `want` literal
// would be updated in the same edit that forgot the assignment, so the
// assertion is reflective: against an aiwf.yaml that turns every knob
// on, no field of Policy may remain at its zero value.
func TestFrom_SetsEveryPolicyField(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeConfig(t, root, `tdd:
  strict: true
areas:
  members:
    - name: kernel
  required: true
docs:
  strict: true
archive:
  sweep_threshold: 7
`)
	got := reflect.ValueOf(Load(root))
	for i := range got.NumField() {
		if got.Field(i).IsZero() {
			t.Errorf("Policy.%s is still its zero value under an aiwf.yaml that sets every knob — From does not read it, so the knob escalates nothing on every surface at once",
				got.Type().Field(i).Name)
		}
	}
}

func writeConfig(t *testing.T, root, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(body), 0o644); err != nil {
		t.Fatalf("writing aiwf.yaml: %v", err)
	}
}
