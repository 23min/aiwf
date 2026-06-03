package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// reallocate_scenarios_test.go — M-0160/AC-1: real-git E2E
// coverage of `aiwf reallocate` against the corpus of historical
// verb invocations surfaced by the M-0159 history-mining audit
// (~19 actual `aiwf-verb: reallocate` commits in this repo's
// history; the audit's broader 26 count includes related work).
//
// AC-1's load-bearing claim per the M-0160 milestone spec:
//
//	"Verify the `aiwf reallocate` path holds under combinatorial
//	 verb-sequence scenarios."
//
// The corpus collapses to seven representative shapes:
//
//  1. Simple single-step renumber — `aiwf reallocate G-X` on an
//     entity with no descendants and no cross-references. Pins the
//     baseline trunk-aware allocator + frontmatter/file rename.
//
//  2. Chained renumber — `aiwf reallocate` applied to an entity
//     whose `prior_ids:` already carries one entry. Pins G-0118's
//     fix (prior_ids populated AND grows oldest-first across
//     multiple renumbers).
//
//  3. Cross-branch merge collision (the CLAUDE.md §"Id-collision
//     resolution at merge time" scenario) — trunk and branch each
//     independently allocated G-NNN; the trunk-aware allocator
//     missed the collision (parallel un-pushed branches); the
//     check rule `ids-unique/trunk-collision` fires; `aiwf
//     reallocate` resolves it and the rule goes silent.
//
//  4. Cross-reference body-prose rewrite — entity A's body
//     mentions entity B's id; reallocate B; A's body is rewritten
//     atomically in the same commit (G-5 invariant). Pins the
//     prose-grammar rewrite at internal/verb/reallocate.go.
//
//  5. aiwf-prior-entity trailer + history bridging — the
//     reallocate commit carries `aiwf-prior-entity: <old-id>`;
//     `aiwf history <old-id>` returns the new entity's lifecycle.
//     Pins the audit-trail guarantee.
//
//  6. Epic-with-milestone directory move — reallocate an epic;
//     the milestone inside the epic's directory has its file Path
//     updated atomically in the same commit. Pins the directory-
//     rename branch at internal/verb/reallocate.go (`pathInside`).
//
//  7. Trunk-allocator skips trunk-side ids (positive baseline) —
//     after trunk has G-NNN, a feature-branch `aiwf add gap`
//     allocates G-NNN+1, not G-NNN. The trunk-aware allocator's
//     normal-path guarantee that scenario 3's collision shape is
//     anomalous (parallel un-pushed branches), not the steady
//     state.
//
// Sabotage discipline: each scenario's load-bearing assertion is
// authored so reverting the underlying production code (trailer-
// driven rename detection in refs.go; prior_ids population in
// reallocate.go; prose rewrite; aiwf-prior-entity trailer) fires
// the scenario. Verified during AC-1 RED phase by replaying each
// sabotage against the worktree-built binary.

// TestReallocateScenarios_AC1_HistoricalCorpus drives all seven
// representative shapes through the M-0159 RunScenarios framework.
// Each Setup composes the fixture inline (real `aiwf add` + `aiwf
// reallocate` invocations against the worktree binary, real
// `git` for filesystem-state assertions); the framework's Expect
// runs `aiwf check --format=json` at the end and asserts no
// leftover findings.
func TestReallocateScenarios_AC1_HistoricalCorpus(t *testing.T) {
	t.Parallel()
	RunScenarios(t, []Scenario{
		// Scenario 1 — simple single-step renumber.
		{
			Name: "single-step renumber preserves id-grammar and allocates next free id (M-0160/AC-1: corpus baseline)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "gap", "--title", "First gap")
				oldPath := findEntityFile(t, env, "G-0001")
				if oldPath == "" {
					t.Fatalf("G-0001 not found after `aiwf add gap`")
				}

				out := env.MustRunBin("reallocate", "G-0001")
				if !strings.Contains(out, "G-0002") {
					t.Errorf("reallocate output does not name new id G-0002: %s", out)
				}

				// Filesystem invariant: G-0001 is gone; G-0002 exists.
				if fileExists(t, env, "G-0001") {
					t.Error("file at old id G-0001 still exists post-reallocate")
				}
				newPath := findEntityFile(t, env, "G-0002")
				if newPath == "" {
					t.Fatalf("G-0002 file not found after reallocate")
				}

				// Frontmatter invariant: id updated, prior_ids carries old id.
				fm := readFrontmatter(t, filepath.Join(env.Root, newPath))
				if !strings.Contains(fm, "id: G-0002") {
					t.Errorf("frontmatter id not updated to G-0002:\n%s", fm)
				}
				if !strings.Contains(fm, "G-0001") {
					t.Errorf("frontmatter does not record prior id G-0001:\n%s", fm)
				}
			},
			// No error findings expected. (Advisory warnings about
			// upstream / body emptiness are acceptable.)
			Expect: Expectation{
				NoFindingWithCode: check.CodeIDsUnique,
			},
		},

		// Scenario 2 — chained renumber; prior_ids grows.
		{
			Name: "chained renumber across two reallocates grows prior_ids oldest-first (M-0160/AC-1: G-0118 invariant)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "gap", "--title", "Chained gap")
				env.MustRunBin("reallocate", "G-0001")
				env.MustRunBin("reallocate", "G-0002")

				finalPath := findEntityFile(t, env, "G-0003")
				if finalPath == "" {
					t.Fatalf("G-0003 file not found after two chained reallocates")
				}
				fm := readFrontmatter(t, filepath.Join(env.Root, finalPath))
				if !strings.Contains(fm, "id: G-0003") {
					t.Errorf("final id not G-0003:\n%s", fm)
				}
				// G-0118 invariant: prior_ids grows. Both old ids must
				// appear, in their historical order (oldest-first).
				if !strings.Contains(fm, "G-0001") || !strings.Contains(fm, "G-0002") {
					t.Errorf("prior_ids does not record full chain (expected both G-0001 and G-0002):\n%s", fm)
				}
				// Ordering: G-0001 appears before G-0002 textually in
				// the frontmatter (the prior_ids list is rendered in
				// allocation order).
				idxOne := strings.Index(fm, "G-0001")
				idxTwo := strings.Index(fm, "G-0002")
				if idxOne < 0 || idxTwo < 0 || idxOne > idxTwo {
					t.Errorf("prior_ids ordering wrong; expected G-0001 before G-0002 (oldest-first per G-0118):\n%s", fm)
				}
			},
			Expect: Expectation{
				NoFindingWithCode: check.CodeIDsUnique,
			},
		},

		// Scenario 3 — cross-branch merge collision.
		{
			Name: "cross-branch merge collision: aiwf check fires trunk-collision; reallocate resolves it (M-0160/AC-1: CLAUDE.md §Id-collision)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()

				// Trunk-side: allocate G-0001 + push origin/main forward.
				env.MustRunBin("add", "gap", "--title", "Trunk-side gap")
				env.MustRunGit("push", "origin", "main")

				// Cut a feature branch from origin/main pre-push (the
				// parallel-allocation scenario), but simulate it
				// without reset: directly compose the colliding state
				// by hand-authoring a different G-0001 entity in a
				// branch cut from the post-push tip. Then manually
				// stage that file as if `aiwf add` had run on a
				// pre-push branch state.
				//
				// The aiwf-allocator on a fresh `aiwf add` would skip
				// G-0001 (trunk-aware); the kernel rule we're
				// exercising fires on the ALREADY-LANDED collision
				// state — two parallel branches that EACH allocated
				// G-0001 before either pushed. The synthetic fixture
				// reproduces that landed state without simulating the
				// time-travel.
				env.MustRunGit("checkout", "-b", "feature/parallel-allocation")

				collidingPath := "work/gaps/G-0001-different-thing-from-feature.md"
				// No `discovered_in:` — the synthetic fixture's
				// purpose is to exhibit the trunk-collision shape,
				// not to model a real cross-reference graph; setting
				// `discovered_in:` to a fabricated id would fire
				// `refs-resolve/unresolved` from the reallocate
				// verb's own pre-flight check.
				collidingBody := "---\nid: G-0001\ntitle: 'Different thing from feature branch'\nstatus: open\n---\n\n## What's missing\n\nFeature-side independently allocated G-0001 before trunk's update reached this branch — the parallel-clones case CLAUDE.md §Id-collision documents.\n"
				if err := os.WriteFile(filepath.Join(env.Root, collidingPath), []byte(collidingBody), 0o644); err != nil {
					t.Fatalf("write colliding gap: %v", err)
				}
				env.MustRunGit("add", collidingPath)
				env.MustRunGit("commit", "-m", "feat: feature-side G-0001 (parallel allocation)")

				// First-pass check: the specific finding `ids-unique`
				// with subcode `trunk-collision` MUST fire. Reviewer
				// nit (M-0160/AC-1 review T-strong-1): bare exit-code
				// assertion would pass spuriously if some OTHER
				// finding fired (refs-resolve, tree-shape, etc.).
				// Parse the envelope and pin both fields against the
				// rule's emit site at internal/check/check.go.
				firstOut, _ := testutil.RunBin(t, env.Root, env.BinDir, nil, "check", "--format=json")
				var firstEnvelope struct {
					Findings []struct {
						Code    string `json:"code"`
						Subcode string `json:"subcode"`
					} `json:"findings"`
				}
				if jErr := json.Unmarshal([]byte(firstOut), &firstEnvelope); jErr != nil {
					t.Fatalf("parse first-pass check envelope: %v\n%s", jErr, firstOut)
				}
				var sawTrunkCollision bool
				for _, f := range firstEnvelope.Findings {
					if f.Code == check.CodeIDsUnique && f.Subcode == "trunk-collision" {
						sawTrunkCollision = true
						break
					}
				}
				if !sawTrunkCollision {
					t.Fatalf("first-pass check did not fire ids-unique/trunk-collision; envelope findings:\n%s", firstOut)
				}

				// Operator resolution: reallocate by path (id is
				// duplicated, so path-disambiguation is required).
				env.MustRunBin("reallocate", collidingPath)

				// Post-reallocate: feature's file is at G-0002 with
				// prior_ids: [G-0001]. The trunk's G-0001 remains
				// untouched.
				newPath := findEntityFile(t, env, "G-0002")
				if newPath == "" {
					t.Fatalf("G-0002 file not found after reallocate")
				}
				fm := readFrontmatter(t, filepath.Join(env.Root, newPath))
				if !strings.Contains(fm, "id: G-0002") || !strings.Contains(fm, "G-0001") {
					t.Errorf("reallocated entity missing canonical frontmatter shape (id: G-0002 + prior G-0001):\n%s", fm)
				}
			},
			Expect: Expectation{
				NoFindingWithCode: check.CodeIDsUnique,
			},
		},

		// Scenario 4 — cross-reference body-prose rewrite (G-5).
		{
			Name: "cross-reference body-prose rewritten atomically on reallocate (M-0160/AC-1: G-5 prose-grammar invariant)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()

				// Two gaps: A references B in its body.
				env.MustRunBin("add", "gap", "--title", "Referencing gap")
				env.MustRunBin("add", "gap", "--title", "Referenced gap")

				gapA := findEntityFile(t, env, "G-0001")
				if gapA == "" {
					t.Fatalf("G-0001 missing")
				}
				// Append a body mention of G-0002 to G-0001 via
				// edit-body in bless mode (working-copy edit).
				bodyPath := filepath.Join(env.Root, gapA)
				current, err := os.ReadFile(bodyPath)
				if err != nil {
					t.Fatalf("read G-0001 body: %v", err)
				}
				updated := string(current) + "\n## Cross-reference\n\nThis gap depends on G-0002 to land first; see G-0002 for the upstream.\n"
				if werr := os.WriteFile(bodyPath, []byte(updated), 0o644); werr != nil {
					t.Fatalf("write G-0001 body: %v", werr)
				}
				env.MustRunBin("edit-body", "G-0001", "--reason", "add cross-reference to G-0002 (AC-1 S-4 fixture)")

				// Now reallocate G-0002 → G-0003. The kernel rewrites
				// G-0001's prose to point at G-0003 in the SAME
				// commit.
				env.MustRunBin("reallocate", "G-0002")

				// Read G-0001 body post-reallocate; assert prose was
				// rewritten.
				rewrittenA := findEntityFile(t, env, "G-0001")
				if rewrittenA == "" {
					t.Fatalf("G-0001 vanished after reallocating G-0002")
				}
				postBody, err := os.ReadFile(filepath.Join(env.Root, rewrittenA))
				if err != nil {
					t.Fatalf("re-read G-0001: %v", err)
				}
				if strings.Contains(string(postBody), "G-0002") {
					t.Errorf("G-0001's body still references G-0002 post-reallocate; prose rewrite (G-5) regressed:\n%s", postBody)
				}
				if !strings.Contains(string(postBody), "G-0003") {
					t.Errorf("G-0001's body does not reference G-0003 post-reallocate; prose rewrite (G-5) regressed:\n%s", postBody)
				}

				// And the rewrite landed atomically: the HEAD commit
				// is the reallocate commit, NOT a separate cleanup.
				headSubject := strings.TrimSpace(env.MustRunGit("log", "-1", "--pretty=%s"))
				if !strings.Contains(headSubject, "reallocate") {
					t.Errorf("HEAD subject is not the reallocate commit; rewrite was not atomic:\n%s", headSubject)
				}
			},
			Expect: Expectation{
				NoFindingWithCode: check.CodeIDsUnique,
			},
		},

		// Scenario 5 — aiwf-prior-entity trailer + history bridging.
		{
			Name: "reallocate commit carries aiwf-prior-entity trailer and aiwf history bridges old to new (M-0160/AC-1: audit-trail invariant)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "gap", "--title", "Bridging gap")
				env.MustRunBin("reallocate", "G-0001")

				// HEAD's trailer set: aiwf-verb: reallocate +
				// aiwf-entity: G-0002 + aiwf-prior-entity: G-0001.
				priorEntity := strings.TrimSpace(env.MustRunGit("log", "-1",
					"--pretty=%(trailers:key=aiwf-prior-entity,valueonly=true,unfold=true)"))
				if priorEntity != "G-0001" {
					t.Errorf("aiwf-prior-entity trailer = %q; want G-0001", priorEntity)
				}
				verb := strings.TrimSpace(env.MustRunGit("log", "-1",
					"--pretty=%(trailers:key=aiwf-verb,valueonly=true,unfold=true)"))
				if verb != "reallocate" {
					t.Errorf("aiwf-verb trailer = %q; want reallocate", verb)
				}

				// Bridge: `aiwf history G-0001` returns the new
				// entity's history (the reallocate commit is the
				// hand-off; queries for the OLD id surface the
				// renumber event AND the subsequent lifecycle of
				// G-0002).
				historyOld := env.MustRunBin("history", "G-0001")
				if !strings.Contains(historyOld, "reallocate") {
					t.Errorf("aiwf history G-0001 missing reallocate event:\n%s", historyOld)
				}
				if !strings.Contains(historyOld, "G-0002") {
					t.Errorf("aiwf history G-0001 does not bridge to G-0002:\n%s", historyOld)
				}
			},
			Expect: Expectation{
				NoFindingWithCode: check.CodeIDsUnique,
			},
		},

		// Scenario 6 — epic-with-milestone directory move.
		{
			Name: "reallocate epic atomically moves contained milestone (M-0160/AC-1: directory-rename invariant)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()

				env.MustRunBin("add", "epic", "--title", "Sample epic")
				// Add a milestone inside the epic.
				env.MustRunBin("add", "milestone", "--epic", "E-0001",
					"--tdd", "advisory", "--title", "Sample milestone")

				// Verify pre-state.
				preEpicDir := findEntityFile(t, env, "E-0001")
				if !strings.Contains(preEpicDir, "E-0001-") {
					t.Fatalf("E-0001 dir not found pre-reallocate; got %q", preEpicDir)
				}
				preMilestone := findEntityFile(t, env, "M-0001")
				if !strings.Contains(preMilestone, "E-0001-") {
					t.Fatalf("M-0001 should live inside E-0001's directory pre-reallocate; got %q", preMilestone)
				}

				// Reallocate the epic.
				env.MustRunBin("reallocate", "E-0001")

				// Post-state: E-0001 dir is gone; E-0002 dir exists
				// and contains M-0001 at its new path.
				if fileExists(t, env, "E-0001") {
					t.Errorf("E-0001 still present after reallocate")
				}
				postEpicDir := findEntityFile(t, env, "E-0002")
				if !strings.Contains(postEpicDir, "E-0002-") {
					t.Errorf("E-0002 dir not found post-reallocate; got %q", postEpicDir)
				}
				postMilestone := findEntityFile(t, env, "M-0001")
				if !strings.Contains(postMilestone, "E-0002-") {
					t.Errorf("M-0001 should now live inside E-0002's directory; got %q", postMilestone)
				}

				// M-0001's `parent:` frontmatter field rewritten from
				// E-0001 to E-0002. The directory-move is `git mv`'s
				// side effect; the parent-rewrite is rewriteEntityRefs'
				// frontmatter edit through the same atomic commit
				// (per internal/verb/reallocate.go). Reviewer nit
				// (M-0160/AC-1 review T-nit-2): without this assertion
				// scenario 6 would still pass even if the parent-field
				// rewrite silently regressed.
				milestoneFM := readFrontmatter(t, filepath.Join(env.Root, postMilestone))
				if !strings.Contains(milestoneFM, "parent: E-0002") {
					t.Errorf("M-0001 frontmatter `parent:` not rewritten to E-0002:\n%s", milestoneFM)
				}
				if strings.Contains(milestoneFM, "parent: E-0001") {
					t.Errorf("M-0001 frontmatter still references old `parent: E-0001`:\n%s", milestoneFM)
				}
			},
			Expect: Expectation{
				NoFindingWithCode: check.CodeIDsUnique,
			},
		},

		// Scenario 7 — trunk-allocator skips trunk-side ids
		// (positive baseline; complement to scenario 3).
		{
			Name: "trunk-aware allocator skips trunk-side ids on feature branch (M-0160/AC-1: positive baseline)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()

				// Trunk-side: allocate G-0001 + push.
				env.MustRunBin("add", "gap", "--title", "Trunk gap")
				env.MustRunGit("push", "origin", "main")

				// Feature branch from the post-push state.
				env.MustRunGit("checkout", "-b", "feature/allocate-after-trunk")

				// `aiwf add gap` on feature: allocator must skip
				// G-0001 (already on origin/main) and produce G-0002.
				out := env.MustRunBin("add", "gap", "--title", "Feature gap")
				if !strings.Contains(out, "G-0002") {
					t.Errorf("feature-side allocator did not skip trunk-side G-0001; got:\n%s", out)
				}

				// Filesystem invariant — the load-bearing assertion
				// (reviewer nit M-0160/AC-1 T-nit-1: the previous
				// output-substring guard was dead code; the verb's
				// subject line never contains the slug shape it was
				// checking against). The feature-side `aiwf add gap`
				// produces G-0002 with a "feature-gap" slug; the
				// trunk-side G-0001 was added with "Trunk gap" title
				// → its slug contains "trunk-gap". Find G-0001 and
				// assert its file's slug is the trunk-side one
				// (collision-free allocator).
				g0001Path := findEntityFile(t, env, "G-0001")
				if g0001Path == "" {
					t.Errorf("G-0001 vanished post-feature-add (allocator regressed)")
				} else if !strings.Contains(g0001Path, "trunk-gap") {
					t.Errorf("G-0001 file path %q does not name the trunk-side slug shape; feature-side may have collided", g0001Path)
				}
				// And G-0002 (the feature-side allocation) exists on
				// the feature branch.
				g0002Path := findEntityFile(t, env, "G-0002")
				if g0002Path == "" {
					t.Errorf("G-0002 missing post-feature-add; allocator did not skip trunk-side G-0001")
				}
			},
			Expect: Expectation{
				NoFindingWithCode: check.CodeIDsUnique,
			},
		},
	})
}

// Frontmatter / filesystem inspection helpers (findEntityFile,
// fileExists, readFrontmatter) lifted to
// branch_scenarios_helpers_test.go at M-0160/AC-3 once the second
// caller (apply_rollback_g0170_test.go) appeared, per the
// AC-1-time docstring promise to lift on the second caller.
