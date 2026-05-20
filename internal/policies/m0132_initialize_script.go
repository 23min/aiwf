package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// PolicyM0132InitializeScript asserts that .devcontainer/initialize.sh
// is the host-side initializeCommand hook, has mode 0755, carries the
// expected bash header, creates the three /tmp symlinks the
// devcontainer mounts reference, and documents the shadow-mount
// workaround for claude-code#31388 in a comment block above the
// symlink section.
//
// Pins M-0132/AC-3. The structural location of the comment block
// matters — per CLAUDE.md's "substring assertions are not structural
// assertions" rule, the URL must appear in a comment line *above*
// the first ln -sfn, not anywhere else in the file. A future change
// that deletes the comment but leaves the URL in a footer comment
// would still fire this check.
func PolicyM0132InitializeScript(root string) ([]Violation, error) {
	const relPath = ".devcontainer/initialize.sh"
	abs := filepath.Join(root, relPath)

	info, err := os.Stat(abs)
	if err != nil {
		return []Violation{{
			Policy: "m0132-initialize-script",
			File:   relPath,
			Detail: fmt.Sprintf("missing or unreadable: %v", err),
		}}, nil
	}
	mode := info.Mode().Perm()

	raw, err := os.ReadFile(abs)
	if err != nil {
		return []Violation{{
			Policy: "m0132-initialize-script",
			File:   relPath,
			Detail: fmt.Sprintf("readable but ReadFile failed: %v", err),
		}}, nil
	}

	var vs []Violation
	report := func(detail string) {
		vs = append(vs, Violation{
			Policy: "m0132-initialize-script",
			File:   relPath,
			Detail: detail,
		})
	}

	if mode != 0o755 {
		report(fmt.Sprintf("mode = %#o, want 0755 (chmod +x .devcontainer/initialize.sh)", mode))
	}

	lines := strings.Split(string(raw), "\n")

	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "#!/usr/bin/env bash" {
		first := ""
		if len(lines) > 0 {
			first = lines[0]
		}
		report(fmt.Sprintf("first line = %q, want \"#!/usr/bin/env bash\"", first))
	}

	if !strings.Contains(string(raw), "set -euo pipefail") {
		report("missing `set -euo pipefail` directive (strict-mode bash; failures shouldn't silently propagate)")
	}

	// The three expected ln -sfn lines. Match the symlink target only
	// (the destination path under /tmp) so source-path reformatting
	// doesn't false-fail; the source is host-relative and varies by
	// $HOME expansion.
	lnPattern := regexp.MustCompile(`(?m)^\s*ln\s+-sfn\s+\S+\s+(/tmp/[A-Za-z0-9._-]+)\s*$`)
	wantTargets := map[string]bool{
		"/tmp/.claude-mount":         false,
		"/tmp/.claude-plugins-mount": false,
		"/tmp/.gh-mount":             false,
	}
	firstLnLine := -1
	for i, line := range lines {
		if m := lnPattern.FindStringSubmatch(line); m != nil {
			if _, ok := wantTargets[m[1]]; ok {
				wantTargets[m[1]] = true
			}
			if firstLnLine == -1 {
				firstLnLine = i
			}
		}
	}
	var missingLn []string
	for tgt, found := range wantTargets {
		if !found {
			missingLn = append(missingLn, tgt)
		}
	}
	if len(missingLn) > 0 {
		sort.Strings(missingLn)
		report(fmt.Sprintf("missing `ln -sfn ... %s` line(s) (the /tmp symlink dance per Q5; mounts in devcontainer.json reference these targets)", strings.Join(missingLn, ", ")))
	}

	// The claude-code#31388 URL must appear in a comment line above
	// the first ln -sfn (structural: URL-in-footer doesn't count).
	const want31388 = "https://github.com/anthropics/claude-code/issues/31388"
	urlFoundAbove := false
	if firstLnLine > 0 {
		for i := 0; i < firstLnLine; i++ {
			line := lines[i]
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "#") {
				continue
			}
			if strings.Contains(line, want31388) {
				urlFoundAbove = true
				break
			}
		}
	}
	if !urlFoundAbove {
		report(fmt.Sprintf("missing comment line containing %s above the first `ln -sfn` (the structural placement matters — URL in a footer comment doesn't count)", want31388))
	}

	// The comment block above the symlinks should also name the
	// concept the workaround addresses, so a reader-in-place
	// understands what the symlinks are for. Either "shadow-mount" or
	// "plugin index" (or both) appearing in a comment line above the
	// first ln -sfn satisfies the structural intent check.
	conceptFound := false
	if firstLnLine > 0 {
		for i := 0; i < firstLnLine; i++ {
			line := lines[i]
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "#") {
				continue
			}
			if strings.Contains(line, "shadow-mount") || strings.Contains(line, "plugin index") {
				conceptFound = true
				break
			}
		}
	}
	if !conceptFound {
		report("comment block above the ln -sfn section must name the concept the workaround addresses (\"shadow-mount\" or \"plugin index\" — readers shouldn't have to chase the URL to understand the structural intent)")
	}

	return vs, nil
}
