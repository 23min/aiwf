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
			// M-0162/AC-3 cell pin: one cell per non-main trunk
			// shape exercised by this matrix. Cell IDs are
			// stable across test renames because they key
			// off the matrix-row name.
			pinCell("branch-cell-m0161-ac1-"+tc.name, t.Name())

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
			cfgBytes = append(cfgBytes, []byte("\nallocate:\n  trunk: "+tc.trunkRef+"\n")...)
			if writeErr := os.WriteFile(cfgPath, cfgBytes, 0o644); writeErr != nil {
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
			if out, err := testutil.RunBin(
				t, root, binDir, nil,
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

// TestAuthorize_AC2_RungPair_Matrix — M-0161/AC-2 (G-0201).
//
// The 16-cell (CurrentBranch rung × --branch rung) matrix per the
// AC-2 body's table. 4 legal pairs accept; 12 illegal pairs refuse.
// Plus 1 sovereign-override scenario exercising --force --reason on
// an illegal pair.
//
// Pre-AC-2 production state: the verb-layer authorize carve-out
// accepts any (ritual or trunk current, ritual target) when
// BranchExists=false (M-0104/AC-4 + M-0105/AC-6's loose union), and
// any --branch value when BranchExists=true (skipping the carve-out
// entirely). AC-2 tightens this with `branchparse.LegalRungPair`
// applied to (RungOf(current, trunk), RungOf(target, trunk))
// regardless of BranchExists — so cross-rung typos, up-the-tree
// shapes, and `--branch <trunk>` all refuse, while the 4 legitimate
// ritual flows accept.
//
// RED-state discrimination:
//   - 4 legal pairs PASS RED (existing carve-out accepts).
//   - 4 (X, trunk) illegal pairs FAIL RED (BranchExists=true bypass
//     → verb accepts → expected refuse).
//   - 8 illegal ritual-target pairs FAIL RED (existing loose carve-out
//     accepts → expected refuse).
//   - 1 sovereign-override PASSES RED (--force bypasses preflight,
//     same as GREEN).
//
// Net: 12 fail, 5 pass.
//
// Per AC-2 §"Single rung-pair check refuses every illegal cell" —
// the predicate applies regardless of BranchExists. The 4 (X, trunk)
// rows where --branch IS the configured trunk's local branch refuse
// via the same rung-pair check that catches the 8 ritual-target
// illegal cells; no separate upstream layer. Per the body, the
// single-check semantic is what AC-2 commits to (the earlier
// "two refusal layers" framing was wrong and dropped in commit
// 2fd84dd4).
func TestAuthorize_AC2_RungPair_Matrix(t *testing.T) {
	t.Parallel()
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)

	// Branch names per rung. The "current" set is what the operator
	// is on; the "target" set is the --branch value (deliberately
	// different from "current" so same-rung typo cells are
	// distinguishable from same-name cells).
	currentBranchByRung := map[string]string{
		"trunk":     "main",
		"epic":      "epic/E-0001-current",
		"milestone": "milestone/M-0007-current",
		"patch":     "patch/g-0099-current",
	}
	targetBranchByRung := map[string]string{
		"trunk":     "main",
		"epic":      "epic/E-0002-target",
		"milestone": "milestone/M-0008-target",
		"patch":     "patch/g-0100-target",
	}
	// The 4 legal pairs per AC-2's body matrix. Every other (rung,
	// rung) pair is illegal.
	legalSet := map[[2]string]bool{
		{"trunk", "epic"}:      true,
		{"epic", "milestone"}:  true,
		{"milestone", "patch"}: true,
		{"epic", "patch"}:      true,
	}
	rungs := []string{"trunk", "epic", "milestone", "patch"}

	// Build the 16-cell scenario list.
	type cell struct {
		currentRung string
		targetRung  string
	}
	var cells []cell
	for _, c := range rungs {
		for _, ta := range rungs {
			cells = append(cells, cell{currentRung: c, targetRung: ta})
		}
	}

	for _, c := range cells {
		c := c
		legal := legalSet[[2]string{c.currentRung, c.targetRung}]
		name := c.currentRung + "_to_" + c.targetRung
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// M-0162/AC-3 cell pin: one cell per rung-pair
			// matrix row. 16 cells from the 4×4 rung-pair
			// grid (current × target) — see runAC2RungPairCell.
			pinCell("branch-cell-m0161-ac2-"+name, t.Name())
			runAC2RungPairCell(
				t, bin, binDir,
				currentBranchByRung[c.currentRung],
				targetBranchByRung[c.targetRung],
				legal,
			)
		})
	}

	// Sovereign-override scenario: an illegal pair (epic→epic
	// cross-epic-typo) + --force --reason "cross-epic intentional"
	// should accept regardless of rung-pair predicate. The
	// authorize commit must carry BOTH aiwf-branch: AND aiwf-force:
	// trailers — pinning the override surface for AC-2's gate.
	t.Run("sovereign_override_force_reason_bypasses_rung_check", func(t *testing.T) {
		t.Parallel()
		// M-0162/AC-3 cell pin: the AC-2 sovereign-override
		// scenario. Same name as the test subtest for traceability.
		pinCell("branch-cell-m0161-ac2-sovereign-override", t.Name())
		runAC2OverrideCell(
			t, bin, binDir,
			currentBranchByRung["epic"],
			targetBranchByRung["epic"],
		)
	})
}

// runAC2RungPairCell drives one (currentBranch, targetBranch) cell
// against the worktree-built binary and asserts accept (legal=true)
// or refuse (legal=false). The fixture bootstraps a fresh temp repo,
// sets allocate.trunk to refs/heads/main, makes a baseline commit,
// cuts the current-rung branch (and switches to it), then runs
// `aiwf authorize E-0001 --to ai/alice --branch <target>`. The
// target branch is NOT cut unless it IS the trunk (in which case it
// already exists from `git init -b main`).
func runAC2RungPairCell(t *testing.T, bin, binDir, currentBranch, targetBranch string, legal bool) {
	t.Helper()
	root := setupAC2RungPairFixture(t, bin, binDir, currentBranch)

	args := []string{
		"authorize", "E-0001",
		"--to", "ai/alice",
		"--branch", targetBranch,
	}
	out, err := testutil.RunBin(t, root, binDir, nil, args...)
	if legal {
		if err != nil {
			t.Fatalf("legal pair (current=%q, target=%q) refused but should accept: %v\n%s",
				currentBranch, targetBranch, err, out)
		}
		// Trailer pins: aiwf-branch records the target; no
		// aiwf-force trailer (the carve-out is the load-bearing
		// accept, not a silent override).
		tr, terr := gitops.HeadTrailers(context.Background(), root)
		if terr != nil {
			t.Fatal(terr)
		}
		hasTrailer(t, tr, "aiwf-verb", "authorize")
		hasTrailer(t, tr, "aiwf-to", "ai/alice")
		hasTrailer(t, tr, "aiwf-branch", targetBranch)
		noTrailer(t, tr, "aiwf-force")
		return
	}
	// Illegal: expect non-zero exit.
	if err == nil {
		t.Fatalf("illegal pair (current=%q, target=%q) accepted but should refuse; output:\n%s",
			currentBranch, targetBranch, out)
	}
	// Stderr quality check: the refusal message should name BOTH
	// rungs (the operator can see which pair is rejected) and the
	// sovereign-override path (--force --reason). Substring scoped
	// to the error context, per CLAUDE.md "Substring assertions"
	// — verb-time errors don't carry structured codes today.
	stderr := out
	if !strings.Contains(stderr, "--force") {
		t.Errorf("refusal stderr should name --force override path; got:\n%s", stderr)
	}
}

// runAC2OverrideCell exercises the sovereign-override scenario:
// an illegal rung pair + --force --reason → accepts with
// aiwf-force: trailer recorded.
func runAC2OverrideCell(t *testing.T, bin, binDir, currentBranch, targetBranch string) {
	t.Helper()
	root := setupAC2RungPairFixture(t, bin, binDir, currentBranch)

	args := []string{
		"authorize", "E-0001",
		"--to", "ai/alice",
		"--branch", targetBranch,
		"--force",
		"--reason", "cross-epic intentional",
	}
	out, err := testutil.RunBin(t, root, binDir, nil, args...)
	if err != nil {
		t.Fatalf("sovereign override refused but should accept: %v\n%s", err, out)
	}
	tr, terr := gitops.HeadTrailers(context.Background(), root)
	if terr != nil {
		t.Fatal(terr)
	}
	hasTrailer(t, tr, "aiwf-verb", "authorize")
	hasTrailer(t, tr, "aiwf-to", "ai/alice")
	hasTrailer(t, tr, "aiwf-branch", targetBranch)
	// The load-bearing override-surface pin: the commit carries
	// aiwf-force with the operator's reason.
	hasTrailer(t, tr, "aiwf-force", "cross-epic intentional")
}

// setupAC2RungPairFixture builds the per-scenario temp repo:
//   - git init -b main + git config
//   - aiwf init
//   - baseline commit (births refs/heads/main so allocate.trunk
//     resolves)
//   - amend aiwf.yaml with allocate.trunk: refs/heads/main
//   - aiwf add epic E-0001 Engine + promote E-0001 active
//   - if currentBranch != "main": cut currentBranch + checkout
//
// Returns the repo root path.
func setupAC2RungPairFixture(t *testing.T, bin, binDir, currentBranch string) string {
	t.Helper()
	root := t.TempDir()

	if out, err := testutil.RunGit(root, "init", "-q", "-b", "main"); err != nil {
		t.Fatalf("git init -b main: %v\n%s", err, out)
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
	// Baseline commit to birth refs/heads/main so allocate.trunk
	// resolves (per AC-1 fixture pattern).
	if out, err := testutil.RunGit(root, "add", "-A"); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	if out, err := testutil.RunGit(root, "commit", "-q", "-m", "aiwf init"); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	// Configure allocate.trunk so the verb's TrunkShort resolves
	// to "main".
	cfgPath := filepath.Join(root, "aiwf.yaml")
	cfgBytes, readErr := os.ReadFile(cfgPath)
	if readErr != nil {
		t.Fatalf("read aiwf.yaml: %v", readErr)
	}
	cfgBytes = append(cfgBytes, []byte("\nallocate:\n  trunk: refs/heads/main\n")...)
	if writeErr := os.WriteFile(cfgPath, cfgBytes, 0o644); writeErr != nil {
		t.Fatalf("rewrite aiwf.yaml: %v", writeErr)
	}
	// Create the epic entity + activate (so authorize has a target).
	if out, err := testutil.RunBin(t, root, binDir, nil, "add", "epic", "--title", "Engine"); err != nil {
		t.Fatalf("aiwf add epic: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "promote", "E-0001", "active"); err != nil {
		t.Fatalf("aiwf promote: %v\n%s", err, out)
	}
	// If the operator's current-rung branch isn't main, cut it and
	// check it out.
	if currentBranch != "main" {
		if out, err := testutil.RunGit(root, "checkout", "-b", currentBranch); err != nil {
			t.Fatalf("git checkout -b %s: %v\n%s", currentBranch, err, out)
		}
	}
	return root
}
