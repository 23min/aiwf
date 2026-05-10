package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/23min/ai-workflow-v2/internal/entity"
)

// markdownLinkPattern matches a markdown link of the form
// [text](path) where path is a relative reference (no scheme).
// Captures (1) text, (2) the path-and-fragment.
var markdownLinkPattern = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

// idLeadingInSegment matches the leading "<kind-prefix>-<digits>"
// portion of a path-component basename. ADR is listed first so the
// alternation does not match D against ADR's leading A. Used by the
// width-tolerant path resolver — same theme as M-081's parser
// tolerance for entity ids (ADR-0008).
var idLeadingInSegment = regexp.MustCompile(`^(ADR|[EMGDC])-(\d+)`)

// canonicalizePathIDs returns p with every path segment whose leading
// token matches `<kind-prefix>-<digits>` rewritten so the digit run is
// zero-padded to entity.CanonicalPad. Used as a fallback when a
// path-form reference doesn't resolve at its authored width — narrow
// legacy `work/epics/E-19-foo/epic.md` resolves via the canonical
// `work/epics/E-0019-foo/epic.md` after M-082's rewidth migrates the
// active tree, and vice versa for any reverse case.
//
// Returns p unchanged when no segment carries an id-shaped leading
// token, when the digit run already meets canonical width, or when
// the path is malformed.
func canonicalizePathIDs(p string) string {
	parts := strings.Split(p, "/")
	changed := false
	for i, part := range parts {
		m := idLeadingInSegment.FindStringSubmatch(part)
		if m == nil {
			continue
		}
		prefix, digits := m[1], m[2]
		if len(digits) >= entity.CanonicalPad {
			continue
		}
		n, err := strconv.Atoi(digits)
		if err != nil {
			continue
		}
		parts[i] = fmt.Sprintf("%s-%0*d", prefix, entity.CanonicalPad, n) + part[len(m[0]):]
		changed = true
	}
	if !changed {
		return p
	}
	return strings.Join(parts, "/")
}

// PolicyDesignDocAnchors scans the docs/pocv3/ tree for relative
// markdown links and asserts every linked file path exists. When
// the link carries a #fragment, we also verify a heading or
// explicit anchor in the target file matches.
//
// Scope: only docs/pocv3/. Other directories may have their own
// link conventions; this policy is specifically about keeping the
// design / plan corpus internally consistent.
//
// Mailto, http://, https:// links are skipped (only relative
// references are validated). Code-fenced blocks aren't filtered
// out — a `[foo](path)` inside ```` ```go ```` would be checked,
// which is rare and acceptable.
func PolicyDesignDocAnchors(root string) ([]Violation, error) {
	docsRoot := filepath.Join(root, "docs", "pocv3")
	mdFiles, err := walkMarkdown(docsRoot)
	if err != nil {
		return nil, err
	}
	var out []Violation
	for _, f := range mdFiles {
		matches := markdownLinkPattern.FindAllSubmatchIndex(f.Contents, -1)
		for _, m := range matches {
			if len(m) < 6 {
				continue
			}
			target := string(f.Contents[m[4]:m[5]])
			if isExternalLink(target) {
				continue
			}
			path, fragment := splitMarkdownLink(target)
			if path == "" {
				continue
			}
			absTarget := resolveDocPath(f.AbsPath, path)
			info, statErr := os.Stat(absTarget)
			if statErr != nil {
				// Width-tolerance fallback: try the canonical-width
				// form of the path. After M-082's `aiwf rewidth`
				// migrates the active tree, narrow legacy references
				// in design docs (which M-083 cleans up) still resolve
				// to the canonical filename — same theme as M-081's
				// parser tolerance for entity ids.
				if canonical := canonicalizePathIDs(absTarget); canonical != absTarget {
					if info2, statErr2 := os.Stat(canonical); statErr2 == nil {
						info, statErr, absTarget = info2, nil, canonical
					}
				}
			}
			if statErr != nil {
				out = append(out, Violation{
					Policy: "design-doc-anchors-valid",
					File:   relTo(root, f.AbsPath),
					Line:   LineOf(f.Contents, m[0]),
					Detail: "broken link: " + target + " (path " + path + " does not exist)",
				})
				continue
			}
			if info.IsDir() || fragment == "" {
				continue
			}
			if !markdownAnchorExists(absTarget, fragment) {
				out = append(out, Violation{
					Policy: "design-doc-anchors-valid",
					File:   relTo(root, f.AbsPath),
					Line:   LineOf(f.Contents, m[0]),
					Detail: "broken anchor: " + target + " (#" + fragment + " not found in target)",
				})
			}
		}
	}
	return out, nil
}

// walkMarkdown returns every .md file under root (recursively),
// FileEntry-like. Used here rather than WalkGoFiles because the
// existing helper is Go-source-specific.
func walkMarkdown(root string) ([]FileEntry, error) {
	var out []FileEntry
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil, nil // no docs tree yet — policy is silent
	}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		out = append(out, FileEntry{
			Path:     path,
			AbsPath:  path,
			Contents: data,
		})
		return nil
	})
	return out, err
}

func isExternalLink(target string) bool {
	t := strings.TrimSpace(target)
	switch {
	case strings.HasPrefix(t, "http://"), strings.HasPrefix(t, "https://"):
		return true
	case strings.HasPrefix(t, "mailto:"), strings.HasPrefix(t, "tel:"):
		return true
	case strings.HasPrefix(t, "#"):
		// Same-document anchor; let the next layer validate via
		// fragment-only path.
		return false
	}
	return false
}

func splitMarkdownLink(target string) (path, fragment string) {
	t := strings.TrimSpace(target)
	idx := strings.Index(t, "#")
	if idx < 0 {
		return t, ""
	}
	return t[:idx], t[idx+1:]
}

func resolveDocPath(fromAbs, rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Clean(filepath.Join(filepath.Dir(fromAbs), rel))
}

// markdownAnchorExists checks that the given target file contains
// either an HTML anchor (<a name="frag"> / id="frag") or a heading
// whose slugified form matches frag. The slug rules used here are
// the GitHub-style: lowercase, spaces → dashes, drop punctuation
// except dashes / underscores.
func markdownAnchorExists(absPath, fragment string) bool {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return false
	}
	body := string(data)
	if strings.Contains(body, `name="`+fragment+`"`) ||
		strings.Contains(body, `id="`+fragment+`"`) {
		return true
	}
	// Heading slug match.
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimLeft(line, "# ")
		if trimmed == line {
			continue // not a heading
		}
		if slugifyHeading(trimmed) == fragment {
			return true
		}
	}
	return false
}

// slugifyHeading turns a markdown heading into a GitHub-style
// fragment. Lowercase, spaces → dashes, drop punctuation except
// `-` and `_`. Inline backticks and emphasis markers are stripped.
func slugifyHeading(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		default:
			// drop
		}
	}
	return b.String()
}

// relTo returns an absolute path made relative to root for nicer
// reporting. Falls back to the absolute path on failure.
func relTo(root, abs string) string {
	if rel, err := filepath.Rel(root, abs); err == nil {
		return filepath.ToSlash(rel)
	}
	return abs
}
