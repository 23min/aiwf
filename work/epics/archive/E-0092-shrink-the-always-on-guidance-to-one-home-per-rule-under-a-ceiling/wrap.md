# Epic wrap — E-0092

**Date:** 2026-10-18
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0092-shrink-the-always-on-guidance-to-one-home-per-rule-under-a-ceiling

## Milestones delivered

- M-0333 — Fence project guidance while preserving managed updates (wrapped `342ba668a`)
- M-0350 — Ship the instruction-file fence in aiwf check (wrapped `7f7bb2bae`)
- M-0349 — Inventory every guidance rule and decide its home and form (wrapped `1c0f85113`)
- M-0334 — Write the rubric and record the baseline (wrapped `542b6be9a`)
- M-0336 — Remove duplicated guidance and move each rule to its audience's home (wrapped `a9a90f7df`)
- M-0335 — Route on-demand guidance through the project router and tighten what moves (wrapped `5ef304323`)
- M-0337 — Audit finding messages and check that guidance paths resolve (wrapped `4fe81f2c7`)
- M-0338 — Record the after observation and re-run the growth report (merged `8962b74de`)
- M-0361 — Separate Go-specific from general development rules and route by activity (merged `3622cf82e`)
- M-0339 — Rewrite the fragment and update the anchors policy: cancelled; shortening the shipped fragment's wording is left to a later epic.

## Changelog entry

### Added — E-0092: `aiwf check` fences commits that change handwritten instructions

`aiwf check` refuses, by default, a commit in the push range that changes handwritten instructions —
the root `CLAUDE.md` or `AGENTS.md` outside aiwf's generated blocks, `.guidance/project.md`, or a
handwritten document it links to — when the commit also carries unrelated files
(`instruction-commit-unrelated-file`), names no entity in an `aiwf-entity` trailer
(`instruction-commit-entity-missing`), or removes text without a disposition block
(`instruction-commit-disposition-missing`). The findings are errors, so the pre-push hook blocks the
push until the commit is split or amended. Set `guidance.fence_instruction_commits: false` in
`aiwf.yaml` to turn the rule off. Commits already on the configured trunk are not judged when that
ref is remote-tracking, as the default `refs/remotes/origin/main` is, and the rule runs only where
the branch has an upstream or `--since <ref>` is passed.

### Changed — E-0092: the shipped guidance states operating rules aiwf's own repository kept to itself

The always-on guidance `aiwf init` and `aiwf update` install now says:

- any deviation from a declared-sequence gate's list — a conflict, a finding, unexpected dirty
  state, an action not on the list — aborts the sequence and re-gates;
- a remote-branch delete is an outward action and never batches;
- a one-at-a-time decision list ends with "something else", the pick is confirmed in one line,
  and a human asking for "Q&A" or "interview me" asks for the same;
- whether a closely related issue is fixed in place or filed, what you noticed is surfaced, not
  silently deferred.

The `aiwf-promote` skill adds that a milestone's `tdd:` setting never waives the evidence an AC
needs and that the obligation binds at the promote, held at review. The `aiwf-reallocate` skill
adds how to rule out a stale branch before reallocating on a trunk collision, and now says the
new id is allocated against every local branch and remote-tracking ref as well as trunk and
that the verb rewrites mentions of the old id in body prose rather than reporting them.

The `id-rename-untrailered` finding and its hint point at that finding's entry in the
`aiwf-check` skill.

The `aiwf-check` skill documents the `commit-msg` hook's refusals and the
`provenance.refuse_coauthors` setting. The rest of E-0092 reorganizes aiwf's own development
guidance and changes nothing a consumer runs.

## Summary

E-0092 gave every guidance rule in this repository one home and one form, except the Go pack rules `go-conventions.md` restates (D-0105): the handwritten text a session loads at start, `CLAUDE.md` outside aiwf's blocks and the router, sits under a ceiling per host (1,288 words for Claude Code and 1,428 for Codex, from 9,647 and 9,776 at the frozen baseline), and the rest is read on demand through `.guidance/project.md` from nine documents under `.guidance/project/`. Operating rules moved into the shipped guidance and skills, and the fence that guards instruction-file commits ships in `aiwf check` for every repository. M-0338's after-observation found one lost effect, `verb-undo` on Claude, which rewording the router corrected, and no violation of a rule that held on every host; M-0361's found one more, the table rule, which stating it in `testing.md` restored. Claude still reaches the Go pack in fewer than half its Go tasks (G-0738).

## Success criteria

Nine of the epic's criteria are met. Two are recorded in the epic as not fully met: hosts reaching task-relevant guidance through the router (G-0738), and development guidance duplicating no language pack (D-0105). Guidance references and enforcement pointers resolve: `guidance-paths-resolve` holds the paths, and, checked once at this wrap, each of the 36 hyphenated names the development guidance cites in backticks resolves to a policy id, a check code, a refusal code, a make target, a skill, a stress scenario, the `commit-msg` hook `aiwf init` installs, the `mutate-hunt` workflow, `golangci-lint` or the `aiwf-entity` trailer key.

## Removal manifest

`git log --format=%B aba8c2732..HEAD | grep '^Disposition:'` lists the 171 disposition blocks since M-0336 began, each naming the inventory row it carries out: 59 relocated to an on-demand document, 50 deleted as a copy of a shipped, pack or gap record, 36 reworded, 24 replaced by a pointer and 2 deleted. The inventory test held every block to its row until this wrap deleted it.

## ADRs ratified

- ADR-0053 — Fence instruction-file edits in aiwf check by default

## Decisions captured

- D-0102 — No criterion is evidenced by a sentence pinned in development guidance
- D-0105 — Restate the depended-on Go pack rules in go-conventions.md

## Follow-ups carried forward

- G-0710 — An assertion outside an if condition passes the shipped-prose ban
- G-0711 — BlobReader reports a missing path containing whitespace as a parse error
- G-0730 — aiwf check runs a whole-repository git fsck once per hinted history finding
- G-0731 — aiwf check reads a shallow boundary commit as adding every file it carries
- G-0732 — The aiwf-add skill's step 1 says allocation scans only working tree and trunk
- G-0733 — The aiwf-add skill says body-prose-id resolves only the working tree and trunk
- G-0734 — Design docs and AllocateID's comment say allocation reads only two trees
- G-0735 — Comments, a test message and an ADR cite CLAUDE.md rules it does not carry
- G-0736 — Contribution templates ask for checks, documents and commands that do not exist
- G-0737 — The builder's Sonnet tier has no observation under the reduced guidance
- G-0738 — Claude reads go-conventions.md in Go tasks far more often than the Go pack
- G-0739 — AC verbs rewrite every same-numbered AC heading in a milestone body

## Doc findings

Clean. Over the 36 documents the epic added or changed, outside `work/` and the archive and other than `CHANGELOG.md` and the generated `ROADMAP.md`: no broken link or anchor, no documentation TODO, and every backticked `aiwf <verb>` resolves to a verb `aiwf --help` lists, except verbs the documents themselves call deferred or hypothetical.

## Handoff

At this wrap the inventory test (`internal/policies/e0092_inventory_test.go`, M-0337's audit test with it) and the observation harness (`scripts/e0092-observe/`) are deleted, and `guidance.enabled` is `true` again, so the next `aiwf update` may refresh the installed packs; whoever refreshes them compares the Go pack with the rules `go-conventions.md` restates (D-0105). The ceiling policy keeps holding each host's primed load at its ceiling. The run records of M-0334, M-0338 and M-0361 are archived outside git in `/workspaces/aiwf/.scratch/e0092-records/`. Open next: Claude's reach to the Go pack (G-0738), the AC-heading rewrite (G-0739), and shortening the shipped fragment's wording, which M-0339 left to a later epic.
