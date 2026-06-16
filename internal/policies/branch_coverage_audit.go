package policies

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// PolicyBranchCoverageAudit is the diff-scoped coverage gate: it
// reports every executable statement on a line changed since a base
// ref that the test suite did not exercise and that is not annotated
// `//coverage:ignore <reason>`.
//
// This is the kernel-side companion to the CI coverage-gate step. It
// makes the wf-tdd-cycle "branch coverage" HARD RULE mechanical rather
// than honor-system (G-0067): a PR or trunk push that adds an untested
// branch fails the gate, naming the file:line, unless the author either
// writes a test or explicitly annotates the line.
//
// What it actually enforces. Go's `-cover` is STATEMENT coverage, not
// branch coverage — the profile records, per basic block, how many
// times the block ran, not which arm of an `if`/`switch` was taken. So
// the audit is "every uncovered CHANGED statement is tested-or-ignored"
// — diff-scoped statement coverage with the `//coverage:ignore` escape.
// True per-arm branch correlation (distinguishing the taken/untaken
// arms of a conditional on a changed line) is a strictly stronger
// property this v1 does not provide; it is tracked as a follow-up.
//
// Inputs come from the environment so the policy keeps the uniform
// `func(root) ([]Violation, error)` shape the runPolicy harness drives:
//
//   - AIWF_COVERAGE_PROFILE — path to a `go test -coverprofile` file
//     generated from the same tree as HEAD. Required; an unset value
//     means "no audit" (the live-tree test t.Skips before calling
//     this, and the synthetic-fixture tests call branchCoverageViolations
//     directly).
//   - AIWF_COVERAGE_BASE — the git ref to diff HEAD against. On a PR
//     this is the merge-base with origin/main; on a trunk push it is
//     github.event.before. An empty or all-zero value means "no
//     comparison point" and the audit no-ops.
func PolicyBranchCoverageAudit(root string) ([]Violation, error) {
	profile := os.Getenv("AIWF_COVERAGE_PROFILE")
	if profile == "" {
		return nil, nil
	}
	base := os.Getenv("AIWF_COVERAGE_BASE")
	return branchCoverageViolations(root, profile, base)
}

// coverBlock is one entry from a coverage profile: a half-open span of
// source lines and the number of times the suite executed it. Only the
// line span and Count are load-bearing here; the columns and statement
// count are parsed for completeness but unused by the line-level audit.
type coverBlock struct {
	StartLine int
	EndLine   int
	Count     int
}

const zeroSHA = "0000000000000000000000000000000000000000"

// branchCoverageViolations is the testable core of the audit. It
// intersects the uncovered blocks in profilePath with the lines changed
// between baseRef and HEAD in root, and reports each uncovered changed
// statement that is not annotated `//coverage:ignore`.
func branchCoverageViolations(root, profilePath, baseRef string) ([]Violation, error) {
	baseRef = strings.TrimSpace(baseRef)
	if baseRef == "" || baseRef == zeroSHA {
		return nil, nil
	}

	mod, err := modulePath(root)
	if err != nil {
		return nil, err
	}

	blocks, err := parseCoverProfile(profilePath, mod)
	if err != nil {
		return nil, err
	}

	changed, err := changedLines(root, baseRef)
	if err != nil {
		return nil, err
	}

	var out []Violation
	// Iterate files deterministically (sorted) so violation order is
	// stable across runs.
	for _, rel := range sortedKeys(blocks) {
		lines, ok := changed[rel]
		if !ok {
			continue
		}
		src, srcErr := readSourceLines(filepath.Join(root, filepath.FromSlash(rel)))
		if srcErr != nil {
			return nil, srcErr
		}
		out = appendFileViolations(out, rel, blocks[rel], lines, src)
	}
	return out, nil
}

// appendFileViolations emits a violation for each uncovered block in
// one file that overlaps a changed line and is not annotated. Pulled
// out of branchCoverageViolations so the only loop touching git stays
// elsewhere (the no-retry-loops-on-git policy keys on for-loops with
// git in the body).
func appendFileViolations(out []Violation, rel string, fileBlocks []coverBlock, changedInFile map[int]bool, src []string) []Violation {
	for _, b := range fileBlocks {
		if b.Count > 0 {
			continue
		}
		if !blockOverlapsChange(b, changedInFile) {
			continue
		}
		if blockHasCoverageIgnore(b, src) {
			continue
		}
		out = append(out, Violation{
			Policy: "branch-coverage-audit",
			File:   rel,
			Line:   b.StartLine,
			Detail: "changed code on this line is not exercised by any test. Add a test that reaches it, or annotate the line `//coverage:ignore <reason>` if it is genuinely unreachable (diff-scoped statement coverage; G-0067).",
		})
	}
	return out
}

// blockOverlapsChange reports whether any line in the block's span was
// changed.
func blockOverlapsChange(b coverBlock, changedInFile map[int]bool) bool {
	for ln := b.StartLine; ln <= b.EndLine; ln++ {
		if changedInFile[ln] {
			return true
		}
	}
	return false
}

// blockHasCoverageIgnore reports whether any source line in the block's
// span carries a `//coverage:ignore` directive. src is 1-indexed via
// src[line-1].
func blockHasCoverageIgnore(b coverBlock, src []string) bool {
	for ln := b.StartLine; ln <= b.EndLine; ln++ {
		if ln < 1 || ln > len(src) {
			continue
		}
		if strings.Contains(src[ln-1], "//coverage:ignore") {
			return true
		}
	}
	return false
}

// modulePath reads the `module` directive from root/go.mod.
func modulePath(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("reading go.mod from %s: %w", root, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(rest), nil
		}
	}
	return "", fmt.Errorf("no module directive in %s/go.mod", root)
}

// coverLinePat matches a coverage-profile data line, capturing the
// full block span (both line and column) so duplicate blocks can be
// merged by an exact key:
//
//	<import-path>/file.go:<startLine>.<startCol>,<endLine>.<endCol> <numStmts> <count>
var coverLinePat = regexp.MustCompile(`^(.+):(\d+)\.(\d+),(\d+)\.(\d+) \d+ (\d+)$`)

// parseCoverProfile reads a `go test -coverprofile` file and returns
// its blocks grouped by repo-relative path (the import path minus the
// module prefix). Blocks whose import path lacks the module prefix are
// skipped.
//
// Duplicate blocks are merged. A `go test -coverpkg=./pkgs ./multi/...`
// run (which CI and `make coverage-gate` use) concatenates one profile
// per test binary, so the same block appears once per binary — count 0
// from binaries that never exercised it and count >0 from the one that
// did. Treating each occurrence independently would flag a covered
// block as uncovered. We key by the exact start.col,end.col span within
// a file and sum the counts, so a block counts as uncovered only when
// no binary reached it. (`go tool cover` performs the same merge.)
func parseCoverProfile(path, mod string) (map[string][]coverBlock, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening coverage profile %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	type blockKey struct{ rel, span string }
	merged := map[blockKey]*coverBlock{}
	order := map[string][]blockKey{} // per-file first-seen order, for stable output

	prefix := mod + "/"
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		m := coverLinePat.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		rel, ok := strings.CutPrefix(m[1], prefix)
		if !ok {
			continue
		}
		start, _ := strconv.Atoi(m[2])
		end, _ := strconv.Atoi(m[4])
		count, _ := strconv.Atoi(m[6])
		k := blockKey{rel: rel, span: m[2] + "." + m[3] + "," + m[4] + "." + m[5]}
		if b := merged[k]; b != nil {
			b.Count += count
			continue
		}
		merged[k] = &coverBlock{StartLine: start, EndLine: end, Count: count}
		order[rel] = append(order[rel], k)
	}
	if scErr := sc.Err(); scErr != nil { //coverage:ignore not portably triggerable: bufio.Scanner over a regular file with a 1MiB buffer errors only on a read fault / token longer than the buffer
		return nil, fmt.Errorf("scanning coverage profile %s: %w", path, scErr)
	}

	out := map[string][]coverBlock{}
	for rel, keys := range order {
		for _, k := range keys {
			out[rel] = append(out[rel], *merged[k])
		}
	}
	return out, nil
}

// hunkHeaderPat matches a unified-diff hunk header and captures the
// new-file start line and (optional) length: `@@ -a,b +c,d @@`.
var hunkHeaderPat = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

// changedLines returns, per repo-relative path, the set of new-file
// line numbers added or modified between baseRef and HEAD. It parses
// `git diff --unified=0`, taking each hunk's new-file range directly
// from the header (with -U0 there are no context lines, so the range is
// exactly the added/modified lines).
func changedLines(root, baseRef string) (map[string]map[int]bool, error) {
	cmd := exec.Command("git", "diff", "--unified=0", "--no-color", baseRef, "HEAD", "--", "*.go")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git diff %s..HEAD in %s: %w\n%s", baseRef, root, err, out)
	}

	result := map[string]map[int]bool{}
	var curFile string
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "+++ "):
			curFile = newFilePath(line)
		case strings.HasPrefix(line, "@@ "):
			if curFile == "" {
				continue
			}
			start, length := parseHunkRange(line)
			if length <= 0 {
				continue
			}
			set := result[curFile]
			if set == nil {
				set = map[int]bool{}
				result[curFile] = set
			}
			for ln := start; ln < start+length; ln++ {
				set[ln] = true
			}
		}
	}
	if scErr := sc.Err(); scErr != nil { //coverage:ignore not portably triggerable: bufio.Scanner over an in-memory string with a 4MiB buffer errors only on a token longer than the buffer
		return nil, fmt.Errorf("scanning git diff output: %w", scErr)
	}
	return result, nil
}

// newFilePath extracts the repo-relative path from a `+++ b/path` diff
// header. Returns "" for a deleted file (`+++ /dev/null`).
func newFilePath(header string) string {
	p := strings.TrimPrefix(header, "+++ ")
	if p == "/dev/null" {
		return ""
	}
	// Git prefixes the new path with `b/` by default.
	p = strings.TrimPrefix(p, "b/")
	return filepath.ToSlash(p)
}

// parseHunkRange returns the new-file start line and length from a hunk
// header. A missing length defaults to 1 per unified-diff convention.
func parseHunkRange(header string) (start, length int) {
	m := hunkHeaderPat.FindStringSubmatch(header)
	if m == nil {
		return 0, 0
	}
	start, _ = strconv.Atoi(m[1])
	if m[2] == "" {
		return start, 1
	}
	length, _ = strconv.Atoi(m[2])
	return start, length
}

// readSourceLines reads a file and returns its lines (without the
// trailing newline), so callers can index by 1-based line number via
// lines[line-1].
func readSourceLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading source %s: %w", path, err)
	}
	return strings.Split(string(data), "\n"), nil
}

// sortedKeys returns the map keys in lexical order.
func sortedKeys(m map[string][]coverBlock) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Small N (changed files per diff); insertion sort keeps it
	// dependency-free and avoids importing sort just for this.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	return keys
}
