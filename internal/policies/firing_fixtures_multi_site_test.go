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

		// trailer-order-matches-constants: each defensive drift line is an
		// early return, so one fixture per class.
		{name: "trailer-order/file-not-found", policy: PolicyTrailerOrderMatchesConstants, files: map[string]string{}},
		{name: "trailer-order/parse-error", policy: PolicyTrailerOrderMatchesConstants, files: map[string]string{"internal/gitops/trailers.go": "package gitops\n\nnot valid go @@@\n"}},
		{name: "trailer-order/no-consts", policy: PolicyTrailerOrderMatchesConstants, files: map[string]string{"internal/gitops/trailers.go": "package gitops\n\nfunc f() {}\n"}},
		{name: "trailer-order/no-order-slice", policy: PolicyTrailerOrderMatchesConstants, files: map[string]string{"internal/gitops/trailers.go": "package gitops\n\nconst TrailerVerb = \"aiwf-verb\"\n"}},
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
