package policies

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/23min/aiwf/internal/projectguidance"
)

// guidanceSpan is one backticked span in guidance text.
var guidanceSpan = regexp.MustCompile("`([^`\n]+)`")

// braceGroup is the first `{a,b}` alternation in a glob.
var braceGroup = regexp.MustCompile(`\{([^{}]*)\}`)

// PolicyGuidancePathsResolve asserts that every repository path the
// development guidance cites in backticks exists, so a reader following
// a citation lands on a file. A path git ignores is not judged: only a
// command such as `aiwf update` or a build creates it, so a fresh
// checkout lacks it whether or not the citation is right.
func PolicyGuidancePathsResolve(root string) ([]Violation, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	roots := map[string]bool{}
	for _, e := range entries {
		roots[e.Name()] = true
	}
	read := repoGuidanceReader(root)
	docs := developmentGuidance(read)
	var vs []Violation
	for _, rel := range slices.Sorted(maps.Keys(docs)) {
		text := docs[rel]
		if rel == projectguidance.ClaudeFile || rel == projectguidance.AgentsFile {
			raw, _ := read(rel) // developmentGuidance listed it, so it reads
			text = projectguidance.HandwrittenLines(raw)
		}
		for i, line := range strings.Split(text, "\n") {
			for _, m := range guidanceSpan.FindAllStringSubmatch(line, -1) {
				p, ok := citedPath(m[1], roots)
				if !ok || pathResolves(root, p) {
					continue
				}
				ignored, err := gitIgnores(root, p)
				if err != nil {
					return nil, err
				}
				if !ignored {
					vs = append(vs, Violation{Policy: "guidance-paths-resolve", File: rel, Line: i + 1, Detail: fmt.Sprintf(
						"cites %s, which does not exist; correct the path or remove the citation", p)})
				}
			}
		}
	}
	return vs, nil
}

// citedPath reports the repository path a backticked span cites: a span
// with no whitespace or angle-bracket placeholder, holding a slash, whose
// first segment names an entry at the repository root. A leading `./` and
// a trailing `/...` package pattern are read as the path they name.
func citedPath(span string, roots map[string]bool) (string, bool) {
	if strings.ContainsAny(span, " \t<") || !strings.Contains(span, "/") {
		return "", false
	}
	p := strings.TrimSuffix(strings.TrimPrefix(span, "./"), "/...")
	first, _, _ := strings.Cut(p, "/")
	if !roots[first] {
		return "", false
	}
	return p, true
}

// pathResolves reports whether a cited path matches the tree: every
// alternative of its brace groups must match a file or directory, `*` and
// `**` matching within one segment.
func pathResolves(root, p string) bool {
	if m := braceGroup.FindStringSubmatchIndex(p); m != nil {
		for _, alt := range strings.Split(p[m[2]:m[3]], ",") {
			if !pathResolves(root, p[:m[0]]+alt+p[m[1]:]) {
				return false
			}
		}
		return true
	}
	matches, _ := filepath.Glob(filepath.Join(root, filepath.FromSlash(p)))
	return len(matches) > 0
}

// gitIgnores reports whether git ignores a path in the repository at root,
// as a file or, for a pattern that names only directories, as a directory.
func gitIgnores(root, p string) (bool, error) {
	for _, q := range []string{p, strings.TrimSuffix(p, "/") + "/"} {
		err := exec.Command("git", "-C", root, "check-ignore", "-q", q).Run()
		var exit *exec.ExitError
		switch {
		case err == nil:
			return true, nil
		case !errors.As(err, &exit) || exit.ExitCode() != 1:
			return false, fmt.Errorf("asking git whether %s is ignored: %w", p, err)
		}
	}
	return false, nil
}
