package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const statuslineRelPath = ".claude/statusline.sh"

// loadStatusline reads the canonical aiwf-aware Claude Code
// statusline script from the repo root and returns its content.
// The script is the test target for M-0153's three content-assertion
// ACs and ships verbatim in the consumer's `.claude/statusline.sh`
// once M-0155's `--statusline` scaffold lands.
func loadStatusline(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, statuslineRelPath))
	if err != nil {
		t.Fatalf("reading %s: %v", statuslineRelPath, err)
	}
	return string(data)
}

// TestStatusline_M0153_AC1_TranscriptWalkPortable asserts M-0153/AC-1:
// the transcript-walk reader uses BSD/macOS `tail -r` as the primary
// command with GNU `tac` as the fallback, so the token segment renders
// correctly on both platforms. macOS lacks `tac`; a bare `tac` invocation
// silently fails to a zero-token read on Darwin.
//
// Two anchored assertions per CLAUDE.md's "substring assertions are not
// structural assertions" rule:
//
//   - Presence: the exact fallback chain
//     `tail -r "$transcript" ... || tac "$transcript"` appears on a single
//     line (structural position, not just two substrings co-occurring).
//   - Absence: no `$(tac "$transcript"` — `tac` may only appear in a
//     fallback position, never as the command-substitution entry point.
//     This is the baseline form the fix removes; a later reflow that
//     drops the `tail -r` half would reintroduce the bug and is caught
//     by this absence rule.
func TestStatusline_M0153_AC1_TranscriptWalkPortable(t *testing.T) {
	t.Parallel()
	body := loadStatusline(t)

	presence := regexp.MustCompile(`tail -r\s+"\$transcript"[^|]*\|\|\s*tac\s+"\$transcript"`)
	if !presence.MatchString(body) {
		t.Errorf("AC-1: statusline.sh must read the transcript via a `tail -r \"$transcript\" ... || tac \"$transcript\"` fallback chain (BSD/macOS first, GNU second); robust form not found")
	}

	bareTac := regexp.MustCompile(`\$\(\s*tac\s+"\$transcript"`)
	if bareTac.MatchString(body) {
		t.Errorf("AC-1: statusline.sh must not invoke `tac` as the command-substitution entry point — `tac` may only appear in a `|| tac` fallback position (the bare form silently produces zero tokens on macOS, which lacks `tac`)")
	}
}

// TestStatusline_M0153_AC2_AheadBehindParseRobust asserts M-0153/AC-2:
// the git ahead/behind sync parse uses `read -r ahead behind <<<"$counts"`,
// which splits on default IFS (space *or* tab) and survives editor /
// copy-paste / patch-tool reflow that converts the source's literal tab to
// spaces. The baseline `${counts%%<TAB>*}` / `${counts##*<TAB>}` parameter
// expansions break the instant the embedded tab is reflowed to spaces —
// the sync indicator silently drops out of the rendered statusline.
//
// Two anchored assertions per CLAUDE.md's "substring assertions are not
// structural assertions" rule:
//
//   - Presence: the exact form `read -r ahead behind <<<"$counts"` appears
//     in the sync block (structural — names both variables in order and
//     pulls from `$counts`, the var the rev-list count populates).
//   - Absence: neither `${counts%%<TAB>*}` nor `${counts##*<TAB>}`
//     literal-tab parameter expansions remain. The regex uses `\t` to
//     match a real tab character — a reflow that converted the tab to
//     spaces would mutate the *baseline* form into one that produces
//     wrong output silently, but the regex below would no longer match
//     either, so a reintroduction in the reflowed form gets caught by the
//     `<<<"$counts"` presence rule remaining required.
func TestStatusline_M0153_AC2_AheadBehindParseRobust(t *testing.T) {
	t.Parallel()
	body := loadStatusline(t)

	presence := regexp.MustCompile(`read -r ahead behind\s*<<<\s*"\$counts"`)
	if !presence.MatchString(body) {
		t.Errorf("AC-2: statusline.sh must parse ahead/behind via `read -r ahead behind <<<\"$counts\"` (default-IFS split survives editor tab→space reflow); robust form not found")
	}

	literalTabAhead := regexp.MustCompile(`\$\{counts%%\t\*\}`)
	if literalTabAhead.MatchString(body) {
		t.Errorf("AC-2: statusline.sh must not use `${counts%%%%<TAB>*}` parameter expansion to extract `ahead` — the literal embedded tab is silently destroyed by any editor/copy-paste reflow that retabs the source")
	}
	literalTabBehind := regexp.MustCompile(`\$\{counts##\*\t\}`)
	if literalTabBehind.MatchString(body) {
		t.Errorf("AC-2: statusline.sh must not use `${counts##*<TAB>}` parameter expansion to extract `behind` — same retab-fragility as the ahead variant")
	}
}

// TestStatusline_M0153_AC3_GitIndexLockHardened asserts M-0153/AC-3:
// the script exports `GIT_OPTIONAL_LOCKS=0` before its first git
// invocation, so no read-only git call in the render path takes
// `.git/index.lock`. The export removes two failure modes a statusline
// renders into a busy repo: contention with concurrent real-write
// commands (`Unable to create '.git/index.lock': File exists`) and
// stale-lock orphaning when a SIGKILLed render dies mid-rename of the
// opportunistic index-refresh write. The export is environment-only;
// output is byte-identical to the unhardened script.
//
// Two assertions:
//
//   - Presence: an `export GIT_OPTIONAL_LOCKS=0` line exists in the
//     script (matched line-by-line, anchored at line-start so a
//     mention inside a comment or string body does not count).
//   - Position: the export line appears strictly *before* the first
//     non-comment line containing a `git ` invocation. Order matters —
//     a child process inherits env from the parent, so the export has
//     to land before the calls it should cover. A comment-line mention
//     of `git` (e.g. the design-note "`git ls-files`") is skipped.
//
// statuslineExportAndFirstGitIdx scans a script body line-by-line and
// returns the zero-based line index of (a) the first `export
// GIT_OPTIONAL_LOCKS=0` statement and (b) the first non-comment line
// containing a `git ` invocation. -1 means "not found".
//
// Factored out of TestStatusline_M0153_AC3_GitIndexLockHardened so the
// position-check failure path — export present but landing *after* the
// first git call — is exercisable from a synthetic body. The live
// statusline currently passes all three branches, so without this
// helper the failure-side branch would never be tested (and CLAUDE.md's
// branch-coverage hard rule would not be satisfied).
func statuslineExportAndFirstGitIdx(body string) (exportIdx, firstGitIdx int) {
	exportLine := regexp.MustCompile(`^\s*export\s+GIT_OPTIONAL_LOCKS=0\b`)
	gitCall := regexp.MustCompile(`\bgit\s+[a-z]`)
	commentLine := regexp.MustCompile(`^\s*#`)
	exportIdx, firstGitIdx = -1, -1
	for i, line := range strings.Split(body, "\n") {
		if exportIdx == -1 && exportLine.MatchString(line) {
			exportIdx = i
		}
		if firstGitIdx == -1 && !commentLine.MatchString(line) && gitCall.MatchString(line) {
			firstGitIdx = i
		}
	}
	return exportIdx, firstGitIdx
}

func TestStatusline_M0153_AC3_GitIndexLockHardened(t *testing.T) {
	t.Parallel()
	body := loadStatusline(t)
	exportIdx, firstGitIdx := statuslineExportAndFirstGitIdx(body)

	if exportIdx == -1 {
		t.Errorf("AC-3: statusline.sh must contain a line `export GIT_OPTIONAL_LOCKS=0` near the top of the script — robust form not found")
	}
	if firstGitIdx == -1 {
		t.Fatalf("AC-3: statusline.sh contains no non-comment `git ` invocation — unexpected (the script's CI / sync / repo / worktree segments all call git)")
	}
	if exportIdx >= 0 && exportIdx >= firstGitIdx {
		t.Errorf("AC-3: statusline.sh must export GIT_OPTIONAL_LOCKS=0 *before* its first git invocation; got export at line %d but first git call at line %d (a child git process inherits env only from a parent that already set it)", exportIdx+1, firstGitIdx+1)
	}
}

// TestStatuslineExportAndFirstGitIdx_BranchCoverage walks every reachable
// branch in statuslineExportAndFirstGitIdx — the helper's correctness
// against five distinguishable shapes the live statusline cannot
// simultaneously hold:
//
//   - both present, export first (canonical green state)
//   - both present, git first (the failure mode the position-check
//     branch in AC-3 exists to catch)
//   - export missing, git present (the failure mode the presence-check
//     branch in AC-3 catches; covered by RED→GREEN on the live file)
//   - git missing, export present (defensive — AC-3 t.Fatal's on this)
//   - git present only as a comment-line mention (skip-comment rule)
//
// Per CLAUDE.md's branch-coverage hard rule — every reachable branch in
// the helper must have an explicit test exercising it.
func TestStatuslineExportAndFirstGitIdx_BranchCoverage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                     string
		body                     string
		wantExport, wantFirstGit int
	}{
		{
			name:       "export before first git (canonical)",
			body:       "set -u\nexport GIT_OPTIONAL_LOCKS=0\ngit rev-parse\n",
			wantExport: 1, wantFirstGit: 2,
		},
		{
			name:       "export after first git (the AC-3 position-check failure)",
			body:       "set -u\ngit rev-parse\nexport GIT_OPTIONAL_LOCKS=0\n",
			wantExport: 2, wantFirstGit: 1,
		},
		{
			name:       "export missing, git present",
			body:       "set -u\ngit rev-parse\n",
			wantExport: -1, wantFirstGit: 1,
		},
		{
			name:       "export present, git missing",
			body:       "set -u\nexport GIT_OPTIONAL_LOCKS=0\necho done\n",
			wantExport: 1, wantFirstGit: -1,
		},
		{
			name:       "git in comment only is skipped",
			body:       "# `git ls-files` — design note\nexport GIT_OPTIONAL_LOCKS=0\necho done\n",
			wantExport: 1, wantFirstGit: -1,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotExport, gotFirstGit := statuslineExportAndFirstGitIdx(tc.body)
			if gotExport != tc.wantExport || gotFirstGit != tc.wantFirstGit {
				t.Errorf("got (export=%d, firstGit=%d); want (%d, %d)", gotExport, gotFirstGit, tc.wantExport, tc.wantFirstGit)
			}
		})
	}
}
