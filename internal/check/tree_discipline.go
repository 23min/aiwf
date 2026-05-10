package check

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// TreeDiscipline reports any file under work/* that the loader walked
// but could not classify as a recognized entity file. This is the
// G40 mechanical guarantee: the LLM must not write directly to the
// entity tree — tree-shape changes go through verbs, body-prose
// edits stay inside existing entity files, and stray hand-written
// files are flagged at validation time.
//
// Filtering, in order:
//
//  1. Files inside a recognized contract's directory
//     (work/contracts/C-NNN-<slug>/) are auto-exempt — contracts
//     legitimately carry schema/fixture artifacts alongside
//     contract.md, and the binding lives in aiwf.yaml.
//  2. Files matching any pattern in `allow` (filepath.Match,
//     forward-slash, repo-relative) are exempt. Consumers configure
//     these via `aiwf.yaml: tree.allow_paths`.
//  3. Everything else surfaces as a finding.
//
// Severity is warning by default. When `strict` is true (consumer
// opts in via `aiwf.yaml: tree.strict: true`), the severity is
// promoted to error so the pre-push hook blocks the push.
//
// Run does *not* call this — it lives outside the standard rule
// chain so render/status callers don't get tree-discipline noise on
// every read. `runCheck` invokes it at the validation chokepoint.
func TreeDiscipline(t *tree.Tree, allow []string, strict bool) []Finding {
	if len(t.Strays) == 0 {
		return nil
	}
	contractDirs := make(map[string]struct{})
	for _, e := range t.Entities {
		if e.Kind != entity.KindContract {
			continue
		}
		dir := filepath.ToSlash(filepath.Dir(e.Path))
		contractDirs[dir] = struct{}{}
	}
	severity := SeverityWarning
	if strict {
		severity = SeverityError
	}
	var findings []Finding
	for _, p := range t.Strays {
		if isInsideContractDir(p, contractDirs) {
			continue
		}
		if matchesAny(p, allow) {
			continue
		}
		findings = append(findings, Finding{
			Code:     "unexpected-tree-file",
			Severity: severity,
			Message: fmt.Sprintf("file %q is under work/ but is not a recognized entity file; tree-shape changes go through `aiwf <verb>`, not direct writes",
				p),
			Path: p,
		})
	}
	return findings
}

// isInsideContractDir reports whether path lives inside any directory
// that contains a recognized contract entity (contract.md). The
// comparison is forward-slash, prefix-based with a trailing slash to
// avoid matching siblings whose names share a prefix.
func isInsideContractDir(path string, contractDirs map[string]struct{}) bool {
	for dir := range contractDirs {
		if strings.HasPrefix(path, dir+"/") {
			return true
		}
	}
	return false
}

// matchesAny reports whether path matches any glob in patterns under
// filepath.Match semantics. Patterns are forward-slash, repo-relative.
// Invalid patterns silently fail-open (no match) — config-shape errors
// belong to the loader, not to this rule.
func matchesAny(path string, patterns []string) bool {
	for _, pat := range patterns {
		ok, err := filepath.Match(pat, path)
		if err == nil && ok {
			return true
		}
	}
	return false
}
