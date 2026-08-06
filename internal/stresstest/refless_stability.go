package stresstest

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/entity"
)

// refless_stability.go — M-0300/AC-2: a verdict is stable under refs the
// tree does not need. The harness computes the verdict twice — once on
// the working checkout, once on a copy stripped of those refs — and
// requires the two to agree.
//
// A verdict that changes when an unrelated ref disappears is reporting
// on the repository's ref graph rather than on the tree, which is the
// shape G-0556 records: a reference resolving against refs only the
// author's machine holds passes locally and fails in every clone.
//
// Agreement here is disposition, not identity, for the same reason it is
// in the read-path property. Stripping a branch that carries a cited id
// legitimately moves the classification from cross-branch-local-only to
// unresolved; both block, so the two runs refine rather than disagree,
// and only a subject that flips between blocking and not is reporting on
// the ref graph.

// refLessStabilityInvariant is M-0300/AC-2's property.
type refLessStabilityInvariant struct{}

// Name identifies the property in a violation message.
func (refLessStabilityInvariant) Name() string { return "ref-less verdict stability" }

// Evaluate computes dir's verdict, then recomputes it on a copy from
// which every ref the tree does not need has been removed.
//
// A repository with no removable ref is left alone: there is nothing to
// take away, so the property has no condition to judge and costs one
// `git for-each-ref`. That is the whole catalog today — a scenario
// composing on one branch with no remote never reaches the copy.
func (refLessStabilityInvariant) Evaluate(aiwfBin, dir, label string) ([]Violation, error) {
	removable, err := removableRefs(dir)
	if err != nil {
		return nil, err
	}
	if len(removable) == 0 {
		return nil, nil
	}

	withRefs, err := runAiwfJSON(aiwfBin, dir, "check")
	if err != nil { //coverage:ignore defensive: same launch-failure class pinned at its source by TestRefLessStabilityInvariant_ErrorsWhenTheBinaryCannotRun
		return nil, fmt.Errorf("running aiwf check on the working checkout: %w", err)
	}

	stripped, cleanup, err := copyRepoWithoutRefs(dir, removable)
	if err != nil { //coverage:ignore defensive: forwards copyRepoWithoutRefs's own failures, each pinned at its source by TestCopyRepoWithoutRefs_Errors*
		return nil, err
	}
	defer cleanup()

	withoutRefs, err := runAiwfJSON(aiwfBin, stripped, "check")
	if err != nil { //coverage:ignore defensive: the binary already launched once against the working checkout above
		return nil, fmt.Errorf("running aiwf check on the ref-stripped copy: %w", err)
	}

	return classifyRefLessStability(label, removable,
		blockingSubjectsFrom(withRefs.Findings),
		blockingSubjectsFrom(withoutRefs.Findings)), nil
}

// blockingSubjectsFrom records, per subject, whether the run classified
// it blocking. A rule that fires more than once on one entity leaves the
// subject blocking if any of those findings blocks, so the disposition
// does not depend on which finding the renderer sorted last.
func blockingSubjectsFrom(findings []verbEnvelopeFinding) map[readPathSubject]bool {
	subjects := make(map[readPathSubject]bool, len(findings))
	for _, f := range findings {
		subj := readPathSubject{EntityID: entity.Canonicalize(f.EntityID), Code: f.Code}
		subjects[subj] = subjects[subj] || f.Severity == severityError
	}
	return subjects
}

// classifyRefLessStability reports one violation per subject whose
// blocking disposition flipped when removed was taken away.
//
// A subject present in only one run is not a flip. A finding that
// genuinely reports on another ref's existence — a cross-branch
// collision, say — disappears with that ref rather than changing its
// mind, and absence is not a claim here any more than it is when two
// surfaces are compared.
func classifyRefLessStability(label string, removed []string, withRefs, withoutRefs map[readPathSubject]bool) []Violation {
	subjects := make([]readPathSubject, 0, len(withRefs))
	for subj := range withRefs {
		if _, ok := withoutRefs[subj]; ok {
			subjects = append(subjects, subj)
		}
	}
	sort.Slice(subjects, func(i, j int) bool { return subjects[i].String() < subjects[j].String() })

	var violations []Violation
	for _, subj := range subjects {
		if withRefs[subj] == withoutRefs[subj] {
			continue
		}
		violations = append(violations, Violation{Message: fmt.Sprintf(
			"%s: %s is %s with the tree's own refs and %s without %s, so the verdict reports on the ref graph rather than on the tree",
			label, subj, blockingWord(withRefs[subj]), blockingWord(withoutRefs[subj]), strings.Join(removed, ", "),
		)})
	}
	return violations
}

// blockingWord renders a disposition for a violation message.
func blockingWord(blocking bool) string {
	if blocking {
		return "blocking"
	}
	return "non-blocking"
}

// removableRefs returns the refs in dir the tree does not need.
//
// Three are needed, and needed for reasons that have nothing to do with
// the resolution tiers this property must hold no model of: the branch
// HEAD is on, because removing it unmakes the checkout; its upstream,
// because that ref defines the provenance audit range; and the
// configured trunk ref, because the uniqueness check compares the
// working tree against it. Every other ref is one the tree does not
// need, whatever it happens to carry.
func removableRefs(dir string) ([]string, error) {
	needed, err := neededRefs(dir)
	if err != nil {
		return nil, err
	}
	all, err := gitRefNames(dir)
	if err != nil {
		return nil, err
	}

	var removable []string
	for _, ref := range all {
		if !needed[ref] {
			removable = append(removable, ref)
		}
	}
	sort.Strings(removable)
	return removable, nil
}

// neededRefs is the set removableRefs excludes.
func neededRefs(dir string) (map[string]bool, error) {
	needed := map[string]bool{}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		return nil, fmt.Errorf("reading the repository at %s: %w", dir, err)
	}

	// A detached HEAD, a branch with no upstream, and an absent trunk ref
	// each fail their query rather than erroring the property: none is a
	// repository state this harness cannot judge, and each simply
	// contributes nothing to the needed set.
	if head, err := gitOutput(dir, "symbolic-ref", "--quiet", "HEAD"); err == nil {
		needed[head] = true
	}
	if upstream, err := gitOutput(dir, "rev-parse", "--symbolic-full-name", "@{u}"); err == nil {
		needed[upstream] = true
	}

	// An absent aiwf.yaml is the state every scenario repo is in, and it
	// means the default trunk ref rather than no trunk — AllocateTrunkRef
	// answers that off a nil receiver, the same way the tree loader reads
	// it.
	cfg, err := config.Load(dir)
	if err != nil && !errors.Is(err, config.ErrNotFound) {
		return nil, fmt.Errorf("reading aiwf.yaml at %s: %w", dir, err)
	}
	trunk, _ := cfg.AllocateTrunkRef()
	needed[trunk] = true

	return needed, nil
}

// gitRefNames returns every ref in dir by full name.
func gitRefNames(dir string) ([]string, error) {
	out, err := gitOutput(dir, "for-each-ref", "--format=%(refname)")
	if err != nil {
		return nil, fmt.Errorf("listing refs in %s: %w", dir, err)
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// gitOutput runs a read-only git query in dir and returns its trimmed
// stdout.
func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// copyRepoWithoutRefs copies dir to a sibling temp directory and deletes
// refs there, returning the copy's path and a cleanup func.
//
// The copy is what keeps the property read-only with respect to the
// repository a scenario is still driving: deleting refs in place and
// restoring them afterwards would leave a killed or panicking run with a
// repository missing refs no later step could tell had been taken away.
func copyRepoWithoutRefs(dir string, refs []string) (path string, cleanup func(), err error) {
	stripped, err := os.MkdirTemp(filepath.Dir(dir), "refless-")
	if err != nil { //coverage:ignore defensive: MkdirTemp beside a directory this scenario is already writing to has no realistic failure mode
		return "", nil, fmt.Errorf("creating the ref-stripped copy's directory: %w", err)
	}
	cleanup = func() { _ = os.RemoveAll(stripped) }

	// CopyFS refuses to write over an existing entry, so it needs the
	// destination to not exist yet — MkdirTemp just made it.
	copyInto := filepath.Join(stripped, "repo")
	if err := os.CopyFS(copyInto, os.DirFS(dir)); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("copying %s for the ref-stripped run: %w", dir, err)
	}

	for _, ref := range refs {
		if _, err := gitOutput(copyInto, "update-ref", "-d", ref); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("deleting %s from the ref-stripped copy: %w", ref, err)
		}
	}
	return copyInto, cleanup, nil
}
