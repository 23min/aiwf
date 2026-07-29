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

// TestRetitle_H1DriftedFromTitle_StillRewrites pins the H1 conjunct. Retitle
// owns the canonical `# <id> — <title>` body H1 outright — no verb lets an
// operator set it independently of the title — so an H1 out of sync with the
// title is work left to do; comparing the title alone would claim success over
// that state.
func TestRetitle_H1DriftedFromTitle_StillRewrites(t *testing.T) {
	t.Parallel()
	const title = "Foundations"
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, title, testActor, verb.AddOptions{}))
	writeEntityH1(r.t, r, "E-0001", "# E-0001 — Stale heading")

	res, err := verb.Retitle(r.ctx, r.tree(), "E-0001", title, testActor, "", 0)
	if err != nil {
		t.Fatalf("retitle over a drifted H1: %v", err)
	}
	if res.NoOp {
		t.Errorf("res.NoOp = true, want false — the H1 retitle writes is out of sync")
	}
	if res.Plan == nil {
		t.Fatal("res.Plan = nil, want a plan resolving the drift")
	}
}

// TestRetitle_OperatorSetSlug_IsPreserved pins the boundary of what retitle
// owns. The slug is retitle's to re-derive only while it still tracks the
// title; `aiwf rename` exists to choose one independently, so re-deriving over
// that choice would leave rename's effect lasting only until the next retitle.
//
// Both arms matter. Under an unchanged title nothing retitle owns is moving,
// so the call converges. Under a changed title the frontmatter is rewritten
// but the operator's slug still stands, which is what makes rename durable
// rather than merely most-recent.
func TestRetitle_OperatorSetSlug_IsPreserved(t *testing.T) {
	t.Parallel()
	const customSlug = "short-path"
	cases := []struct {
		name      string
		newTitle  string
		wantNoOp  bool
		wantTitle string
	}{
		{
			name:      "an unchanged title moves nothing retitle owns",
			newTitle:  "Foundations",
			wantNoOp:  true,
			wantTitle: "Foundations",
		},
		{
			name:      "a changed title rewrites the title and keeps the slug",
			newTitle:  "Foundations And Then Some",
			wantNoOp:  false,
			wantTitle: "Foundations And Then Some",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newRunner(t)
			r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
			r.must(verb.Rename(r.ctx, r.tree(), "E-0001", customSlug, testActor, 0))

			res, err := verb.Retitle(r.ctx, r.tree(), "E-0001", tc.newTitle, testActor, "", 0)
			if err != nil {
				t.Fatalf("retitle over an operator-set slug: %v", err)
			}
			if res.NoOp != tc.wantNoOp {
				t.Errorf("res.NoOp = %v, want %v", res.NoOp, tc.wantNoOp)
			}
			if !tc.wantNoOp {
				if res.Plan == nil {
					t.Fatal("res.Plan = nil, want a plan rewriting the title")
				}
				for _, op := range res.Plan.Ops {
					if op.Type == verb.OpMove {
						t.Errorf("plan moves %s -> %s, want the operator's slug untouched", op.Path, op.NewPath)
					}
				}
				if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
					t.Fatalf("apply: %v", applyErr)
				}
			}

			e := r.tree().ByID("E-0001")
			if e == nil {
				t.Fatal("E-0001 vanished")
			}
			if got := filepath.Base(filepath.Dir(e.Path)); got != "E-0001-"+customSlug {
				t.Errorf("on-disk dir = %q, want %q — retitle re-derived over an operator-set slug",
					got, "E-0001-"+customSlug)
			}
			if e.Title != tc.wantTitle {
				t.Errorf("title = %q, want %q", e.Title, tc.wantTitle)
			}
		})
	}
}

// TestRetitle_NonASCIITitle_SurfacesSlugWarning completes the slug-dropped
// warning across the three verbs that derive a slug. `add` and `rename` each
// surface what normalization discarded; retitle derives a slug too, so an
// operator who retitles to a title carrying non-ASCII characters must see the
// same warning rather than silently getting a lossy path.
func TestRetitle_NonASCIITitle_SurfacesSlugWarning(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))

	res, err := verb.Retitle(r.ctx, r.tree(), "E-0001", "Café Bar", testActor, "", 0)
	if err != nil {
		t.Fatalf("retitle to a non-ASCII title: %v", err)
	}
	hasWarning := false
	for _, f := range res.Findings {
		if f.Code == "slug-dropped-chars" {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Errorf("expected slug-dropped-chars warning on retitle; got %+v", res.Findings)
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
