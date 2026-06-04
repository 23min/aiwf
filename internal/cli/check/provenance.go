package check

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/scope"
	"github.com/23min/aiwf/internal/skills"
	"github.com/23min/aiwf/internal/tree"
)

// RunProvenanceCheck walks every commit reachable from HEAD that
// carries any `aiwf-*` trailer and runs the I2.5 standing rules
// against the result. It also runs the step-7b untrailered-entity-
// commit warning, scoped per the rules in ResolveUntrailedRange:
//   - --since <ref> on the verb wins.
//   - Otherwise `@{u}..HEAD` when an upstream is configured.
//   - Otherwise the audit is SKIPPED with a single
//     `provenance-untrailered-scope-undefined` advisory; the
//     fallback used to be "all of HEAD," which on long-lived
//     branches floods with warnings against commits already
//     merged in from trunk. See issue #5 sub-item 2.
//
// Returns a single concatenated finding slice; transport errors
// propagate.
//
// Why grep on `^aiwf-` for the standing rules: every rule is keyed
// on at least one aiwf trailer (actor, principal, scope-ends, etc.).
// Untrailered commits are handled by the separate step-7b audit pass,
// which uses a different filter (range scoped per ResolveUntrailedRange,
// no trailer grep).
//
// M-0159/AC-3: ackedSHAs is the gather-layer-computed map of
// retroactively-acknowledged commit SHAs (via
// check.WalkAcknowledgedSHAs called once at check.go::Run).
// Passed through to three rules that consume it from this gather
// (check.RunIsolationEscape, check.RunTrailerVerbUnknown,
// check.RunIDRenameUntrailered — the third added at M-0160/AC-4);
// the fourth consumer (check.FSMHistoryConsistent) is called
// directly from check.go::Run with the same map.
func RunProvenanceCheck(ctx context.Context, root string, t *tree.Tree, since string, registeredVerbs map[string]struct{}, ackedSHAs map[string]bool) ([]check.Finding, error) {
	if !cliutil.HasCommits(ctx, root) {
		return nil, nil
	}
	commits, err := readProvenanceCommits(ctx, root)
	if err != nil {
		return nil, err
	}
	findings := check.RunProvenance(commits, t)
	// M-0106: isolation-escape rule. Wire the git-backed BranchOracle
	// (built once per check invocation across the ritual-branch set)
	// to the kernel rule. An oracle-construction error is non-fatal —
	// the rule degrades to "unknown branch, silent" rather than
	// blocking the entire check pass, because branch-policing is one
	// rule among many and a failure here should not mask the others.
	//
	// M-0159/AC-6: cherry-pick gather-side wired via
	// check.WalkCherryPicks. Closes G-0202 — the parked gather that
	// left this call passing nil. The walker walks HEAD's reachable
	// history once per check invocation, applies the both-signals
	// contract (marker AND committer-vs-author email gap), and
	// returns the set of sovereign-human cherry-pick re-author SHAs
	// the rule should exempt. The rule's docstring at
	// internal/check/isolation_escape.go:67-78 pins the contract;
	// the walker is the gather-side derivation.
	if oracle, oErr := newGitBranchOracle(ctx, root); oErr == nil {
		cherryPicked := check.WalkCherryPicks(ctx, root)
		findings = append(findings, check.RunIsolationEscape(commits, oracle, cherryPicked, ackedSHAs)...)
		// M-0161/AC-5: force-push orphan detection. Walk each
		// ritual ref's reflog for non-fast-forward updates;
		// surface orphaned AI-actor commits as
		// isolation-escape-orphaned-ai-commit warnings. Composes
		// with M-0159/AC-3 acknowledge-illegal via the shared
		// ackedSHAs map.
		orphans := check.WalkOrphanedAICommits(ctx, root)
		findings = append(findings, check.RunOrphanedAICommits(orphans, ackedSHAs)...)
		// M-0161/AC-8 (G-0209): promote-on-wrong-branch detection.
		// Activating-promote commits (epic → active, milestone →
		// in_progress) must land on the parent branch per ADR-0010.
		// Build the expected-branch map from the loaded tree +
		// the configured trunk short-name (AC-1 composition);
		// missing parents stay out of the map → rule silent
		// (fail-shut on correctness).
		expectedBranches := expectedParentBranchesForPromote(t, cliutil.ConfiguredTrunkBranchShortName(root))
		findings = append(findings, check.RunPromoteOnWrongBranch(commits, expectedBranches, oracle, ackedSHAs)...)
		// M-0161/AC-3 + AC-4 + AC-5 / D-0019: surface oracle
		// coverage gaps. Per-ref failures emit isolation-escape
		// -oracle-failure (advisory; AC-3); the shallow-clone
		// capability emits the separate isolation-escape-
		// shallow-clone warning (AC-4); the reflog-disabled
		// capability rides AC-3's advisory per AC-5 body
		// line 350. All ride fail-shut on correctness — the
		// isolation-escape rule does not fire on commits whose
		// branch resolution lost coverage.
		for _, oe := range oracle.OracleErrors() {
			switch oe.Capability {
			case "shallow-clone":
				findings = append(findings, check.Finding{
					Code:     check.CodeIsolationEscapeShallowClone.ID,
					Severity: check.SeverityWarning,
					Message:  "isolation-escape coverage is incomplete: this repository is a shallow clone (rev-list returns commits only within the shallow boundary).",
					Hint:     "unshallow with `git fetch --unshallow`, or in CI use `actions/checkout@vN` with `fetch-depth: 0`; after unshallowing, re-run `aiwf check`.",
				})
			case "reflog-disabled":
				findings = append(findings, check.Finding{
					Code:     check.CodeIsolationEscapeOracleFailure.ID,
					Severity: check.SeverityWarning,
					Message:  "branch oracle could not run force-push orphan detection: core.logAllRefUpdates=false (reflog disabled); isolation-escape-orphaned-ai-commit coverage is incomplete",
					Hint: fmt.Sprintf(
						"%v",
						oe.Err,
					),
				})
			default:
				findings = append(findings, check.Finding{
					Code:     check.CodeIsolationEscapeOracleFailure.ID,
					Severity: check.SeverityWarning,
					Message: fmt.Sprintf(
						"branch oracle could not index ritual ref %q (%s); isolation-escape coverage is incomplete for commits reachable only via this ref",
						oe.Ref, oe.Capability,
					),
					Hint: fmt.Sprintf(
						"investigate ref %q: %v; the isolation-escape rule still polices every healthy ref",
						oe.Ref, oe.Err,
					),
				})
			}
		}
	}

	// M-0160/AC-4: id-rename-untrailered rule. Walk
	// merge-base(HEAD, trunk)..HEAD for commits that rename
	// id-bearing entity files without an aiwf-verb trailer in the
	// rename-class closed set (retitle/rename/reallocate/archive/
	// move). Catches the CLAUDE.md §"Id-collision resolution at
	// merge time" failure mode where an operator used inline
	// `git mv` instead of `aiwf reallocate`. ackedSHAs (M-0159/AC-3
	// helper-lift) exempts retroactively acknowledged commits.
	//
	// Wired BEFORE the untrailered-range resolution because this
	// rule is independent of the untrailered-audit scope — it uses
	// the trunk ref directly (same as the trunk-collision rule),
	// not @{u}..HEAD. Putting it after would mean it gets
	// short-circuited by the `provenance-untrailered-scope-undefined`
	// advisory on feature branches with no upstream.
	//
	// A nil/missing TrunkRef means the trunk-view computation was
	// skipped (no remotes configured, or trunk ref unresolved); the
	// rule degrades to "no cross-tree view, silent" just as the
	// trunk-collision rule does.
	if t != nil && t.TrunkRef != "" {
		renames := check.WalkUntrailedIDRenames(ctx, root, t.TrunkRef)
		findings = append(findings, check.RunIDRenameUntrailered(renames, ackedSHAs)...)
	}

	rangeArg, advisory, rErr := ResolveUntrailedRange(ctx, root, since)
	if rErr != nil {
		return nil, rErr
	}
	if advisory != nil {
		findings = append(findings, *advisory)
		return findings, nil
	}
	untrailed, uErr := ReadUntrailedCommits(ctx, root, rangeArg)
	if uErr != nil {
		return nil, uErr
	}
	findings = append(findings, check.RunUntrailedAudit(untrailed)...)
	// G-0150: warn on any `aiwf-verb:` trailer whose value is not in
	// the running binary's Cobra command tree, scoped to the same
	// `@{u}..HEAD` window as the untrailered audit. The chokepoint
	// catches fabricated trailers (e.g. an LLM-invented
	// `aiwf-verb: implement` on a hand-rolled `feat(...)` commit) at
	// pre-push; historical fabrications already on trunk stay out of
	// scope (rewriting their SHAs would invalidate the kernel's
	// addressed_by_commit references; the rule's job is to stop the
	// bleed, not retroactively flag what can't be repaired).
	//
	// G-0190: the ritual-verb allowlist is derived from the embedded
	// ritual snapshot via skills.RitualTrailerVerbs so it stays in
	// lock-step with what the rituals actually stamp. An extraction
	// error degrades to the empty set — the rule then flags ritual
	// stamps as unknown, which is preferable to silently allowing
	// arbitrary values.
	ritualVerbs, _ := skills.RitualTrailerVerbs()
	findings = append(findings, check.RunTrailerVerbUnknown(asScopeCommits(untrailed), registeredVerbs, ritualVerbs, ackedSHAs)...)
	return findings, nil
}

// asScopeCommits adapts the untrailered-audit's commit shape to the
// scope.Commit shape the trailer-verb rule reads. Both carry SHA +
// parsed trailers; the rule needs nothing else.
func asScopeCommits(in []check.UntrailedCommit) []scope.Commit {
	if len(in) == 0 {
		return nil
	}
	out := make([]scope.Commit, len(in))
	for i, c := range in {
		out[i] = scope.Commit{SHA: c.SHA, Trailers: c.Trailers}
	}
	return out
}

// ResolveUntrailedRange picks the `git log` range for the step-7b
// untrailered-entity audit. Three branches:
//
//  1. since != "" — the operator's explicit choice wins. Validates
//     the ref shape via `git rev-parse --verify`; an unrecognized
//     ref returns a usage-error advisory finding so the audit is
//     still skipped (rather than failing the whole check verb).
//  2. else, an upstream is configured — return `@{u}..HEAD`.
//  3. else — return ("", advisory) so the caller skips the scan
//     and surfaces the undefined-scope warning.
func ResolveUntrailedRange(ctx context.Context, root, since string) (string, *check.Finding, error) {
	if since != "" {
		// Verify the ref before trusting it: a typo in `--since`
		// would otherwise cause a `git log` failure that aborts
		// the whole `aiwf check` run.
		verify := exec.CommandContext(ctx, "git", "rev-parse", "--verify", since+"^{commit}")
		verify.Dir = root
		if err := verify.Run(); err != nil {
			advisory := &check.Finding{
				Code:     check.CodeProvenanceUntrailedScopeUndefined,
				Severity: check.SeverityWarning,
				Message: fmt.Sprintf("--since %q does not resolve to a commit; provenance audit skipped",
					since),
			}
			return "", advisory, nil
		}
		return since + "..HEAD", nil, nil
	}
	upstream := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	upstream.Dir = root
	if err := upstream.Run(); err == nil {
		return "@{u}..HEAD", nil, nil
	}
	advisory := &check.Finding{
		Code:     check.CodeProvenanceUntrailedScopeUndefined,
		Severity: check.SeverityWarning,
		Message:  "no upstream configured and no --since <ref>; provenance audit skipped",
	}
	return "", advisory, nil
}

// ReadUntrailedCommits returns the commits in rangeArg (e.g.
// `@{u}..HEAD`, or `<sha>..HEAD` from --since) along with their
// trailer set and the relative paths each commit touched.
//
// The range is supplied by the caller (ResolveUntrailedRange);
// ReadUntrailedCommits is purely the git-log invocation +
// parsing. An empty range (HEAD == @{u}) returns no commits,
// no findings.
//
// `-m --first-parent` walks the integration-branch view (G32):
// merge commits surface their introduced changes (against their
// first parent) so the audit pass sees entity-file paths brought
// in by `git merge`, while feature-branch commits not on
// first-parent ancestry are correctly excluded (those are the
// feature branch's own warning scope, not the integration
// branch's). Without `-m` the default is "show no diff for merge
// commits," which silently bypassed the audit for merges that
// absorbed entity-file changes from a feature branch.
func ReadUntrailedCommits(ctx context.Context, root, rangeArg string) ([]check.UntrailedCommit, error) {
	const fieldSep = "\x1f"
	const recSep = "\x1e"
	args := []string{
		"log",
		"--reverse",
		"-m",
		"--first-parent",
		rangeArg,
		"--name-only",
		"--pretty=tformat:" + recSep + "%H" + fieldSep + "%s" + fieldSep + "%(trailers:only=true,unfold=true)" + fieldSep,
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git log: %w\n%s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git log: %w", err)
	}
	return ParseUntrailedCommits(string(out)), nil
}

// ParseUntrailedCommits unpacks the multi-record stream produced by
// ReadUntrailedCommits. The format is:
//
//	<RS>{SHA}<US>{subject}<US>{trailers}<US>
//	{file1}
//	{file2}
//	...
//	<RS>{SHA}<US>...
//
// Trailers and file lists are both newline-delimited. Subject is
// the commit's first line, used for the squash-merge specialization
// (G31). Empty input (no unpushed commits) returns nil.
func ParseUntrailedCommits(s string) []check.UntrailedCommit {
	const fieldSep = "\x1f"
	const recSep = "\x1e"
	var out []check.UntrailedCommit
	for _, rec := range strings.Split(s, recSep) {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, fieldSep, 4)
		if len(parts) < 4 {
			continue
		}
		var paths []string
		for _, line := range strings.Split(parts[3], "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			paths = append(paths, line)
		}
		out = append(out, check.UntrailedCommit{
			SHA:      strings.TrimSpace(parts[0]),
			Subject:  strings.TrimSpace(parts[1]),
			Trailers: gitops.ParseTrailers(parts[2]),
			Paths:    paths,
		})
	}
	return out
}

// readProvenanceCommits returns every commit reachable from HEAD whose
// message carries any aiwf-* trailer, oldest-first. The output shape
// matches scope.Commit (SHA + parsed trailers) so check.RunProvenance
// stays I/O-free.
//
// The grep pattern is a basic regex anchored to the start of a
// trailer line. `git log -E` enables ERE so the anchor is honored.
func readProvenanceCommits(ctx context.Context, root string) ([]scope.Commit, error) {
	const fieldSep = "\x1f"
	const recSep = "\x1e\n"
	args := []string{
		"log",
		"--reverse",
		"-E",
		"--grep", "^aiwf-[a-z-]+:",
		"--pretty=tformat:%H" + fieldSep + "%(trailers:only=true,unfold=true)\x1e",
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git log: %w\n%s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git log: %w", err)
	}
	var commits []scope.Commit
	for _, rec := range strings.Split(string(out), recSep) {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, fieldSep, 2)
		if len(parts) < 2 {
			continue
		}
		commits = append(commits, scope.Commit{
			SHA:      strings.TrimSpace(parts[0]),
			Trailers: gitops.ParseTrailers(parts[1]),
		})
	}
	return commits, nil
}

// expectedParentBranchesForPromote builds the AC-8 map of
// entity id → expected parent branch name for the activating
// promote rule (M-0161/AC-8 / G-0209).
//
//   - Epics: expected branch is trunk (trunkShort, from
//     Config.TrunkBranchShortName() per AC-1).
//   - Milestones: expected branch is the parent epic's
//     ritual branch (epic/<parent-dir>); derived from the
//     entity's on-disk path which already carries the
//     E-NNNN-<slug> shape.
//   - Other kinds: not in the map → rule treats as "no
//     expectation, silent" (fail-shut per D-0019).
//
// Missing trunk name (empty trunkShort) means the trunk
// expectation is unresolvable; epic entries stay out of the
// map → epic-side promote-on-wrong-branch detection is
// silent. Same fail-shut posture as AC-1's empty-trunk
// path.
func expectedParentBranchesForPromote(t *tree.Tree, trunkShort string) map[string]string {
	if t == nil {
		return nil
	}
	expected := map[string]string{}
	if trunkShort != "" {
		for _, e := range t.ByKind(entity.KindEpic) {
			expected[e.ID] = trunkShort
		}
	}
	for _, m := range t.ByKind(entity.KindMilestone) {
		if m.Parent == "" {
			continue
		}
		parent := t.ByID(m.Parent)
		if parent == nil || parent.Kind != entity.KindEpic {
			continue // parent lookup failed → silent (fail-shut)
		}
		// Branch name follows the parent epic's on-disk dirname
		// (work/epics/E-NNNN-<slug>/epic.md → dirname
		// "E-NNNN-<slug>" → branch "epic/E-NNNN-<slug>"). Derive
		// from the parent's Path to honor whatever slug the
		// operator chose at creation time without recomputing
		// from title.
		parentDir := filepath.Base(filepath.Dir(parent.Path))
		if parentDir == "" || parentDir == "." {
			continue
		}
		expected[m.ID] = "epic/" + parentDir
	}
	return expected
}
