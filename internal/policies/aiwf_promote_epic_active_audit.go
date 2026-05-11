package policies

import (
	"fmt"
	"io/fs"
	"regexp"
	"strings"
)

// auditUnforcedEpicActivate is the static chokepoint complementing
// the runtime sovereign-act rule landed in M-0095 (`internal/verb/
// promote_sovereign_epic_active.go`). Where M-0095 refuses non-
// `human/` actors at verb invocation time, this audit refuses
// *automation-shaped source* — CI workflow files, scripts, Makefiles
// — that statically invoke `aiwf promote E-<id> active` without the
// `--force --reason "..."` override. The two chokepoints layer: a
// CI/script line that escapes static review still fails at runtime,
// but the static check surfaces the problem at PR time rather than
// at deploy time.
//
// M-0097/AC-1.

// epicActivateRegex matches `aiwf promote E-<id> active` (case-
// sensitive, whitespace-flexible). The token after `promote` must
// start with `E-`; the trailing word must be `active`. Other promote
// edges (`done`, `cancelled`) and other kinds (M-, G-, C-, ADR-) are
// out of scope per AC-1 — the sovereign-act rule M-0095 itself is
// scoped to `epic / proposed → active`.
var epicActivateRegex = regexp.MustCompile(`aiwf\s+promote\s+E-\S+\s+active`)

// auditUnforcedEpicActivate scans the named paths under fsys for
// lines invoking `aiwf promote E-<id> active` without `--force` on
// the same line. Returns one human-readable finding per offender of
// the form `<path>:<line-number>: <trimmed line content>`.
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
func auditUnforcedEpicActivate(fsys fs.FS, paths []string) []string {
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
				if !epicActivateRegex.MatchString(line) {
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
