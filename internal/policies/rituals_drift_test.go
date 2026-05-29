package policies

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// vendoredRitualsDir is the repo-relative path where the pinned
// ai-workflow-rituals snapshot is vendored for embedding into the aiwf
// binary (E-0038 / M-0148). The drift check below pins it against the
// upstream ref recorded in rituals.lock.
const vendoredRitualsDir = "internal/skills/embedded-rituals/plugins"

// ritualsLockFile is the single discoverable record of the pinned
// upstream commit (AC-2): the URL the snapshot was vendored from and the
// exact ref it was vendored at. `make sync-rituals` reads it; this test
// reads it; both agree on one source of truth.
const ritualsLockFile = "rituals.lock"

// parseRitualsLock reads rituals.lock at the repo root into its url/ref
// fields. The format is intentionally trivial — `key=value` lines, `#`
// comments — so the shell sync script and this Go test parse the same
// file without a shared library.
func parseRitualsLock(t *testing.T, root string) (url, ref string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, ritualsLockFile))
	if err != nil {
		t.Fatalf("read %s: %v", ritualsLockFile, err)
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "url":
			url = strings.TrimSpace(v)
		case "ref":
			ref = strings.TrimSpace(v)
		}
	}
	return url, ref
}

// TestRituals_LockPinsUpstreamRef asserts AC-2: rituals.lock records a
// non-empty upstream URL and a full 40-char hex commit SHA.
func TestRituals_LockPinsUpstreamRef(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	url, ref := parseRitualsLock(t, root)
	if url == "" {
		t.Error("rituals.lock: url is empty")
	}
	if len(ref) != 40 || strings.Trim(ref, "0123456789abcdef") != "" {
		t.Errorf("rituals.lock: ref %q is not a 40-char lowercase-hex commit SHA", ref)
	}
}

// TestRituals_VendoredSnapshotPresent asserts AC-1: the vendored tree
// exists as committed files and carries the load-bearing artifact kinds
// (skills, agents, templates) from both plugins.
func TestRituals_VendoredSnapshotPresent(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	want := []string{
		"aiwf-extensions/skills/aiwfx-plan-epic/SKILL.md",
		"aiwf-extensions/skills/aiwfx-wrap-epic/SKILL.md",
		"aiwf-extensions/agents/builder.md",
		"aiwf-extensions/templates/epic-spec.md",
		"wf-rituals/skills/wf-tdd-cycle/SKILL.md",
	}
	for _, rel := range want {
		if _, err := os.Stat(filepath.Join(root, vendoredRitualsDir, rel)); err != nil {
			t.Errorf("vendored rituals missing %s: %v", rel, err)
		}
	}
}

// TestRituals_VendoredMatchesUpstream asserts AC-3: the vendored snapshot
// is byte-identical to the upstream plugins/ tree at the pinned ref. It
// fetches upstream@ref into a temp dir and compares.
//
// Skips cleanly when the upstream is absent: under -short, when git is
// unavailable, or on any fetch failure (offline / CI without network).
// A transient network error is a skip, not a failure — only a successful
// fetch that *differs* fails, so the check never flakes.
func TestRituals_VendoredMatchesUpstream(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("upstream rituals drift check skipped under -short")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available; cannot fetch upstream rituals")
	}
	root := repoRoot(t)
	url, ref := parseRitualsLock(t, root)

	tmp := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"remote", "add", "origin", url},
		{"fetch", "-q", "--depth", "1", "origin", ref},
		{"checkout", "-q", "FETCH_HEAD"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmp
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("upstream unreachable (git %s): %v\n%s", strings.Join(args, " "), err, out)
		}
	}

	compareTrees(t, filepath.Join(tmp, "plugins"), filepath.Join(root, vendoredRitualsDir))
}

// compareTrees reads both trees from disk and reports every divergence
// via t.Errorf. The comparison logic itself lives in diffRitualsTrees
// (pure, unit-tested below) so the missing/differ/extra branches are
// exercised deterministically without touching the filesystem or network.
func compareTrees(t *testing.T, want, got string) {
	t.Helper()
	for _, problem := range diffRitualsTrees(collectFiles(t, want), collectFiles(t, got)) {
		t.Error(problem)
	}
}

// diffRitualsTrees returns one problem message per file that is present
// upstream but missing or differing in the vendored snapshot, plus one per
// extra vendored file with no upstream counterpart. Empty result == in sync.
// The remediation is always the same: re-run `make sync-rituals`.
func diffRitualsTrees(want, got map[string][]byte) []string {
	var problems []string
	for rel, wb := range want {
		gb, ok := got[rel]
		if !ok {
			problems = append(problems, "vendored rituals missing "+rel+" (present upstream) — run `make sync-rituals`")
			continue
		}
		if !bytes.Equal(wb, gb) {
			problems = append(problems, "vendored rituals differ from upstream at "+rel+" — run `make sync-rituals`")
		}
	}
	for rel := range got {
		if _, ok := want[rel]; !ok {
			problems = append(problems, "vendored rituals has extra file "+rel+" (not upstream) — run `make sync-rituals`")
		}
	}
	return problems
}

// TestRituals_DiffTrees exercises every branch of diffRitualsTrees:
// identical (no problems), missing-in-vendored, extra-in-vendored,
// differing-content, and a mixed case that hits all three at once.
func TestRituals_DiffTrees(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		want, got map[string][]byte
		problems  int
	}{
		{"identical", map[string][]byte{"a": {1}}, map[string][]byte{"a": {1}}, 0},
		{"missing", map[string][]byte{"a": {1}}, map[string][]byte{}, 1},
		{"extra", map[string][]byte{}, map[string][]byte{"a": {1}}, 1},
		{"differ", map[string][]byte{"a": {1}}, map[string][]byte{"a": {2}}, 1},
		{"mixed", map[string][]byte{"a": {1}, "b": {1}}, map[string][]byte{"a": {2}, "c": {1}}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if n := len(diffRitualsTrees(tt.want, tt.got)); n != tt.problems {
				t.Errorf("diffRitualsTrees(%v, %v) = %d problems, want %d", tt.want, tt.got, n, tt.problems)
			}
		})
	}
}

// collectFiles maps every regular file under dir to its bytes, keyed by
// path relative to dir.
func collectFiles(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(dir, p)
		if relErr != nil {
			return relErr
		}
		b, readErr := os.ReadFile(p)
		if readErr != nil {
			return readErr
		}
		out[rel] = b
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return out
}
