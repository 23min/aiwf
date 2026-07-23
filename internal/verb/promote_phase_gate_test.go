package verb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/verb"
)

// newPhaseGateFixture builds a git-backed repo carrying a milestone M-0001 with
// one AC at the pre-cycle empty phase, ready for a `--phase red` promote. When
// testPaths is non-empty it writes and commits an aiwf.yaml declaring those
// tdd.test_paths globs (activating the red/green diff-shape gate) so the tree
// stays clean and a test controls exactly which paths are dirty; when empty it
// writes no aiwf.yaml, leaving the gate inactive (opt-in).
func newPhaseGateFixture(t *testing.T, testPaths []string) *runner {
	t.Helper()
	r := newRunner(t)
	if len(testPaths) > 0 {
		var b strings.Builder
		b.WriteString("tdd:\n  test_paths:\n")
		for _, p := range testPaths {
			b.WriteString("    - \"" + p + "\"\n")
		}
		if err := os.WriteFile(filepath.Join(r.root, "aiwf.yaml"), []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := gitops.Add(r.ctx, r.root, "aiwf.yaml"); err != nil {
			t.Fatalf("add aiwf.yaml: %v", err)
		}
		if err := gitops.Commit(r.ctx, r.root, "configure test_paths", "", nil); err != nil {
			t.Fatalf("commit aiwf.yaml: %v", err)
		}
	}
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "required"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "First criterion", testActor))
	return r
}

// TestPromoteACPhase_RedGate_DiffShape pins AC-3 of M-0276: with tdd.test_paths
// configured, `--phase red` refuses when any non-test path is dirty (naming it),
// succeeds when only test paths are dirty, and refuses when nothing is dirty;
// with no test_paths configured the gate is inactive and red proceeds.
func TestPromoteACPhase_RedGate_DiffShape(t *testing.T) {
	t.Parallel()
	globs := []string{"*_test.go", "**/*_test.go"}
	cases := []struct {
		name      string
		testPaths []string
		dirty     map[string]string
		wantErr   bool
		wantInErr string
	}{
		{
			name:      "only test paths dirty succeeds",
			testPaths: globs,
			dirty:     map[string]string{"foo_test.go": "package foo\n"},
		},
		{
			name:      "non-test path dirty refuses and names it",
			testPaths: globs,
			dirty:     map[string]string{"foo_test.go": "package foo\n", "impl.go": "package foo\n"},
			wantErr:   true,
			wantInErr: "impl.go",
		},
		{
			name:      "nothing dirty refuses",
			testPaths: globs,
			wantErr:   true,
			wantInErr: "no test",
		},
		{
			name:      "unconfigured test_paths leaves the gate inactive",
			testPaths: nil,
			dirty:     map[string]string{"impl.go": "package foo\n"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newPhaseGateFixture(t, tc.testPaths)
			for name, content := range tc.dirty {
				if err := os.WriteFile(filepath.Join(r.root, name), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			_, err := verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", entity.TDDPhaseRed, testActor, "", false, nil)
			if tc.wantErr {
				if err == nil {
					t.Fatal("want a diff-shape refusal, got nil")
				}
				if tc.wantInErr != "" && !strings.Contains(err.Error(), tc.wantInErr) {
					t.Errorf("refusal %q does not mention %q", err, tc.wantInErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("want success, got refusal: %v", err)
			}
		})
	}
}
