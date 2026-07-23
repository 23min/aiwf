package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestG0443_AuditSourceColumnGoPathsResolve guards the drift G-0443 fixed:
// the audit catalog's §§1-9 Source column names the file that enforces each
// rule, but nothing checked those paths still existed — so the cmd/aiwf →
// internal/cli restructure (M-0116) left ~35 rows citing files that had
// moved. This asserts every Source-column value that is a concrete Go file
// path (has a directory separator, ends in .go, no glob metacharacter)
// resolves on disk. Bare filenames, `*.go` category globs, ADR ids, doc
// names, and category words (FSM, Verb) are skipped, so the check is scoped
// to the one unambiguous shape and never false-positives on prose or
// historical references — which is why it is a targeted catalog guard, not a
// general doc-path linter.
func TestG0443_AuditSourceColumnGoPathsResolve(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	body := loadAuditCatalog(t)

	end := strings.Index(body, "## 10. Consolidated rules")
	if end == -1 {
		t.Fatal("cannot locate §10 boundary in catalog")
	}
	section := body[:end]

	rowPattern := regexp.MustCompile(`(?m)^\| R-AUDIT-\d{4} \|`)
	checked := 0
	for _, rs := range rowPattern.FindAllStringIndex(section, -1) {
		lineEnd := strings.IndexByte(section[rs[0]:], '\n')
		if lineEnd == -1 {
			lineEnd = len(section) - rs[0]
		}
		fields := splitTableRow(section[rs[0] : rs[0]+lineEnd])
		if len(fields) != 6 {
			continue // malformed rows are TestM0121_AC3's concern
		}
		src := strings.Trim(strings.TrimSpace(fields[1]), "`")
		if !isConcreteGoPath(src) {
			continue
		}
		checked++
		if _, err := os.Stat(filepath.Join(root, src)); err != nil {
			t.Errorf("row %s Source column cites Go path %q which does not resolve on disk (moved? update the source attribution)",
				strings.TrimSpace(fields[0]), src)
		}
	}
	if checked == 0 {
		t.Fatal("guard resolved zero Source-column Go paths — the extraction is broken (it must cover the internal/cli/<verb> rows)")
	}
}

// isConcreteGoPath reports whether s is a repo-root-relative Go source path
// worth resolving on disk: it ends in .go, carries no glob metacharacter (a
// `*.go` category glob names a set, not a file), and starts at a real
// top-level source dir (`internal/` or `cmd/`). Category-relative shorthand
// under a section's own base — e.g. §2's `policies/foo.go` (base
// `internal/policies/`) or a bare `transition.go` under §1 — uses a different,
// legitimate convention and is deliberately skipped: this guard targets the
// repo-root full-path style (the `cmd/aiwf/ → internal/cli/` class G-0443
// fixed), not every path shape in the catalog.
func isConcreteGoPath(s string) bool {
	if !strings.HasSuffix(s, ".go") || strings.Contains(s, "*") {
		return false
	}
	return strings.HasPrefix(s, "internal/") || strings.HasPrefix(s, "cmd/")
}
