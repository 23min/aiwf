package policies

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// PolicySkillEditStructuralTestBackstop is the diff-scoped backstop for
// the skill-edit → structural-test discipline (G-0220). Shipped ritual
// content under internal/skills/embedded-rituals/**/SKILL.md is
// materialized into consumer repos by `aiwf init` / `aiwf update`; the
// kernel design requires each prescriptive edit to be pinned by a
// structural test under internal/policies/ that fails if the prescription
// drifts. This policy flags any commit that modifies an embedded-rituals
// SKILL.md whose edit is not referenced by any such test — the M-0160
// failure mode, where a drive-by skill edit passed pre-commit and
// pre-push and was caught only by human review.
//
// It is a Go policy test (CI tier), not an `aiwf check` finding, because
// the property it polices — "this aiwf-repo skill edit has a paired
// internal/policies/ test" — is an aiwf-repo development invariant,
// meaningless in a consumer tree where rituals are materialized rather
// than authored.
//
// Like PolicyBranchCoverageAudit, it is diff-scoped and reads its base
// ref from the environment so it keeps the uniform `func(root)
// ([]Violation, error)` shape the runPolicy harness drives:
//
//   - AIWF_COVERAGE_BASE — the git ref to diff the working tree against.
//     An empty or all-zero value (the default in the broad `go test ./...`
//     job, and a brand-new branch's github.event.before) means "no comparison point"
//     and the audit no-ops. The authoritative invocation is the dedicated
//     CI coverage-gate step and `make coverage-gate`, both of which set it.
//
// v1 granularity is file-existence + skill-reference: the edited SKILL.md
// path appears as a string literal in some internal/policies/*_test.go
// source. The stronger "the test references the changed section" property
// is deferred to a follow-up gap.
func PolicySkillEditStructuralTestBackstop(root string) ([]Violation, error) {
	base := strings.TrimSpace(os.Getenv("AIWF_COVERAGE_BASE"))
	return skillEditBackstopViolations(root, base)
}

// skillRitualsDir is the embedded-ritual authoring tree whose SKILL.md
// edits this policy backstops. The verb-skill tree (embedded/) is out of
// scope — G-0220 is about rituals.
const skillRitualsDir = "internal/skills/embedded-rituals"

// skillEditBackstopViolations is the testable IO core: it resolves the
// changed embedded-rituals SKILL.md paths between baseRef and the working
// tree (untracked ones included), scans
// the policy test sources for path references, and delegates the per-path
// decision to detectUnbackedSkillEdits.
func skillEditBackstopViolations(root, baseRef string) ([]Violation, error) {
	baseRef = strings.TrimSpace(baseRef)
	if baseRef == "" || baseRef == zeroSHA {
		return nil, nil
	}
	changed, err := changedSkillFiles(root, baseRef)
	if err != nil {
		return nil, err
	}
	if len(changed) == 0 {
		return nil, nil
	}
	refs, err := policyTestRefs(root)
	if err != nil {
		return nil, err
	}
	return detectUnbackedSkillEdits(changed, refs), nil
}

// detectUnbackedSkillEdits is the pure core. For each changed
// embedded-rituals SKILL.md path it emits a Violation unless the path
// appears verbatim in policyTestSources (the concatenated
// internal/policies/*_test.go bytes) — the structural tests reference the
// skill by its repo-relative path constant (e.g. aiwfxWrapEpicFixturePath),
// which is the exact form `git diff --name-only` emits.
func detectUnbackedSkillEdits(changedSkillPaths []string, policyTestSources string) []Violation {
	var out []Violation
	for _, p := range changedSkillPaths {
		if strings.Contains(policyTestSources, p) {
			continue
		}
		out = append(out, Violation{
			Policy: "skill-edit-structural-test-backstop",
			File:   p,
			Detail: "this ritual SKILL.md was modified but no structural test under internal/policies/ references its path; add a structural test (see internal/policies/aiwfx_wrap_epic_test.go for the heading-walk template) that pins the prescribed content, or the edit ships to consumers with no mechanical backstop (G-0220).",
		})
	}
	return out
}

// changedSkillFiles returns the embedded-rituals SKILL.md paths added or
// modified between baseRef and the working tree, sorted for deterministic
// output. Deletions (--diff-filter excludes D) don't need a backstop test.
//
// The comparison target is the working tree, matching the coverage audit
// and the comment scan: `make ci` runs this gate before the ritual's
// commit step, so a HEAD-scoped diff would report green on an edit that
// has not been committed yet — vacuous at exactly the moment it is asked.
// Where the tree is clean, working tree and HEAD are the same tree.
// An untracked skill file is listed separately, because `git diff`
// cannot see one at any revision — and a ritual skill that has just been
// written is exactly the case the backstop exists for.
func changedSkillFiles(root, baseRef string) ([]string, error) {
	diffed, err := gitSkillPaths(root, "diff", "--name-only", "--diff-filter=AM", baseRef, "--", skillRitualsDir)
	if err != nil {
		return nil, err
	}
	untracked, err := gitSkillPaths(root, "ls-files", "--others", "--exclude-standard", "--", skillRitualsDir)
	if err != nil {
		return nil, err //coverage:ignore unreachable in practice: the diff above already ran git against this same repo, and ls-files takes no ref that could be invalid
	}

	// The lists cannot overlap: --diff-filter=AM drops deletions, and a
	// path that ls-files reports as untracked is absent from the index,
	// so the diff can only carry it as one.
	paths := make([]string, 0, len(diffed)+len(untracked))
	paths = append(paths, diffed...)
	paths = append(paths, untracked...)
	sort.Strings(paths)
	return paths, nil
}

// gitSkillPaths runs a git command that emits one path per line and keeps
// the SKILL.md entries, slash-normalized.
func gitSkillPaths(root string, args ...string) ([]string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		// The whole argv, not just the subcommand: an unresolvable base
		// ref is the realistic failure and it is only visible there.
		return nil, fmt.Errorf("git %s in %s: %w\n%s", strings.Join(args, " "), root, err, out)
	}
	var paths []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasSuffix(line, "/SKILL.md") {
			continue
		}
		paths = append(paths, filepath.ToSlash(line))
	}
	return paths, nil
}

// policyTestRefs returns the concatenated contents of every _test.go file
// under internal/policies/. The scan is restricted to test files because
// the backstop the gap requires is a structural *test*, not an incidental
// mention in a production policy source.
func policyTestRefs(root string) (string, error) {
	dir := filepath.Join(root, "internal", "policies")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", dir, err)
	}
	var b strings.Builder
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return "", fmt.Errorf("reading %s: %w", p, rerr) //coverage:ignore os.ReadFile failing on a path os.ReadDir just listed needs a TOCTOU race (the file deleted between the two syscalls); not deterministically reachable.
		}
		b.Write(data)
		b.WriteByte('\n')
	}
	return b.String(), nil
}
