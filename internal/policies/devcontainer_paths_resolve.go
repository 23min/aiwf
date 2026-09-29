package policies

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// devcontainerPathRe matches a .devcontainer/ path as prose, scripts and Go
// name it: letters, digits, `.`, `_`, `-`, `/` and a `*` glob, ending on a
// name character so trailing punctuation is not part of the path.
var devcontainerPathRe = regexp.MustCompile(`\.devcontainer/[A-Za-z0-9_.*/-]*[A-Za-z0-9_*]`)

// devcontainerReferenceFiles lists the files a reader follows to the
// development container: the root guides, the Normative docs tier, scripts,
// CI, and Go source. Archival and exploratory docs keep the paths they were
// written with, and tests name fixture paths that exist only in a temporary
// tree, so neither is scanned.
func devcontainerReferenceFiles(root string) ([]string, error) {
	files := []string{"CLAUDE.md", "README.md", "CONTRIBUTING.md", "docs/architecture.md", "docs/overview.md", "docs/workflows.md", "docs/skill-author-guide.md"}
	walks := []struct {
		dir  string
		keep func(rel string) bool
	}{
		{"docs/adr", func(rel string) bool {
			return strings.HasSuffix(rel, ".md") && !strings.HasPrefix(rel, "docs/adr/archive/")
		}},
		{"docs/design", func(string) bool { return true }},
		{"docs/reference", func(string) bool { return true }},
		{"scripts", func(string) bool { return true }},
		{".github", func(string) bool { return true }},
		{"internal", isGoSource},
		{"cmd", isGoSource},
	}
	for _, w := range walks {
		err := filepath.WalkDir(filepath.Join(root, w.dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				//coverage:ignore defensive: every walked path lies under root, so Rel cannot fail
				return err
			}
			if rel = filepath.ToSlash(rel); w.keep(rel) {
				files = append(files, rel)
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("walking %s: %w", w.dir, err)
		}
	}
	return files, nil
}

func isGoSource(rel string) bool {
	return strings.HasSuffix(rel, ".go") && !strings.HasSuffix(rel, "_test.go")
}

// PolicyDevcontainerPathsResolve asserts that every .devcontainer/ path the
// guides, Normative docs, scripts, CI and Go source name exists, or matches a
// file when it is a glob. Replacing the container's files moves and removes
// paths; a reference left pointing at one sends the reader nowhere.
func PolicyDevcontainerPathsResolve(root string) ([]Violation, error) {
	files, err := devcontainerReferenceFiles(root)
	if err != nil {
		return nil, err
	}
	var vs []Violation
	for _, rel := range files {
		raw, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			continue // a listed guide the tree does not carry names nothing
		}
		for i, line := range strings.Split(string(raw), "\n") {
			for _, ref := range devcontainerPathRe.FindAllString(line, -1) {
				if matches, _ := filepath.Glob(filepath.Join(root, ref)); len(matches) == 0 {
					vs = append(vs, Violation{Policy: "devcontainer-paths-resolve", File: rel, Line: i + 1, Detail: fmt.Sprintf(
						"names %s, which does not exist", ref)})
				}
			}
		}
	}
	return vs, nil
}
