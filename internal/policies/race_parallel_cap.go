package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PolicyRaceParallelCap asserts that every race-mode `go test`
// invocation in this repo's Makefile and GitHub workflows carries
// a `-parallel 8` cap.
//
// On macOS, `go test -race` at the default -parallel=GOMAXPROCS
// fan-outs the per-test git subprocess work faster than the host
// can absorb, producing a SIGSEGV-shaped flake (~50% of runs at
// GOMAXPROCS=20 on the development host; see G-0097 evidence).
// Capping at 8 is reliable. CI Linux runners are less affected but
// the same cap applies — one rule across host shapes, not three.
//
// The policy is structural: each named file must contain at least
// one `go test ... -race ...` line, and every such line must also
// carry `-parallel 8`. Drop the cap to 4 in one place and the
// policy fires on that line; same if a new race-test surface is
// added without the cap.
//
// Pins M-0091 AC-1. Drift-prevention test for the cap itself,
// not for the underlying parallelism rollout (AC-2 onward).
func PolicyRaceParallelCap(root string) ([]Violation, error) {
	targets := []string{
		"Makefile",
		filepath.Join(".github", "workflows", "go.yml"),
		filepath.Join(".github", "workflows", "flake-hunt.yml"),
	}
	raceLine := regexp.MustCompile(`go\s+test\b[^\n]*-race\b`)
	capPresent := regexp.MustCompile(`-parallel\s+8\b`)

	var vs []Violation
	for _, rel := range targets {
		path := filepath.Join(root, rel)
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		lines := strings.Split(string(b), "\n")
		sawRace := false
		for i, line := range lines {
			// Skip Makefile / YAML comment lines — the policy
			// targets executable invocations, not prose about
			// them.
			if trimmed := strings.TrimSpace(line); strings.HasPrefix(trimmed, "#") {
				continue
			}
			if !raceLine.MatchString(line) {
				continue
			}
			sawRace = true
			if !capPresent.MatchString(line) {
				vs = append(vs, Violation{
					Policy: "race-parallel-cap",
					File:   rel,
					Line:   i + 1,
					Detail: "race-mode go test invocation must carry `-parallel 8` (M-0091 AC-1)",
				})
			}
		}
		if !sawRace {
			vs = append(vs, Violation{
				Policy: "race-parallel-cap",
				File:   rel,
				Detail: "expected a `go test ... -race ...` line; none found (M-0091 AC-1)",
			})
		}
	}
	return vs, nil
}
