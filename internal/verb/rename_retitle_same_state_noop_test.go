package verb_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestRename_SameSlug_ReturnsNoOp pins the rename half of M-0281/AC-5: renaming
// an entity to the slug it already carries converges to a NoOp instead of the
// "matches the current slug" error.
func TestRename_SameSlug_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))

	res, err := verb.Rename(r.ctx, r.tree(), "E-0001", "foundations", testActor, 0)
	if err != nil {
		t.Fatalf("rename to the current slug returned a Go error, want a NoOp: %v", err)
	}
	if !res.NoOp {
		t.Errorf("res.NoOp = false, want true (renaming to the current slug is a no-op)")
	}
	if res.Plan != nil {
		t.Errorf("res.Plan = %+v, want nil (a NoOp produces no commit)", res.Plan)
	}
}

// TestRetitle_SameTitle_ReturnsNoOp pins the retitle half of M-0281/AC-5:
// retitling an entity to the title it already carries converges to a NoOp
// instead of the "title already" error.
func TestRetitle_SameTitle_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))

	res, err := verb.Retitle(r.ctx, r.tree(), "E-0001", "Foundations", testActor, "", 0)
	if err != nil {
		t.Fatalf("retitle to the current title returned a Go error, want a NoOp: %v", err)
	}
	if !res.NoOp {
		t.Errorf("res.NoOp = false, want true (retitling to the current title is a no-op)")
	}
	if res.Plan != nil {
		t.Errorf("res.Plan = %+v, want nil (a NoOp produces no commit)", res.Plan)
	}
}

// TestRenameRetitleAC_SameTitle_ReturnsNoOp extends AC-5's convergence to the
// composite-id (acceptance-criterion) variants of both verbs: an AC carries a
// title but no slug, so `rename M-NNNN/AC-N` and `retitle M-NNNN/AC-N` both
// operate on that title, and both must converge on a same-title input rather
// than keeping the "title already" error the entity-level paths just shed.
func TestRenameRetitleAC_SameTitle_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	const acTitle = "cache warms on boot"
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", acTitle, testActor))

	t.Run("rename", func(t *testing.T) {
		t.Parallel()
		res, err := verb.Rename(r.ctx, r.tree(), "M-0001/AC-1", acTitle, testActor, 0)
		if err != nil {
			t.Fatalf("rename AC to its current title returned a Go error, want a NoOp: %v", err)
		}
		if !res.NoOp {
			t.Errorf("res.NoOp = false, want true")
		}
		if res.Plan != nil {
			t.Errorf("res.Plan = %+v, want nil", res.Plan)
		}
	})

	t.Run("retitle", func(t *testing.T) {
		t.Parallel()
		res, err := verb.Retitle(r.ctx, r.tree(), "M-0001/AC-1", acTitle, testActor, "", 0)
		if err != nil {
			t.Fatalf("retitle AC to its current title returned a Go error, want a NoOp: %v", err)
		}
		if !res.NoOp {
			t.Errorf("res.NoOp = false, want true")
		}
		if res.Plan != nil {
			t.Errorf("res.Plan = %+v, want nil", res.Plan)
		}
	})
}

// TestRetitle_TitleMatchesButOtherSurfacesDrifted_StillRewrites pins the two
// conjuncts beyond the title. Retitle writes three surfaces — the frontmatter
// title, the slug-derived filename, and the canonical `# <id> — <title>` body
// H1 — so drift in either of the other two leaves work to do, and comparing
// the title alone claimed success over exactly that state.
func TestRetitle_TitleMatchesButOtherSurfacesDrifted_StillRewrites(t *testing.T) {
	t.Parallel()
	const title = "Foundations"
	cases := []struct {
		name  string
		drift func(r *runner)
	}{
		{
			name: "the filename drifted away from the slug this title derives",
			drift: func(r *runner) {
				r.must(verb.Rename(r.ctx, r.tree(), "E-0001", "totally-different-slug", testActor, 0))
			},
		},
		{
			name: "the body H1 drifted away from the title",
			drift: func(r *runner) {
				writeEntityH1(r.t, r, "E-0001", "# E-0001 — Stale heading")
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newRunner(t)
			r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, title, testActor, verb.AddOptions{}))
			tc.drift(r)

			res, err := verb.Retitle(r.ctx, r.tree(), "E-0001", title, testActor, "", 0)
			if err != nil {
				t.Fatalf("retitle over drifted state: %v", err)
			}
			if res.NoOp {
				t.Errorf("res.NoOp = true, want false — a surface retitle writes is out of sync")
			}
			if res.Plan == nil {
				t.Fatal("res.Plan = nil, want a plan resolving the drift")
			}
		})
	}
}

// writeEntityH1 replaces an entity body's canonical H1 with an arbitrary line.
func writeEntityH1(t *testing.T, r *runner, id, heading string) {
	t.Helper()
	e := r.tree().ByID(id)
	if e == nil {
		t.Fatalf("%s missing from the fixture tree", id)
	}
	path := filepath.Join(r.root, e.Path)
	raw, err := os.ReadFile(path) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("reading %s: %v", id, err)
	}
	patched := regexp.MustCompile(`(?m)^# `+regexp.QuoteMeta(id)+` — .*$`).ReplaceAllString(string(raw), heading)
	if patched == string(raw) {
		// The template may ship no canonical H1; add one so the drift is real.
		patched = string(raw) + "\n" + heading + "\n"
	}
	if writeErr := os.WriteFile(path, []byte(patched), 0o600); writeErr != nil {
		t.Fatalf("writing %s: %v", id, writeErr)
	}
}
