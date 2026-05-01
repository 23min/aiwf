// Package contractconfig validates that the schema and fixtures
// paths configured in aiwf.yaml resolve to locations inside the
// consumer repo. Both `..` traversal and out-of-repo symlinks are
// rejected before any path is stat'd or passed to a validator.
//
// The package is the single point of truth for path containment
// across contractcheck (which reports the finding) and contractverify
// (which refuses to invoke a validator on an escaped path).
package contractconfig

import (
	"fmt"
	"path/filepath"

	"github.com/23min/ai-workflow-v2/tools/internal/aiwfyaml"
	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/pathutil"
)

// Resolved is the post-validation view of one contracts.entries[]
// entry. SchemaPath and FixturesPath are absolute paths with
// symlinks evaluated, suitable for direct use by callers.
//
// Skip is true when at least one path-escape finding was raised for
// this entry; callers must treat Skip == true as "do not stat, do
// not invoke the validator, do not proceed."
type Resolved struct {
	Entry        aiwfyaml.Entry
	SchemaPath   string
	FixturesPath string
	Skip         bool
}

// Resolve validates every entry's configured paths, returns the
// safe-to-use resolved forms, and emits one or two `contract-config`
// findings (subcode `path-escape`) per entry whose paths escape the
// repo root. repoRoot must be absolute.
//
// A nil contracts argument yields nil resolved and nil findings.
//
// The returned Resolved slice is in the same order as
// contracts.Entries. Entries whose paths are clean have Skip == false
// and absolute resolved paths populated; entries with any escape have
// Skip == true and the escaping path field empty.
func Resolve(repoRoot string, entries []aiwfyaml.Entry) ([]Resolved, []check.Finding) {
	if entries == nil {
		return nil, nil
	}

	rootResolved := resolveRepoRoot(repoRoot)

	resolved := make([]Resolved, len(entries))
	var findings []check.Finding

	for i, e := range entries {
		r := Resolved{Entry: e}

		schema, ok, fs := resolvePath(rootResolved, e, "schema", e.Schema, i)
		findings = append(findings, fs...)
		if ok {
			r.SchemaPath = schema
		} else {
			r.Skip = true
		}

		fixtures, ok, fs := resolvePath(rootResolved, e, "fixtures", e.Fixtures, i)
		findings = append(findings, fs...)
		if ok {
			r.FixturesPath = fixtures
		} else {
			r.Skip = true
		}

		resolved[i] = r
	}
	return resolved, findings
}

// resolveRepoRoot returns repoRoot with symlinks evaluated. If
// resolution fails (broken symlink, loop), it falls back to the
// cleaned form so per-entry containment checks still produce useful
// (if stricter) results rather than crashing. repoRoot is assumed
// absolute; relative input is cleaned but not made absolute, so the
// containment check naturally rejects all entries.
func resolveRepoRoot(repoRoot string) string {
	cleaned := filepath.Clean(repoRoot)
	if !filepath.IsAbs(cleaned) {
		return cleaned
	}
	resolved, err := pathutil.Resolve(cleaned)
	if err != nil {
		return cleaned
	}
	return resolved
}

// resolvePath joins, resolves, and verifies one configured path.
// Returns the resolved absolute path on success, or zero value plus a
// path-escape finding on any failure (empty input, escape, broken
// symlink, loop).
func resolvePath(rootResolved string, e aiwfyaml.Entry, kind, configured string, index int) (resolved string, ok bool, findings []check.Finding) {
	if configured == "" {
		return "", false, []check.Finding{escapeFinding(e, kind, configured, index)}
	}
	native := filepath.FromSlash(configured)
	// An absolute configured path is suspicious by itself — joining
	// would silently rebase it under the repo root (filepath.Join
	// does not treat an absolute second argument specially). Resolve
	// it as-is so the Inside check can reject it.
	var joined string
	if filepath.IsAbs(native) {
		joined = native
	} else {
		joined = filepath.Join(rootResolved, native)
	}
	r, err := pathutil.Resolve(joined)
	if err != nil {
		return "", false, []check.Finding{escapeFinding(e, kind, configured, index)}
	}
	if !pathutil.Inside(rootResolved, r) {
		return "", false, []check.Finding{escapeFinding(e, kind, configured, index)}
	}
	return r, true, nil
}

// escapeFinding constructs the path-escape finding. The message
// quotes the *configured* path verbatim and never references the
// resolved/host path, to avoid leaking absolute filesystem layout
// into output.
func escapeFinding(e aiwfyaml.Entry, kind, configured string, index int) check.Finding {
	return check.Finding{
		Code:     "contract-config",
		Severity: check.SeverityError,
		Subcode:  "path-escape",
		EntityID: e.ID,
		Path:     "aiwf.yaml",
		Message: fmt.Sprintf(
			"contracts.entries[%d] (id=%s): %s path %q resolves outside the repo root (check for `..` segments or out-of-repo symlinks)",
			index, e.ID, kind, configured,
		),
	}
}
