package policies

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// changelogDelta is one commit in the audited range that touched a
// shipped surface, together with the `aiwf-entity` trailer it carried.
// Entity is empty when the commit carried none.
type changelogDelta struct {
	SHA     string
	Subject string
	Entity  string
}

// uncitedEntity is one entity that shipped a surface change nothing
// under `[Unreleased]` cites, with the commits behind it. The commits
// ride along because the id alone does not tell an operator what to
// write: they have to see what changed before they can describe it.
type uncitedEntity struct {
	ID      string
	Commits []changelogDelta
}

// changelogAudit is the audit's verdict, split by what each half
// establishes rather than by how much it matters (D-0087).
//
// Uncited is a proven violation of the citation rule and fails the
// release: the audit read both sides and nothing names the entity.
//
// Unattributed is a shipped-surface commit carrying no `aiwf-entity`
// trailer. It establishes nothing about the changelog — with no entity
// named, the audit cannot say whether an entry covers it — so it is
// reported rather than blocking. Dropping it instead would let the audit
// under-report in silence, which is the failure it is least able to
// notice about itself.
type changelogAudit struct {
	Uncited      []uncitedEntity
	Unattributed []changelogDelta
}

// detectUncitedDeltas is the pure core. It rolls each delta's trailer up
// to the entity that owes a citation, then reports every such entity the
// `[Unreleased]` section does not name.
//
// A composite id is owed by its parent, so a delta attributed to
// `M-0001/AC-2` is owed by M-0001 before the rollup runs; ids compare
// canonicalized, since a narrower legacy width names the same entity
// (`E-01` is `E-0001`). Both happen here rather than inside owner:
// what counts as the same id is a question about the rule, answerable
// without loading a tree.
//
// A delta carrying no trailer names no entity, so it cannot be uncited.
// What becomes of it is a separate question with a separate answer.
//
// Order follows first appearance in the range, which is `git log`'s
// order, so the report reads newest-first like the log it came from.
func detectUncitedDeltas(deltas []changelogDelta, owner func(string) string, cited func(string) bool) changelogAudit {
	var order []string
	behind := map[string][]changelogDelta{}

	out := changelogAudit{}
	for _, d := range deltas {
		id := strings.TrimSpace(d.Entity)
		if id == "" {
			out.Unattributed = append(out.Unattributed, d)
			continue
		}
		owed := owner(entity.Canonicalize(entity.CompositeRoot(id)))
		if cited(owed) {
			continue
		}
		if _, seen := behind[owed]; !seen {
			order = append(order, owed)
		}
		behind[owed] = append(behind[owed], d)
	}

	out.Uncited = make([]uncitedEntity, 0, len(order))
	for _, id := range order {
		out.Uncited = append(out.Uncited, uncitedEntity{ID: id, Commits: behind[id]})
	}
	return out
}

// unattributedNotes renders the reported-but-not-blocking half, one line
// per commit the audit could not attribute.
//
// The subject rides along because the repair needs it: an operator has
// to work out which entity the commit belonged to before they can decide
// whether an entry already covers it, and a bare list of SHAs is one
// nobody acts on.
func unattributedNotes(a changelogAudit) []string {
	notes := make([]string, 0, len(a.Unattributed))
	for _, d := range a.Unattributed {
		notes = append(notes, fmt.Sprintf(
			"%s changed a shipped surface but carries no aiwf-entity trailer, so this audit cannot say whether [Unreleased] covers it: %s",
			d.SHA, d.Subject))
	}
	return notes
}

// uncitedViolations renders the blocking half for the policy harness.
// The Detail names the entity and the commits behind it, because the
// repair is to write an entry describing them and the operator cannot do
// that from an id.
func uncitedViolations(a changelogAudit) []Violation {
	out := make([]Violation, 0, len(a.Uncited))
	for _, u := range a.Uncited {
		out = append(out, Violation{
			Policy: "changelog-completeness",
			File:   "CHANGELOG.md",
			Detail: fmt.Sprintf("%s changed a shipped surface but nothing under [Unreleased] cites it; add an entry naming %s (%s).",
				u.ID, u.ID, summarizeCommits(u.Commits)),
		})
	}
	return out
}

// summarizeCommitsCap bounds how many commits a Detail lists. A release
// range can carry dozens behind one epic, and a finding an operator
// cannot read is one they skip.
const summarizeCommitsCap = 3

// summarizeCommits renders the commits behind an entity for a Detail,
// capped so a long range stays readable and counted so the cap does not
// hide how much is behind it.
func summarizeCommits(commits []changelogDelta) string {
	shown := commits[:min(len(commits), summarizeCommitsCap)]
	parts := make([]string, 0, len(shown))
	for _, c := range shown {
		parts = append(parts, c.SHA+" "+c.Subject)
	}
	rendered := strings.Join(parts, "; ")
	if len(commits) > len(shown) {
		rendered += fmt.Sprintf("; and %d more", len(commits)-len(shown))
	}
	return rendered
}

// changelogBaseEnv names the release to audit forward from, and its
// presence is also what turns the audit on.
//
// It is this audit's own variable rather than the coverage gate's,
// because the two run at different boundaries: the coverage gate asks
// about a branch's changes at every push, and this asks what a release
// is about to ship. A milestone's delta is legitimately absent from
// `[Unreleased]` until its epic wraps, so asking at push time would need
// an in-flight-epic exemption and asking at the release needs none.
//
// The audit lives here as a Go policy rather than as an `aiwf check`
// finding because the property is an aiwf-repo development invariant:
// the embedded trees exist only in this repo's source, so a consumer
// running the same audit would compare an empty range against its own
// changelog.
//
// Unlike the other policies in this package it has no
// `func(root) ([]Violation, error)` entry point, because its result has
// two halves and the runPolicy harness reads a violation slice as the
// whole verdict. TestPolicy_ChangelogCompleteness renders both.
const changelogBaseEnv = "AIWF_CHANGELOG_BASE"

// changelogBaseAuto is the AIWF_CHANGELOG_BASE value meaning "work the
// base out from history". It is a sentinel rather than the unset
// default, because unset has to keep meaning "do not run": the audit is
// a release-boundary check, and a default that resolved a base would
// turn it on for every `go test ./...` — where a milestone's delta is
// legitimately absent from `[Unreleased]` until its epic wraps.
//
// An explicit ref still overrides, which is what makes a past range
// auditable without moving a tag.
const changelogBaseAuto = "auto"

// tagsAtHEAD returns every tag pointing at HEAD.
//
// All of them matter to the base, and only the release-shaped one
// matters to the section — two different questions that would be one
// bug if answered together. `git describe` answers with a tag on HEAD
// whatever that tag is called, so *any* tag there empties the range and
// has to be excluded; but only a `v*` tag means the entries have moved
// out of `[Unreleased]`, so only that one changes which section is read.
// Conflating them would leave the audit vacuous whenever HEAD carried a
// tag of some other shape.
func tagsAtHEAD(root string) []string {
	cmd := exec.Command("git", "tag", "--points-at", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	return strings.Fields(string(out))
}

// releaseTagAtHEAD returns the release tag pointing at HEAD, or the
// empty string when HEAD carries none.
//
// Its presence is what tells the audit which shape it is in. Before the
// release commit HEAD carries no release tag, and the notes under test
// are `[Unreleased]`. On a pushed tag HEAD carries one, the entries have
// already moved into that version's heading, and that is the section to
// read instead.
func releaseTagAtHEAD(root string) string {
	var release string
	for _, t := range tagsAtHEAD(root) {
		if strings.HasPrefix(t, "v") {
			// More than one release tag on one commit is a re-tagging
			// accident rather than a shape to interpret. The last is the
			// highest by git's own ordering, which is the release a
			// reader would name.
			release = t
		}
	}
	return release
}

// changelogSectionFor returns the heading whose section holds the notes
// under test: the version being released when HEAD carries a release
// tag, and `[Unreleased]` otherwise.
func changelogSectionFor(releaseTag string) string {
	if releaseTag == "" {
		return unreleasedHeading
	}
	return releaseHeadingPrefix + strings.TrimPrefix(releaseTag, "v") + "]"
}

// resolveChangelogBase returns the release this commit's changes are
// measured against: the newest tag reachable from HEAD, excluding one
// that points at HEAD itself.
//
// Excluding HEAD's own tags is what makes the audit work at the moment
// it matters. On a pushed tag `git describe` answers with that same tag,
// so the range would be empty and the audit would report nothing however
// incomplete the notes are — a check that passes by construction exactly
// where it is wired to run. Every tag on HEAD is excluded, not just the
// release-shaped one, because `describe` does not care what a tag is
// called.
//
// Reachability rather than recency is the rest of it. `git describe`
// walks history, so a tag on a branch HEAD cannot reach is not a
// candidate however new it is or however high it sorts — the case a
// trunk-only test never produces, since on trunk the newest tag and the
// newest reachable tag are usually the same commit.
//
// With no tag at all the base is the root commit. A first release has no
// predecessor, and failing here would leave the audit unrunnable for
// exactly the release with the most undescribed history behind it.
func resolveChangelogBase(root string) (string, error) {
	args := []string{"describe", "--tags", "--abbrev=0"}
	for _, t := range tagsAtHEAD(root) {
		args = append(args, "--exclude="+t)
	}
	describe := exec.Command("git", args...)
	describe.Dir = root
	if out, err := describe.Output(); err == nil {
		if tag := strings.TrimSpace(string(out)); tag != "" {
			return tag, nil
		}
	}

	// `--max-parents=0` can list more than one root in a repo built by
	// grafting histories; the last is the earliest, which is the base
	// that leaves no commit unaudited.
	roots := exec.Command("git", "rev-list", "--max-parents=0", "HEAD")
	roots.Dir = root
	out, err := roots.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("resolving a base for %s: no reachable tag, and git rev-list failed: %w\n%s", root, err, out)
	}
	lines := strings.Fields(string(out))
	if len(lines) == 0 {
		//coverage:ignore git exits non-zero when HEAD does not resolve —
		// measured at 128 on an unborn HEAD — and when it does resolve the
		// walk always reaches a root, so exit 0 with no output cannot
		// happen. The guard stays because the alternative is indexing an
		// empty slice.
		return "", fmt.Errorf("resolving a base for %s: no reachable tag and no root commit", root)
	}
	return lines[len(lines)-1], nil
}

// changelogShippedDir holds the embedded trees that materialize into
// consumer repos via `aiwf init` / `aiwf update`. A change to their
// bytes reaches every consumer on upgrade, which is what makes it a
// release delta — and what makes a `docs(` prefix misleading here, since
// it means "nothing user-visible" in most repos and the opposite in
// this one.
//
// It is the whole directory rather than the `embedded*` subdirectories
// by name, so a tree added later is watched without an edit here. The
// Go sitting beside them is not shipped content; shipsSomething is
// where that line is drawn, and why.
const changelogShippedDir = "internal/skills"

// changelogFile is the release notes the audit reads, and the file a
// finding names.
const changelogFile = "CHANGELOG.md"

// unreleasedHeading opens the section holding deltas that have not
// shipped; releaseHeadingPrefix opens every section that has.
const (
	unreleasedHeading    = "## [Unreleased]"
	releaseHeadingPrefix = "## ["
)

// changelogSection returns the body under the named heading: the text
// between it and the next release heading, or the empty string when the
// file carries no such heading.
//
// Bounding at the next release heading is what keeps a citation in one
// release's notes from satisfying another's delta. Without the bound the
// whole file would count, and every entity ever released would read as
// cited — the audit would pass on any input.
//
// Headings count only at the start of a line and only outside a fenced
// code block. Both matter, and a changelog is exactly the document that
// breaks them: this file's own preamble names `[Unreleased]` in prose,
// and its entries quote heading shapes in fences to describe the release
// format. Unanchored, the preamble mention opens the section and the
// prose after it is read as notes; fence-blind, a quoted heading closes
// the section early and everything below it reads as uncited.
func changelogSection(doc, heading string) string {
	var body []string
	inFence, inSection := false, false
	for _, line := range strings.Split(doc, "\n") {
		if isFenceLine(line) {
			inFence = !inFence
			if inSection {
				body = append(body, line)
			}
			continue
		}
		if !inFence && strings.HasPrefix(line, releaseHeadingPrefix) {
			if inSection {
				break
			}
			if strings.HasPrefix(line, heading) {
				inSection = true
			}
			continue
		}
		if inSection {
			body = append(body, line)
		}
	}
	if !inSection {
		return ""
	}
	return strings.Join(body, "\n")
}

// isFenceLine reports whether the line opens or closes a fenced code
// block, in either of Markdown's two spellings.
func isFenceLine(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~")
}

// changelogCitedIn returns the predicate reporting whether an id is
// cited in the given section text.
//
// The match runs per id through entity.IDGrepAlternation rather than by
// pulling id-shaped tokens out of the prose. The widths that name an
// entity are then the ones the kernel says name it, instead of a second
// copy of the id grammar sitting here to drift from it — so an entry
// written at a legacy width still satisfies a delta whose trailer is
// canonical, and a token like `M-7` that names no milestone cannot
// satisfy anything.
//
// Citation is a rule this audit imposes, not a property it observes: an
// entry describing a change while naming no id is indistinguishable from
// no entry at all, so the finding says "nothing cites this" rather than
// "this is undocumented" (D-0087).
func changelogCitedIn(section string) func(string) bool {
	return func(id string) bool {
		alt := entity.IDGrepAlternation(id)
		if alt == "" {
			return false
		}
		// The alternation is a wrapped group of regex-quoted literals,
		// so it cannot fail to compile — the construction reallocate's
		// citation rewriter relies on for the same reason.
		return regexp.MustCompile(`\b` + alt + `\b`).MatchString(section)
	}
}

// changelogAuditFor is the testable IO core: it resolves the
// shipped-surface commits between baseRef and HEAD, reads what
// `[Unreleased]` cites, loads the tree for the rollup, and returns both
// halves of the verdict.
//
// Callers that gate a release take the blocking half through
// changelogViolations; the caller that reports takes both from here. One
// computation serves both, so the two can never disagree about the range
// they read.
func changelogAuditFor(root, baseRef string) (changelogAudit, error) {
	baseRef = strings.TrimSpace(baseRef)
	if baseRef == "" || baseRef == zeroSHA {
		return changelogAudit{}, nil
	}
	if baseRef == changelogBaseAuto {
		resolved, err := resolveChangelogBase(root)
		if err != nil {
			return changelogAudit{}, err
		}
		baseRef = resolved
	}
	deltas, err := shippedDeltasInRange(root, baseRef)
	if err != nil {
		return changelogAudit{}, err
	}
	if len(deltas) == 0 {
		return changelogAudit{}, nil
	}
	doc, err := os.ReadFile(filepath.Join(root, changelogFile))
	if err != nil {
		return changelogAudit{}, fmt.Errorf("reading %s in %s: %w", changelogFile, root, err)
	}
	owner, err := changelogOwner(root)
	if err != nil {
		return changelogAudit{}, err
	}
	section := changelogSection(string(doc), changelogSectionFor(releaseTagAtHEAD(root)))
	return detectUncitedDeltas(deltas, owner, changelogCitedIn(section)), nil
}

// changelogViolations is the release-gating half: the findings that fail
// a release, in the shape the runPolicy harness drives.
func changelogViolations(root, baseRef string) ([]Violation, error) {
	audit, err := changelogAuditFor(root, baseRef)
	if err != nil {
		return nil, err
	}
	return uncitedViolations(audit), nil
}

// shippedDeltasInRange returns one changelogDelta per non-merge commit
// between baseRef and HEAD that touched the shipped tree.
//
// `--no-merges` states the exclusion rather than establishing it: `git
// log --name-only` emits no file list for a merge without an explicit
// `--diff-merges`, so the path check below already drops one. Measured,
// that holds even for a merge whose resolution introduces content, and
// no `log.diffMerges` setting changes it. The flag is there so the
// exclusion does not rest on that default alone; no test can tell the
// two apart, and the merge test pins the outcome rather than the flag.
//
// One consequence is worth knowing rather than discovering. Content that
// exists only because of a merge resolution belongs to no non-merge
// commit, so this audit cannot see it — the same blind spot the
// shipped-ritual provenance backstop records under G-0602.
//
// Test files under the shipped tree are excluded: they are not
// materialized into a consumer repo, so a change to one ships nothing.
func shippedDeltasInRange(root, baseRef string) ([]changelogDelta, error) {
	format := skillEditRecSep + "%H" + skillEditFldSep + "%s" + skillEditFldSep +
		"%(trailers:key=" + gitops.TrailerEntity + ",valueonly,separator=" + skillEditFldSep + ")"
	cmd := exec.Command("git", "-c", "core.quotePath=false", "log",
		"--format="+format, "--name-only", "--no-merges",
		baseRef+"..HEAD", "--", changelogShippedDir)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		// The whole argv, not just the subcommand: an unresolvable base
		// ref is the realistic failure and is only visible there.
		return nil, fmt.Errorf("git log %s..HEAD in %s: %w\n%s", baseRef, root, err, out)
	}
	return parseShippedDeltaLog(string(out)), nil
}

// parseShippedDeltaLog turns the range scan into deltas, keeping one per
// commit that touched a shipping file. `--name-only` lists every path
// the commit touched under the pathspec, so a commit touching only test
// files under the shipped tree yields nothing.
func parseShippedDeltaLog(out string) []changelogDelta {
	var deltas []changelogDelta
	for _, rec := range strings.Split(out, skillEditRecSep) {
		lines := strings.Split(strings.TrimSpace(rec), "\n")
		if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
			continue
		}
		header := strings.Split(lines[0], skillEditFldSep)
		d := changelogDelta{SHA: shortSkillSHA(strings.TrimSpace(header[0]))}
		if len(header) > 1 {
			d.Subject = strings.TrimSpace(header[1])
		}
		// A commit may carry several aiwf-entity trailers; the first is
		// the owner, and a second would not change the verdict for it.
		if len(header) > 2 {
			d.Entity = strings.TrimSpace(header[2])
		}
		if !shipsSomething(lines[1:]) {
			continue
		}
		deltas = append(deltas, d)
	}
	return deltas
}

// shipsSomething reports whether any of the commit's listed paths is
// content a consumer receives.
//
// Go is the exception the pathspec cannot express. The watched tree
// holds two unlike things: the embedded directories, whose bytes are
// materialized into a consumer's repo, and the Go that materializes
// them. A change to the latter can change aiwf's behaviour, but that is
// a kernel-surface delta rather than a shipped-content one — the class
// this audit does not cover — and demanding a changelog entry for a
// rename in the kernel's own source blocks releases over nothing a
// consumer reads.
//
// Excluding Go loses no shipped content: every `go:embed` in the
// package names a path under an `embedded*` directory, none of which
// holds a `.go` file, so no shipped byte reaches a consumer through a
// Go source file.
func shipsSomething(paths []string) bool {
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" || strings.HasSuffix(p, ".go") {
			continue
		}
		return true
	}
	return false
}

// changelogOwner loads the tree once and returns the rollup: the entity
// that owes a citation for work attributed to a given id.
func changelogOwner(root string) (func(string) string, error) {
	t, _, err := tree.Load(context.Background(), root)
	if err != nil {
		return nil, fmt.Errorf("loading entity tree at %s: %w", root, err)
	}
	return changelogOwnerFor(t), nil
}

// changelogOwnerFor is the rollup over an already-loaded tree. A
// milestone's user-visible delta lands in its parent epic's entry, never
// its own, so a milestone rolls up and everything else owes its own
// citation.
//
// Resolution goes through the loader, which reaches archived entities.
// Most milestones that shipped in a past range are archived by the time
// a release is cut, so a scan of the active epic directories would leave
// their ids un-rolled and report each as an entity owing a citation of
// its own — naming an id no entry was ever expected to carry.
//
// `prior_ids` are consulted so an entity renumbered by `aiwf reallocate`
// keeps its older commit trailers rolling up: the verb rewrites
// references in the tree, but it cannot rewrite a commit message.
//
// An id the tree does not carry, and a milestone with no parent, are
// both returned unchanged. Neither is this audit's to diagnose — a stale
// trailer and a hand-edited milestone are `aiwf check`'s subjects — and
// returning an empty owner would report a finding naming no entity.
func changelogOwnerFor(t *tree.Tree) func(string) string {
	return func(id string) string {
		canon := entity.Canonicalize(id)
		e := t.ResolveByCurrentOrPriorID(canon)
		if e == nil || e.Kind != entity.KindMilestone || e.Parent == "" {
			return canon
		}
		return entity.Canonicalize(e.Parent)
	}
}
