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

// id_rename_untrailered_scenarios_test.go — M-0160/AC-4: real-git
// E2E coverage of the new id-rename-untrailered kernel rule.
//
// The rule (defined in internal/check/id_rename_untrailered.go,
// wired via the CLI gather layer) fires when a commit between
// merge-base(HEAD, origin/main) and HEAD renames an id-bearing
// entity file AND lacks an aiwf-verb trailer in the rename-class
// closed set (retitle / rename / reallocate / archive / move).
//
// The integration tests below exercise the binary-level seam:
// real git mv against a real entity file, real aiwf check via
// subprocess, real envelope JSON. Without this seam, a regression
// in the gather wire-up (e.g., the new walker's nil-pass shape,
// the M-0106/F-1 anti-pattern) would surface only at user-push
// time.
//
// AC-4 is a genuine TDD cycle (not regression-pinning):
//  1. RED: tests fail because the rule isn't yet implemented
//     OR isn't yet wired through provenance.go.
//  2. GREEN: implementation + gather wiring; tests pass.
//  3. REFACTOR: hint + SKILL.md + acknowledge-illegal silencing
//     + drift policies.

// TestIDRenameUntrailered_AC4_InlineGitMvFiresFinding pins the
// load-bearing claim: an inline `git mv` of an id-bearing entity
// file, committed without an aiwf-verb trailer, fires the new
// rule. Mirrors CLAUDE.md §"Id-collision resolution at merge
// time" — the operator-discipline failure mode where someone
// resolves a trunk-collision via direct git mv instead of
// `aiwf reallocate <id-or-path>`.
//
// Setup:
//   - Trunk-side: aiwf add gap creates G-0001 with one slug;
//     push origin/main.
//   - Feature branch: git mv renames the file to a different slug,
//     commit with conventional-commits subject + no aiwf-verb
//     trailer.
//
// Expected envelope:
//   - id-rename-untrailered finding fires (warning severity, per
//     the M-0106 / G-0150 precedent for chokepoint rules — error
//     tightening deferred to a future D-NNN).
//   - ids-unique/trunk-collision does NOT fire (the inline git mv
//     is cleanly paired by gitops.RenamesFromRef's cumulative
//     -M50 fallback; G-0109's pre-existing behavior).
func TestIDRenameUntrailered_AC4_InlineGitMvFiresFinding(t *testing.T) {
	t.Parallel()
	RunScenarios(t, []Scenario{
		{
			Name: "inline git mv of an id-bearing entity file with no aiwf-verb trailer fires id-rename-untrailered (M-0160/AC-4: CLAUDE.md §Id-collision)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()

				// Trunk-side: aiwf add gap → G-0001 at the original slug.
				env.MustRunBin("add", "gap", "--title", "Trunk gap original slug")
				env.MustRunGit("push", "origin", "main")

				// Feature branch from origin/main's tip.
				env.MustRunGit("checkout", "-b", "feature/inline-rename")

				// Locate the entity file post-`add`.
				oldRel := findEntityFile(t, env, "G-0001")
				if oldRel == "" {
					t.Fatal("G-0001 not found after `aiwf add gap`")
				}

				// Inline `git mv` to a different slug. The new path's
				// id (G-0001) is preserved; only the slug changes —
				// modeling the "resolved a trunk-collision via inline
				// git mv without reaching for aiwf reallocate"
				// operator-discipline gap CLAUDE.md documents.
				newRel := "work/gaps/G-0001-renamed-via-inline-git-mv.md"
				env.MustRunGit("mv", oldRel, newRel)

				// Commit with NO aiwf-verb trailer. Conventional-commits
				// subject only — what an operator would naturally type
				// if they hadn't reached for `aiwf reallocate`.
				env.MustRunGit("commit", "-m", "chore: rename G-0001 slug")

				// Discrimination assertion (reviewer S3 follow-up
				// pre-GREEN): inline `git mv` similarity is high, so
				// gitops.RenamesFromRef's pass 2 (-M50 cumulative)
				// pairs the rename. The trunk-collision rule MUST
				// stay silent — this is the docstring-pinned claim
				// at the file header. Without this assertion, a
				// future GREEN regression that accidentally coupled
				// id-rename-untrailered emission to trunk-collision
				// firing would pass spuriously. The framework's
				// Expectation cannot constrain `NoFindingWithCode`
				// and `FindingPresent` by different subcodes in one
				// row, so the assertion lands inline here (the
				// AC-1/scenario-3 envelope-parse pattern).
				envOut, _ := testutil.RunBin(t, env.Root, env.BinDir, nil, "check", "--format=json")
				var envelope struct {
					Findings []struct {
						Code    string `json:"code"`
						Subcode string `json:"subcode"`
					} `json:"findings"`
				}
				if jErr := json.Unmarshal([]byte(envOut), &envelope); jErr != nil {
					t.Fatalf("parse discrimination-check envelope: %v\n%s", jErr, envOut)
				}
				for _, f := range envelope.Findings {
					if f.Code == check.CodeIDsUnique && f.Subcode == "trunk-collision" {
						t.Errorf("trunk-collision finding fired on inline git mv (G-0109 fallback regressed, or AC-4's rule accidentally coupled to ids-unique); finding: %+v\nenvelope:\n%s", f, envOut)
					}
				}
			},
			Expect: Expectation{
				FindingPresent:  "id-rename-untrailered",
				FindingSeverity: "warning",
			},
		},

		// Positive control / discrimination: when the same rename
		// IS done via `aiwf rename` (which stamps an aiwf-verb:
		// rename trailer), the new rule must NOT fire — proves the
		// rule's trailer-aware suppression works end-to-end.
		{
			Name: "aiwf rename (with aiwf-verb: rename trailer) does NOT fire id-rename-untrailered (M-0160/AC-4: positive control)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()

				env.MustRunBin("add", "gap", "--title", "Trunk gap for rename control")
				env.MustRunGit("push", "origin", "main")

				env.MustRunGit("checkout", "-b", "feature/aiwf-rename-control")

				// `aiwf rename G-0001 <new-slug>` stamps aiwf-verb:
				// rename. The rule's trailer check skips it.
				env.MustRunBin("rename", "G-0001", "renamed-via-aiwf-verb")
			},
			Expect: Expectation{
				NoFindingWithCode: "id-rename-untrailered",
			},
		},
	})
}

// TestIDRenameUntrailered_AC4_AcknowledgeIllegalSilences pins
// the M-0159/AC-3 ack-helper-lift integration for the new rule:
// driving `aiwf acknowledge-illegal <sha> --reason "..."` against
// the violating commit silences the rule's finding without
// rewriting history. The unit-level pin lives at
// TestIDRenameUntrailered_AckedSHAExempted; this binary-level
// scenario exercises the gather → ackedSHAs map → rule wiring
// end-to-end through the live `aiwf check` envelope.
//
// REFACTOR-phase deliverable (RED-phase reviewer N7).
func TestIDRenameUntrailered_AC4_AcknowledgeIllegalSilences(t *testing.T) {
	t.Parallel()
	RunScenarios(t, []Scenario{
		{
			Name: "aiwf acknowledge-illegal silences id-rename-untrailered for the specific SHA (M-0160/AC-4: ack-helper-lift integration)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()

				// Trunk-side: gap with one slug + push.
				env.MustRunBin("add", "gap", "--title", "Trunk gap original slug")
				env.MustRunGit("push", "origin", "main")

				// Feature branch + inline git mv with no trailer
				// — the rule fires on this commit.
				env.MustRunGit("checkout", "-b", "feature/ack-silenced-rename")
				oldRel := findEntityFile(t, env, "G-0001")
				if oldRel == "" {
					t.Fatal("G-0001 not found after `aiwf add gap`")
				}
				newRel := "work/gaps/G-0001-renamed-for-ack-fixture.md"
				env.MustRunGit("mv", oldRel, newRel)
				env.MustRunGit("commit", "-m", "chore: rename G-0001 slug (will be acked)")

				// Capture the violating commit's SHA — this is the
				// argument `aiwf acknowledge-illegal` consumes.
				violatingSHA := strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))

				// Sovereign-human acknowledgment via the kernel verb.
				// AcknowledgeIllegal (the M-0159 helper at
				// branch_scenarios_helpers_test.go:954) wraps
				// `aiwf acknowledge-illegal <sha> --reason "..."`
				// — same shape M-0159/AC-4 established for the
				// three pre-existing ack-consuming rules.
				AcknowledgeIllegal(t, env, violatingSHA,
					"M-0160/AC-4 REFACTOR fixture: silencing the inline-git-mv rename for the specific SHA")
			},
			// The rule must be silent on the violating commit
			// after the acknowledgment lands. The framework's
			// `NoFindingWithCode` asserts no finding with code
			// id-rename-untrailered remains in the envelope —
			// the per-SHA closed-set scoping the M-0159/AC-3
			// helper-lift guarantees.
			Expect: Expectation{
				NoFindingWithCode: "id-rename-untrailered",
			},
		},
	})
}

// TestIDRenameUntrailered_AC4_NonEntityFileIgnored pins the
// negative case: an inline `git mv` of a NON-entity file
// (anything outside the kernel's id-bearing path patterns) must
// NOT fire the rule. Catches over-broad regex regression.
func TestIDRenameUntrailered_AC4_NonEntityFileIgnored(t *testing.T) {
	t.Parallel()
	RunScenarios(t, []Scenario{
		{
			Name: "inline git mv of a non-entity file does NOT fire id-rename-untrailered (M-0160/AC-4: non-entity exclusion)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()

				// Create a non-entity file in the repo.
				readmePath := filepath.Join(env.Root, "README.md")
				if err := os.WriteFile(readmePath, []byte("# README\n\nSome prose.\n"), 0o644); err != nil {
					t.Fatalf("write README.md: %v", err)
				}
				env.MustRunGit("add", "README.md")
				env.MustRunGit("commit", "-m", "chore: add README")
				env.MustRunGit("push", "origin", "main")

				env.MustRunGit("checkout", "-b", "feature/non-entity-rename")

				// Rename README.md to a different name. The path does
				// not match any of entity.PathKind's id-bearing
				// patterns. The rule must NOT fire on it.
				env.MustRunGit("mv", "README.md", "DOCS.md")
				env.MustRunGit("commit", "-m", "chore: rename README to DOCS")
			},
			Expect: Expectation{
				NoFindingWithCode: "id-rename-untrailered",
			},
		},
	})
}
