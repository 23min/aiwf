package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
)

// trunk_rename_g0167_test.go — M-0160/AC-2: binary-level real-git
// E2E regression pin for G-0167.
//
// G-0167 (closed by commit `8b56ba1c`) hardened
// `internal/gitops/refs.go::RenamesFromRef` to detect entity-file
// renames via the `aiwf-verb: retitle` / `rename` / `reallocate` /
// `archive` / `move` commit trailers in addition to git's default
// `-M50` cumulative similarity heuristic.
//
// The failure shape this AC pins (M-0125/G-0139, originally
// surfaced on the `epic/E-0033-...` push):
//
//	G-0139 was retitled (slug + frontmatter title) AND its body
//	tripled in subsequent edit-body commits within the same
//	feature branch. The cumulative diff between merge-base and
//	HEAD dropped below 50% similarity — git's default
//	`-M` rename detection missed it; the `ids-unique` check rule
//	then fired a false-positive `trunk-collision` on the entity
//	(one file on trunk at the OLD slug, one on branch at the NEW
//	slug, both with `id: G-0139`). Pre-push hook blocked the
//	push; the only escape was `git push --no-verify`.
//
// The fix's load-bearing claim: walk merge-base..HEAD for
// `aiwf-verb: retitle` (or other rename-class) trailers, and for
// each such commit, lift the rename from its per-commit diff —
// the per-commit diff has high similarity because the body did
// not change in the retitle commit; git's `--diff-filter=R` runs
// cleanly. Chains forward through multiple retitles.
//
// Unit coverage already comprehensive
// (`TestRenamesFromRef_DetectsTrailerDrivenRenameAcrossBodyEdits`
// + four siblings in `internal/gitops/refs_test.go`). What this
// AC adds: the **binary-level seam** through `aiwf check` →
// `RunProvenanceCheck` → `RenamesFromRef` → `ids-unique` rule.
// Without this seam pin, a regression that detached
// `RenamesFromRef` from the rule's input (e.g., the CLI gather
// layer passing `nil` for `TrunkRenames`, exactly the M-0106 / F-1
// pattern this milestone's epic was conceived to prevent recurring)
// would surface only at user-push time. Catching it at CI time
// requires driving the verbs as subprocess and parsing the
// envelope.

// TestTrunkRenameScenarios_AC2_G0167TrailerDrivenRescue drives
// one real-git scenario through the M-0159 RunScenarios framework:
//
//  1. Trunk-side: a gap entity with a short slug and a short body
//     pushed to origin/main.
//  2. Feature branch: `aiwf retitle` rewrites the slug + frontmatter
//     title, then `aiwf edit-body` substantially enriches the body.
//  3. Cumulative diff between origin/main and feature HEAD: well
//     below 50% similarity (the body-enrichment dominates the
//     diff; git's default rename detection cannot pair the
//     old-slug and new-slug files).
//  4. `aiwf check` MUST NOT fire `ids-unique/trunk-collision`
//     — the trailer-driven rename detection in
//     `internal/gitops/refs.go` is the rescue path.
//
// Sabotage-verified at AC-2 RED: reverting pass 1 of
// `RenamesFromRef` (the trailer-walk arm at refs.go:177-241) fires
// the scenario with envelope showing the trunk-collision finding.
// Discrimination confirmed end-to-end through the binary.
func TestTrunkRenameScenarios_AC2_G0167TrailerDrivenRescue(t *testing.T) {
	t.Parallel()
	RunScenarios(t, []Scenario{
		{
			Name: "retitle + multi-commit body enrichment cumulative similarity below 50%; trailer-driven rename detection prevents trunk-collision false positive (M-0160/AC-2: G-0167)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()

				// Trunk-side: a gap entity with a moderate initial body.
				// The body must be large enough that the retitle
				// commit's per-commit similarity is well ABOVE 50%
				// (so the trailer-driven detection lifts it cleanly
				// from `git show -M`), and the subsequent body
				// enrichment must be large enough that the cumulative
				// origin/main..HEAD similarity drops BELOW 50% (so
				// git's default -M50 cumulative detection misses it
				// and the G-0167 rescue is genuinely necessary).
				//
				// Use `aiwf add gap --body-file` to seed a moderate
				// body (the default `aiwf add` stub is too small —
				// the title-line change alone drops per-commit
				// similarity to ~48%, below the -M default).
				seedFile := filepath.Join(t.TempDir(), "seed-body.md")
				if err := os.WriteFile(seedFile, []byte(seedBodyAC2), 0o644); err != nil {
					t.Fatalf("write seed body fixture: %v", err)
				}
				env.MustRunBin("add", "gap",
					"--title", "Trunk gap original short title",
					"--body-file", seedFile)
				env.MustRunGit("push", "origin", "main")

				// Cut a feature branch from origin/main's tip.
				env.MustRunGit("checkout", "-b", "feature/g0167-shape")

				// Step 1: retitle. The verb renames the file to the
				// new slug AND rewrites the frontmatter `title:`
				// AND stamps an `aiwf-verb: retitle` trailer on the
				// commit. The retitle commit's per-commit diff is
				// almost entirely the file move (body unchanged in
				// this commit), so its `git show --diff-filter=R`
				// pairs the old and new paths cleanly.
				env.MustRunBin("retitle", "G-0001",
					"Trunk gap retitled to a substantially longer and richer title shape",
					"--reason", "AC-2 G-0167 fixture: retitle the gap so the slug changes")

				// Step 2: body enrichment. Replace the body with a
				// long prose block via `aiwf edit-body --body-file`.
				// The new body is ~25x the original 3-line stub —
				// enough that the cumulative origin/main..HEAD diff
				// drops well below the -M50 similarity threshold.
				bodyFile := filepath.Join(t.TempDir(), "enriched-body.md")
				if err := os.WriteFile(bodyFile, []byte(longEnrichedBodyAC2), 0o644); err != nil {
					t.Fatalf("write enriched body fixture: %v", err)
				}
				env.MustRunBin("edit-body", "G-0001",
					"--body-file", bodyFile,
					"--reason", "AC-2 G-0167 fixture: substantial body enrichment pushing similarity below -M50")

				// Sanity-check the fixture's similarity is actually
				// below 50%. If a future kernel change made the
				// fixture too small to trigger the failure mode, the
				// scenario would silently start exercising the
				// non-G-0167 path. Run `git diff -M50 --diff-filter=R
				// origin/main HEAD -- work/gaps/` and assert the
				// retitle is NOT detected by default git rename
				// detection at the -M50 threshold (so the rescue
				// path is genuinely necessary).
				out := env.MustRunGit("diff", "-M50", "--name-status",
					"--diff-filter=R", "origin/main", "HEAD", "--", "work/gaps/")
				if strings.TrimSpace(out) != "" {
					t.Fatalf("fixture sanity check failed: git -M50 detected the rename without the kernel's trailer-driven rescue\n(diff output: %q)\n— the body enrichment was insufficient to push cumulative similarity below 50%%; the AC-2 scenario would pass spuriously even with G-0167's fix reverted; increase the size of `longEnrichedBodyAC2` so the cumulative diff dominates the seed body", out)
				}

				// Per-commit sanity check: the retitle commit's
				// per-commit similarity (via `git show -M` default
				// 50%) MUST resolve to a rename pair. The
				// trailer-driven pass at internal/gitops/refs.go uses
				// exactly this command shape to lift the rename; if
				// the per-commit similarity is below 50%, the fix's
				// rescue path silently does nothing and the scenario
				// would fail to discriminate (sabotage probe couldn't
				// produce a meaningful RED). Verified during AC-2
				// fixture authoring: per-commit similarity is ~91%
				// when the initial body is the seedBodyAC2 stub.
				perCommit := env.MustRunGit("show", "-M", "--diff-filter=R",
					"--name-status", "--format=", "HEAD~1")
				if !strings.Contains(perCommit, "\tR") && !strings.HasPrefix(perCommit, "R") {
					t.Fatalf("fixture sanity check failed: per-commit rename detection on the retitle commit (HEAD~1) does not produce an R<score> line\n(output: %q)\n— G-0167's trailer-driven rescue at internal/gitops/refs.go uses `git show -M --diff-filter=R` per commit; if this returns empty, the rescue silently does nothing and the scenario fails to discriminate. Increase the size of `seedBodyAC2` so the title-line change is a smaller fraction of the retitle commit's per-commit diff (target per-commit similarity > 50%%)", perCommit)
				}
			},
			// AC-promotion evidence anchor (per CLAUDE.md §"AC
			// promotion requires mechanical evidence"): the
			// load-bearing mechanical assertion for AC-2 lives in
			// the Expect block below. The pair (Code:
			// ids-unique, Subcode: trunk-collision) being ABSENT
			// from the envelope is the regression-pin. If G-0167's
			// trailer-driven detection at refs.go pass 1 regresses
			// (drops, reorders, gets nil-passed by the CLI gather
			// layer), this finding fires in the envelope and the
			// assertion flips from absent to present.
			// Sabotage-verified during AC-2 RED phase by reverting
			// the pass 1 block at refs.go:247-253 — the test fired
			// with the expected envelope shape.
			// The load-bearing assertion: aiwf check exits with NO
			// `ids-unique/trunk-collision` finding. Pre-G-0167 this
			// fires (the false positive). Post-G-0167 the trailer-
			// driven detection lifts the retitle rename from the
			// per-commit diff and the rule's rename-map exemption
			// suppresses the false collision.
			Expect: Expectation{
				NoFindingWithCode: check.CodeIDsUnique,
				FindingSubcode:    "trunk-collision",
			},
		},
	})
}

// seedBodyAC2 is the initial gap body on the trunk side. Sized to
// be moderate (~25 lines) so the retitle commit's per-commit
// similarity is well above the -M50 threshold (the title-line
// change is a small fraction of the file). Subsequent body
// enrichment (longEnrichedBodyAC2) is ~5x this size; the
// cumulative origin/main..HEAD diff drops below 50% similarity,
// which is the failure mode G-0167 rescues.
const seedBodyAC2 = `
## Problem

Initial seed body for a gap that will later get retitled and
enriched. The seed prose establishes the diagnostic surface in
a few short sections so a subsequent body enrichment grows the
file substantially in a way that drops cumulative origin..HEAD
similarity below the kernel's rename-detection default.

## Why it matters

Gaps frequently evolve from a stub recording the initial
observation to a fuller body capturing rationale, fix shape,
test surface, and closing notes. The retitle-with-enrichment
pattern is normal; the chokepoint must handle it.

## Notes

Synthetic prose; not anonymized from any real entity. Sized
to make the per-commit similarity of the subsequent retitle
commit well above 50% while leaving room for the body
enrichment to drop the cumulative diff below 50%.
`

// longEnrichedBodyAC2 is the post-retitle body for the G-0167
// fixture. Sized to make the cumulative origin/main..HEAD diff
// drop below the -M50 similarity threshold (the fixture sanity
// check inside the scenario asserts that git's default -M50
// rename detection cannot pair the files, so the trailer-driven
// rescue at refs.go is genuinely the load-bearing path under
// test).
//
// Content is synthetic prose patterned after the M-0125/G-0139
// shape — a substantial set of rationale sections (Problem,
// Why it matters, Proposed fix, Test surface, History,
// Workaround, Closing notes) that an authoring operator would
// add when an initial gap stub is elaborated post-retitle.
// Synthetic content per testdata discipline; not anonymized from
// any real entity.
const longEnrichedBodyAC2 = `
## Problem

The original gap stub was a placeholder; the elaboration here
reframes the diagnostic surface in substantially more detail.
A reader landing on this entity should be able to reconstruct
the failure mode, the chokepoint that misses it today, and the
remediation path without cross-referencing other documents.

## Why it matters

This particular shape sits at the intersection of several
mechanical chokepoints in the kernel: the trunk-aware allocator,
the ids-unique rule, the trailer-driven rename detection in
refs.go, and the operator-discipline gap that CLAUDE.md
documents as the manual-reallocate-after-merge dance. Each
piece is individually correct; the failure surfaces when the
operator's mental model differs from the kernel's.

## Proposed fix shape

Two viable paths, both of which preserve the kernel's
correctness-must-not-depend-on-LLM-behavior principle:

Path A — Lower the cumulative similarity threshold. Mechanically
straightforward but fragile: empirical evidence in this repo
shows the legitimate-rename and false-collision similarity
ranges overlap, so no single threshold catches the rename and
rejects the collision.

Path B — Use the kernel's existing operator-intent trailers as
ground truth. Every rename-class verb (retitle, rename,
reallocate, archive, move) stamps an aiwf-verb trailer on its
commit. That trailer is the authoritative signal for "this
commit was a rename"; git's similarity heuristic was always
just a convenient proxy.

Path B is the kernel-correctness path. The implementation walks
merge-base..HEAD for the rename-class trailers, lifts the
per-commit rename from each (the per-commit diff has high
similarity because the body did not change in the retitle
commit), and chains forward through multiple retitles within
the same branch.

## Test surface

Both unit and binary-level coverage land alongside the fix:

  - Unit: TestRenamesFromRef_DetectsTrailerDrivenRenameAcrossBodyEdits
    constructs the M-0125/G-0139 shape against the live git
    helper and asserts the rename is detected.
  - Unit: TestRenamesFromRef_ChainsForwardThroughMultipleRetitles
    pins the chain-forward property across two retitles in the
    same branch.
  - Unit: TestRenamesFromRef_IgnoresUncommittedRename pins the
    negative case (un-committed rename does not register).
  - Unit: TestRenamesFromRef_IgnoresParallelClonesG37Case pins
    the G-0109 / G37 case where two parallel clones each created
    a fresh entity at the same id; git's rename detection must
    NOT pair them (different bodies, different intent).
  - Binary E2E: this scenario.

## History

The failure mode was first surfaced on the epic/E-0033 push
during M-0125 wrap. The operator workaround at the time was
git push --no-verify, which bypasses the pre-push hook and
defeats the framework's correctness guarantees. The fix landed
shortly after as commit 8b56ba1c, closing G-0167. The binary-
level regression pin (this scenario) lands at M-0160/AC-2 as
part of the operational-pain regression milestone.

## Workaround (historical)

Pre-fix, operators encountering this pushed their work with
git push --no-verify (explicit human approval per CLAUDE.md).
The first epic/E-0033 push to origin used this workaround.
Post-fix, the workaround is no longer needed; the kernel
detects the rename via the operator-intent trailer.

## Closing this gap

When the fix lands and the binary-level regression test
exists, the gap closes with status: addressed,
addressed_by_commit: [8b56ba1c]. The CLAUDE.md section on
id-collision resolution documents the operator-discipline
path that remains for the genuinely-different-entity case
(merge-time parallel allocation, where aiwf reallocate is the
right remediation).

## Notes

This body is intentionally long to push the file's content
several multiples beyond the original stub. The diff against
origin/main is dominated by added prose, not by the slug
rename — exactly the M-0125/G-0139 shape that defeats default
git rename detection.
`
