package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PolicyFiringFixturePresence is the meta-chokepoint over the policy
// corpus itself: it fails when a policy's Violation construction site —
// the line that runs only when the policy fires — is covered by no test.
// An uncovered construction site means there is zero evidence the policy
// can detect the regression it exists to catch: a refactor could turn it
// into a no-op and CI would stay green (G-0259, "vacuous chokepoints").
//
// Definition of "dark" is identical to the audit that motivated this
// gate: a `Policy: "<id>"` construction line whose enclosing coverage
// block ran zero times. Sharing the definition means the chokepoint
// cannot drift from the finding — it is the finding, made permanent and
// total instead of the diff-scoped coverage of G-0067 (which only ever
// gated the construction lines a change happened to touch).
//
// Inputs come from the environment, like PolicyBranchCoverageAudit, so
// the policy keeps the uniform func(root) shape runPolicy drives:
//
//   - AIWF_COVERAGE_PROFILE — a `go test -coverprofile` file from the
//     same tree as HEAD, instrumented with -coverpkg=./internal/... so
//     the policies package's own firing lines are recorded. Unset means
//     "no audit" (the live test t.Skips; the synthetic tests call
//     firingFixtureCore directly).
//
// Grandfathering: policies dark at the G-0259 audit are listed in
// grandfatherDark and tolerated; G-0262 burns that list to empty. A
// policy added after this gate landed is NOT grandfathered — it has no
// allowlist entry, so its dark construction line fails the gate until a
// firing fixture covers it. That is the ratchet.
func PolicyFiringFixturePresence(root string) ([]Violation, error) {
	profile := os.Getenv("AIWF_COVERAGE_PROFILE")
	if profile == "" {
		return nil, nil
	}
	return firingFixtureViolations(root, profile, grandfatherDark)
}

// firingFixtureViolations is the testable core: it reports a violation
// for every dark construction site whose policy id is not in allow.
// PolicyFiringFixturePresence passes grandfatherDark; the synthetic
// tests pass their own allowlist against a fixture tree. Keeping the
// Violation construction here (not in the env-gated wrapper) means a
// synthetic firing fixture covers this gate's own construction line —
// so firing-fixture-presence never goes dark on a clean live tree the
// way it would if the append lived behind the allowlist filter in the
// wrapper.
func firingFixtureViolations(root, profilePath string, allow map[string]bool) ([]Violation, error) {
	dark, err := firingFixtureCore(root, profilePath)
	if err != nil {
		return nil, err
	}
	var out []Violation
	for _, s := range dark {
		if allow[s.id] {
			continue
		}
		out = append(out, Violation{
			Policy: "firing-fixture-presence",
			File:   s.file,
			Line:   s.line,
			Detail: "policy " + s.id + " has a Violation construction site no test covers: nothing proves this policy can fire, so a refactor could silently make it a no-op (a vacuous chokepoint, G-0259). Add a firing test beside the policy that drives it to return >=1 violation; if it is a structure-auditor (fires only by mutating a hardcoded Go structure), grandfather it in grandfatherDark with a mutate-hunt note (G-0262).",
		})
	}
	return out, nil
}

// constructionSite is a `Policy: "<id>"` literal in a policy source
// file: the id it stamps, and the file:line where the literal sits.
type constructionSite struct {
	id   string
	file string // repo-relative, forward-slash
	line int
}

// constructionLinePat matches a Violation construction site: the
// `Policy: "<kebab-id>"` field literal every policy stamps when it
// fires. The captured id is the stable policy id; the line is the firing
// line whose coverage proves the policy can fire. A match inside a
// comment or string is harmless — a non-executable line falls in no
// coverage block, so lineInZeroBlock never reports it dark. A policy that
// set the field from a constant or variable instead of a string literal
// would be invisible to this scan (a false negative); none does today,
// and the closed-set-status / enum-literal policies keep id construction
// literal — so this stays YAGNI rather than a parsed-AST walk.
var constructionLinePat = regexp.MustCompile(`Policy:\s*"([a-z0-9-]+)"`)

// firingFixtureCore reports every Violation construction site in
// internal/policies whose enclosing coverage block is uncovered. It is
// the testable core: PolicyFiringFixturePresence applies the allowlist
// on top, and the synthetic-fixture tests drive this directly.
func firingFixtureCore(root, profilePath string) ([]constructionSite, error) {
	mod, err := modulePath(root)
	if err != nil {
		return nil, err
	}
	blocks, err := parseCoverProfile(profilePath, mod)
	if err != nil {
		return nil, err
	}
	// Fail closed: the profile must carry internal/policies coverage,
	// else every construction line falls in no block and the audit
	// silently passes everything. This fires if a future edit drops
	// -coverpkg=./internal/... or narrows the instrumented scope.
	if !hasPoliciesBlocks(blocks) {
		return nil, fmt.Errorf("coverage profile %s carries no internal/policies blocks; the firing-fixture audit needs the policies package instrumented (is -coverpkg=./internal/... still set?)", profilePath)
	}
	sites, err := constructionSites(root)
	if err != nil {
		return nil, err
	}
	var dark []constructionSite
	for _, s := range sites {
		if lineInZeroBlock(blocks[s.file], s.line) {
			dark = append(dark, s)
		}
	}
	return dark, nil
}

// hasPoliciesBlocks reports whether the parsed profile carries at least
// one block for an internal/policies source file.
func hasPoliciesBlocks(blocks map[string][]coverBlock) bool {
	for rel := range blocks {
		if strings.HasPrefix(rel, "internal/policies/") {
			return true
		}
	}
	return false
}

// constructionSites scans every non-test .go file under
// internal/policies for `Policy: "<id>"` literals and returns one site
// per match.
func constructionSites(root string) ([]constructionSite, error) {
	dir := filepath.Join(root, "internal", "policies")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	var out []constructionSite
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, rerr := os.ReadFile(filepath.Join(dir, name))
		if rerr != nil {
			return nil, fmt.Errorf("reading %s: %w", name, rerr) //coverage:ignore os.ReadFile failing on a path os.ReadDir just listed needs a TOCTOU race (the file deleted between the two syscalls); not deterministically reachable.
		}
		rel := "internal/policies/" + name
		for i, line := range strings.Split(string(data), "\n") {
			if m := constructionLinePat.FindStringSubmatch(line); m != nil {
				out = append(out, constructionSite{id: m[1], file: rel, line: i + 1})
			}
		}
	}
	return out, nil
}

// lineInZeroBlock reports whether line falls within a coverage block
// that ran zero times. Blocks in a Go profile are disjoint, so a line is
// in at most one block; a line in no block (e.g. a comment) is not dark.
func lineInZeroBlock(fileBlocks []coverBlock, line int) bool {
	for _, b := range fileBlocks {
		if b.Count == 0 && line >= b.StartLine && line <= b.EndLine {
			return true
		}
	}
	return false
}

// grandfatherDark lists the policy ids whose firing branch was dark at
// the G-0259 audit (a Violation construction site no test covers). The
// meta-gate tolerates these; G-0262 burns the list to empty by adding a
// firing fixture per policy — or, for a structure-auditor (one that can
// fire only by mutating a hardcoded Go structure, e.g. fsm-invariants or
// trailer-order-matches-constants), routing it through mutate-hunt and
// keeping it here with that note.
//
// A policy leaves this list the moment a test covers its construction
// line. The stale-entry test (TestPolicy_FiringFixtureNoStaleAllowlist)
// fails if an id here is no longer dark, so adding a firing fixture
// forces the matching deletion — the ledger cannot rot, and it shrinks
// monotonically toward zero.
var grandfatherDark = map[string]bool{
	// Seeded from the G-0259 audit (44 policies with >=1 dark
	// construction line), burned down to empty-but-one by G-0262 / M-0166:
	// AC-1 lit the 25 single-dark-site policies, AC-2 the 16 multi-site
	// policies, AC-3 the 11-site acks-helper-lift — each via a firing
	// fixture (firing_fixtures_{single,multi}_site_test.go,
	// firing_fixtures_acks_helper_test.go). Only fsm-invariants remains: a
	// structure-auditor no fixture can reach.
	"fsm-invariants": true, // structure-auditor: routes through mutate-hunt, not a firing fixture — introspects compiled-in entity FSM tables (discards root), so no fixture reaches its construction line; stays grandfathered permanently (G-0262 exempts this class).
}
