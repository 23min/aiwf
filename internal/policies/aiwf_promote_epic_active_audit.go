package policies

import (
	"fmt"
	"io/fs"
	"regexp"
	"strings"

	"github.com/23min/aiwf/internal/entity"
)

// auditUnforcedSovereignActPromote is the static chokepoint
// complementing the runtime sovereign-act rule (`internal/verb/
// promote_sovereign_act.go`). Where the runtime gate refuses non-
// `human/` actors at verb invocation time, this audit refuses
// *automation-shaped source* — CI workflow files, scripts, Makefiles
// — that statically invoke `aiwf promote <prefix>-<id> <to>` against
// any sovereign-act-shape transition without the `--force --reason
// "..."` override on the same line. The two chokepoints layer: a
// CI/script line that escapes static review still fails at runtime,
// but the static check surfaces the problem at PR time rather than
// at deploy time.
//
// The set of (kind, to) pairs to scan for is derived from
// `entity.SovereignActShapes()` at call time — adding a new entry
// to the kernel's closed set automatically widens the audit's reach
// without policy-side changes. M-0095 was the first such entry (epic
// proposed → active per G-0063); M-0130 consolidated the kernel
// property into `internal/entity/sovereign.go` and made this audit
// list-driven.
//
// M-0097/AC-1 (original chokepoint); M-0130 (consolidation).

// sovereignActPromoteRegexes builds the regexes matching automation-
// shaped invocations of a kernel-declared sovereign-act-shape
// transition. Every entry yields a `aiwf promote <prefix>-<id> <to>`
// regex, at the entry's own index, so the leading len(shapes) elements
// align positionally with `entity.SovereignActShapes()`. Entries
// reachable through `aiwf cancel` yield a second regex for that
// spelling, appended after the promote block.
//
// Both spellings are needed because a transition is named by the state
// it reaches, not by the verb that reaches it: ADR-0047 places both
// epic cancel edges in the closed set, and a human spells those `aiwf
// cancel <id>`. Matching only the promote form would leave the audit
// blind to the natural spelling, and blind silently — an unmatched
// line produces no finding.
//
// The cancel spelling names no status, so it cannot discriminate on
// From the way the promote form does: one regex per kind is all it can
// express. That makes it wider than the closed set for any kind whose
// sovereign edges are not all cancel-reachable — a scripted `aiwf
// cancel` of such a kind would be reported though the kernel permits
// it. No kind is in that position today, and the audit's subject is
// automation-shaped source, where the finding names the file and line
// and is cheap to answer. Narrowing the emission by consulting
// entity.CancelTarget was built and removed: against a closed set whose
// entries are all one kind it produced byte-identical output, so the
// rule cost more to constrain than the over-match it prevented.
//
// Built on-demand rather than as a package-level var so a future
// kernel-side addition lands in the same compilation unit without a
// stale-package gotcha; the cost is one map walk per audit call (the
// closed set is tiny — single-digit entries — so the overhead is
// negligible).
func sovereignActPromoteRegexes() []*regexp.Regexp {
	shapes := entity.SovereignActShapes()
	out := make([]*regexp.Regexp, 0, len(shapes))
	var cancelForms []*regexp.Regexp
	seenPrefix := map[string]bool{}
	for _, s := range shapes {
		prefix := entity.IDPrefix(s.Kind)
		if prefix == "" {
			// Defensive: a kernel entry with an unknown kind would
			// be a closed-set-invariant violation (see
			// TestSovereignActShapes_AllFSMLegal). Skip silently
			// here; the kernel-side invariant test is the
			// authoritative chokepoint.
			continue
		}
		// regexp.QuoteMeta both pieces — neither prefix nor status
		// names are user input today, but the discipline keeps the
		// helper safe if either ever derives from less-controlled
		// data.
		pattern := `aiwf\s+promote\s+` + regexp.QuoteMeta(prefix) + `\S+\s+` + regexp.QuoteMeta(string(s.To))
		out = append(out, regexp.MustCompile(pattern))

		if seenPrefix[prefix] {
			continue
		}
		seenPrefix[prefix] = true
		cancelForms = append(cancelForms, regexp.MustCompile(`aiwf\s+cancel\s+`+regexp.QuoteMeta(prefix)+`\S+`))
	}
	return append(out, cancelForms...)
}

// auditUnforcedSovereignActPromote scans the named paths under fsys
// for lines invoking `aiwf promote <prefix>-<id> <to>` (for any
// kernel-declared sovereign-act-shape transition) without `--force`
// on the same line. Returns one human-readable finding per offender
// of the form `<path>:<line-number>: <trimmed line content>`.
//
// The same-line `--force` rule is intentionally strict: heredoc /
// multi-line invocations that split the override across lines are
// not common in CI workflow files (which prefer single-line `run:`
// values), and treating them as exempt would weaken the audit's
// guarantee. If a future legitimate multi-line case surfaces, the
// rule can be relaxed deliberately rather than absorbed silently.
//
// Each entry in `paths` is a path relative to fsys's root. A missing
// path is silently skipped (the caller decides which paths to probe).
// Walk errors on individual files are silently skipped — the audit's
// job is to surface *findable* offenders, not to fight the filesystem.
func auditUnforcedSovereignActPromote(fsys fs.FS, paths []string) []string {
	regexes := sovereignActPromoteRegexes()
	if len(regexes) == 0 {
		return nil
	}
	var findings []string
	for _, p := range paths {
		_ = fs.WalkDir(fsys, p, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			data, readErr := fs.ReadFile(fsys, path)
			if readErr != nil {
				return nil
			}
			for i, line := range strings.Split(string(data), "\n") {
				if !lineMatchesAnySovereignActRegex(line, regexes) {
					continue
				}
				if strings.Contains(line, "--force") {
					continue
				}
				findings = append(findings, fmt.Sprintf("%s:%d: %s", path, i+1, strings.TrimSpace(line)))
			}
			return nil
		})
	}
	return findings
}

// lineMatchesAnySovereignActRegex reports whether `line` matches any
// of the supplied sovereign-act-shape promote-line regexes. Extracted
// so the test suite can drive the predicate directly with a
// controlled regex set.
func lineMatchesAnySovereignActRegex(line string, regexes []*regexp.Regexp) bool {
	for _, re := range regexes {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}
