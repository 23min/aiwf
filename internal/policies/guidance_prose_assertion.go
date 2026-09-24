package policies

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// PolicyGuidanceProseAssertion enforces D-0091: no acceptance criterion is
// evidenced by a sentence pinned in this repository's development guidance
// — a CLAUDE.md or AGENTS.md at any depth, the project router, or a
// document the router links to. A test added or modified since
// AIWF_COVERAGE_BASE that reads one of those and asserts a phrase it
// wrote is present fails, naming the test. An absence assertion is a ban
// and pins no reading, so it stands; so does a relationship check, whose
// needle comes from code or another artefact.
//
// It runs the shipped-prose-assertion engine over a second surface, diff-
// scoped because pins already in the tree are the shrink's to retire: each
// is carried in guidanceProseLedger until its milestone re-aims or retires
// it, and TestGuidanceProseLedger_NoStaleEntries forces the entry out when
// its pin goes, so the ledger only shrinks. The policy is internal to this
// repository (ADR-0053).
func PolicyGuidanceProseAssertion(root string) ([]Violation, error) {
	return guidanceProseViolations(root, os.Getenv("AIWF_COVERAGE_BASE"), guidanceProseLedger)
}

// guidanceProseLedger names the tests pinning development-guidance prose
// when the ban landed. Each is re-aimed at a relationship or retired, with
// its reason recorded, by the E-0092 milestone that meets its passage.
var guidanceProseLedger = map[string]string{
	"TestM0154_AC2_CLAUDEMDOperatorSetupAmended":            "pins the operator-setup passage of CLAUDE.md",
	"TestM0195_AC5_SkillBodyDisciplineInClaudeMd":           "pins the shipped-surface discipline passage of CLAUDE.md",
	"TestM0209_AC1_GeneralizedGateInClaudeMd":               "pins the declared-sequence gate passage of CLAUDE.md",
	"TestM0211_AC1_ClaudeMdIdCollisionSplitInPlace":         "pins the id-collision resolution passage of CLAUDE.md",
	"TestM0211_AC3_AuthoringRuleNamesDividingPrinciple":     "pins the audience-dividing passage of CLAUDE.md",
	"TestM0234_AC4_ClaudeMdWorktreeSectionsCiteWorktreeAdd": "pins the worktree passages of CLAUDE.md",
}

// namesGuidancePath reports whether a string literal names a development
// guidance document: a CLAUDE.md or AGENTS.md at any depth, the project
// router, or one of the documents the router links to.
func namesGuidancePath(s string, routed map[string]bool) bool {
	s = strings.TrimPrefix(path.Clean(filepath.ToSlash(s)), "./")
	switch path.Base(s) {
	case fenceClaudeMD, fenceAgentsMD:
		return true
	}
	return s == fenceRouter || strings.HasSuffix(s, "/"+fenceRouter) || routed[s]
}

// guidanceSurface is D-0091's surface. ledger names the tests it does not
// judge.
func guidanceSurface(namesPath func(string) bool, ledger map[string]string) proseSurface {
	return proseSurface{
		policy:       "guidance-prose-assertion",
		namesPath:    namesPath,
		rooted:       true,
		presenceOnly: true,
		exempt: func(fn string) bool {
			_, ok := ledger[fn]
			return ok
		},
		noun:   "development guidance",
		remedy: "D-0091 bars evidencing a criterion with a sentence pinned in CLAUDE.md, AGENTS.md, the project router or a document it routes to: state the claim as a relationship check — a pointer resolved against what it names, an expectation derived by running the code — or record it as an observation.",
	}
}

// detectGuidanceProseAssertions is the pure core over one parsed package.
func detectGuidanceProseAssertions(fset *token.FileSet, files []*ast.File, paths map[*ast.File]string, namesPath func(string) bool, ledger map[string]string) []Violation {
	return proseViolations(detectProseFindings(guidanceSurface(namesPath, ledger), fset, files, paths))
}

// repoNamesGuidance builds the path predicate for the tree at root, with
// the router's routes read from disk.
func repoNamesGuidance(root string) func(string) bool {
	routed := map[string]bool{}
	if router, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(fenceRouter))); err == nil {
		for _, p := range routedDocuments(string(router)) {
			routed[p] = true
		}
	}
	return func(s string) bool { return namesGuidancePath(s, routed) }
}

// guidanceProseViolations scans each test package holding a test file
// changed between base and the working tree, uncommitted and untracked
// files included, and keeps what it finds in those files. The whole
// package is parsed so a helper in an unchanged file still carries its
// reads to the assertion.
func guidanceProseViolations(root, base string, ledger map[string]string) ([]Violation, error) {
	base = strings.TrimSpace(base)
	if base == "" || base == zeroSHA {
		return nil, nil
	}
	changed, err := changedTestFiles(root, base)
	if err != nil {
		return nil, err
	}
	dirs := map[string]bool{}
	for f := range changed {
		dirs[path.Dir(f)] = true
	}
	surface := guidanceSurface(repoNamesGuidance(root), ledger)
	ordered := make([]string, 0, len(dirs))
	for d := range dirs {
		ordered = append(ordered, d)
	}
	sort.Strings(ordered)
	var out []Violation
	for _, d := range ordered {
		found, err := scanPackageFor(root, d, surface)
		if err != nil { //coverage:ignore git lists a changed test file only when it can read the file, and so the directory holding it
			return nil, err
		}
		for _, f := range found {
			if changed[f.v.File] {
				out = append(out, f.v)
			}
		}
	}
	return out, nil
}

// changedTestFiles lists the Go test files that differ between base and
// the working tree, plus untracked ones, repo-relative.
func changedTestFiles(root, base string) (map[string]bool, error) {
	out := map[string]bool{}
	for _, args := range [][]string{
		{"-c", "core.quotePath=false", "diff", "--name-only", "--diff-filter=AMR", base},
		{"-c", "core.quotePath=false", "ls-files", "--others", "--exclude-standard"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		raw, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("git %s in %s: %w\n%s", strings.Join(args, " "), root, err, raw)
		}
		for _, line := range strings.Split(string(raw), "\n") {
			if line = strings.TrimSpace(line); strings.HasSuffix(line, "_test.go") {
				out[line] = true
			}
		}
	}
	return out, nil
}

// guidanceProseFlaggedTests scans every test package with no ledger and
// returns the names of the tests the rule refuses.
func guidanceProseFlaggedTests(root string) (map[string]bool, error) {
	dirs, err := testPackageDirs(root)
	if err != nil {
		return nil, err
	}
	surface := guidanceSurface(repoNamesGuidance(root), nil)
	out := map[string]bool{}
	for _, d := range dirs {
		found, err := scanPackageFor(root, d, surface)
		if err != nil { //coverage:ignore testPackageDirs just read every one of these directories
			return nil, err
		}
		for _, f := range found {
			out[f.fn] = true
		}
	}
	return out, nil
}
