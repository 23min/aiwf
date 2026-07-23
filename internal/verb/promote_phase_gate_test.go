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

// TestPromoteACPhase_GreenGate_DiffShape pins AC-4 of M-0276: with tdd.test_paths
// configured, `--phase green` refuses when no non-test (implementation) path is
// dirty and succeeds once one is. The AC is force-promoted to red for setup
// (force skips the gate), isolating this test to the green arm.
func TestPromoteACPhase_GreenGate_DiffShape(t *testing.T) {
	t.Parallel()
	globs := []string{"*_test.go", "**/*_test.go"}
	cases := []struct {
		name    string
		dirty   map[string]string
		wantErr bool
	}{
		{
			name:    "no non-test path dirty refuses",
			dirty:   map[string]string{"foo_test.go": "package foo\n"},
			wantErr: true,
		},
		{
			name:  "non-test path dirty succeeds",
			dirty: map[string]string{"impl.go": "package foo\n"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newPhaseGateFixture(t, globs)
			// Force to red so the green transition is FSM-legal; force skips the
			// diff-shape gate, so this test exercises only the green arm.
			r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", entity.TDDPhaseRed, testActor, "setup", true, nil))
			for name, content := range tc.dirty {
				if err := os.WriteFile(filepath.Join(r.root, name), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			_, err := verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", entity.TDDPhaseGreen, testActor, "", false, nil)
			if tc.wantErr {
				if err == nil {
					t.Fatal("want a diff-shape refusal, got nil")
				}
				if !strings.Contains(err.Error(), "green") {
					t.Errorf("refusal %q is not a green-gate message", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("want success, got refusal: %v", err)
			}
		})
	}
}

// TestPromoteACPhase_ForceBypassesDiffShapeGate pins AC-5 of M-0276: --force
// (force=true) skips the diff-shape gate entirely (it runs only under !force),
// so a promote that would otherwise be refused lands. --force's human-only
// property is enforced at the provenance-decoration layer by the existing
// coherence rule, not re-checked here.
func TestPromoteACPhase_ForceBypassesDiffShapeGate(t *testing.T) {
	t.Parallel()
	globs := []string{"*_test.go", "**/*_test.go"}

	t.Run("red that would refuse succeeds with force", func(t *testing.T) {
		t.Parallel()
		r := newPhaseGateFixture(t, globs)
		// A dirty non-test path makes an unforced --phase red refuse.
		if err := os.WriteFile(filepath.Join(r.root, "impl.go"), []byte("package foo\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", entity.TDDPhaseRed, testActor, "override", true, nil); err != nil {
			t.Fatalf("forced --phase red: want success, got refusal: %v", err)
		}
	})

	t.Run("green that would refuse succeeds with force", func(t *testing.T) {
		t.Parallel()
		r := newPhaseGateFixture(t, globs)
		// Force to red for FSM legality, then a green with only a test path
		// dirty (no implementation) would refuse unforced.
		r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", entity.TDDPhaseRed, testActor, "setup", true, nil))
		if err := os.WriteFile(filepath.Join(r.root, "foo_test.go"), []byte("package foo\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", entity.TDDPhaseGreen, testActor, "override", true, nil); err != nil {
			t.Fatalf("forced --phase green: want success, got refusal: %v", err)
		}
	})
}

// TestPromoteACPhase_RedGate_ExcludesPlanningPaths pins AC-6 of M-0276: the
// gate's path universe excludes planning/entity files (work/** and docs/**), so
// a legitimate red promote — a written test alongside a dirty planning file —
// does not self-refuse by counting the planning file as implementation.
func TestPromoteACPhase_RedGate_ExcludesPlanningPaths(t *testing.T) {
	t.Parallel()
	globs := []string{"*_test.go", "**/*_test.go"}
	cases := []struct {
		name        string
		planningRel string
	}{
		{name: "docs path excluded", planningRel: "docs/note.md"},
		{name: "work path excluded", planningRel: "work/scratch.tmp"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newPhaseGateFixture(t, globs)
			// Load the tree while it is clean, before the scratch planning file
			// exists, so tree.Load never sees a non-entity path under work/.
			tr := r.tree()
			// A legitimate red: the test is written (test path dirty) alongside a
			// dirty planning file that must not count as implementation.
			if err := os.WriteFile(filepath.Join(r.root, "foo_test.go"), []byte("package foo\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			abs := filepath.Join(r.root, filepath.FromSlash(tc.planningRel))
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(abs, []byte("scratch\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := verb.PromoteACPhase(r.ctx, tr, "M-0001/AC-1", entity.TDDPhaseRed, testActor, "", false, nil); err != nil {
				t.Fatalf("red with a dirty %s planning path + test: want success, got refusal: %v", tc.planningRel, err)
			}
		})
	}
}
