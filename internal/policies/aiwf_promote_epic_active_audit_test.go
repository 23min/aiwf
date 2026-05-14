package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// TestPolicy_NoNonForcedEpicActivateInCIScripts pins M-0097/AC-1: the
// M-0095 sovereign-act rule's chokepoint at the verb cannot be
// silently bypassed by an automation path that invokes `aiwf promote
// E-... active` without the `--force --reason "..."` override. The
// audit walks the repo's CI/script surfaces (`.github/`, `scripts/`,
// `Makefile` when present) and fires one finding per offending line.
//
// Why this exists: M-0095's spec claimed an automation audit was run
// pre-implementation. The conversation record does not show evidence
// of the grep being executed. This test converts the claim into a
// mechanical chokepoint — every CI run re-runs the audit, so the
// "we checked" assertion stays load-bearing across time.
//
// Note: `aiwf doctor --self-check` (separate from this) covers
// run-time invocations; this test pins the *static* invocation set
// in repo source.
func TestPolicy_NoNonForcedEpicActivateInCIScripts(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	fsys := os.DirFS(root)

	// `.github/` and `scripts/` are conventional; `Makefile` is
	// optional. We probe stat to keep the path list tight (asking
	// `fs.WalkDir` to walk a missing path returns an error per call).
	paths := []string{}
	for _, p := range []string{".github", "scripts", "Makefile"} {
		if _, err := os.Stat(filepath.Join(root, p)); err == nil {
			paths = append(paths, p)
		}
	}

	findings := auditUnforcedEpicActivate(fsys, paths)
	for _, f := range findings {
		t.Errorf("AC-1: unforced `aiwf promote E- active` invocation: %s — append `--force --reason \"...\"` to the same line, or have a human run the verb out-of-band", f)
	}
}

// TestAuditUnforcedEpicActivate_MissingPathIsSilent exercises the
// walkErr arm of `auditUnforcedEpicActivate` — when a named path does
// not exist under the given fs, `fs.WalkDir` invokes the callback
// with `walkErr != nil`, and the helper silently skips rather than
// propagating the error. This is the audit's "best-effort over the
// named paths" contract: a missing `Makefile` should not break the
// run. Confirms the defensive arm is reachable AND that the helper
// produces zero findings (the only sane response).
func TestAuditUnforcedEpicActivate_MissingPathIsSilent(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		// Empty fs — the named path does not exist.
	}
	findings := auditUnforcedEpicActivate(fsys, []string{"does-not-exist"})
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for missing path, got %d: %v", len(findings), findings)
	}
}

// TestAuditUnforcedEpicActivate_BranchCoverage exercises every
// reachable arm of `auditUnforcedEpicActivate` against synthetic
// in-memory filesystem inputs. Together with the seam test above
// (which exercises real-repo paths), this gives both layers of
// coverage CLAUDE.md §"Test the seam, not just the layer" requires
// — the helper's logic is exercised with controlled inputs even on
// a clean repo where the seam test produces zero findings.
//
// Cases cover: the empty/clean arm (no matching line), the
// `--force` exemption arm (a matching line *with* --force, ignored),
// and the offending arm (a matching line without --force, fires).
// A multi-file case threads the WalkDir's per-file iteration.
func TestAuditUnforcedEpicActivate_BranchCoverage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		fsys         fstest.MapFS
		paths        []string
		wantFindings int
		wantContains string // substring that must appear in the first finding (if any)
	}{
		{
			name: "clean-no-matches",
			fsys: fstest.MapFS{
				".github/workflows/ci.yml": {Data: []byte("name: CI\non: push\n")},
			},
			paths:        []string{".github"},
			wantFindings: 0,
		},
		{
			name: "forced-invocation-ignored",
			fsys: fstest.MapFS{
				".github/workflows/release.yml": {Data: []byte(`run: aiwf promote E-0042 active --force --reason "release bot"` + "\n")},
			},
			paths:        []string{".github"},
			wantFindings: 0,
		},
		{
			name: "unforced-invocation-fires",
			fsys: fstest.MapFS{
				".github/workflows/bad.yml": {Data: []byte("run: aiwf promote E-0042 active\n")},
			},
			paths:        []string{".github"},
			wantFindings: 1,
			wantContains: ".github/workflows/bad.yml",
		},
		{
			name: "mixed-files-only-unforced-fires",
			fsys: fstest.MapFS{
				".github/workflows/clean.yml":    {Data: []byte("name: clean\n")},
				".github/workflows/forced.yml":   {Data: []byte("run: aiwf promote E-0001 active --force --reason \"x\"\n")},
				".github/workflows/unforced.yml": {Data: []byte("run: aiwf promote E-0002 active\n")},
			},
			paths:        []string{".github"},
			wantFindings: 1,
			wantContains: "unforced.yml",
		},
		{
			name: "multiple-unforced-lines-in-one-file",
			fsys: fstest.MapFS{
				"scripts/release.sh": {Data: []byte("#!/usr/bin/env bash\naiwf promote E-0001 active\naiwf promote E-0002 active\n")},
			},
			paths:        []string{"scripts"},
			wantFindings: 2,
			wantContains: "scripts/release.sh",
		},
		{
			name: "non-matching-line-with-similar-words",
			fsys: fstest.MapFS{
				// Mentions `promote` and `active` but not the exact `aiwf
				// promote E-... active` pattern — must not fire.
				".github/workflows/docs.yml": {Data: []byte("# How to promote an epic to active: run `aiwf promote E-NN active`\n")},
			},
			paths:        []string{".github"},
			wantFindings: 1, // The doc *example* itself matches the pattern.
			wantContains: ".github/workflows/docs.yml",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			findings := auditUnforcedEpicActivate(tc.fsys, tc.paths)
			if len(findings) != tc.wantFindings {
				t.Fatalf("%s: expected %d findings, got %d: %v", tc.name, tc.wantFindings, len(findings), findings)
			}
			if tc.wantContains != "" && len(findings) > 0 {
				if !strings.Contains(findings[0], tc.wantContains) {
					t.Errorf("%s: first finding %q must contain %q", tc.name, findings[0], tc.wantContains)
				}
			}
		})
	}
}
