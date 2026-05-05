// Package policies implements repo-wide invariants that are too
// codebase-specific for golangci-lint and too important to leave
// to convention. Each policy is a pure function from "the source
// tree at a path" to a slice of Violation values; tests in
// policies_test.go iterate them and surface findings via t.Errorf.
//
// The same package is meant to back a pre-commit hook: from the
// repo root, `go test ./internal/policies/...` runs every
// policy and exits non-zero on any violation.
//
// What's a policy here?
//
//   - "Provenance" rules: trailer keys come from constants, sovereign
//     acts gate on human/ actors, empty-diff commits carry a marker.
//   - "Audit-trail" rules: every finding has a hint, read-only verbs
//     never mutate state, every kernel concept appears in `--help`
//     or an embedded skill.
//
// Violations carry a Policy name (stable id), the source File +
// Line where the issue lives, and a human-readable Detail. Tests
// fail with one t.Errorf per violation so a CI run reads as a
// punch list.
package policies

import (
	"os"
	"path/filepath"
	"strings"
)

// Violation is one rule infraction. Policy is a stable id for the
// rule (e.g. "trailer-keys-via-constants"); File + Line locate the
// offender; Detail describes what to fix.
type Violation struct {
	Policy string
	File   string
	Line   int
	Detail string
}

// FileEntry is a (relative-path, contents) pair produced by walking
// the repo. Policies that just scan text consume this; AST-aware
// policies parse the source themselves.
type FileEntry struct {
	Path     string // forward-slash, repo-relative (e.g. "cmd/aiwf/main.go")
	AbsPath  string
	Contents []byte
}

// WalkGoFiles returns every .go file under root, excluding vendored
// trees, generated files, and the policies package itself (so a
// policy doesn't fire on its own assertion strings). Test files are
// included by default — pass excludeTests=true to drop them.
func WalkGoFiles(root string, excludeTests bool) ([]FileEntry, error) {
	var out []FileEntry
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == "node_modules" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if excludeTests && strings.HasSuffix(path, "_test.go") {
			return nil
		}
		// Skip the policies package itself: assertion strings inside
		// scanners would otherwise trip the policies they implement.
		rel, _ := filepath.Rel(root, path)
		relSlash := filepath.ToSlash(rel)
		if strings.HasPrefix(relSlash, "internal/policies/") {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		out = append(out, FileEntry{
			Path:     relSlash,
			AbsPath:  path,
			Contents: data,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// LineOf returns the 1-based line number of byte offset off in
// data. Returns 1 when off is out of range; never panics.
func LineOf(data []byte, off int) int {
	if off < 0 || off > len(data) {
		return 1
	}
	line := 1
	for i := 0; i < off; i++ {
		if data[i] == '\n' {
			line++
		}
	}
	return line
}

// FindAllOffsets returns every byte offset in data where needle
// appears. Used by simple grep-style policies.
func FindAllOffsets(data []byte, needle string) []int {
	var out []int
	if needle == "" {
		return out
	}
	s := string(data)
	start := 0
	for {
		i := strings.Index(s[start:], needle)
		if i < 0 {
			return out
		}
		out = append(out, start+i)
		start += i + len(needle)
	}
}
