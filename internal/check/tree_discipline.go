package check

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// CodeUnexpectedTreeFile is the finding code emitted by TreeDiscipline
// for files under work/ that don't match a recognized entity shape.
// Typed per G-0129.
const CodeUnexpectedTreeFile = "unexpected-tree-file"

// TreeDiscipline reports any file under work/{epics,gaps,decisions,
// contracts}/ that the loader walked but could not classify as a
// recognized entity file. This is the G40 mechanical guarantee for
// the entity-bearing subtrees: the LLM must not write directly
// inside them — tree-shape changes go through verbs, body-prose
// edits stay inside existing entity files, and stray hand-written
// files in those subtrees are flagged at validation time.
//
// Scope is the four entity-bearing subdirs, not all of work/. Files
// at the work/ root or under non-entity sibling dirs (work/migration/,
// work/scratch/, etc.) are outside the loader's walk roots, never
// registered as strays, and never flagged here. The threat model is
// the LLM mistaking the entity tree as a scratch space, not the
// operator parking a transient file alongside it.
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
		// M-0086: archive scoping per ADR-0004 §"Check shape rules".
		// unexpected-tree-file is in the shape-and-health group;
		// strays under archive/ are historical artifacts and out of
		// scope for active tree-discipline linting. Tree-integrity
		// rules (ids-unique, parse-level errors) still traverse
		// archive in full — a malformed frontmatter under archive
		// is still a problem the loader surfaces as a load-error.
		if entity.IsArchivedPath(p) {
			continue
		}
		if matchesAny(p, allow) {
			continue
		}
		findings = append(findings, Finding{
			Code:     CodeUnexpectedTreeFile,
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
