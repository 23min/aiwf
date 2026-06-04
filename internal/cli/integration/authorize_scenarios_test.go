package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/gitops"
)

// authorize_scenarios_test.go — M-0161/AC-1 (G-0200).
//
// Real-git E2E for the trunk-name configurability arc: the verb-layer
// carve-out at internal/verb/authorize.go (the `currentIsRitualContext
// := opts.CurrentBranch == "main"` site) must honor aiwf.yaml.allocate.trunk's
// configured trunk short-name rather than the literal "main". Drives
// the worktree-built binary as subprocess against fresh temp repos
// with explicit allocate.trunk + matching local-branch shapes.
//
// Pre-AC-1 production state: the carve-out compares against the
// literal `"main"`. The 3 non-main scenarios refuse in RED via the
// existing "must be ritual or main" preflight; the main scenario
// passes (matches the literal). Post-AC-1 GREEN replaces the literal
// with cfg.TrunkBranchShortName() and all 4 scenarios accept.
//
// Sabotage discipline: reverting the helper-derivation at the
// authorize.go call site (re-pinning to "main") makes the 3 non-main
// scenarios refuse again — the integration test discriminates the new
// code path, not the old.
//
// Per AC-1 §"Auxiliary unit test" — the per-Config unit table at
// internal/config/config_test.go::TestTrunkBranchShortName is
// diagnostic; THIS file's scenarios are the load-bearing E2E.

// TestAuthorize_AC1_NonMainTrunkNames_Accept exercises 4 trunk-name
// shapes per the AC-1 mechanical assertion table:
//
//   - Default: refs/remotes/origin/main → local "main"
//   - GitHub-classic: refs/remotes/origin/master → local "master"
//   - Operator-chosen: refs/remotes/origin/dev → local "dev"
//   - Bare-heads: refs/heads/trunk → local "trunk"
//
// Each scenario bootstraps a fresh temp repo with the named aiwf.yaml +
// matching local trunk branch, runs `aiwf authorize <epic-id> --to
// ai/alice --branch epic/E-0001-engine` against the worktree-built
// binary, and asserts:
//   - exit 0
//   - HEAD trailer aiwf-branch: epic/E-0001-engine
//   - HEAD trailer aiwf-verb: authorize
//   - HEAD trailer aiwf-to: ai/alice
//   - HEAD remains on the configured trunk (carve-out does not change
//     checkout — step 5 of aiwfx-start-epic does that).
//
// The "default-main" sub-case is the baseline and the existing
// TestRunAuthorize_AITarget_MainPlusRitualFutureBranch_Accepts
// already pins it; this table-driven test re-pins it alongside the
// 3 non-main shapes to keep the per-scenario assertions uniform and
// catch any regression where the helper derivation diverges from the
// existing carve-out's behavior on the default path.
func TestAuthorize_AC1_NonMainTrunkNames_Accept(t *testing.T) {
	t.Parallel()
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)

	// Each case configures aiwf.yaml.allocate.trunk to a refs/heads/<X>
	// shape and creates that local branch via `git init -b <X>`. This
	// keeps the test setup self-contained (no upstream remote required)
	// while exercising the helper's last-path-segment derivation for
	// 4 distinct trunk-name shapes.
	//
	// The orthogonal refs/remotes/<remote>/<name> tracking-ref shape is
	// covered exhaustively by the auxiliary unit table at
	// internal/config/config_test.go::TestTrunkBranchShortName (10 rows
	// including alternate-remote-upstream). The seam this E2E pins is
	// the verb-layer call site reading cfg.TrunkBranchShortName() and
	// matching it against opts.CurrentBranch — which is the same
	// regardless of whether the trunk ref is heads-shaped or
	// tracking-shaped.
	cases := []struct {
		name string
		// trunkRef goes into aiwf.yaml's allocate.trunk.
		trunkRef string
		// localBranch is the operator's local trunk branch (matches
		// the trunkRef's last path segment).
		localBranch string
	}{
		{
			name:        "main",
			trunkRef:    "refs/heads/main",
			localBranch: "main",
		},
		{
			name:        "github-classic-master",
			trunkRef:    "refs/heads/master",
			localBranch: "master",
		},
		{
			name:        "operator-chosen-dev",
			trunkRef:    "refs/heads/dev",
			localBranch: "dev",
		},
		{
			name:        "operator-chosen-trunk",
			trunkRef:    "refs/heads/trunk",
			localBranch: "trunk",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			// Initialize git with the configured trunk as the
			// initial branch so the operator's CurrentBranch is
			// deterministically `tc.localBranch` regardless of
			// the host's init.defaultBranch.
			if out, err := testutil.RunGit(root, "init", "-q", "-b", tc.localBranch); err != nil {
				t.Fatalf("git init -b %s: %v\n%s", tc.localBranch, err, out)
			}
			for _, args := range [][]string{
				{"config", "user.email", "peter@example.com"},
				{"config", "user.name", "Peter Test"},
			} {
				if out, err := testutil.RunGit(root, args...); err != nil {
					t.Fatalf("git %v: %v\n%s", args, err, out)
				}
			}
			if out, err := testutil.RunBin(t, root, binDir, nil, "init"); err != nil {
				t.Fatalf("aiwf init: %v\n%s", err, out)
			}

			// `aiwf init` does NOT commit — it leaves aiwf.yaml staged
			// in the working tree. The configured trunk ref check is
			// hard-fail at tree-load time, so refs/heads/<X> must
			// already resolve BEFORE we set allocate.trunk. Make a
			// baseline commit to birth refs/heads/<localBranch>.
			if out, err := testutil.RunGit(root, "add", "-A"); err != nil {
				t.Fatalf("git add: %v\n%s", err, out)
			}
			if out, err := testutil.RunGit(root, "commit", "-q", "-m", "aiwf init"); err != nil {
				t.Fatalf("git commit baseline: %v\n%s", err, out)
			}

			// Configure aiwf.yaml.allocate.trunk to the named
			// ref. Per the AC-1 contract: the verb-layer carve-out
			// derives its trunk short-name from this value via
			// cfg.TrunkBranchShortName(), not from the literal
			// "main". The amend lands as the working-tree state
			// the next verb invocation reads — no commit needed
			// (the aiwf.yaml change is config-only, not
			// entity-tree state).
			cfgPath := filepath.Join(root, "aiwf.yaml")
			cfgBytes, readErr := os.ReadFile(cfgPath)
			if readErr != nil {
				t.Fatalf("read aiwf.yaml: %v", readErr)
			}
			amended := append(cfgBytes, []byte("\nallocate:\n  trunk: "+tc.trunkRef+"\n")...)
			if writeErr := os.WriteFile(cfgPath, amended, 0o644); writeErr != nil {
				t.Fatalf("rewrite aiwf.yaml: %v", writeErr)
			}

			if out, err := testutil.RunBin(t, root, binDir, nil,
				"add", "epic", "--title", "Engine"); err != nil {
				t.Fatalf("aiwf add: %v\n%s", err, out)
			}
			if out, err := testutil.RunBin(t, root, binDir, nil,
				"promote", "E-0001", "active"); err != nil {
				t.Fatalf("aiwf promote: %v\n%s", err, out)
			}

			// The AC-1 invocation: from the configured trunk,
			// name a future epic branch (step-4 pattern of
			// aiwfx-start-epic). The carve-out must accept this
			// regardless of trunk-name shape.
			// No --force, no --reason: AC-1's contract is that the
			// carve-out accepts cleanly without sovereign override.
			// The negative-trailer assertion below pins this — if a
			// future regression made the carve-out fail and an
			// implicit force-override kicked in, the silent
			// `aiwf-force:` trailer would catch it.
			if out, err := testutil.RunBin(t, root, binDir, nil,
				"authorize", "E-0001",
				"--to", "ai/alice",
				"--branch", "epic/E-0001-engine",
			); err != nil {
				t.Fatalf("aiwf authorize from trunk %q (allocate.trunk=%s): %v\n%s",
					tc.localBranch, tc.trunkRef, err, out)
			}

			// Trailer pins: same shape as the existing main-trunk
			// scenario — aiwf-verb, aiwf-to, aiwf-branch are all
			// required. Plus a negative assertion that NO
			// aiwf-force: trailer lands — the carve-out path is
			// the load-bearing accept, not a silent force-override.
			tr, err := gitops.HeadTrailers(context.Background(), root)
			if err != nil {
				t.Fatal(err)
			}
			hasTrailer(t, tr, "aiwf-verb", "authorize")
			hasTrailer(t, tr, "aiwf-to", "ai/alice")
			hasTrailer(t, tr, "aiwf-branch", "epic/E-0001-engine")
			noTrailer(t, tr, "aiwf-force")

			// HEAD stays on the configured trunk (carve-out does
			// not change checkout). Pins the "step 5 cuts the
			// branch, not step 4" invariant for each trunk shape.
			if out, err := testutil.RunGit(root, "symbolic-ref", "--short", "HEAD"); err != nil {
				t.Fatalf("git symbolic-ref --short HEAD: %v\n%s", err, out)
			} else if got := strings.TrimSpace(out); got != tc.localBranch {
				t.Errorf("HEAD = %q, want %q (carve-out should not move off trunk)",
					got, tc.localBranch)
			}
		})
	}
}
