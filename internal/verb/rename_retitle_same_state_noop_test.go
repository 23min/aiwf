package verb_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
// owns the CANONICAL `# <id> — <title>` heading: it states the entity's own
// title, so one that disagrees is stale rather than chosen, and an operator who
// wants their own heading writes a non-canonical one that retitle leaves alone.
// An out-of-sync canonical H1 is therefore work left to do; comparing the title
// alone would claim success over that state.
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

// TestRetitle_TitleWithDollarToken_SurvivesTheH1Rewrite pins the H1 rewrite
// against regexp replacement-template expansion. The replacement string carries
// the operator's title, and `Regexp.ReplaceAll` expands `$name` / `${name}`
// inside a replacement. The H1 pattern has no capture groups, so every such
// reference would expand to the empty string and silently delete part of the
// title.
//
// The same-title arm is the one that bites twice: the convergence guard compares
// the body against what rewriteEntityH1 would write, so a mangled H1 matches its
// own mangling and every later retitle converges over the damage.
func TestRetitle_TitleWithDollarToken_SurvivesTheH1Rewrite(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		title string
	}{
		{"a positional-looking token", "Cost is $1 per unit"},
		{"a braced token", "Use ${TOKEN} in env"},
		{"a named token", "Budget in $USD only"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newRunner(t)
			r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, tc.title, testActor, verb.AddOptions{}))
			writeEntityH1(r.t, r, "E-0001", "# E-0001 — "+tc.title)

			// Retitling to the title already stored must leave the H1 alone.
			res, err := verb.Retitle(r.ctx, r.tree(), "E-0001", tc.title, testActor, "", 0)
			if err != nil {
				t.Fatalf("retitle to the stored title: %v", err)
			}
			if !res.NoOp {
				t.Errorf("res.NoOp = false, want true — the H1 already reads as requested")
			}
			if got := readEntityBody(t, r, "E-0001"); !strings.Contains(got, "# E-0001 — "+tc.title) {
				t.Errorf("H1 lost part of the title.\n got: %q\nwant it to contain: %q",
					got, "# E-0001 — "+tc.title)
			}
		})
		t.Run(tc.name+", retitling to it", func(t *testing.T) {
			t.Parallel()
			// The converging case above never applies a plan, so it cannot see
			// a mangled replacement — it reads back the H1 the fixture wrote.
			// This drives the write itself: the token has to survive landing in
			// the heading, which is where `$`-expansion actually corrupts it.
			r := newRunner(t)
			r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Placeholder title", testActor, verb.AddOptions{}))
			writeEntityH1(r.t, r, "E-0001", "# E-0001 — Placeholder title")

			res, err := verb.Retitle(r.ctx, r.tree(), "E-0001", tc.title, testActor, "", 0)
			if err != nil {
				t.Fatalf("retitle to a $-bearing title: %v", err)
			}
			if res.NoOp {
				t.Fatal("res.NoOp = true, want a real retitle — the title changed")
			}
			if res.Plan == nil {
				t.Fatal("res.Plan = nil, want a plan writing the new H1")
			}
			if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
				t.Fatalf("apply: %v", applyErr)
			}
			if got := readEntityBody(t, r, "E-0001"); !strings.Contains(got, "# E-0001 — "+tc.title) {
				t.Errorf("the $-token did not survive the H1 write.\n got: %q\nwant it to contain: %q",
					got, "# E-0001 — "+tc.title)
			}
		})
	}
}

// TestRetitle_NarrowIDSpelling_StillRepairsTheH1 pins R1 for the H1 comparison.
// Parsers accept narrower legacy widths, so `E-01` names the very entity stored
// as `E-0001`. Comparing against the operator's spelling matches no H1 carrying
// the canonical id, which reports a drifted heading as already-consistent and
// converges over it — a NoOp that is false about the surface it claims to have
// checked.
func TestRetitle_NarrowIDSpelling_StillRepairsTheH1(t *testing.T) {
	t.Parallel()
	const title = "Alpha epic"
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, title, testActor, verb.AddOptions{}))
	writeEntityH1(r.t, r, "E-0001", "# E-0001 — Stale drifted heading")

	res, err := verb.Retitle(r.ctx, r.tree(), "E-01", title, testActor, "", 0)
	if err != nil {
		t.Fatalf("retitle at a narrow id width: %v", err)
	}
	if res.NoOp {
		t.Fatalf("res.NoOp = true, want false — E-01 is E-0001, whose H1 has drifted")
	}
	if res.Plan == nil {
		t.Fatal("res.Plan = nil, want a plan repairing the H1")
	}
	if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: %v", applyErr)
	}
	if got := readEntityBody(t, r, "E-0001"); !strings.Contains(got, "# E-0001 — "+title) {
		t.Errorf("H1 not repaired.\ngot: %q", got)
	}
}

// readEntityBody returns an entity file's contents as a string.
func readEntityBody(t *testing.T, r *runner, id string) string {
	t.Helper()
	e := r.tree().ByID(id)
	if e == nil {
		t.Fatalf("%s missing from the fixture tree", id)
	}
	raw, err := os.ReadFile(filepath.Join(r.root, e.Path)) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("reading %s: %v", id, err)
	}
	return string(raw)
}

// TestRetitle_PathWithoutIDPrefix_Refuses pins the refusal for a name
// renamePaths cannot rewrite. The slug substitution splits the on-disk name on
// its leading id, so a directory that carries no id prefix has nothing to
// substitute — and the verb must say so rather than proceed.
func TestRetitle_PathWithoutIDPrefix_Refuses(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	writeLooseEpicOnly(t, r.root, "plainname", "E-0001", "Hand authored epic")

	_, err := verb.Retitle(r.ctx, r.tree(), "E-0001", "Another title", testActor, "", 0)
	if err == nil {
		t.Fatal("retitle over a path with no id prefix returned no error, want a refusal")
	}
	// Naming the path shape keeps this from passing on an unrelated refusal —
	// an entity that failed to load would also return non-nil.
	if !strings.Contains(err.Error(), "no id prefix") {
		t.Errorf("err = %v, want the path-shape refusal", err)
	}
}

// TestRetitle_StoredTitleSlugifiesToNothing_PreservesTheSlug pins the polarity
// of slugTracksTitle's empty-derived arm. A stored title that normalizes away
// entirely cannot be the source of the slug on disk, so that slug is the
// operator's and survives the retitle. Answering the other way would re-derive
// over it.
func TestRetitle_StoredTitleSlugifiesToNothing_PreservesTheSlug(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	writeLooseEpicOnly(t, r.root, "E-0001-legacy-slug", "E-0001", "日本語のタイトル")

	res, err := verb.Retitle(r.ctx, r.tree(), "E-0001", "Now An ASCII Title", testActor, "", 0)
	if err != nil {
		t.Fatalf("retitle over a title that slugifies to nothing: %v", err)
	}
	if res.Plan == nil {
		t.Fatal("res.Plan = nil, want a plan rewriting the title")
	}
	for _, op := range res.Plan.Ops {
		if op.Type == verb.OpMove {
			t.Errorf("plan moves %s -> %s, want the slug preserved", op.Path, op.NewPath)
		}
	}
}

// TestRetitle_PreservedSlug_SuppressesTheSlugDroppedWarning pins the negative
// half of the warning. The notice names the slug a title derives; on a path
// where that slug is never written, reporting it would point an operator at a
// path that does not exist.
func TestRetitle_PreservedSlug_SuppressesTheSlugDroppedWarning(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Rename(r.ctx, r.tree(), "E-0001", "short-path", testActor, 0))

	res, err := verb.Retitle(r.ctx, r.tree(), "E-0001", "Café Bar", testActor, "", 0)
	if err != nil {
		t.Fatalf("retitle a preserved-slug entity to a non-ASCII title: %v", err)
	}
	for _, f := range res.Findings {
		if f.Code == "slug-dropped-chars" {
			t.Errorf("got %s naming a slug this call never writes: %+v", f.Code, f)
		}
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
	// ReplaceAllLiteralString, not ReplaceAllString: heading carries test titles
	// that contain `$1` / `${TOKEN}`, which the template form would expand away —
	// silently mangling the fixture the $-token test depends on.
	patched := regexp.MustCompile(`(?m)^# `+regexp.QuoteMeta(id)+` — .*$`).ReplaceAllLiteralString(string(raw), heading)
	if patched == string(raw) {
		// The template may ship no canonical H1; add one so the drift is real.
		patched = string(raw) + "\n" + heading + "\n"
	}
	if writeErr := os.WriteFile(path, []byte(patched), 0o600); writeErr != nil {
		t.Fatalf("writing %s: %v", id, writeErr)
	}
}

// TestRetitleAndRename_LineBreakTitle_Refused pins the verb-level consequence of
// the single-line title rule. A line break in a title is not merely malformed:
// it makes the H1 and AC-heading rewrites permanently non-convergent, because
// their patterns are line-anchored and match only the first fragment, so every
// invocation appends the remainder again and the same-state guard can never see
// a match. Unbounded growth, at exit 0, invisible to `aiwf check`.
//
// All four surfaces that accept a title are covered — the entity and composite
// paths of both verbs — because `rename` dispatches to its AC path before its
// own validation runs, so the entity-level guard does not cover it.
func TestRetitleAndRename_LineBreakTitle_Refused(t *testing.T) {
	t.Parallel()
	const bad = "Alpha\nBeta"
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "cache warms on boot", testActor))

	t.Run("retitle an entity", func(t *testing.T) {
		t.Parallel()
		_, err := verb.Retitle(r.ctx, r.tree(), "E-0001", bad, testActor, "", 0)
		if err == nil {
			t.Fatal("retitle accepted a title containing a line break, want a refusal")
		}
		if !strings.Contains(err.Error(), "line break") {
			t.Errorf("retitle refused for the wrong reason: %v", err)
		}
	})
	t.Run("retitle an AC", func(t *testing.T) {
		t.Parallel()
		_, err := verb.Retitle(r.ctx, r.tree(), "M-0001/AC-1", bad, testActor, "", 0)
		if err == nil {
			t.Fatal("retitle accepted an AC title containing a line break, want a refusal")
		}
		if !strings.Contains(err.Error(), "line break") {
			t.Errorf("retitle on an AC refused for the wrong reason: %v", err)
		}
	})
	t.Run("rename an AC", func(t *testing.T) {
		t.Parallel()
		_, err := verb.Rename(r.ctx, r.tree(), "M-0001/AC-1", bad, testActor, 0)
		if err == nil {
			t.Fatal("rename accepted an AC title containing a line break, want a refusal")
		}
		if !strings.Contains(err.Error(), "line break") {
			t.Errorf("rename on an AC refused for the wrong reason: %v", err)
		}
	})
	t.Run("add an entity", func(t *testing.T) {
		t.Parallel()
		// KindEpic, not KindGap: a gap is born-complete (G-0326), so creating
		// one with no body is refused for that reason alone and the assertion
		// would hold whether or not the title was ever inspected. Asserting on
		// the message closes the same gap from the other side.
		_, err := verb.Add(r.ctx, r.tree(), entity.KindEpic, bad, testActor, verb.AddOptions{})
		if err == nil {
			t.Fatal("add accepted a title containing a line break, want a refusal")
		}
		if !strings.Contains(err.Error(), "line break") {
			t.Errorf("add refused for the wrong reason: %v", err)
		}
	})
}
