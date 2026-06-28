package policies_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// gitleaks_enforcement_test.go — G-0292 chokepoint pins.
//
// "Make the gitleaks gate real" (G-0292): the secret / path-leak scan must
// be enforced operator-independently (CI) as well as locally (pre-push),
// with the pre-enforcement historical findings accepted by fingerprint.
// These tests pin that wiring so it cannot silently rot back into the
// decorative state that let 67 path leaks accumulate in history undetected:
//
//   - the CI workflow exists and runs `gitleaks git --config=.gitleaks.toml`
//     over full history — the operator-independent chokepoint;
//   - .gitleaksignore exists and lists the accepted historical fingerprints;
//   - the devcontainer installs gitleaks (so the local pre-push hook fires);
//   - the pinned gitleaks version agrees between CI and the devcontainer.

func gitleaksFile(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRootForHook(t), rel))
	if err != nil {
		t.Fatalf("reading %s: %v", rel, err)
	}
	return string(b)
}

func TestGitleaksEnforcement_CIWorkflow(t *testing.T) {
	t.Parallel()
	wf := gitleaksFile(t, ".github/workflows/gitleaks.yml")
	// The chokepoint: CI runs gitleaks against the repo config, UNDEFANGED.
	// A trailing `|| true` / `--exit-code 0` / a `continue-on-error` / a
	// disabling `if: false` would leave the command substring intact while
	// the job stops failing on a finding — the "silently decorative" mode
	// G-0292 exists to prevent — so assert the run line ends right at
	// --no-banner. NOTE: exact-match pin; adding a gitleaks flag (e.g.
	// --redact) means updating this regex.
	runLine := regexp.MustCompile(`(?m)^\s*run: gitleaks git --config=\.gitleaks\.toml --no-banner\s*$`)
	if !runLine.MatchString(wf) {
		t.Error("gitleaks.yml must run exactly `gitleaks git --config=.gitleaks.toml --no-banner` (undefanged) as the secret-scan chokepoint")
	}
	if strings.Contains(wf, "--exit-code") || strings.Contains(wf, "continue-on-error") || strings.Contains(wf, "if: false") {
		t.Error("gitleaks.yml must not defang the scan (no --exit-code / continue-on-error / if: false)")
	}
	// Full history, else `gitleaks git` sees only the shallow default clone
	// and misses commits.
	if !strings.Contains(wf, "fetch-depth: 0") {
		t.Error("gitleaks.yml checkout must use fetch-depth: 0 so `gitleaks git` scans full history")
	}
	// Must trigger on every push (any branch) AND every PR, UNFILTERED: a
	// secret is exposed at push-to-origin on a public repo regardless of
	// merge/PR, so a `paths:` or `branches:` filter would silently narrow
	// coverage (the seam the reviewers flagged). Forbid both.
	if !strings.Contains(wf, "pull_request:") {
		t.Error("gitleaks.yml must trigger on pull_request")
	}
	if !strings.Contains(wf, "push:") {
		t.Error("gitleaks.yml must trigger on push")
	}
	if strings.Contains(wf, "paths:") || strings.Contains(wf, "branches:") {
		t.Error("gitleaks.yml must not filter triggers by paths/branches — every push and PR scans every file")
	}
}

func TestGitleaksEnforcement_RulesetIntact(t *testing.T) {
	t.Parallel()
	cfg := gitleaksFile(t, ".gitleaks.toml")
	// The gate scans with --config=.gitleaks.toml; gutting the path-leak
	// rules would silently weaken it while CI stays green. Pin the rule ids
	// (.gitleaks.toml is the single source of scan rules).
	for _, id := range []string{
		"path-leak-darwin-home",
		"path-leak-linux-home",
		"path-leak-windows-userprofile",
	} {
		if !strings.Contains(cfg, id) {
			t.Errorf(".gitleaks.toml must retain the %q rule (the gate scans with this config)", id)
		}
	}
}

func TestGitleaksEnforcement_Gitleaksignore(t *testing.T) {
	t.Parallel()
	body := gitleaksFile(t, ".gitleaksignore")
	// Each non-comment line must be a gitleaks git fingerprint
	// (<40-hex-commit>:<file>:<rule>:<line>), NOT a pattern / path allowlist
	// — which would broaden suppression to FUTURE leaks. Shape-checking keeps
	// the file from silently degrading into something over-broad.
	fp := regexp.MustCompile(`^[0-9a-f]{40}:.+:[a-z0-9-]+:\d+$`)
	fingerprints := 0
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !fp.MatchString(line) {
			t.Errorf(".gitleaksignore line is not a commit:file:rule:line fingerprint: %q", line)
		}
		fingerprints++
	}
	if fingerprints == 0 {
		t.Error(".gitleaksignore must list the accepted historical fingerprints; found none")
	}
}

func TestGitleaksEnforcement_DevcontainerInstallsGitleaks(t *testing.T) {
	t.Parallel()
	init := gitleaksFile(t, ".devcontainer/init.sh")
	if !strings.Contains(init, "github.com/zricethezav/gitleaks/v8@") {
		t.Error(".devcontainer/init.sh must install gitleaks so the local pre-push hook actually fires")
	}
}

func TestGitleaksEnforcement_PinnedVersionConsistent(t *testing.T) {
	t.Parallel()
	atRe := regexp.MustCompile(`gitleaks/v8@(v8\.\d+\.\d+)`)
	devRe := regexp.MustCompile(`GITLEAKS_VERSION="(v8\.\d+\.\d+)"`)
	ci := atRe.FindStringSubmatch(gitleaksFile(t, ".github/workflows/gitleaks.yml"))
	dev := devRe.FindStringSubmatch(gitleaksFile(t, ".devcontainer/init.sh"))
	hint := atRe.FindStringSubmatch(gitleaksFile(t, "scripts/git-hooks/pre-push"))
	if ci == nil {
		t.Fatal("no pinned gitleaks version (gitleaks/v8@vX.Y.Z) in .github/workflows/gitleaks.yml")
	}
	if dev == nil {
		t.Fatal(`no pinned GITLEAKS_VERSION="vX.Y.Z" in .devcontainer/init.sh`)
	}
	if hint == nil {
		t.Fatal("no pinned gitleaks version (gitleaks/v8@vX.Y.Z) in scripts/git-hooks/pre-push install hint")
	}
	// CI, devcontainer, and the pre-push install hint must all agree so a
	// version bump can't leave one site stale.
	if ci[1] != dev[1] || ci[1] != hint[1] {
		t.Errorf("pinned gitleaks version must agree: gitleaks.yml=%s, init.sh=%s, pre-push hint=%s", ci[1], dev[1], hint[1])
	}
}
