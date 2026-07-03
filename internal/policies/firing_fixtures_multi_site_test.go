package policies

import (
	"path/filepath"
	"testing"
)

// TestFiringFixtures_MultiSite is the G-0262 burn-down positive control
// for the multi-dark-site policies (M-0166/AC-2). Each policy has several
// Violation construction lines (one per violation class), and the
// firing-fixture-presence gate tracks darkness per line — so a policy needs
// one fixture per dark class. Each row drives exactly one class and asserts
// the policy returns >=1 Violation; coverage then confirms every dark line
// is lit.
func TestFiringFixtures_MultiSite(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		policy func(string) ([]Violation, error)
		files  map[string]string
	}{
		// capture-stdout-singleton: a CaptureStdout defined outside the
		// canonical testutil prefix fires both "defined outside" and the
		// post-loop "canonical not found" lines.
		{
			name:   "capture-stdout/outside-canonical",
			policy: PolicyCaptureStdoutSingleton,
			files:  map[string]string{"cmd/aiwf/cap.go": "package main\n\nfunc CaptureStdout() {}\n"},
		},

		// design-doc-anchors-valid: broken link + broken anchor.
		{
			name:   "design-doc-anchors/broken-link-and-anchor",
			policy: PolicyDesignDocAnchors,
			files: map[string]string{
				"docs/pocv3/a.md":    "# A\n\n[x](./missing.md)\n\n[y](./real.md#nope)\n",
				"docs/pocv3/real.md": "# Real\n\nbody\n",
			},
		},

		// embedded-rituals: a work/tracking/ reference + a "tracking doc"
		// mention without "v1".
		{
			name:   "embedded-rituals/tracking-refs",
			policy: PolicyEmbeddedRitualsNoRetiredTrackingDoc,
			files:  map[string]string{"internal/skills/embedded-rituals/x.md": "see work/tracking/foo here\n\nthis is a tracking doc mention\n"},
		},

		// finding-codes-have-tests: an untested code const + an untested
		// code literal (no _test.go references either).
		{
			name:   "finding-codes-have-tests/const-and-literal",
			policy: PolicyFindingCodesHaveTests,
			files:  map[string]string{"internal/check/x.go": "package check\n\nconst MadeUpCode = \"made-up-code\"\n\nvar _ = Finding{Code: \"other-made-up-code\"}\n"},
		},

		// m0132-claude-md-devcontainer-section: missing file + present
		// but missing the ### Devcontainer subsection.
		{name: "m0132-claude-section/missing", policy: PolicyM0132ClaudeMdDevcontainerSection, files: map[string]string{}},
		{
			name:   "m0132-claude-section/no-subsection",
			policy: PolicyM0132ClaudeMdDevcontainerSection,
			files:  map[string]string{"CLAUDE.md": "# X\n\n## Operator setup\n\nbody without the devcontainer subsection\n"},
		},

		// m0228-skills-policy-broadened-principle: missing file (read-error
		// site) + present-but-no-section (missing-section report branch) +
		// section-present-but-no-markers (marker report branch). The three
		// together light both construction sites and cover every not-green
		// branch of the section walk.
		{name: "m0228-skills-policy/missing", policy: PolicyM0228SkillsPolicyBroadenedPrinciple, files: map[string]string{}},
		{
			name:   "m0228-skills-policy/no-section",
			policy: PolicyM0228SkillsPolicyBroadenedPrinciple,
			files:  map[string]string{"CLAUDE.md": "# X\n\n## Go conventions\n\nbody with no ### Skills policy subsection\n"},
		},
		{
			name:   "m0228-skills-policy/no-markers",
			policy: PolicyM0228SkillsPolicyBroadenedPrinciple,
			files:  map[string]string{"CLAUDE.md": "# X\n\n## Go conventions\n\n### Skills policy\n\nshipped skill bodies cite no real entity id, and nothing else here\n"},
		},

		// m0132-devcontainer-readme: missing + missing-required-sections.
		{name: "m0132-readme/missing", policy: PolicyM0132DevcontainerReadme, files: map[string]string{}},
		{
			name:   "m0132-readme/missing-sections",
			policy: PolicyM0132DevcontainerReadme,
			files:  map[string]string{".devcontainer/README.md": "# Devcontainer\n\n## Some Unrelated Section\n\nbody\n"},
		},

		// m0132-devcontainer-shape: missing + invalid-JSON + empty-object.
		{name: "m0132-shape/missing", policy: PolicyM0132DevcontainerShape, files: map[string]string{}},
		{name: "m0132-shape/bad-json", policy: PolicyM0132DevcontainerShape, files: map[string]string{".devcontainer/devcontainer.json": "this is not json"}},
		{name: "m0132-shape/empty-object", policy: PolicyM0132DevcontainerShape, files: map[string]string{".devcontainer/devcontainer.json": "{}"}},

		// m0132-init-script: missing + present-minimal + unreadable. The
		// unreadable case makes the path a directory, so os.Stat succeeds
		// but os.ReadFile fails (the "ReadFile failed" construction line).
		{name: "m0132-init/missing", policy: PolicyM0132InitScript, files: map[string]string{}},
		{name: "m0132-init/minimal", policy: PolicyM0132InitScript, files: map[string]string{".devcontainer/init.sh": "#!/usr/bin/env bash\n"}},
		{name: "m0132-init/unreadable", policy: PolicyM0132InitScript, files: map[string]string{".devcontainer/init.sh/keep": "x"}},

		// m0132-initialize-script: missing + present-minimal + unreadable
		// (directory in place of the script file).
		{name: "m0132-initialize/missing", policy: PolicyM0132InitializeScript, files: map[string]string{}},
		{name: "m0132-initialize/minimal", policy: PolicyM0132InitializeScript, files: map[string]string{".devcontainer/initialize.sh": "#!/usr/bin/env bash\n"}},
		{name: "m0132-initialize/unreadable", policy: PolicyM0132InitializeScript, files: map[string]string{".devcontainer/initialize.sh/keep": "x"}},

		// m0132-devcontainer-lock: missing + bad-json + no-devcontainer-json
		// + bad-devcontainer-json + feature-mismatch.
		{name: "m0132-lock/missing", policy: PolicyM0132DevcontainerLock, files: map[string]string{}},
		{name: "m0132-lock/bad-json", policy: PolicyM0132DevcontainerLock, files: map[string]string{".devcontainer/devcontainer-lock.json": "not json"}},
		{name: "m0132-lock/no-devcontainer-json", policy: PolicyM0132DevcontainerLock, files: map[string]string{".devcontainer/devcontainer-lock.json": "{}"}},
		{
			name:   "m0132-lock/bad-devcontainer-json",
			policy: PolicyM0132DevcontainerLock,
			files: map[string]string{
				".devcontainer/devcontainer-lock.json": "{}",
				".devcontainer/devcontainer.json":      "not json",
			},
		},
		{
			name:   "m0132-lock/feature-mismatch",
			policy: PolicyM0132DevcontainerLock,
			files: map[string]string{
				".devcontainer/devcontainer-lock.json": "{\"features\":{}}",
				".devcontainer/devcontainer.json":      "{\"features\":{\"ghcr.io/x/y:1\":{}}}",
			},
		},

		// m0134-claude-md-test-running-sections: missing + present-malformed.
		{name: "m0134/missing", policy: PolicyM0134ClaudeMdTestRunningSections, files: map[string]string{}},
		{
			name:   "m0134/present-malformed",
			policy: PolicyM0134ClaudeMdTestRunningSections,
			files:  map[string]string{"CLAUDE.md": "# X\n\n#### Running tests in the devcontainer (primary)\n\nbody with no required markers\n"},
		},

		// m0137-ac3-batched-walker: a rule file lacking BulkRevwalk and
		// BlobReader and still defining a banned per-entity helper lights
		// all three accumulating lines.
		{
			name:   "m0137/all-three",
			policy: PolicyM0137AC3BatchedWalker,
			files:  map[string]string{"internal/check/fsm_history_consistent.go": "package check\n\nfunc walkOneEntity() {}\n"},
		},

		// race-parallel-cap: a -race line without -parallel 8, plus a
		// target with no -race line. All three target files must exist.
		{
			name:   "race-parallel-cap/cap-and-missing",
			policy: PolicyRaceParallelCap,
			files: map[string]string{
				"Makefile":                         "test:\n\tgo test -race ./...\n",
				".github/workflows/go.yml":         "name: x\n",
				".github/workflows/flake-hunt.yml": "name: y\n# valid: go test -race -parallel 8 ./...\nrun: go test -race -parallel 8 ./...\n",
			},
		},

		// read-only-verbs: a read-only verb that mutates lights the
		// mutation line; the other expected verbs being absent lights the
		// "not found" line.
		{
			name:   "read-only-verbs/mutation-and-missing",
			policy: PolicyReadOnlyVerbsDoNotMutate,
			files:  map[string]string{"internal/cli/check/check.go": "package check\n\nfunc Run() { os.WriteFile() }\n"},
		},

		// test-setup-presence: missing setup_test.go + unparseable
		// setup_test.go + setup_test.go without TestMain.
		{
			name:   "test-setup-presence/three-classes",
			policy: PolicyTestSetupPresence,
			files: map[string]string{
				"internal/a/a_test.go":     "package a\n",
				"internal/b/b_test.go":     "package b\n",
				"internal/b/setup_test.go": "package b\n\nthis is not valid go @@@\n",
				"internal/c/c_test.go":     "package c\n",
				"internal/c/setup_test.go": "package c\n",
			},
		},

		// m0202-devcontainer-onboarding: missing files (both report sites)
		// + a retired marker in each file + the banner missing its
		// verification pointer. Together these light every report site.
		{name: "m0202-onboarding/missing", policy: PolicyM0202DevcontainerOnboarding, files: map[string]string{}},
		{
			name:   "m0202-onboarding/init-retired-marker",
			policy: PolicyM0202DevcontainerOnboarding,
			files: map[string]string{
				".devcontainer/init.sh":   "aiwf doctor rituals:\n/plugin marketplace add 23min/ai-workflow-rituals\n",
				".devcontainer/README.md": "clean\n",
			},
		},
		{
			name:   "m0202-onboarding/readme-retired-marker",
			policy: PolicyM0202DevcontainerOnboarding,
			files: map[string]string{
				".devcontainer/init.sh":   "aiwf doctor rituals:\n",
				".devcontainer/README.md": "install both plugins at PROJECT scope\n",
			},
		},
		{
			name:   "m0202-onboarding/banner-missing-pointer",
			policy: PolicyM0202DevcontainerOnboarding,
			files: map[string]string{
				".devcontainer/init.sh":   "aiwf devcontainer ready.\n",
				".devcontainer/README.md": "clean\n",
			},
		},

		// trailer-order-matches-constants: each defensive drift line is an
		// early return, so one fixture per class.
		{name: "trailer-order/file-not-found", policy: PolicyTrailerOrderMatchesConstants, files: map[string]string{}},
		{name: "trailer-order/parse-error", policy: PolicyTrailerOrderMatchesConstants, files: map[string]string{"internal/gitops/trailers.go": "package gitops\n\nnot valid go @@@\n"}},
		{name: "trailer-order/no-consts", policy: PolicyTrailerOrderMatchesConstants, files: map[string]string{"internal/gitops/trailers.go": "package gitops\n\nfunc f() {}\n"}},
		{name: "trailer-order/no-order-slice", policy: PolicyTrailerOrderMatchesConstants, files: map[string]string{"internal/gitops/trailers.go": "package gitops\n\nconst TrailerVerb = \"aiwf-verb\"\n"}},

		// m0210-trailer-commit-drift: the trailered-commit prescription drift
		// chokepoint has one Violation construction line (a single report
		// closure), so any firing fixture lights it; the cases below each
		// target a distinct violation branch so the diff-scoped coverage gate
		// sees every arm exercised.
		//
		// required-missing: no ritual files -> both required wraps absent
		// (AC-1 presence guard, `if !ok`).
		{name: "m0210/required-missing", policy: PolicyM0210TrailerCommitDrift, files: map[string]string{}},
		// required-no-block: wrap-epic present but carries no trailered-commit
		// block (AC-1 `if !hasTrailerBlock`, the G-0219 missing-block mode).
		{
			name:   "m0210/required-no-block",
			policy: PolicyM0210TrailerCommitDrift,
			files:  map[string]string{aiwfxWrapEpicFixturePath: "# wrap-epic\n\nThis ritual documents no trailered-commit block.\n"},
		},
		// required-missing-key: wrap-epic block omits the aiwf-actor trailer
		// flag (AC-1 all-three-keys guard).
		{
			name:   "m0210/required-missing-key",
			policy: PolicyM0210TrailerCommitDrift,
			files: map[string]string{aiwfxWrapEpicFixturePath: `git merge --no-ff --no-commit epic/E-NN-<slug>

git commit -m "chore(epic): wrap E-NNNN" \
  --trailer "aiwf-verb: wrap-epic" \
  --trailer "aiwf-entity: E-NNNN"

Resolve identity from git config user.email; do not hardcode the id.
Variant casings such as Aiwf-Verb fail the kernel's trailer-keys policy.
`},
		},
		// wrap-missing-caveat: wrap-epic composes a full trailered commit but
		// drops the canonical variant-casings caveat (AC-2 caveat accompaniment).
		{
			name:   "m0210/wrap-missing-caveat",
			policy: PolicyM0210TrailerCommitDrift,
			files: map[string]string{aiwfxWrapEpicFixturePath: `git merge --no-ff --no-commit epic/E-NN-<slug>

git commit -m "chore(epic): wrap E-NNNN" \
  --trailer "aiwf-verb: wrap-epic" \
  --trailer "aiwf-entity: E-NNNN" \
  --trailer "aiwf-actor: human/<id>"

Resolve identity from git config user.email; do not hardcode the id.
`},
		},
		// merge-missing-identity: wrap-epic stages a --no-commit merge and
		// composes a trailered commit but drops the git-config identity rule
		// (AC-2 identity accompaniment at merge sites).
		{
			name:   "m0210/merge-missing-identity",
			policy: PolicyM0210TrailerCommitDrift,
			files: map[string]string{aiwfxWrapEpicFixturePath: `git merge --no-ff --no-commit epic/E-NN-<slug>

git commit -m "chore(epic): wrap E-NNNN" \
  --trailer "aiwf-verb: wrap-epic" \
  --trailer "aiwf-entity: E-NNNN" \
  --trailer "aiwf-actor: human/<id>"

Variant casings such as Aiwf-Verb fail the kernel's trailer-keys policy.
`},
		},
		// unreadable: a directory in place of wrap-epic's SKILL.md makes the
		// glob match but os.ReadFile fail (the "unreadable ritual" line).
		{name: "m0210/unreadable", policy: PolicyM0210TrailerCommitDrift, files: map[string]string{aiwfxWrapEpicFixturePath + "/keep": "x"}},

		// m0211-guidance-operating-anchors: the drift chokepoint over the
		// shipped guidance has two Violation construction lines — the
		// unreadable/absent-file line and the per-anchor loop line. One fixture
		// per line.
		//
		// missing-file: no guidance source -> os.ReadFile fails (the absent-file
		// line).
		{name: "m0211/missing-file", policy: PolicyM0211GuidanceOperatingAnchors, files: map[string]string{}},
		// missing-anchor: a guidance source that carries every curated anchor
		// except the cross-branch allocation rule (no `--fetch` / push-promptly)
		// -> the per-anchor loop line fires.
		{
			name:   "m0211/missing-anchor",
			policy: PolicyM0211GuidanceOperatingAnchors,
			files: map[string]string{m0209GuidanceFixturePath: `# guidance

- **Each mutating action is its own approval gate.** don't bundle.
- **On an id collision, run aiwf reallocate, not git mv.**
- **Promote an AC to met only with mechanical evidence.**
- **Decide one thing at a time.**
- **Never suggest the human pause.**
- The body-prose-id rule enforces id shapes.
`},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			for rel, content := range tc.files {
				mustWrite(t, filepath.Join(root, rel), content)
			}
			vs, err := tc.policy(root)
			if err != nil {
				t.Fatalf("%s: policy returned error: %v", tc.name, err)
			}
			if len(vs) == 0 {
				t.Errorf("%s: policy did not fire on its fixture (0 violations)", tc.name)
			}
		})
	}
}
