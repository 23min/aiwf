package check

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"regexp"
)

// resolveLines fills in Line on every finding that has a Path. Files
// are read at most once: the first finding for a given path triggers
// a scan that maps `<key>:` lines to their 1-based numbers; later
// findings for the same path are answered from the cache.
//
// When the finding's Field is empty, or the field can't be located,
// Line falls back to 1 — editors still get a clickable file:line link
// to the entity file.
func resolveLines(root string, findings []Finding) {
	cache := make(map[string]map[string]int)
	for i := range findings {
		f := &findings[i]
		if f.Path == "" {
			continue
		}
		idx, ok := cache[f.Path]
		if !ok {
			idx = scanFieldLines(filepath.Join(root, f.Path))
			cache[f.Path] = idx
		}
		if f.Field != "" {
			if line, ok := idx[f.Field]; ok {
				f.Line = line
				continue
			}
		}
		f.Line = 1
	}
}

// fieldKeyPattern matches a YAML mapping key at the start of a line,
// with optional indentation. Captures the key name. We don't try to
// parse YAML — we just want approximate line numbers good enough for
// editor navigation.
var fieldKeyPattern = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*:`)

// scanFieldLines reads the file at path and returns a map from YAML
// key (first occurrence at the start of a line) to its 1-based line
// number. Returns an empty map when the file cannot be read; callers
// fall back to line 1 in that case.
func scanFieldLines(path string) map[string]int {
	out := map[string]int{}
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		m := fieldKeyPattern.FindSubmatch(scanner.Bytes())
		if m == nil {
			continue
		}
		key := string(m[1])
		if _, seen := out[key]; seen {
			continue
		}
		out[key] = line
	}
	return out
}
