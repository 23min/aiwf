package policies

import (
	"strings"
)

// PolicyTestsRealCloneNotUpdateRef flags integration tests that
// fabricate trunk-tracking refs via `git update-ref refs/remotes/...`
// instead of using a real bare repo + clone. Synthetic single-repo
// trunk setups can pass while the real first-push flow against an
// empty bare repo fails (and did, during G37 layer (a)) — the
// allocator/check logic happens to read the right refs but the
// surrounding policy is never actually exercised against the no-
// tracking-refs case.
//
// Heuristic: any *_test.go file under tools/cmd/aiwf/ that contains
// `update-ref refs/remotes/` is flagged unless the same file (or one
// of its same-package siblings) also contains `git init --bare`. The
// real-clone form is what proves the test exercises a fetched-ref
// state rather than a hand-stitched one.
//
// The policy targets cmd-level integration tests only. Lower-level
// unit tests in tools/internal/* may legitimately use update-ref to
// pin a single refs.Read code path; they don't claim to be testing
// end-to-end consumer flows.
func PolicyTestsRealCloneNotUpdateRef(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, false)
	if err != nil {
		return nil, err
	}

	// Collect every cmd-level test file's body for both the
	// per-file violation pass and the package-wide bare-repo
	// allowance check.
	var packageBody strings.Builder
	type entry struct {
		path string
		body string
	}
	var cmdTests []entry
	for _, f := range files {
		if !strings.HasPrefix(f.Path, "tools/cmd/aiwf/") {
			continue
		}
		if !strings.HasSuffix(f.Path, "_test.go") {
			continue
		}
		body := string(f.Contents)
		cmdTests = append(cmdTests, entry{path: f.Path, body: body})
		packageBody.WriteString(body)
		packageBody.WriteByte('\n')
	}
	pkg := packageBody.String()
	packageHas := strings.Contains(pkg, `"--bare"`) || strings.Contains(pkg, `--bare`)

	var out []Violation
	for _, e := range cmdTests {
		if !strings.Contains(e.body, "update-ref refs/remotes/") &&
			!strings.Contains(e.body, `"update-ref", "refs/remotes/`) {
			continue
		}
		if packageHas {
			// At least one sibling test in the same package proves
			// the real-clone form is exercised. The flagged file may
			// still be a synthetic shortcut, but the package as a
			// whole isn't relying solely on it for trunk coverage.
			continue
		}
		out = append(out, Violation{
			Policy: "tests-real-clone-not-update-ref",
			File:   e.path,
			Detail: "uses `git update-ref refs/remotes/...` to fabricate a trunk-tracking ref but no test in tools/cmd/aiwf/ uses `git init --bare` for a real clone scenario; synthetic single-repo trunk setups can pass while real first-push flows fail",
		})
	}
	return out, nil
}
