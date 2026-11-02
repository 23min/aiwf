---
id: G-0753
title: aiwf reallocate does not rewrite a contract's linked_adrs
status: open
priority: medium
---
## What's missing

`aiwf reallocate` renumbers an entity and then rewrites the references other entities hold to it in `rewriteEntityRefs` (`internal/verb/reallocate.go:491`). That function rewrites `parent`, `superseded_by`, `discovered_in`, `depends_on`, `supersedes`, `addressed_by` and `relates_to`, and not `linked_adrs`, the contract field naming the ADRs that motivate a contract, which `internal/entity/refs.go:60` declares as a reference. The verb's skill says it does: step 3 of `internal/skills/embedded/aiwf-reallocate/SKILL.md` (line 38) reads "Rewrites every reference field in every other entity's frontmatter".

Reproduced 2026-10-31 with aiwf built from `c06ffdaf9` (`vcs.modified=false`, the code `main` holds at `35d7cf109`), git 2.51.1, Linux, with the hooks `aiwf init` installs. A bare remote holds a fresh `aiwf init` tree, and two clones are made before either does any work. Alpha runs `aiwf add adr --title "Alpha decision"` on the default branch and pushes. Beta, on a branch, runs `aiwf add adr --title "Beta decision"`, which takes the same id, and `aiwf add contract --linked-adr ADR-0001 --title "Beta contract"`, linking its own ADR. After `git fetch`, Beta reallocates before merging the default branch in:

```
$ aiwf check
docs/adr/ADR-0001-beta-decision.md:2: error ids-unique/trunk-collision: id "ADR-0001" is allocated on this branch (docs/adr/ADR-0001-beta-decision.md) and on trunk (docs/adr/ADR-0001-alpha-decision.md) for different entities — hint: …
2 findings (1 errors, 1 warnings)
$ aiwf reallocate ADR-0001
aiwf reallocate ADR-0001 -> ADR-0002
$ aiwf check
refs-resolve (warning) × 1 — contract field "linked_adrs" references "ADR-0001", known only on refs/remotes/origin/main (not yet merged into this branch)
2 findings (0 errors, 2 warnings)
$ git push -q -u origin feature/B; echo "exit $?"
exit 0
$ git merge -q --no-edit origin/main; echo "exit $?"
exit 0
$ grep -A1 '^linked_adrs' work/contracts/C-0001-beta-contract/contract.md
linked_adrs:
    - ADR-0001
$ aiwf show ADR-0001
ADR-0001 · Alpha decision · status: proposed

  Referenced by (1):
    C-0001
$ aiwf check
ok — no findings
```

The first hint is cut; the second `aiwf check` and the hooked push also printed the scratch tree's `provenance-untrailered-scope-undefined` warning, left out above. With both ADRs merged into one tree instead, `aiwf reallocate docs/adr/ADR-0001-beta-decision.md` leaves the contract on `ADR-0001`, now Alpha's, and `aiwf check` prints `ok — no findings`.

Where nothing aiwf reads still holds the old id, as after reallocating an ADR with no collision, which the skill advises against (`SKILL.md:48`), `aiwf check` raises `refs-resolve/unresolved` on the contract at error severity, and the pre-push hook refuses the push.

## Why it matters

Resolving an ADR id collision with `aiwf reallocate`, whether on the branch before merging the default branch in or with both ADRs in one tree, leaves a contract that linked the renumbered ADR linking the one that kept the id. Once both ADRs are in the tree, `aiwf check` reports nothing, the push passes, and `aiwf show` lists the contract among the other ADR's referrers.

## Related

[G-0745](G-0745-aiwf-reallocate-rewrites-references-meant-for-the-other-colliding-entity.md) records `aiwf reallocate` rewriting the other reference fields and body prose for the referrers of both colliding entities, and reallocating on the branch before merging the default branch in as giving the right result; for `linked_adrs` that path gives the wrong one.
