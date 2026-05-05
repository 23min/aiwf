package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// markdownLinkPattern matches a markdown link of the form
// [text](path) where path is a relative reference (no scheme).
// Captures (1) text, (2) the path-and-fragment.
var markdownLinkPattern = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

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
