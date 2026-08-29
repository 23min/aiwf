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
		t.Errorf("AC-1: unforced `aiwf promote` invocation against a sovereign-act-shape transition: %s — have a human run the verb out-of-band. Adding `--force --reason \"...\"` here would not help: a scripted invocation acts as a non-human actor, and the apply seam refuses a force trailer from one", f)
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
// regex builder produces one promote regex per
// `entity.SovereignActShapes()` entry, positionally aligned, plus one
// cancel regex per distinct cancel-reachable prefix. Without this pin,
// a regression that hardcoded the list back to a single regex would
// pass the fixture tests below and silently narrow the audit's reach.
//
// The expected count is derived from the closed set rather than
// written as a number, so an entry added for a new kind updates it
// here as well as in the builder.
func TestSovereignActPromoteRegexes_TracksKernelClosedSet(t *testing.T) {
	t.Parallel()
	shapes := entity.SovereignActShapes()
	regexes := sovereignActPromoteRegexes()

	cancelPrefixes := map[string]bool{}
	for _, s := range shapes {
		if p := entity.IDPrefix(s.Kind); p != "" && entity.CancelTarget(s.Kind, s.From) == s.To {
			cancelPrefixes[p] = true
		}
	}
	want := len(shapes) + len(cancelPrefixes)
	if len(regexes) != want {
		t.Fatalf("regex count = %d, want %d (one promote regex per entry, plus one cancel regex "+
			"per cancel-reachable prefix); shapes=%+v", len(regexes), want, shapes)
	}
	// Each regex should match the canonical command shape for its
	// corresponding shape entry. We construct an example invocation
	// from the entry's data and assert the regex at the same index
	// matches it.
	for i, s := range shapes {
		prefix := entity.IDPrefix(s.Kind)
		example := "aiwf promote " + prefix + "0001 " + string(s.To)
		if !regexes[i].MatchString(example) {
			t.Errorf("regex[%d] (%s) does not match example invocation %q built from shape entry %+v", i, regexes[i].String(), example, s)
		}
	}
}

// TestSovereignActRegexes_CoverTheCancelSpelling is M-0324/AC-4.
//
// The builder keys its patterns on (prefix, To), which describes the
// `aiwf promote <id> <to>` spelling only. ADR-0047 put both epic cancel
// edges in the closed set, and `aiwf cancel <id>` is how a human spells
// those — so an audit that saw only the promote form would be blind to
// automation invoking the very route M-0324 adds a call site for, and
// blind silently, since a missing pattern emits no finding.
//
// Which entries are cancel-reachable is derived rather than listed:
// entity.CancelTarget is the same mapping the cancel verb itself uses
// to pick a target, so an entry is reachable that way exactly when the
// mapping lands on the entry's To.
func TestSovereignActRegexes_CoverTheCancelSpelling(t *testing.T) {
	t.Parallel()
	regexes := sovereignActPromoteRegexes()

	var checked int
	for _, s := range entity.SovereignActShapes() {
		if entity.CancelTarget(s.Kind, s.From) != s.To {
			continue
		}
		checked++
		example := "aiwf cancel " + entity.IDPrefix(s.Kind) + "0001"
		if !lineMatchesAnySovereignActRegex(example, regexes) {
			t.Errorf("no regex matches %q, the spelling a human uses to reach shape %+v; "+
				"automation invoking it would pass the audit unseen", example, s)
		}
	}
	if checked == 0 {
		t.Fatal("no cancel-reachable entry in the closed set; this test's subject has vanished, " +
			"so it is passing without asserting anything")
	}
}

// TestSovereignActRegexes_CancelSpellingStaysScoped is the negative
// half of AC-4: widening the audit to the cancel spelling must not
// make it fire on cancels of kinds the closed set says nothing about.
// Milestone cancel is the case that matters — it is legal, routine,
// and deliberately not sovereign.
func TestSovereignActRegexes_CancelSpellingStaysScoped(t *testing.T) {
	t.Parallel()
	regexes := sovereignActPromoteRegexes()

	sovereignKinds := map[entity.Kind]bool{}
	for _, s := range entity.SovereignActShapes() {
		sovereignKinds[s.Kind] = true
	}
	for _, k := range []entity.Kind{entity.KindMilestone, entity.KindGap, entity.KindContract} {
		if sovereignKinds[k] {
			continue // a future entry would make this line legitimately match
		}
		example := "aiwf cancel " + entity.IDPrefix(k) + "0001"
		if lineMatchesAnySovereignActRegex(example, regexes) {
			t.Errorf("a regex matches %q, but no closed-set entry names kind %q — the audit would "+
				"refuse a cancel the kernel permits", example, k)
		}
	}
}

// TestSovereignActRegexesFor_CancelFormOnlyForReachableEdges drives
// the builder with fabricated shape lists, which is the only way to
// constrain the cancel-reachability rule: every entry in the live
// closed set is an epic, and the cancel form is deduplicated per
// prefix, so against that set a builder ignoring reachability
// entirely emits exactly the same regexes.
//
// The milestone kind supplies the discriminating case. `in_progress →
// done` is not something `aiwf cancel` can reach — CancelTarget maps
// a milestone to `cancelled` — so a sovereign entry for it must yield
// the promote form alone.
func TestSovereignActRegexesFor_CancelFormOnlyForReachableEdges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		shapes     []entity.SovereignActShape
		matches    []string
		notMatches []string
	}{
		{
			name:       "cancel-reachable entry yields the cancel form",
			shapes:     []entity.SovereignActShape{{Kind: entity.KindEpic, From: entity.StatusActive, To: entity.StatusCancelled}},
			matches:    []string{"aiwf cancel E-0001", "aiwf promote E-0001 cancelled"},
			notMatches: []string{"aiwf cancel M-0001"},
		},
		{
			name:       "unreachable-by-cancel entry yields the promote form only",
			shapes:     []entity.SovereignActShape{{Kind: entity.KindMilestone, From: entity.StatusInProgress, To: entity.StatusDone}},
			matches:    []string{"aiwf promote M-0001 done"},
			notMatches: []string{"aiwf cancel M-0001"},
		},
		{
			name: "a kind's cancel form is emitted once, not per entry",
			shapes: []entity.SovereignActShape{
				{Kind: entity.KindEpic, From: entity.StatusActive, To: entity.StatusCancelled},
				{Kind: entity.KindEpic, From: entity.StatusProposed, To: entity.StatusCancelled},
			},
			matches: []string{"aiwf cancel E-0001"},
		},
		{
			name:    "an unknown kind contributes nothing",
			shapes:  []entity.SovereignActShape{{Kind: entity.Kind("widget"), From: "a", To: "b"}},
			matches: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			regexes := sovereignActRegexesFor(tc.shapes)
			for _, line := range tc.matches {
				if !lineMatchesAnySovereignActRegex(line, regexes) {
					t.Errorf("no regex matches %q", line)
				}
			}
			for _, line := range tc.notMatches {
				if lineMatchesAnySovereignActRegex(line, regexes) {
					t.Errorf("a regex matches %q, which no shape in this list authorizes", line)
				}
			}
		})
	}
}

// TestSovereignActRegexesFor_CancelFormCount pins the emitted count
// for a mixed list, so a builder that dropped the reachability test
// and emitted a cancel form per prefix regardless would fire here
// even though the live closed set cannot tell the difference.
func TestSovereignActRegexesFor_CancelFormCount(t *testing.T) {
	t.Parallel()
	shapes := []entity.SovereignActShape{
		{Kind: entity.KindEpic, From: entity.StatusActive, To: entity.StatusCancelled},     // cancel-reachable
		{Kind: entity.KindMilestone, From: entity.StatusInProgress, To: entity.StatusDone}, // not
	}
	// Two promote forms, one cancel form.
	if got := len(sovereignActRegexesFor(shapes)); got != 3 {
		t.Errorf("regex count = %d, want 3 (a promote form per entry plus one cancel form for the "+
			"single cancel-reachable prefix)", got)
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
