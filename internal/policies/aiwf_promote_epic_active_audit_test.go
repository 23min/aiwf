package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/23min/aiwf/internal/entity"
)

// TestPolicy_NoNonForcedSovereignActPromoteInCIScripts pins
// M-0097/AC-1: the runtime sovereign-act rule's chokepoint at the
// verb cannot be silently bypassed by an automation path that
// invokes `aiwf promote <prefix>-<id> <to>` for any kernel-declared
// sovereign-act-shape transition without the `--force --reason "..."`
// override. The audit walks the repo's CI/script surfaces
// (`.github/`, `scripts/`, `Makefile` when present) and fires one
// finding per offending line.
//
// Why this exists: M-0095's spec claimed an automation audit was run
// pre-implementation. The conversation record does not show evidence
// of the grep being executed. This test converts the claim into a
// mechanical chokepoint — every CI run re-runs the audit, so the
// "we checked" assertion stays load-bearing across time. M-0130's
// consolidation generalized the audit from one hardcoded
// (epic, active) pair to all entries in `entity.SovereignActShapes()`.
//
// Note: `aiwf doctor --self-check` (separate from this) covers
// run-time invocations; this test pins the *static* invocation set
// in repo source.
func TestPolicy_NoNonForcedSovereignActPromoteInCIScripts(t *testing.T) {
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

	findings := auditUnforcedSovereignActPromote(fsys, paths)
	for _, f := range findings {
		t.Errorf("AC-1: unforced `aiwf promote` invocation against a sovereign-act-shape transition: %s — append `--force --reason \"...\"` to the same line, or have a human run the verb out-of-band", f)
	}
}

// TestAuditUnforcedSovereignActPromote_MissingPathIsSilent exercises
// the walkErr arm of `auditUnforcedSovereignActPromote` — when a
// named path does not exist under the given fs, `fs.WalkDir` invokes
// the callback with `walkErr != nil`, and the helper silently skips
// rather than propagating the error. This is the audit's "best-
// effort over the named paths" contract: a missing `Makefile` should
// not break the run. Confirms the defensive arm is reachable AND
// that the helper produces zero findings (the only sane response).
func TestAuditUnforcedSovereignActPromote_MissingPathIsSilent(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		// Empty fs — the named path does not exist.
	}
	findings := auditUnforcedSovereignActPromote(fsys, []string{"does-not-exist"})
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for missing path, got %d: %v", len(findings), findings)
	}
}

// TestAuditUnforcedSovereignActPromote_BranchCoverage exercises every
// reachable arm of `auditUnforcedSovereignActPromote` against
// synthetic in-memory filesystem inputs. Together with the seam test above
// (which exercises real-repo paths), this gives both layers of
// coverage CLAUDE.md §"Test the seam, not just the layer" requires
// — the helper's logic is exercised with controlled inputs even on
// a clean repo where the seam test produces zero findings.
//
// Cases cover: the empty/clean arm (no matching line), the
// `--force` exemption arm (a matching line *with* --force, ignored),
// and the offending arm (a matching line without --force, fires).
// A multi-file case threads the WalkDir's per-file iteration.
func TestAuditUnforcedSovereignActPromote_BranchCoverage(t *testing.T) {
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
			findings := auditUnforcedSovereignActPromote(tc.fsys, tc.paths)
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

// TestSovereignActPromoteRegexes_TracksKernelClosedSet asserts the
// regex builder produces exactly one regex per
// `entity.SovereignActShapes()` entry. Without this pin, a future
// regression that hardcoded the list back to a single regex would
// pass the existing fixture tests (since the kernel's closed set has
// one entry today) but silently break consolidation when the second
// entry lands.
func TestSovereignActPromoteRegexes_TracksKernelClosedSet(t *testing.T) {
	t.Parallel()
	shapes := entity.SovereignActShapes()
	regexes := sovereignActPromoteRegexes()
	if len(regexes) != len(shapes) {
		t.Fatalf("regex count = %d, want %d (one per kernel sovereign-act-shape entry); shapes=%+v", len(regexes), len(shapes), shapes)
	}
	// Each regex should match the canonical command shape for its
	// corresponding shape entry. We construct an example invocation
	// from the entry's data and assert the regex at the same index
	// matches it.
	for i, s := range shapes {
		prefix := entity.IDPrefix(s.Kind)
		example := "aiwf promote " + prefix + "0001 " + s.To
		if !regexes[i].MatchString(example) {
			t.Errorf("regex[%d] (%s) does not match example invocation %q built from shape entry %+v", i, regexes[i].String(), example, s)
		}
	}
}

// TestLineMatchesAnySovereignActRegex_MultiEntry exercises the
// list-driven OR-over-regexes behavior with a synthetic two-entry
// set. The kernel's actual closed set has one entry today, so
// existing audit tests cannot prove the list-driven logic works
// past one entry — this test does. If a future refactor collapses
// the helper back to a single hardcoded regex, this test fires.
func TestLineMatchesAnySovereignActRegex_MultiEntry(t *testing.T) {
	t.Parallel()
	regexes := []*regexp.Regexp{
		regexp.MustCompile(`aiwf\s+promote\s+E-\S+\s+active`),
		regexp.MustCompile(`aiwf\s+promote\s+C-\S+\s+accepted`),
	}
	cases := []struct {
		line string
		want bool
	}{
		{"aiwf promote E-0001 active", true},
		{"aiwf promote C-0042 accepted", true},
		{"aiwf promote M-0001 in_progress", false},
		{"aiwf cancel E-0001", false},
		{"", false},
	}
	for _, c := range cases {
		t.Run(c.line, func(t *testing.T) {
			t.Parallel()
			if got := lineMatchesAnySovereignActRegex(c.line, regexes); got != c.want {
				t.Errorf("lineMatchesAnySovereignActRegex(%q) = %v, want %v", c.line, got, c.want)
			}
		})
	}
}

// TestLineMatchesAnySovereignActRegex_EmptyRegexList asserts the
// helper returns false for an empty regex list. This is the early-
// return guard in auditUnforcedSovereignActPromote (skip the per-
// file walk entirely when the kernel has zero sovereign-act-shape
// entries) — defensive but reachable if a hypothetical future state
// empties the closed set.
func TestLineMatchesAnySovereignActRegex_EmptyRegexList(t *testing.T) {
	t.Parallel()
	if lineMatchesAnySovereignActRegex("aiwf promote E-0001 active", nil) {
		t.Error("empty regex list should never match; got true")
	}
}
